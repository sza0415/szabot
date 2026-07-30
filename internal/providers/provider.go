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
	ID        string
	Name      string
	Arguments json.RawMessage
}

// Message is one item in a model conversation. Assistant messages retain their
// calls and tool messages retain the matching call ID so provider protocols can
// replay the required assistant → tool sequence.
type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
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
	// Chat 发起一次对话调用。
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
