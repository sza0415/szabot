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
