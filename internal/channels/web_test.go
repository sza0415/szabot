package channels

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ziangsun/szabot/internal/bus"
)

// newTestWeb 造一个已初始化 subscribers 的 WebChannel（不真正监听端口），
// 方便直接调 handler 与 deliver 做单元测试。
func newTestWeb(b *bus.MessageBus) *WebChannel {
	return &WebChannel{
		ID:          "web",
		Bus:         b,
		subscribers: make(map[string]map[*subscriber]struct{}),
	}
}

// TestHandleSendPublishesInbound 验证 POST /api/send 会把消息翻译成
// InboundMessage 推进 bus，且带上正确的 session/channel。
func TestHandleSendPublishesInbound(t *testing.T) {
	b := bus.New(16)
	w := newTestWeb(b)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	body, _ := json.Marshal(sendRequest{Session: "web:abc", Text: "你好"})
	req := httptest.NewRequest(http.MethodPost, "/api/send", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	w.handleSend(ctx)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	select {
	case in := <-b.Inbound():
		if in.ChannelID != "web" {
			t.Errorf("ChannelID = %q, want web", in.ChannelID)
		}
		if in.SessionID != "web:abc" {
			t.Errorf("SessionID = %q, want web:abc", in.SessionID)
		}
		if in.Text != "你好" {
			t.Errorf("Text = %q, want 你好", in.Text)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for inbound message")
	}
}

// TestHandleSendRejectsEmpty 验证空文本被拒。
func TestHandleSendRejectsEmpty(t *testing.T) {
	b := bus.New(16)
	w := newTestWeb(b)
	ctx := context.Background()

	body, _ := json.Marshal(sendRequest{Session: "s", Text: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/send", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	w.handleSend(ctx)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// TestDeliverRoutesBySession 验证出站消息只投给匹配 SessionID 的订阅者，
// 其他 session 的订阅者收不到——这是 Web 多连接场景的核心正确性。
func TestDeliverRoutesBySession(t *testing.T) {
	b := bus.New(16)
	w := newTestWeb(b)

	subA := &subscriber{sessionID: "A", events: make(chan bus.OutboundMessage, 4)}
	subB := &subscriber{sessionID: "B", events: make(chan bus.OutboundMessage, 4)}
	w.addSubscriber(subA)
	w.addSubscriber(subB)

	w.deliver(bus.OutboundMessage{ChannelID: "web", SessionID: "A", Text: "for-a", Delta: true})

	select {
	case out := <-subA.events:
		if out.Text != "for-a" {
			t.Errorf("subA got %q, want for-a", out.Text)
		}
	case <-time.After(time.Second):
		t.Fatal("subA should have received the message")
	}

	select {
	case out := <-subB.events:
		t.Fatalf("subB should NOT receive session A's message, got %q", out.Text)
	case <-time.After(50 * time.Millisecond):
		// 正确：B 收不到。
	}
}

// TestRemoveSubscriber 验证注销后不再投递。
func TestRemoveSubscriber(t *testing.T) {
	b := bus.New(16)
	w := newTestWeb(b)

	sub := &subscriber{sessionID: "A", events: make(chan bus.OutboundMessage, 4)}
	w.addSubscriber(sub)
	w.removeSubscriber(sub)

	w.deliver(bus.OutboundMessage{ChannelID: "web", SessionID: "A", Text: "x", Delta: true})

	select {
	case <-sub.events:
		t.Fatal("removed subscriber should not receive messages")
	case <-time.After(50 * time.Millisecond):
		// 正确。
	}
}

// TestDispatchStreamsToSSE 端到端验证：dispatch 从 bus 读到属于 web 的出站消息，
// 经由 SSE handler 以 data: 行推给"浏览器"。用 httptest server 起真实 HTTP。
func TestDispatchStreamsToSSE(t *testing.T) {
	b := bus.New(16)
	w := newTestWeb(b)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.dispatch(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/stream", w.handleStream)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// 起 SSE 连接。
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/stream?session=S", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("stream request error: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)

	// 先读到 ready 事件，确认订阅者已注册。
	waitForLine(t, reader, "event: ready")

	// 通过 bus 推一条属于 web 的 Delta，应该经 dispatch → SSE 出现在流里。
	_ = b.PublishOutbound(ctx, bus.OutboundMessage{
		ChannelID: "web", SessionID: "S", Text: "hello-sse", Delta: true,
	})

	line := waitForLine(t, reader, "hello-sse")
	if !strings.HasPrefix(line, "data: ") {
		t.Fatalf("expected data line, got %q", line)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &payload); err != nil {
		t.Fatalf("bad json payload %q: %v", line, err)
	}
	if payload["text"] != "hello-sse" || payload["delta"] != true {
		t.Fatalf("unexpected payload: %v", payload)
	}
}

// waitForLine 从 SSE 流里逐行读，直到某行包含 want 或超时。
func waitForLine(t *testing.T, r *bufio.Reader, want string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read stream error: %v", err)
		}
		line = strings.TrimRight(line, "\n")
		if strings.Contains(line, want) {
			return line
		}
	}
	t.Fatalf("timeout waiting for line containing %q", want)
	return ""
}
