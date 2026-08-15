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

	// 出站分片的统一发送器：按 Kind 区分正文 / 推理 / 工具调用 / 工具结果。
	// 都以 Delta=true 的分片形式流过 bus，channel 可据 Kind 分区渲染。
	emit := func(kind bus.OutboundKind, text string) {
		if text == "" {
			return
		}
		out := bus.OutboundMessage{
			SessionID: in.SessionID,
			ChannelID: in.ChannelID,
			Text:      text,
			Kind:      kind,
			Delta:     true,
			Time:      time.Now(),
		}
		if err := l.Bus.PublishOutbound(ctx, out); err != nil {
			log.Printf("[loop] publish %s error: %v", kindLabel(kind), err)
		}
	}

	// Runner 把一轮内部发生的每类事件都回调出来：
	//   - 正文增量：边收边显示，也是最终答案；
	//   - 推理增量：推理型模型的思考过程，单独一路推送供前端折叠展示；
	//   - 工具调用/结果：让用户看到 agent 正在"做什么"，而非只有最终文本。
	sink := newLoopSink(emit)

	result, err := l.Runner.RunCollect(ctx, messages, sink)
	if err != nil {
		log.Printf("[loop] runner error session=%s: %v", in.SessionID, err)
		return
	}

	// 3. 回写历史：把本轮 user 以及 Runner 产生的全部消息（含推理过程、
	//    工具调用与工具结果）追加进 Store。这样 session jsonl 不再只剩
	//    纯文本，而是完整记录了推理与工具调用轨迹，下一轮 Load 也能带上。
	if l.Store != nil {
		history := make([]providers.Message, 0, len(result.Messages)+1)
		history = append(history, userMsg)
		history = append(history, result.Messages...)
		if err := l.Store.Append(in.SessionID, history...); err != nil {
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

// newLoopSink 把统一的 emit 发送器适配成 Runner 需要的 StreamSink：
// 每类事件各走一条对应 Kind 的出站分片。工具调用/结果会被格式化成
// 简短可读的文本，供 channel 直接展示。
func newLoopSink(emit func(bus.OutboundKind, string)) StreamSink {
	return StreamSink{
		OnContentDelta: func(delta string) { emit(bus.KindAnswer, delta) },
		OnReasoningDelta: func(delta string) {
			emit(bus.KindReasoning, delta)
		},
		OnToolCall: func(call providers.ToolCall) {
			emit(bus.KindToolCall, formatToolCall(call))
		},
		OnToolResult: func(call providers.ToolCall, result string) {
			emit(bus.KindToolResult, formatToolResult(call, result))
		},
	}
}

// formatToolCall 把一次工具调用渲染成 "name(arguments)" 形式。
func formatToolCall(call providers.ToolCall) string {
	args := string(call.Arguments)
	if args == "" || args == "null" {
		return call.Name + "()"
	}
	return call.Name + "(" + args + ")"
}

// formatToolResult 把一次工具结果渲染成 "name -> result" 形式，
// 过长时截断，避免刷屏。
func formatToolResult(call providers.ToolCall, result string) string {
	const maxResultRunes = 500
	trimmed := result
	if r := []rune(result); len(r) > maxResultRunes {
		trimmed = string(r[:maxResultRunes]) + "…(truncated)"
	}
	return call.Name + " -> " + trimmed
}

// kindLabel 给日志用的可读标签。
func kindLabel(kind bus.OutboundKind) string {
	switch kind {
	case bus.KindReasoning:
		return "reasoning"
	case bus.KindToolCall:
		return "tool_call"
	case bus.KindToolResult:
		return "tool_result"
	default:
		return "delta"
	}
}
