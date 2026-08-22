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

	"github.com/ziangsun/szabot/internal/agent"
	"github.com/ziangsun/szabot/internal/bus"
	tracing "github.com/ziangsun/szabot/internal/trace"
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

	// Trace 是只读的 Run 轨迹查询器，供 Web Trace 工作台使用。
	Trace tracing.Reader
	// Snapshots 提供跨 Run 的任务摘要查询，包含没有完整 Trace 的中断 Run。
	Snapshots RunSnapshotReader

	// Addr 是 HTTP 监听地址，如 ":8080"。默认 ":8080"。
	Addr string

	// GracePeriod 是"最后一个 SSE 连接断开"后、真正判定客户端离开之前的宽限期。
	//
	// 为什么必须要有它：SSE/EventSource 的断开极其频繁且多为良性——手机锁屏、
	// 切后台、网络抖动、WiFi 切蜂窝、中间代理回收空闲连接……每一次都会触发
	// 一次断开，而前端 EventSource 会立刻用同一个 SessionID 自动重连。若"一断
	// 就取消"，用户锁屏几秒就会误杀正在跑的 agent 任务。因此断开后先等宽限期，
	// 期间有重连就撤销，到点仍无人才真正回调 OnDisconnect。默认 20s。
	GracePeriod time.Duration

	// OnDisconnect 在某 SessionID 的最后一个订阅者断开、且过宽限期仍无重连时
	// 被调用（带该 SessionID）。wiring 通常把它接到 Loop.CancelSession，从而
	// 取消该会话下游正在运行的 Runner/LLM 请求。为 nil 时不做任何取消（仅感知）。
	OnDisconnect func(sessionID string)

	// mu 保护 subscribers 与 graceTimers。
	mu sync.RWMutex
	// subscribers 按 SessionID 记录当前在线的 SSE 连接。
	// 一个 SessionID 理论上可能有多个连接（同一会话开了多个标签页），
	// 所以 value 是一个集合。
	subscribers map[string]map[*subscriber]struct{}
	// graceTimers 记录各 SessionID 正在倒计时的宽限 timer（断开后启动）。
	// 重连时据此撤销对应 timer，避免误取消。
	graceTimers map[string]*time.Timer
}

// defaultGracePeriod 是断连取消的默认宽限期：覆盖锁屏/网络抖动/刷新等常见
// 良性断开，同时不至于长到白烧太多算力。
const defaultGracePeriod = 20 * time.Second

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
	if w.GracePeriod <= 0 {
		w.GracePeriod = defaultGracePeriod
	}
	w.subscribers = make(map[string]map[*subscriber]struct{})
	w.graceTimers = make(map[string]*time.Timer)

	// 出站分发：全局唯一的 goroutine 读 bus，按 SessionID 投递。
	go w.dispatch(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/", w.handleIndex)
	mux.HandleFunc("/api/send", w.handleSend(ctx))
	mux.HandleFunc("/api/stream", w.handleStream)
	mux.HandleFunc("/api/traces", w.handleTraces)
	mux.HandleFunc("/api/traces/run", w.handleTraceRun)
	mux.HandleFunc("/api/runs", w.handleRuns)

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

	// 重连撤销：该 session 若有正在倒计时的宽限 timer（说明此前刚断开、
	// 正等着判定离开），此刻有连接接回来了，立即停掉 timer，取消判定。
	if t := w.graceTimers[s.sessionID]; t != nil {
		t.Stop()
		delete(w.graceTimers, s.sessionID)
	}
}

func (w *WebChannel) removeSubscriber(s *subscriber) {
	w.mu.Lock()
	defer w.mu.Unlock()
	set := w.subscribers[s.sessionID]
	if set == nil {
		return
	}
	delete(set, s)
	if len(set) > 0 {
		// 该 session 还有别的活连接（如多标签页），不触发任何取消。
		return
	}
	delete(w.subscribers, s.sessionID)

	// 最后一个订阅者也走了：启动宽限 timer，而非立即取消。
	// OnDisconnect 为 nil 时无需取消逻辑，直接跳过。
	if w.OnDisconnect == nil {
		return
	}
	grace := w.GracePeriod
	if grace <= 0 {
		grace = defaultGracePeriod
	}
	if w.graceTimers == nil {
		w.graceTimers = make(map[string]*time.Timer)
	}
	// 若已有 timer 在跑（理论上不该发生，防御性处理），先停掉再重置。
	if t := w.graceTimers[s.sessionID]; t != nil {
		t.Stop()
	}
	sessionID := s.sessionID
	w.graceTimers[sessionID] = time.AfterFunc(grace, func() {
		w.onGraceExpired(sessionID)
	})
}

