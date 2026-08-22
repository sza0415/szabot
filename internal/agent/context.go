package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ziangsun/szabot/internal/providers"
)

// ContextManager builds a bounded context while keeping raw conversation intact.
type ContextManager struct {
	Store            *SessionStore
	Provider         providers.Provider
	Model            string
	MaxContextTokens int
	RecentMessages   int
	SummaryTimeout   time.Duration
}

type ContextResult struct {
	Messages        []providers.Message
	HistoryCount    int
	Compacted       bool
	EstimatedTokens int
	Compaction      *CompactionResult
}

type CompactionResult struct {
	CoveredBefore  int
	CoveredAfter   int
	BeforeTokens   int
	AfterTokens    int
	RecentMessages int
	Summary        string
	Duration       time.Duration
}

func (m *ContextManager) Build(ctx context.Context, sessionID, systemPrompt string, user providers.Message) (ContextResult, error) {
	var history []providers.Message
	var summary string
	var covered int
	if m.Store != nil {
		var err error
		history, err = m.Store.Load(sessionID)
		if err != nil {
			return ContextResult{}, err
		}
		summary, covered, err = m.Store.LoadSummary(sessionID)
		if err != nil {
			return ContextResult{}, err
		}
		boundedSummary := truncateSummary(summary, m.MaxContextTokens)
		if boundedSummary != summary {
			summary = boundedSummary
			if err := m.Store.SaveSummary(sessionID, summary, covered); err != nil {
				return ContextResult{}, err
			}
		}
	}
	if covered > len(history) {
		covered = len(history)
	}
	base := make([]providers.Message, 0, len(history)+3)
	if systemPrompt != "" {
		base = append(base, providers.Message{Role: providers.RoleSystem, Content: systemPrompt})
	}
	if summary != "" {
		base = append(base, providers.Message{Role: providers.RoleSystem, Content: "Conversation summary:\n" + summary})
	}
	base = append(base, history[covered:]...)
	base = append(base, user)
	if m.MaxContextTokens <= 0 || estimateMessagesTokens(base) <= m.MaxContextTokens {
		return ContextResult{Messages: base, HistoryCount: len(history), EstimatedTokens: estimateMessagesTokens(base)}, nil
	}
	recent := m.RecentMessages
	if recent <= 0 {
		recent = 8
	}
	if recent > len(history) {
		recent = len(history)
	}
	// A tiny budget must still make progress when the history is shorter than
	// the normal recent window: keep the newest message and summarize the rest.
	if recent == len(history) && len(history) > 1 {
		recent = 1
	}
	cut := len(history) - recent
	if cut <= covered {
		return ContextResult{Messages: base, HistoryCount: len(history), EstimatedTokens: estimateMessagesTokens(base)}, nil
	}
	if m.Provider == nil {
		return ContextResult{}, fmt.Errorf("agent: context exceeds budget and summary provider is nil")
	}
	started := time.Now()
	summaryCtx := ctx
	var cancel context.CancelFunc
	if m.SummaryTimeout > 0 {
		summaryCtx, cancel = context.WithTimeout(ctx, m.SummaryTimeout)
		defer cancel()
	}
	newSummary, err := summarizeMessages(summaryCtx, m.Provider, m.Model, summary, history[covered:cut])
	if err != nil {
		return ContextResult{}, err
	}
	if m.Store == nil {
		return ContextResult{}, fmt.Errorf("agent: context compaction requires session store")
	}
	// A summarizer may itself return a long answer. Keep the persisted summary
	// bounded so it cannot become the next source of context overflow.
	newSummary = truncateSummary(newSummary, m.MaxContextTokens)
	if err := m.Store.SaveSummary(sessionID, newSummary, cut); err != nil {
		return ContextResult{}, err
	}
	result := make([]providers.Message, 0, recent+3)
	if systemPrompt != "" {
		result = append(result, providers.Message{Role: providers.RoleSystem, Content: systemPrompt})
	}
	result = append(result, providers.Message{Role: providers.RoleSystem, Content: "Conversation summary:\n" + newSummary})
	remaining := append([]providers.Message(nil), history[cut:]...)
	// If the recent window is still too large, discard its oldest entries while
	// always retaining the current user message.
	for len(remaining) > 0 && estimateContextTokens(systemPrompt, newSummary, remaining, user) > m.MaxContextTokens {
		remaining = remaining[1:]
	}
	result = append(result, remaining...)
	result = append(result, user)
	return ContextResult{Messages: result, HistoryCount: len(history), Compacted: true, EstimatedTokens: estimateMessagesTokens(result), Compaction: &CompactionResult{
		CoveredBefore: covered, CoveredAfter: cut, BeforeTokens: estimateMessagesTokens(base), AfterTokens: estimateMessagesTokens(result), RecentMessages: recent, Summary: newSummary,
		Duration: time.Since(started),
	}}, nil
}

func truncateSummary(summary string, maxTokens int) string {
	if maxTokens <= 0 {
		return summary
	}
	maxChars := maxTokens * 4 / 3
	if maxChars < 256 {
		maxChars = 256
	}
	runes := []rune(summary)
	if len(runes) <= maxChars {
		return summary
	}
	return string(runes[:maxChars]) + "\n[summary truncated]"
}

func estimateContextTokens(systemPrompt, summary string, history []providers.Message, user providers.Message) int {
	msgs := make([]providers.Message, 0, len(history)+3)
	if systemPrompt != "" {
		msgs = append(msgs, providers.Message{Role: providers.RoleSystem, Content: systemPrompt})
	}
	if summary != "" {
		msgs = append(msgs, providers.Message{Role: providers.RoleSystem, Content: "Conversation summary:\n" + summary})
	}
	msgs = append(msgs, history...)
	msgs = append(msgs, user)
	return estimateMessagesTokens(msgs)
}

func summarizeMessages(ctx context.Context, p providers.Provider, model, previous string, messages []providers.Message) (string, error) {
	prompt := "Summarize the conversation for future turns. Preserve concrete facts, user constraints, decisions, unfinished tasks, and important file paths. Be concise and do not invent facts."
	if previous != "" {
		prompt += "\nExisting summary:\n" + previous
	}
	input := []providers.Message{{Role: providers.RoleSystem, Content: prompt}}
	input = append(input, messages...)
	resp, err := p.Chat(ctx, providers.ChatRequest{Model: model, Messages: input})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Content), nil
}

func estimateMessagesTokens(messages []providers.Message) int {
	n := 0
	for _, msg := range messages {
		n += len(msg.Content) + len(msg.Reasoning) + len(msg.ToolCallID)
		for _, call := range msg.ToolCalls {
			n += len(call.ID) + len(call.Name) + len(call.Arguments)
		}
	}
	return (n + 3) / 4
}
