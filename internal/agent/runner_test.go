package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ziangsun/szabot/internal/providers"
	"github.com/ziangsun/szabot/internal/tools"
)

type scriptedProvider struct {
	responses []providers.ChatResponse
	requests  []providers.ChatRequest
}

func (p *scriptedProvider) Name() string { return "scripted" }

func (p *scriptedProvider) Chat(_ context.Context, request providers.ChatRequest) (providers.ChatResponse, error) {
	p.requests = append(p.requests, request)
	response := p.responses[0]
	p.responses = p.responses[1:]
	return response, nil
}

// streamingScriptedProvider 在 scriptedProvider 基础上实现 StreamingProvider，
// 把脚本回复的正文按 rune 逐段回调，用于验证 RunStream 的增量透传。
type streamingScriptedProvider struct {
	scriptedProvider
}

func (p *streamingScriptedProvider) ChatStream(
	_ context.Context,
	request providers.ChatRequest,
	onChunk func(providers.StreamChunk) error,
) (providers.ChatResponse, error) {
	p.requests = append(p.requests, request)
	response := p.responses[0]
	p.responses = p.responses[1:]

	for _, r := range response.Content {
		if err := onChunk(providers.StreamChunk{ContentDelta: string(r)}); err != nil {
			return providers.ChatResponse{}, err
		}
	}
	if err := onChunk(providers.StreamChunk{
		Done:         true,
		ToolCalls:    response.ToolCalls,
		FinishReason: response.FinishReason,
	}); err != nil {
		return providers.ChatResponse{}, err
	}
	return response, nil
}

type echoTool struct{}

