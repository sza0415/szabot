// Package providers 抽象了"和某个 LLM 厂商对话"这件事。
//
// 设计要点：
//   - AgentRunner 只依赖 Provider 接口，不直接调任何 SDK；
//   - 想换 OpenAI / DeepSeek / Anthropic / 本地 Ollama 时，
//     只需要新增一个实现，Runner 一行不用改。
package providers

import (
	"context"
	"encoding/json"
)

// Role 是对话消息的角色，沿用 OpenAI 兼容惯例。
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ToolDefinition describes one function the model is allowed to request.
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  json.RawMessage
}

// ToolCall is one function request returned by a model.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Message is one item in a model conversation. Assistant messages retain their
// calls and tool messages retain the matching call ID so provider protocols can
// replay the required assistant → tool sequence.
//
// JSON tags are explicit so that the on-disk session log (jsonl) has a stable,
// readable schema independent of Go field names.
type Message struct {
	Role       Role       `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"toolCalls,omitempty"`
	ToolCallID string     `json:"toolCallId,omitempty"`
}

// ChatRequest is one request to a model provider.
type ChatRequest struct {
	Model    string
	Messages []Message
	Tools    []ToolDefinition
}

// ChatResponse is a model reply. A non-empty ToolCalls list asks the Runner to
// execute local tools and submit their results in a follow-up request.
type ChatResponse struct {
	Content      string
	ToolCalls    []ToolCall
	FinishReason string
}

// Provider 是 LLM 厂商的统一接口。
type Provider interface {
	// Name 仅用于日志/调试。
	Name() string
	// Chat 发起一次对话调用（非流式，一次性拿到完整回复）。
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

// StreamChunk 是流式响应里的一个增量片段。
//
// 一次流式调用会回调多次：
//   - 文本增量：ContentDelta 非空（拼起来就是完整回复正文）；
//   - 工具调用：provider 把分片拼装完成后，通过 ToolCalls 一次性给出；
//   - 结束标记：Done=true，并带上 FinishReason。
type StreamChunk struct {
	ContentDelta string
	ToolCalls    []ToolCall
	Done         bool
	FinishReason string
}

// StreamingProvider 是"可选能力"：实现了它的 Provider 支持流式输出。
//
// 之所以做成独立接口而非塞进 Provider，是为了不强迫所有实现都支持流式
// （比如 EchoProvider 没必要）。调用方用类型断言探测：
//
//	if sp, ok := provider.(StreamingProvider); ok { 走流式 } else { 回退 Chat }
type StreamingProvider interface {
	Provider
	// ChatStream 发起一次流式对话。每收到一个增量就调用 onChunk 回调；
	// onChunk 返回错误则中止流式并把该错误返回。
	// 无论是否流式，最终都要返回一个累积好的完整 ChatResponse，
	// 供上层记录历史、判断 tool_calls 与停止条件。
	ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk) error) (ChatResponse, error)
}
