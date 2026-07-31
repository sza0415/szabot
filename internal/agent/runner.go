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
func (r *Runner) Run(ctx context.Context, messages []providers.Message) (string, error) {
	if r.Provider == nil {
		return "", fmt.Errorf("agent: provider is nil")
	}

	conversation := append([]providers.Message(nil), messages...)
	maxTurns := r.MaxToolTurns
	if maxTurns <= 0 {
		maxTurns = defaultMaxToolTurns
	}

	for turn := 0; turn < maxTurns; turn++ {
		response, err := r.Provider.Chat(ctx, providers.ChatRequest{
			Model:    r.Model,
			Messages: conversation,
			Tools:    providerToolDefinitions(r.Tools),
		})
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
