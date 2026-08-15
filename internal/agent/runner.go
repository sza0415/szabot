// Package agent 实现 szabot 的核心循环。
//
// 这里有两个角色：
//   - Runner：对内（朝 LLM）。负责跟 Provider 来回打交道、（将来）执行 tool call、判断停止条件。
//   - Loop：对外（朝 channel）。负责消费 bus 入站消息、加载/保存 session、把回复推回 bus。
//
// 第一阶段的 Runner 极度简化：单轮调用，不做工具、不做多轮。
// 等接入真实 LLM 和 tool 之后，这里会演进为真正的"循环"。
package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/ziangsun/szabot/internal/providers"
	"github.com/ziangsun/szabot/internal/tools"
)

const defaultMaxToolTurns = 12

// Runner coordinates a model conversation and the explicit local tool allowlist.
type Runner struct {
	Provider     providers.Provider
	Model        string
	Tools        *tools.Registry
	MaxToolTurns int
}

// StreamSink 是 Runner 向上层实时汇报"一轮对话内部发生了什么"的回调集合。
//
// 设计动机：此前 Runner 只把最终正文按增量吐给 onDelta，推理过程与工具调用
// 在 Runner 内部被消化后就丢失了，既无法展示也无法落盘。StreamSink 把这些
// 中间事件也暴露出来，让 Loop 能分类推送到 channel 并写入 session。
//
// 所有回调均可为 nil，缺省即"不关心该类事件"。
type StreamSink struct {
	// OnContentDelta 接收面向用户的正文增量（拼起来即最终答案）。
	OnContentDelta func(string)
	// OnReasoningDelta 接收推理型模型的思考过程增量（普通模型永不触发）。
	OnReasoningDelta func(string)
	// OnToolCall 在模型请求某次工具调用时触发（执行前）。
	OnToolCall func(providers.ToolCall)
	// OnToolResult 在某次工具调用执行完成后触发，result 为其结果/错误文本。
	OnToolResult func(call providers.ToolCall, result string)
}

// RunResult 汇总一轮 Run 的产物。
//
//   - Answer 是面向用户的最终正文（与旧 Run 的返回字符串语义一致）；
//   - Messages 是本轮**新增**的全部对话消息，按发生顺序排列：
//     每个 tool-call 轮的 assistant 消息（含 Reasoning 与 ToolCalls）、
//     对应的 tool 结果消息，以及最终那条 assistant 正文消息。
//     Loop 应把 Messages 整体追加进 session，从而推理过程与工具调用
//     都被完整持久化，而不再只剩一条最终正文。
type RunResult struct {
	Answer   string
	Messages []providers.Message
}

// Run 是非流式入口，等价于 RunStream(ctx, messages, nil)。
func (r *Runner) Run(ctx context.Context, messages []providers.Message) (string, error) {
	return r.RunStream(ctx, messages, nil)
}

// RunStream 与 Run 相同，但会把模型正文的增量通过 onDelta 实时回调出去。
//
// 它是 RunCollect 的兼容封装：只关心正文增量、只返回最终答案字符串，
// 供既有调用方/测试沿用旧签名。需要推理过程、工具调用事件或完整消息
// 序列的调用方（如 Loop）应改用 RunCollect。
func (r *Runner) RunStream(
	ctx context.Context,
	messages []providers.Message,
	onDelta func(string),
) (string, error) {
	result, err := r.RunCollect(ctx, messages, StreamSink{OnContentDelta: onDelta})
	if err != nil {
		return "", err
	}
	return result.Answer, nil
}

