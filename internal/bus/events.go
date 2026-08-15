// Package bus 定义系统中流转的"统一消息"格式。
//
// 设计要点：
//   - 不管消息从 CLI、Telegram 还是 Web 来，进入 bus 之前都被翻译成 InboundMessage；
//   - 不管要发回哪个平台，发出去之前都先变成 OutboundMessage；
//   - 这样 AgentLoop 只需要面对这两种统一类型，不用关心来源/去向。
package bus

import "time"

// InboundMessage 表示从某个 channel 进入系统的一条消息。
type InboundMessage struct {
	// SessionID 用于标识"这是哪一轮对话"，
	// 通常由 channel 决定（CLI 固定一个；Telegram 用 chat_id 等）。
	SessionID string

	// ChannelID 标识消息来自哪个 channel 实例，
	// AgentLoop 处理完后要按这个 ID 把回复送回去。
	ChannelID string

	// UserID 是平台内的用户标识（CLI 可以固定 "local"）。
	UserID string

	// Text 是用户的文本输入。后续要支持图片/文件再扩展即可。
	Text string

	// Time 是消息进入 bus 的时间。
	Time time.Time

	// Meta 用于放各 channel 的原生附加信息，AgentLoop 一般不读。
	Meta map[string]any
}

// OutboundMessage 表示由 agent 产生、要发回某个 channel 的消息。
//
// 流式支持：一段完整回复会被拆成"多条分片 + 一条结束标记"流过 bus：
//   - 分片消息：Delta=true，Text 是本次新增的一小段正文；
//   - 结束消息：Done=true，Text 通常为空（正文已由前面的分片给完）；
//   - 非流式/回退：也可以只发一条 Delta=false、Done=false 的完整消息，
//     channel 直接把它当作一整段回复处理即可（向后兼容）。
type OutboundMessage struct {
	SessionID string
	ChannelID string
	Text      string
	// Delta 为 true 表示这是一段增量（流式输出中的一小块）。
	Delta bool
	// Done 为 true 表示本轮回复到此结束（流式收尾标记）。
	Done bool
	Time time.Time
	Meta map[string]any
}
