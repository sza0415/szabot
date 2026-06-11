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

	"github.com/ziangsun/szabot/internal/providers"
)

// Runner 负责"跟 LLM 对话"这件事。
type Runner struct {
	Provider providers.Provider
	Model    string
}

// Run 接收一段已经拼好的对话历史，返回 LLM 的最终回复文本。
//
// 第一阶段实现：直接转发给 Provider，拿到 content 就返回。
// 后续会演进为：
//  1. 调 Provider；
//  2. 如果回复包含 tool call → 执行 tool → 把结果塞回 messages → 回到第 1 步；
//  3. 直到 Provider 给出 final answer 才返回。
func (r *Runner) Run(ctx context.Context, messages []providers.Message) (string, error) {
	resp, err := r.Provider.Chat(ctx, providers.ChatRequest{
		Model:    r.Model,
		Messages: messages,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}
