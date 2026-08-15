package channels

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ziangsun/szabot/internal/bus"
)

// webIndex 把前端单页 HTML 直接嵌进二进制里，
// 这样 szabot 依然是"一个可执行文件、零外部资源"，跟设计宪法一致。
//
//go:embed web/index.html
var webIndex embed.FS

// WebChannel 是基于 HTTP 的 channel：浏览器通过它跟 agent 对话。
//
// 为什么用 SSE（Server-Sent Events）而不是 WebSocket？
//   - go.mod 不引第三方依赖，标准库没有 WebSocket，SSE 用 net/http 就能做；
//   - agent 的输出本就是"服务端单向流式推送"，SSE 天生贴合这个形态；
//   - 用户发消息走普通 POST 即可，不需要双向长连接。
//
// 关键难点——出站消息的分发（fan-out）：
//
//	bus.Outbound() 是一条被所有 channel 共享的 channel，多个消费者一起读会
//	互相抢消息。CLIChannel 独占终端时无所谓，但 Web 会同时有多个浏览器连接
//	（每个浏览器 = 一个 SessionID）。因此 WebChannel 用"单读多分发"模型：
//	  - 只有 dispatch() 这一个 goroutine 读 bus.Outbound()；
//	  - 它按 OutboundMessage.SessionID 找到对应的订阅者，把消息投递过去；
//	  - 每个 SSE 连接在建立时注册一个订阅者，断开时注销。
type WebChannel struct {
	// ID 就是 ChannelID，出站消息靠它区分归属。默认 "web"。
	ID string

	// Bus 是消息总线引用。
	Bus *bus.MessageBus

	// Addr 是 HTTP 监听地址，如 ":8080"。默认 ":8080"。
	Addr string

	// mu 保护 subscribers。
	mu sync.RWMutex
	// subscribers 按 SessionID 记录当前在线的 SSE 连接。
	// 一个 SessionID 理论上可能有多个连接（同一会话开了多个标签页），
	// 所以 value 是一个集合。
	subscribers map[string]map[*subscriber]struct{}
}

// subscriber 代表一个在线的 SSE 连接。
// events 是投递该连接的出站消息队列；dispatch 往里写，SSE handler 往外读。
type subscriber struct {
	sessionID string
	events    chan bus.OutboundMessage
}

// webSessionCookie 是给浏览器分配会话的 cookie 名。
const webSessionCookie = "szabot_session"

// sendRequest 是 POST /api/send 的请求体。
type sendRequest struct {
	Session string `json:"session"`
	Text    string `json:"text"`
}

// Start 起 HTTP 服务与出站分发 goroutine。
//
// 注意 ctx 取消时会优雅关停 HTTP server，避免端口泄漏。
func (w *WebChannel) Start(ctx context.Context) error {
	if w.ID == "" {
		w.ID = "web"
	}
	if w.Addr == "" {
		w.Addr = ":8080"
	}
	w.subscribers = make(map[string]map[*subscriber]struct{})

	// 出站分发：全局唯一的 goroutine 读 bus，按 SessionID 投递。
	go w.dispatch(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/", w.handleIndex)
	mux.HandleFunc("/api/send", w.handleSend(ctx))
	mux.HandleFunc("/api/stream", w.handleStream)

	server := &http.Server{Addr: w.Addr, Handler: mux}

	// ctx 取消 → 关 server。用一个短超时的独立 ctx 做 Shutdown。
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	// 在独立 goroutine 里跑，避免阻塞调用方（跟 CLIChannel.Start 语义一致）。
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[web] server error: %v", err)
		}
	}()

	return nil
}

// dispatch 是唯一读 bus.Outbound() 的 goroutine：按 SessionID 把出站消息
// 投递给对应的所有订阅者。找不到订阅者（连接已断/尚未建立）时直接丢弃。
func (w *WebChannel) dispatch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case out, ok := <-w.Bus.Outbound():
			if !ok {
				return
			}
			// 只处理属于自己的消息。
			if out.ChannelID != w.ID {
				continue
			}
			w.deliver(out)
		}
	}
}

// deliver 把一条出站消息投递给指定 session 的全部订阅者。
func (w *WebChannel) deliver(out bus.OutboundMessage) {
	w.mu.RLock()
	subs := w.subscribers[out.SessionID]
	targets := make([]*subscriber, 0, len(subs))
	for s := range subs {
		targets = append(targets, s)
	}
	w.mu.RUnlock()

	for _, s := range targets {
		select {
		case s.events <- out:
		default:
			// 订阅者的队列满了（前端消费不过来）就丢弃这一条，
			// 保证 dispatch 永不阻塞，不拖垮整个出站链路。
		}
	}
}

