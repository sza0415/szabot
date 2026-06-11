package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAICompatibleProvider 是一个通用的 OpenAI 兼容 Provider。
//
// 凡是接口形如 POST {BaseURL}/chat/completions、请求/响应遵循 OpenAI
// chat completions 规范的服务，都可以用它。已知兼容：
//   - OpenAI       BaseURL = https://api.openai.com/v1
//   - DeepSeek     BaseURL = https://api.deepseek.com/v1
//   - Moonshot     BaseURL = https://api.moonshot.cn/v1
//   - 本地 Ollama   BaseURL = http://localhost:11434/v1
//
// 设计要点：
//   - 不引入任何官方 SDK，纯 net/http + encoding/json，依赖最少；
//   - 错误信息尽量带上 HTTP 状态码和原始响应体，便于排查；
//   - 请求超时由调用方通过 ctx 控制；HTTPClient.Timeout 作为兜底。
type OpenAICompatibleProvider struct {
	// ProviderName 用于日志展示，例如 "deepseek"、"openai"。
	ProviderName string

	// BaseURL 不要带末尾斜杠，例如 "https://api.deepseek.com/v1"。
	BaseURL string

	// APIKey 走 Authorization: Bearer 头。
	APIKey string

	// HTTPClient 可注入；为空时使用 30s 超时的默认 client。
	HTTPClient *http.Client
}

// Name 返回 provider 名字（日志用）。
func (p *OpenAICompatibleProvider) Name() string {
	if p.ProviderName == "" {
		return "openai-compatible"
	}
	return p.ProviderName
}

// ---- 与 OpenAI 一致的请求/响应结构（先只覆盖纯文本对话所需字段） ----

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model    string              `json:"model"`
	Messages []openAIChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// Chat 发起一次 chat completions 调用。
func (p *OpenAICompatibleProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if p.BaseURL == "" {
		return ChatResponse{}, errors.New("provider: BaseURL is empty")
	}
	if p.APIKey == "" {
		return ChatResponse{}, errors.New("provider: APIKey is empty")
	}
	if req.Model == "" {
		return ChatResponse{}, errors.New("provider: model is empty")
	}

	// 1. 把内部 Message 转成 OpenAI wire format。
	wireMsgs := make([]openAIChatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		wireMsgs = append(wireMsgs, openAIChatMessage{
			Role:    string(m.Role),
			Content: m.Content,
		})
	}

	body, err := json.Marshal(openAIChatRequest{
		Model:    req.Model,
		Messages: wireMsgs,
		Stream:   false,
	})
	if err != nil {
		return ChatResponse{}, fmt.Errorf("provider: marshal request: %w", err)
	}

	// 2. 发起 HTTP 请求。
	url := p.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("provider: new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := p.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("provider: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("provider: read response: %w", err)
	}

	// 3. 非 2xx 直接返回带原始内容的错误，便于排查 401/429/400 等。
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ChatResponse{}, fmt.Errorf(
			"provider: http %d: %s",
			resp.StatusCode,
			truncate(string(respBody), 500),
		)
	}

	// 4. 解析 JSON。
	var parsed openAIChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return ChatResponse{}, fmt.Errorf("provider: unmarshal response: %w; body=%s",
			err, truncate(string(respBody), 500))
	}
	if parsed.Error != nil {
		return ChatResponse{}, fmt.Errorf("provider: api error: %s (%s)",
			parsed.Error.Message, parsed.Error.Code)
	}
	if len(parsed.Choices) == 0 {
		return ChatResponse{}, errors.New("provider: no choices in response")
	}

	return ChatResponse{Content: parsed.Choices[0].Message.Content}, nil
}

// truncate 截断超长字符串，避免把整段响应糊到错误里。
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
