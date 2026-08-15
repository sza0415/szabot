package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ziangsun/szabot/internal/providers"
)

// SessionStore 按 SessionID 持久化对话历史（M8）。
//
// 存储形态：每个 session 一个 jsonl 文件（<dir>/<sessionID>.jsonl），
// 每行是一条 providers.Message 的 JSON。选择"一 session 一文件 + 追加写"
// 是因为它天然匹配对话"只在末尾增长"的特性：Append 就是往文件尾追加一行，
// 无需读改写整个文件。
//
// 注意：这里只存"对话历史"（user / assistant / tool），不存 system prompt。
// system prompt 在启动时构建、全程不变，由 Loop 在每次请求时恒定拼在最前，
// 既避免把它反复写进磁盘，也保证前缀稳定、对 KV Cache 友好。
type SessionStore struct {
	dir string

	mu    sync.Mutex
	cache map[string][]providers.Message
}

// NewSessionStore 在 dir 下创建/使用会话目录。dir 不存在会被自动创建。
func NewSessionStore(dir string) (*SessionStore, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("agent: session store dir is empty")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("agent: create session dir %q: %w", dir, err)
	}
	return &SessionStore{
		dir:   dir,
		cache: make(map[string][]providers.Message),
	}, nil
}

// Load 返回某个 session 的完整历史（不含 system prompt）。
// 首次访问时从磁盘读入并缓存；之后走内存缓存，避免每轮都读盘。
// session 不存在时返回空切片（而非错误）——新会话就是一段空历史。
func (s *SessionStore) Load(sessionID string) ([]providers.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if history, ok := s.cache[sessionID]; ok {
		return cloneMessages(history), nil
	}

	history, err := s.readFile(sessionID)
	if err != nil {
		return nil, err
	}
	s.cache[sessionID] = history
	return cloneMessages(history), nil
}

// Append 把若干条消息追加到某个 session（内存缓存 + 磁盘各追加一份）。
// 典型调用：一轮结束后追加 [本轮 user, 本轮 assistant 回复]。
func (s *SessionStore) Append(sessionID string, messages ...providers.Message) error {
	if len(messages) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 确保缓存已从磁盘热身，避免缓存与文件不一致。
	if _, ok := s.cache[sessionID]; !ok {
		history, err := s.readFile(sessionID)
		if err != nil {
			return err
		}
		s.cache[sessionID] = history
	}

	if err := s.appendFile(sessionID, messages); err != nil {
		return err
	}
	s.cache[sessionID] = append(s.cache[sessionID], messages...)
	return nil
}

func (s *SessionStore) path(sessionID string) string {
	// 清洗 sessionID，避免路径穿越（如 "../x"）。
	safe := filepath.Base(filepath.Clean("/" + sessionID))
	if safe == "." || safe == "/" || safe == "" {
		safe = "default"
	}
	return filepath.Join(s.dir, safe+".jsonl")
}

func (s *SessionStore) readFile(sessionID string) ([]providers.Message, error) {
	f, err := os.Open(s.path(sessionID))
	if err != nil {
		if os.IsNotExist(err) {
			return []providers.Message{}, nil
		}
		return nil, fmt.Errorf("agent: open session %q: %w", sessionID, err)
	}
	defer f.Close()

	var history []providers.Message
	scanner := bufio.NewScanner(f)
	// 单条消息可能较长（工具结果/长回复），放宽单行上限到 1MB。
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg providers.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return nil, fmt.Errorf("agent: parse session %q: %w", sessionID, err)
		}
		history = append(history, msg)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("agent: read session %q: %w", sessionID, err)
	}
	return history, nil
}

func (s *SessionStore) appendFile(sessionID string, messages []providers.Message) error {
	f, err := os.OpenFile(s.path(sessionID), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("agent: open session %q for append: %w", sessionID, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("agent: marshal session message: %w", err)
		}
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("agent: write session %q: %w", sessionID, err)
		}
		if err := w.WriteByte('\n'); err != nil {
			return fmt.Errorf("agent: write session %q: %w", sessionID, err)
		}
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("agent: flush session %q: %w", sessionID, err)
	}
	// fsync 保证进程/机器异常时历史不丢。
	if err := f.Sync(); err != nil {
		return fmt.Errorf("agent: sync session %q: %w", sessionID, err)
	}
	return nil
}

func cloneMessages(src []providers.Message) []providers.Message {
	if len(src) == 0 {
		return []providers.Message{}
	}
	return append([]providers.Message(nil), src...)
}
