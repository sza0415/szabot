package agent

import (
	"context"
	"log"
	"time"

	"github.com/ziangsun/szabot/internal/bus"
	"github.com/ziangsun/szabot/internal/providers"
)

// Loop 是"对外"的那一层：把 bus 入站消息接住，调用 Runner，再把结果推回 bus。
//
// 设计要点：
//   - Loop 不知道 LLM 怎么调（那是 Runner 的事）；
//   - Loop 不知道消息从哪个平台来（那是 Channel 的事）；
//   - Loop 唯一的职责是"协调上下文 + 路由回复"。
//
// 上下文构造（system 恒在最前）：
//
//	system prompt(固定) + 历史(从 Store 按 SessionID 加载) + 本轮 user
//
// 一轮结束后，把"本轮 user"和"assistant 回复"追加回 Store，
// 从而同一 session 的后续请求都能带上此前的对话历史。
type Loop struct {
	Bus    *bus.MessageBus
	Runner *Runner
	// Store 按 SessionID 持久化对话历史（不含 system prompt）。
	// 为 nil 时退化为无历史模式（每条消息独立处理）。
	Store *SessionStore
	// SystemPrompt 是一段固定的系统提示，作为每轮对话的首条 system 消息。
	// 技能系统的 L1 摘要（以及 always 技能正文）就拼在这里注入 —— 它在进程
	// 启动时构建一次、全程不变，从而对 KV Cache 友好（动态内容只追加在末尾）。
	SystemPrompt string
}

// Start 起一个 goroutine 持续消费入站消息。
// ctx 取消时退出。
func (l *Loop) Start(ctx context.Context) {
	go l.run(ctx)
}

func (l *Loop) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case in, ok := <-l.Bus.Inbound():
			if !ok {
				return
			}
			l.handle(ctx, in)
		}
	}
}

func (l *Loop) handle(ctx context.Context, in bus.InboundMessage) {
	userMsg := providers.Message{Role: providers.RoleUser, Content: in.Text}

	// 1. 加载历史（不含 system prompt）。Store 为空时按无历史处理。
	var history []providers.Message
	if l.Store != nil {
		loaded, err := l.Store.Load(in.SessionID)
		if err != nil {
			log.Printf("[loop] load session=%s error: %v", in.SessionID, err)
			// 加载失败不致命：退化成本轮无历史，至少不阻断当前对话。
		} else {
			history = loaded
		}
	}

	// 2. 组装本次请求：system(恒在最前) + 历史 + 本轮 user。
	//    system 不进 Store，仅在发送前拼接，保证前缀稳定、对 KV Cache 友好。
	messages := make([]providers.Message, 0, len(history)+2)
	if l.SystemPrompt != "" {
		messages = append(messages, providers.Message{
			Role:    providers.RoleSystem,
			Content: l.SystemPrompt,
		})
	}
	messages = append(messages, history...)
	messages = append(messages, userMsg)

	// 流式：Runner 每吐一段正文增量，就作为一条 Delta 分片推回 bus，
	// channel 收到后边收边显示。Runner 内部会累积出完整回复作为返回值。
	onDelta := func(delta string) {
		if delta == "" {
			return
		}
		out := bus.OutboundMessage{
			SessionID: in.SessionID,
			ChannelID: in.ChannelID,
			Text:      delta,
			Delta:     true,
			Time:      time.Now(),
		}
		if err := l.Bus.PublishOutbound(ctx, out); err != nil {
			log.Printf("[loop] publish delta error: %v", err)
		}
	}

	reply, err := l.Runner.RunStream(ctx, messages, onDelta)
	if err != nil {
		log.Printf("[loop] runner error session=%s: %v", in.SessionID, err)
		return
	}

	// 3. 回写历史：把本轮 user 和 assistant 回复追加进 Store，
	//    下一轮 Load 就能带上它们。system prompt 不写入。
	if l.Store != nil {
		if err := l.Store.Append(in.SessionID,
			userMsg,
			providers.Message{Role: providers.RoleAssistant, Content: reply},
		); err != nil {
			log.Printf("[loop] append session=%s error: %v", in.SessionID, err)
		}
	}

	// 4. 结束标记：告诉 channel 本轮回复到此结束（换行、重打提示符等）。
	//    Text 为空，因为正文已由前面的 Delta 分片给完。
	done := bus.OutboundMessage{
		SessionID: in.SessionID,
		ChannelID: in.ChannelID,
		Done:      true,
		Time:      time.Now(),
	}
	if err := l.Bus.PublishOutbound(ctx, done); err != nil {
		log.Printf("[loop] publish done error: %v", err)
	}
}