// addSubscriber / removeSubscriber 维护 session → 连接集合的映射。
func (w *WebChannel) addSubscriber(s *subscriber) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.subscribers[s.sessionID] == nil {
		w.subscribers[s.sessionID] = make(map[*subscriber]struct{})
	}
	w.subscribers[s.sessionID][s] = struct{}{}
}

func (w *WebChannel) removeSubscriber(s *subscriber) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if set := w.subscribers[s.sessionID]; set != nil {
		delete(set, s)
		if len(set) == 0 {
			delete(w.subscribers, s.sessionID)
		}
	}
}

// handleIndex 返回内嵌的前端页面。
func (w *WebChannel) handleIndex(rw http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(rw, r)
		return
	}
	data, err := webIndex.ReadFile("web/index.html")
	if err != nil {
		http.Error(rw, "index not found", http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = rw.Write(data)
}

// handleSend 接收浏览器发来的用户消息，翻译成 InboundMessage 推进 bus。
//
// 用闭包捕获 Start 的 ctx：请求处理需要在系统关停时能被取消，避免卡在
// PublishInbound 上。
func (w *WebChannel) handleSend(ctx context.Context) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req sendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(rw, "bad request", http.StatusBadRequest)
			return
		}
		if req.Text == "" {
			http.Error(rw, "empty text", http.StatusBadRequest)
			return
		}

		// session 优先取请求体，其次取 cookie，最后兜底分配一个。
		session := req.Session
		if session == "" {
			if c, err := r.Cookie(webSessionCookie); err == nil {
				session = c.Value
			}
		}
		if session == "" {
			session = newSessionID()
		}

		in := bus.InboundMessage{
			ChannelID: w.ID,
			SessionID: session,
			UserID:    session, // Web 场景没有独立用户体系，用 session 兜底。
			Text:      req.Text,
			Time:      time.Now(),
		}
		if err := w.Bus.PublishInbound(ctx, in); err != nil {
			http.Error(rw, "publish failed", http.StatusServiceUnavailable)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(map[string]string{"session": session})
	}
}

// handleStream 建立 SSE 长连接，把该 session 的出站消息实时推给浏览器。
//
// SSE 协议：响应头 Content-Type: text/event-stream，每条事件形如
//
//	data: {...json...}\n\n
//
// 我们把 OutboundMessage 里前端关心的字段（text/delta/done）序列化成 JSON 发出去。
func (w *WebChannel) handleStream(rw http.ResponseWriter, r *http.Request) {
	session := r.URL.Query().Get("session")
	if session == "" {
		http.Error(rw, "missing session", http.StatusBadRequest)
		return
	}

	flusher, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")

	sub := &subscriber{
		sessionID: session,
		// 缓冲给足，突发的流式增量不至于因为瞬时消费慢而被 deliver 丢弃。
		events: make(chan bus.OutboundMessage, 256),
	}
	w.addSubscriber(sub)
	defer w.removeSubscriber(sub)

	// 先发一个 ready 事件，让前端确认连接已就绪。
	fmt.Fprintf(rw, "event: ready\ndata: {\"session\":%q}\n\n", session)
	flusher.Flush()

	// 心跳：定期发注释行，避免中间代理把空闲连接掐断。
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			// 浏览器断开连接。
			return
		case <-heartbeat.C:
			fmt.Fprint(rw, ": keep-alive\n\n")
			flusher.Flush()
		case out := <-sub.events:
			kind := string(out.Kind)
			if kind == "" {
				kind = "answer"
			}
			payload, err := json.Marshal(map[string]any{
				"text":  out.Text,
				"kind":  kind,
				"delta": out.Delta,
				"done":  out.Done,
			})
			if err != nil {
				continue
			}
			fmt.Fprintf(rw, "data: %s\n\n", payload)
			flusher.Flush()
		}
	}
}

// newSessionID 生成一个基于时间戳的会话 ID。
// Web 场景对唯一性要求不高（本地单机为主），时间戳纳秒足够区分。
func newSessionID() string {
	return fmt.Sprintf("web:%d", time.Now().UnixNano())
}
