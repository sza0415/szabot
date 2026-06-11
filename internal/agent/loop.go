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
// 第一阶段：还没有 session 存储，所以每条消息独立处理（无历史）。
// 等加上 SessionStore 之后，这里会按 SessionID 加载/保存历史。
type Loop struct {
	Bus    *bus.MessageBus
	Runner *Runner
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
	// 第一阶段：上下文就是用户这一句话本身。
	// 后面接 SessionStore 之后，这里会变成：
	//   history := store.Load(in.SessionID)
	//   history = append(history, userMsg)
	messages := []providers.Message{
		{Role: providers.RoleUser, Content: in.Text},
	}

	reply, err := l.Runner.Run(ctx, messages)
	if err != nil {
		log.Printf("[loop] runner error session=%s: %v", in.SessionID, err)
		return
	}

	// 关键：ChannelID 和 SessionID 都原样填回去，
	// 这两个 ID 是"回信地址"，channel 监听 outbound 时靠 ChannelID 决定要不要处理。
	out := bus.OutboundMessage{
		SessionID: in.SessionID,
		ChannelID: in.ChannelID,
		Text:      reply,
		Time:      time.Now(),
	}
	if err := l.Bus.PublishOutbound(ctx, out); err != nil {
		log.Printf("[loop] publish outbound error: %v", err)
	}
}
