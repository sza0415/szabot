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

// Run continues until the model returns a normal answer or the tool-call limit
// is reached. Tool errors are returned to the model as tool results so it can
// adjust its parameters or choose another capability.
//
// Run 是非流式入口，等价于 RunStream(ctx, messages, nil)。
func (r *Runner) Run(ctx context.Context, messages []providers.Message) (string, error) {
	return r.RunStream(ctx, messages, nil)
}

// RunStream 与 Run 相同，但会把模型正文的增量通过 onDelta 实时回调出去。
//
// 流式约定：
//   - onDelta 为 nil 时退化为非流式行为；
//   - 若 Provider 实现了 providers.StreamingProvider，则走真正的 SSE，
//     正文逐段回调；否则回退到一次性的 Chat，拿到完整正文后一次性回调。
//   - tool-call 中间轮的正文通常为空（模型此时只发工具调用），因此直接透传
//     增量不会污染最终答案；真正给用户看的就是最后一轮（无 tool_calls）的正文。
//
// 无论流式与否，函数最终都返回完整的答案字符串，语义与原 Run 一致，
// 从而 Loop 记录 session 历史、判断停止条件的逻辑完全不变。
func (r *Runner) RunStream(
	ctx context.Context,
	messages []providers.Message,
	onDelta func(string),
) (string, error) {
	if r.Provider == nil {
		return "", fmt.Errorf("agent: provider is nil")
	}

	conversation := append([]providers.Message(nil), messages...)
	maxTurns := r.MaxToolTurns
	if maxTurns <= 0 {
		maxTurns = defaultMaxToolTurns
	}

	for turn := 0; turn < maxTurns; turn++ {
		response, err := r.chatOnce(ctx, conversation, onDelta)
		if err != nil {
			return "", err
		}
		if len(response.ToolCalls) == 0 {
			return response.Content, nil
		}

		conversation = append(conversation, providers.Message{
			Role:      providers.RoleAssistant,
			Content:   response.Content,
			ToolCalls: response.ToolCalls,
		})
		for _, call := range response.ToolCalls {
			if strings.TrimSpace(call.ID) == "" {
				return "", fmt.Errorf("agent: provider returned a tool call without an ID")
			}

			result, err := r.Tools.Execute(ctx, call.Name, call.Arguments)
			if err != nil {
				result = "Error: " + err.Error()
			}
			conversation = append(conversation, providers.Message{
				Role:       providers.RoleTool,
				ToolCallID: call.ID,
				Content:    result,
			})
		}
	}

	return "", fmt.Errorf("agent: exceeded maximum tool turns (%d)", maxTurns)
}

// chatOnce 发起单轮模型调用。若 provider 支持流式且 onDelta 非 nil，走 SSE；
// 否则走一次性 Chat（并在需要时把完整正文一次性回调，保证调用方拿到统一体验）。
func (r *Runner) chatOnce(
	ctx context.Context,
	conversation []providers.Message,
	onDelta func(string),
) (providers.ChatResponse, error) {
	request := providers.ChatRequest{
		Model:    r.Model,
		Messages: conversation,
		Tools:    providerToolDefinitions(r.Tools),
	}

	streamer, ok := r.Provider.(providers.StreamingProvider)
	if ok && onDelta != nil {
		return streamer.ChatStream(ctx, request, func(chunk providers.StreamChunk) error {
			if chunk.ContentDelta != "" {
				onDelta(chunk.ContentDelta)
			}
			return nil
		})
	}

	response, err := r.Provider.Chat(ctx, request)
	if err != nil {
		return providers.ChatResponse{}, err
	}
	// 非流式回退：如果调用方想要增量但 provider 不支持，把完整正文当作
	// 一个"大增量"回调出去，让上层的流式展示逻辑无需分支处理。
	if onDelta != nil && len(response.ToolCalls) == 0 && response.Content != "" {
		onDelta(response.Content)
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