func (echoTool) Name() string        { return "echo_tool" }
func (echoTool) Description() string { return "Echo one value." }
func (echoTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"value":{"type":"string"}},"required":["value"]}`)
}
func (echoTool) Execute(_ context.Context, arguments json.RawMessage) (string, error) {
	var input struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(arguments, &input); err != nil {
		return "", err
	}
	return "tool result: " + input.Value, nil
}

func TestRunnerExecutesToolAndContinuesConversation(t *testing.T) {
	registry := tools.NewRegistry()
	if err := registry.Register(echoTool{}); err != nil {
		t.Fatal(err)
	}
	provider := &scriptedProvider{responses: []providers.ChatResponse{
		{
			ToolCalls: []providers.ToolCall{{
				ID:        "call_1",
				Name:      "echo_tool",
				Arguments: json.RawMessage(`{"value":"hello"}`),
			}},
		},
		{Content: "final answer"},
	}}
	runner := &Runner{Provider: provider, Model: "test", Tools: registry}

	answer, err := runner.Run(context.Background(), []providers.Message{{
		Role: providers.RoleUser, Content: "use the tool",
	}})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if answer != "final answer" {
		t.Fatalf("Run() answer = %q, want final answer", answer)
	}
	if len(provider.requests) != 2 {
		t.Fatalf("Chat calls = %d, want 2", len(provider.requests))
	}
	if len(provider.requests[0].Tools) != 1 || provider.requests[0].Tools[0].Name != "echo_tool" {
		t.Fatalf("first request tools = %#v, want echo_tool", provider.requests[0].Tools)
	}

	messages := provider.requests[1].Messages
	if len(messages) != 3 {
		t.Fatalf("second request messages = %d, want 3", len(messages))
	}
	if len(messages[1].ToolCalls) != 1 || messages[1].ToolCalls[0].ID != "call_1" {
		t.Fatalf("assistant tool calls = %#v, want call_1", messages[1].ToolCalls)
	}
	if messages[2].Role != providers.RoleTool || messages[2].ToolCallID != "call_1" || messages[2].Content != "tool result: hello" {
		t.Fatalf("tool result message = %#v", messages[2])
	}
}

// TestRunStreamDeliversDeltas 验证：provider 支持流式时，RunStream 会把最终
// 答案的正文按增量回调出去，且返回的完整答案与增量拼接一致。
func TestRunStreamDeliversDeltas(t *testing.T) {
	provider := &streamingScriptedProvider{
		scriptedProvider: scriptedProvider{responses: []providers.ChatResponse{
			{Content: "你好世界"},
		}},
	}
	runner := &Runner{Provider: provider, Model: "test", Tools: tools.NewRegistry()}

	var got string
	answer, err := runner.RunStream(context.Background(),
		[]providers.Message{{Role: providers.RoleUser, Content: "hi"}},
		func(delta string) { got += delta })
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if answer != "你好世界" {
		t.Fatalf("answer = %q, want 你好世界", answer)
	}
	if got != "你好世界" {
		t.Fatalf("streamed deltas = %q, want 你好世界", got)
	}
}

// TestRunStreamFallbackWhenNotStreaming 验证：provider 不支持流式时，
// RunStream 回退到 Chat，并把完整正文当作一个增量回调出去。
func TestRunStreamFallbackWhenNotStreaming(t *testing.T) {
	provider := &scriptedProvider{responses: []providers.ChatResponse{{Content: "plain"}}}
	runner := &Runner{Provider: provider, Model: "test", Tools: tools.NewRegistry()}

	var got string
	answer, err := runner.RunStream(context.Background(),
		[]providers.Message{{Role: providers.RoleUser, Content: "hi"}},
		func(delta string) { got += delta })
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if answer != "plain" || got != "plain" {
		t.Fatalf("answer=%q streamed=%q, want both plain", answer, got)
	}
}

// TestRunCollectCapturesReasoningAndToolMessages 是本次改动的核心测试：
// RunCollect 必须把推理过程、工具调用、工具结果都收进 RunResult.Messages，
// 并通过 StreamSink 把各类事件实时汇报出去，而不再只剩最终正文。
func TestRunCollectCapturesReasoningAndToolMessages(t *testing.T) {
	registry := tools.NewRegistry()
	if err := registry.Register(echoTool{}); err != nil {
		t.Fatal(err)
	}
	provider := &scriptedProvider{responses: []providers.ChatResponse{
		{
			Reasoning: "先想想需要调用工具",
			ToolCalls: []providers.ToolCall{{
				ID:        "call_1",
				Name:      "echo_tool",
				Arguments: json.RawMessage(`{"value":"hello"}`),
			}},
		},
		{Content: "final answer", Reasoning: "拿到结果后总结"},
	}}
	runner := &Runner{Provider: provider, Model: "test", Tools: registry}

	var (
		reasoning   string
		toolCalls   []string
		toolResults []string
	)
	result, err := runner.RunCollect(context.Background(),
		[]providers.Message{{Role: providers.RoleUser, Content: "use the tool"}},
		StreamSink{
			OnReasoningDelta: func(s string) { reasoning += s },
			OnToolCall:       func(c providers.ToolCall) { toolCalls = append(toolCalls, c.Name) },
			OnToolResult:     func(_ providers.ToolCall, r string) { toolResults = append(toolResults, r) },
		})
	if err != nil {
		t.Fatalf("RunCollect() error = %v", err)
	}

	if result.Answer != "final answer" {
		t.Fatalf("answer = %q, want final answer", result.Answer)
	}

	// 事件回调：两轮的推理都应汇报，工具调用/结果各一次。
	if reasoning != "先想想需要调用工具拿到结果后总结" {
		t.Fatalf("reasoning stream = %q", reasoning)
	}
	if len(toolCalls) != 1 || toolCalls[0] != "echo_tool" {
		t.Fatalf("tool calls = %#v, want [echo_tool]", toolCalls)
	}
	if len(toolResults) != 1 || toolResults[0] != "tool result: hello" {
		t.Fatalf("tool results = %#v", toolResults)
	}

	// 完整消息序列：assistant(带推理+tool_calls) → tool 结果 → assistant(最终答案)。
	if len(result.Messages) != 3 {
		t.Fatalf("result messages = %d, want 3: %#v", len(result.Messages), result.Messages)
	}
	first := result.Messages[0]
	if first.Role != providers.RoleAssistant || first.Reasoning != "先想想需要调用工具" ||
		len(first.ToolCalls) != 1 || first.ToolCalls[0].ID != "call_1" {
		t.Fatalf("messages[0] = %#v", first)
	}
	second := result.Messages[1]
	if second.Role != providers.RoleTool || second.ToolCallID != "call_1" || second.Content != "tool result: hello" {
		t.Fatalf("messages[1] = %#v", second)
	}
	third := result.Messages[2]
	if third.Role != providers.RoleAssistant || third.Content != "final answer" || third.Reasoning != "拿到结果后总结" {
		t.Fatalf("messages[2] = %#v", third)
	}
}
