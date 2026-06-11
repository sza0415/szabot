// Package providers 抽象了"和某个 LLM 厂商对话"这件事。
//
// 设计要点：
//   - AgentRunner 只依赖 Provider 接口，不直接调任何 SDK；
//   - 想换 OpenAI / DeepSeek / Anthropic / 本地 Ollama 时，
//     只需要新增一个实现，Runner 一行不用改。
package providers

import "context"

// Role 是对话消息的角色，沿用 OpenAI 兼容惯例。
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message 是发给/收自 LLM 的一条消息。
// 第一阶段先只支持纯文本，后面要支持 tool_call / 多模态再扩展。
type Message struct {
	Role    Role
	Content string
}

// ChatRequest 是一次"调用 LLM"的输入。
type ChatRequest struct {
	Model    string
	Messages []Message
}

// ChatResponse 是 LLM 的回复。
// 第一阶段先只放 Content；后面接入 tool calling 再加字段。
type ChatResponse struct {
	Content string
}

// Provider 是 LLM 厂商的统一接口。
type Provider interface {
	// Name 仅用于日志/调试。
	Name() string
	// Chat 发起一次对话调用。
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