// onGraceExpired 在宽限期到点时执行：**复查**该 session 是否真的仍无任何活
// 订阅者（防止 timer 快到点时刚好有人重连的竞态窗口），确认无人才回调
// OnDisconnect 触发取消。
func (w *WebChannel) onGraceExpired(sessionID string) {
	w.mu.Lock()
	// timer 已被撤销（重连）或被更晚的 timer 取代，则本次作废。
	// 注意：无法直接比对 timer 指针（闭包未捕获），但撤销时会 delete，
	// 因此"当前 graceTimers 里已无本 session 条目"即代表已被撤销。
	if _, pending := w.graceTimers[sessionID]; !pending {
		w.mu.Unlock()
		return
	}
	delete(w.graceTimers, sessionID)
	// 复查订阅者集合：到点这一刻若又有连接接入，则不取消。
	stillEmpty := len(w.subscribers[sessionID]) == 0
	w.mu.Unlock()

	if stillEmpty && w.OnDisconnect != nil {
		w.OnDisconnect(sessionID)
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

type traceRunView struct {
	RunID      string          `json:"run_id"`
	SessionID  string          `json:"session_id"`
	AgentID    string          `json:"agent_id"`
	Status     string          `json:"status,omitempty"`
	StartedAt  time.Time       `json:"started_at,omitempty"`
	FinishedAt time.Time       `json:"finished_at,omitempty"`
	EventCount int             `json:"event_count"`
	Events     []tracing.Event `json:"events,omitempty"`
}

type RunSnapshotReader interface {
	List(sessionID string) ([]agent.RunSnapshot, error)
	Load(runID string) (agent.RunSnapshot, error)
}

type runSummaryView struct {
	RunID        string            `json:"run_id"`
	SessionID    string            `json:"session_id"`
	AgentID      string            `json:"agent_id"`
	Status       agent.RunStatus   `json:"status"`
	StatusReason string            `json:"status_reason,omitempty"`
	Error        string            `json:"error,omitempty"`
	QueuedAt     time.Time         `json:"queued_at"`
	StartedAt    time.Time         `json:"started_at,omitempty"`
	FinishedAt   time.Time         `json:"finished_at,omitempty"`
	UpdatedAt    time.Time         `json:"updated_at"`
	ModelStatus  string            `json:"model_status,omitempty"`
	ToolStatuses map[string]string `json:"tool_statuses,omitempty"`
	ModelCalls   int               `json:"model_calls"`
	ToolCalls    int               `json:"tool_calls"`
	EventCount   int               `json:"event_count"`
	Events       []tracing.Event   `json:"events,omitempty"`
}

func writeJSON(rw http.ResponseWriter, value any) {
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(rw).Encode(value)
}

// handleTraces 返回一个 Session 下按 Run 分组的 Trace 摘要。
func (w *WebChannel) handleTraces(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(rw, "missing session", http.StatusBadRequest)
		return
	}
	if w.Trace == nil {
		http.Error(rw, "trace reader unavailable", http.StatusServiceUnavailable)
		return
	}
	events, err := w.Trace.ReadSession(sessionID)
	if err != nil {
		http.Error(rw, "read trace failed", http.StatusInternalServerError)
		return
	}
	writeJSON(rw, map[string]any{"session_id": sessionID, "runs": groupTraceRuns(events, false)})
}

// handleRuns returns snapshot-backed run summaries. Unlike /api/traces, this
// also includes interrupted runs that may not have a complete trace file.
func (w *WebChannel) handleRuns(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if w.Snapshots == nil {
		http.Error(rw, "run snapshot reader unavailable", http.StatusServiceUnavailable)
		return
	}
	sessionID := r.URL.Query().Get("session")
	statusFilter := r.URL.Query().Get("status")
	snapshots, err := w.Snapshots.List(sessionID)
	if err != nil {
		http.Error(rw, "list runs failed", http.StatusInternalServerError)
		return
	}
	runs := make([]runSummaryView, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if statusFilter != "" && string(snapshot.Status) != statusFilter {
			continue
		}
		runs = append(runs, w.summaryFromSnapshot(snapshot, false))
	}
	writeJSON(rw, map[string]any{"session_id": sessionID, "runs": runs})
}

// handleTraceRun 返回一个 Run 的完整事件，供右侧详情面板读取。
func (w *WebChannel) handleTraceRun(rw http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	runID := r.URL.Query().Get("run_id")
	if runID == "" {
		http.Error(rw, "missing run_id", http.StatusBadRequest)
		return
	}
	if w.Snapshots != nil {
		if snapshot, err := w.Snapshots.Load(runID); err == nil {
			writeJSON(rw, w.summaryFromSnapshot(snapshot, true))
			return
		}
	}
	if w.Trace != nil {
		events, err := w.Trace.ReadRun(runID)
		if err != nil {
			http.Error(rw, "read trace failed", http.StatusInternalServerError)
			return
		}
		if len(events) > 0 {
			runs := groupTraceRuns(events, true)
			writeJSON(rw, runs[0])
			return
		}
	}
	http.NotFound(rw, r)
}

func (w *WebChannel) summaryFromSnapshot(snapshot agent.RunSnapshot, includeEvents bool) runSummaryView {
	view := runSummaryView{
		RunID: snapshot.ID, SessionID: snapshot.SessionID, AgentID: snapshot.AgentID,
		Status: snapshot.Status, StatusReason: snapshot.StatusReason, Error: snapshot.Error,
		QueuedAt: snapshot.QueuedAt, StartedAt: snapshot.StartedAt, FinishedAt: snapshot.FinishedAt,
		UpdatedAt: snapshot.UpdatedAt, ToolStatuses: make(map[string]string),
		ModelCalls: snapshot.Usage.ModelCalls, ToolCalls: snapshot.Usage.ToolCalls,
	}
	if w.Trace == nil {
		return view
	}
	events, err := w.Trace.ReadRun(snapshot.ID)
	if err != nil {
		return view
	}
	view.EventCount = len(events)
	for _, event := range events {
		switch event.Type {
		case tracing.EventModelRequestStarted, tracing.EventModelStatusChanged:
			view.ModelStatus = event.Status
		case tracing.EventModelResponseFinished, tracing.EventModelRequestFailed:
			view.ModelStatus = event.Status
		case tracing.EventToolStatusChanged:
			if id, ok := event.Data["tool_call_id"].(string); ok && id != "" {
				view.ToolStatuses[id] = event.Status
			}
		case tracing.EventToolExecutionStarted:
			if id, ok := event.Data["tool_call_id"].(string); ok && id != "" {
				if _, exists := view.ToolStatuses[id]; !exists {
					view.ToolStatuses[id] = event.Status
				}
			}
		}
	}
	if !includeEvents {
		return view
	}
	view.Events = events
	return view
}

func groupTraceRuns(events []tracing.Event, includeEvents bool) []traceRunView {
	byRun := make(map[string]*traceRunView)
	order := make([]string, 0)
	for _, event := range events {
		run := byRun[event.RunID]
		if run == nil {
			run = &traceRunView{RunID: event.RunID, SessionID: event.SessionID, AgentID: event.AgentID}
			byRun[event.RunID] = run
			order = append(order, event.RunID)
		}
		run.EventCount++
		if event.Status != "" {
			run.Status = event.Status
		}
		if event.Type == tracing.EventRunStarted && run.StartedAt.IsZero() {
			run.StartedAt = event.Timestamp
		}
		if event.Type == tracing.EventRunFinished {
			run.FinishedAt = event.Timestamp
		}
		if includeEvents {
			run.Events = append(run.Events, event)
		}
	}
	result := make([]traceRunView, 0, len(order))
	for i := len(order) - 1; i >= 0; i-- {
		result = append(result, *byRun[order[i]])
	}
	return result
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
// 出站事件对齐 AG-UI 协议：每条事件形如
//
//	id: <TYPE>_<ts>
//	data: {"type":"<TYPE>",...}
//
// 内部扁平的 OutboundMessage 分片由 aguiTranslator 在此翻译成带生命周期的
// AG-UI 事件（TEXT_MESSAGE_*/TOOL_CALL_*/REASONING_MESSAGE_*/RUN_* 等）。
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

	// AG-UI 翻译器：每连接一个，负责把分片重建成带边界的 AG-UI 事件。
	translator := newAGUITranslator(session, &sseEmitter{w: rw, flusher: flusher})
	// 先发 SESSION 事件，让前端确认连接已就绪（替代旧的 event: ready）。
	if err := translator.start(); err != nil {
		return
	}

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
			if err := translator.handle(out); err != nil {
				// 写失败通常意味着连接已断，退出让 defer 注销订阅者。
				return
			}
		}
	}
}

// newSessionID 生成一个基于时间戳的会话 ID。
// Web 场景对唯一性要求不高（本地单机为主），时间戳纳秒足够区分。
func newSessionID() string {
	return fmt.Sprintf("web:%d", time.Now().UnixNano())
}