// RunCollect 是真正的对话引擎：持续调用 Provider，执行工具，直到模型给出
// 一条不含 tool_calls 的正常回复，或达到工具轮数上限。
//
// 与旧实现的关键区别：它不再"用完即弃"中间过程，而是把本轮新增的每一条
// 消息（含推理过程、工具调用、工具结果）都收进 RunResult.Messages，并通过
// sink 把推理增量、工具调用/结果实时汇报出去。这样上层既能完整落盘，也能
// 分类渲染。
//
// 流式约定与旧版一致：
//   - 若 Provider 实现 StreamingProvider 且 sink 有正文/推理回调，走真正的 SSE；
//   - 否则回退到一次性 Chat，再把完整正文/推理当作单个增量回调出去。
func (r *Runner) RunCollect(
	ctx context.Context,
	messages []providers.Message,
	sink StreamSink,
) (RunResult, error) {
	if r.Provider == nil {
		return RunResult{}, fmt.Errorf("agent: provider is nil")
	}

	conversation := append([]providers.Message(nil), messages...)
	// produced 只收集"本轮新增"的消息，供上层追加进 session。
	produced := make([]providers.Message, 0, 4)

	maxTurns := r.MaxToolTurns
	if maxTurns <= 0 {
		maxTurns = defaultMaxToolTurns
	}

	for turn := 0; turn < maxTurns; turn++ {
		response, err := r.chatOnce(ctx, conversation, sink)
		if err != nil {
			return RunResult{}, err
		}

		// 无工具调用 = 本轮的最终答案。记录这条 assistant 正文（含推理）后收尾。
		if len(response.ToolCalls) == 0 {
			answer := providers.Message{
				Role:      providers.RoleAssistant,
				Content:   response.Content,
				Reasoning: response.Reasoning,
			}
			produced = append(produced, answer)
			return RunResult{Answer: response.Content, Messages: produced}, nil
		}

		// 有工具调用：先记录这条 assistant 消息（可能带思考过程 + tool_calls），
		// 再逐个执行工具、记录结果。这些都会进入 produced 从而被持久化。
		assistant := providers.Message{
			Role:      providers.RoleAssistant,
			Content:   response.Content,
			Reasoning: response.Reasoning,
			ToolCalls: response.ToolCalls,
		}
		conversation = append(conversation, assistant)
		produced = append(produced, assistant)

		for _, call := range response.ToolCalls {
			if strings.TrimSpace(call.ID) == "" {
				return RunResult{}, fmt.Errorf("agent: provider returned a tool call without an ID")
			}
			if sink.OnToolCall != nil {
				sink.OnToolCall(call)
			}

			result, err := r.Tools.Execute(ctx, call.Name, call.Arguments)
			if err != nil {
				result = "Error: " + err.Error()
			}
			if sink.OnToolResult != nil {
				sink.OnToolResult(call, result)
			}

			toolMsg := providers.Message{
				Role:       providers.RoleTool,
				ToolCallID: call.ID,
				Content:    result,
			}
			conversation = append(conversation, toolMsg)
			produced = append(produced, toolMsg)
		}
	}

	return RunResult{}, fmt.Errorf("agent: exceeded maximum tool turns (%d)", maxTurns)
}

// chatOnce 发起单轮模型调用。若 provider 支持流式且 sink 关心正文/推理增量，
// 走 SSE；否则走一次性 Chat（并在需要时把完整正文/推理一次性回调，保证调用方
// 拿到统一体验）。
func (r *Runner) chatOnce(
	ctx context.Context,
	conversation []providers.Message,
	sink StreamSink,
) (providers.ChatResponse, error) {
	request := providers.ChatRequest{
		Model:    r.Model,
		Messages: conversation,
		Tools:    providerToolDefinitions(r.Tools),
	}

	streamer, ok := r.Provider.(providers.StreamingProvider)
	wantsStream := sink.OnContentDelta != nil || sink.OnReasoningDelta != nil
	if ok && wantsStream {
		return streamer.ChatStream(ctx, request, func(chunk providers.StreamChunk) error {
			if chunk.ReasoningDelta != "" && sink.OnReasoningDelta != nil {
				sink.OnReasoningDelta(chunk.ReasoningDelta)
			}
			if chunk.ContentDelta != "" && sink.OnContentDelta != nil {
				sink.OnContentDelta(chunk.ContentDelta)
			}
			return nil
		})
	}

	response, err := r.Provider.Chat(ctx, request)
	if err != nil {
		return providers.ChatResponse{}, err
	}
	// 非流式回退：如果调用方想要增量但 provider 不支持，把完整推理/正文
	// 当作一个"大增量"回调出去，让上层的流式展示逻辑无需分支处理。
	//
	// 推理过程即便伴随 tool_calls 也应展示，故 reasoning 无条件回调；
	// 正文只在无 tool_calls 时回调，避免把中间轮通常为空/无意义的 content
	// 混进最终答案（保持旧语义）。
	if response.Reasoning != "" && sink.OnReasoningDelta != nil {
		sink.OnReasoningDelta(response.Reasoning)
	}
	if len(response.ToolCalls) == 0 && response.Content != "" && sink.OnContentDelta != nil {
		sink.OnContentDelta(response.Content)
	}
	return response, nil
}

func providerToolDefinitions(registry *tools.Registry) []providers.ToolDefinition {
	definitions := registry.Definitions()
	if len(definitions) == 0 {
		return nil
	}

	result := make([]providers.ToolDefinition, 0, len(definitions))
	for _, definition := range definitions {
		result = append(result, providers.ToolDefinition{
			Name:        definition.Name,
			Description: definition.Description,
			Parameters:  definition.Parameters,
		})
	}
	return result
}
