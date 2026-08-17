package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ziangsun/szabot/internal/providers"
	"github.com/ziangsun/szabot/internal/skillreview"
)

// llmExtractor 用一次 LLM 调用把 SKILL.md 正文抽成结构化 Path。
//
// 为什么用 LLM：真实 skill 的"工具"形态五花八门（MCP 调用 mcporter call、
// CLI 命令 kbcli kb-search、工具表格里的 MCP 工具名……），正则启发式抓不全。
// 交给模型读语义、直接产出 PathDefinition 更准。
//
// 复用 szabot 已有的 providers 抽象：只依赖 Provider.Chat 接口，
// 想换 DeepSeek / OpenAI / 本地模型都不用改这里。
type llmExtractor struct {
	provider providers.Provider
	model    string
}

// newLLMExtractor 按与 szabot 一致的环境变量约定构造 Provider。
//
//	SZABOT_PROVIDER=deepseek 且 DEEPSEEK_API_KEY 存在 → 用 DeepSeek 抽取；
//	其余情况（未配置 / echo）→ 返回 nil，调用方回退到正则版 derivePath。
//
// 之所以在"echo"时返回 nil：echo provider 只回声、产不出合法 JSON，
// 走 LLM 抽取只会失败，不如直接回退。
func newLLMExtractor() *llmExtractor {
	switch os.Getenv("SZABOT_PROVIDER") {
	case "deepseek":
		key := os.Getenv("DEEPSEEK_API_KEY")
		if key == "" {
			return nil
		}
		baseURL := envOr("DEEPSEEK_BASE_URL", "https://api.deepseek.com/v1")
		model := envOr("DEEPSEEK_MODEL", "deepseek-chat")
		return &llmExtractor{
			provider: &providers.OpenAICompatibleProvider{
				ProviderName: "deepseek", BaseURL: baseURL, APIKey: key,
			},
			model: model,
		}
	case "openai":
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			return nil
		}
		baseURL := envOr("OPENAI_BASE_URL", "https://api.openai.com/v1")
		model := envOr("OPENAI_MODEL", "gpt-4o-mini")
		return &llmExtractor{
			provider: &providers.OpenAICompatibleProvider{
				ProviderName: "openai", BaseURL: baseURL, APIKey: key,
			},
			model: model,
		}
	default:
		return nil
	}
}

// envOr 返回环境变量，缺省时用 fallback。
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// pathExtractSystemPrompt 指导模型把 SKILL.md 抽成一条 PathDefinition。
// 关键约束：只输出 JSON、节点类型受限枚举、工具节点必须带 tool 名。
const pathExtractSystemPrompt = `你是 Skill 评审系统的路径抽取器。给你一个 Skill 的 SKILL.md 正文，
请抽取出这个 Skill 从"意图命中"到"产出结果"的完整执行路径（Path），只输出一个 JSON 对象，不要任何解释文字、不要 markdown 代码块围栏。

JSON 结构（严格遵守字段名）：
{
  "path_id": "path_<skill名，非字母数字换成下划线>",
  "name": "<skill名> 完整路径",
  "entry_conditions": ["触发该 Skill 的条件/关键词，如 @expert:market、用户提到侵权查询 等"],
  "nodes": [
    {"id": "唯一英文小写下划线id", "kind": "input|validation|decision|tool|output|fallback", "tool": "工具名(仅 kind=tool 时必填)", "condition": "该节点做什么/进入条件", "required": true, "notes": ["注意事项(可选)"]}
  ],
  "exit": "最后一个节点id"
}

节点抽取规则：
- 第一个节点固定 kind=input、id=match_intent，表示"模型据 description 命中该 Skill"。
- 工具调用要抽成 kind=tool 的节点，tool 字段填真实工具名。真实工具形态包括：
    * MCP 调用：mcporter call 'server.tool'  → tool 填 tool 部分（如 mcp_exec_sql、kb_search、cms_search_assets）；
    * CLI 命令：如 kbcli kb-search、kbcli kb-recall  → tool 填命令（如 kbcli kb-search）；
    * bash 脚本：bash path/to/script  → tool 填脚本名。
  同一 Skill 可能有多个工具节点，按正文里的调用顺序排列。
- "必须先读 references/xxx""先读后调"等前置动作 → kind=validation。
- 有条件分支/优先级判断（如按品类走不同链路）→ kind=decision。
- "重要规则/禁止/不得/铁律"等约束 → 放进相关节点的 notes，或单独一个 kind=validation 节点。
- 异常处理/重试/兜底 → kind=fallback。
- 最后一个节点 kind=output，表示产出结果。
- required：核心必经节点 true，可选/兜底节点 false。

只输出 JSON。`

// extract 调用 LLM 抽取 Path。失败时返回 error，调用方据此回退到正则版。
func (e *llmExtractor) extract(ctx context.Context, name, md string) (skillreview.PathDefinition, error) {
	if e == nil || e.provider == nil {
		return skillreview.PathDefinition{}, fmt.Errorf("llm extractor unavailable")
	}
	// 给足超时，但不无限等。
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	user := fmt.Sprintf("Skill 名：%s\n\nSKILL.md 正文：\n---\n%s\n---", name, md)
	resp, err := e.provider.Chat(ctx, providers.ChatRequest{
		Model: e.model,
		Messages: []providers.Message{
			{Role: providers.RoleSystem, Content: pathExtractSystemPrompt},
			{Role: providers.RoleUser, Content: user},
		},
	})
	if err != nil {
		return skillreview.PathDefinition{}, fmt.Errorf("llm chat: %w", err)
	}
	path, err := parsePathJSON(resp.Content)
	if err != nil {
		return skillreview.PathDefinition{}, fmt.Errorf("parse llm output: %w", err)
	}
	return path, nil
}

// parsePathJSON 从模型输出里提取并解析出 PathDefinition。
// 模型偶尔会包一层 ```json 围栏或带前后缀文字，这里做容错：截取第一个
// '{' 到最后一个 '}' 之间的内容再解析。
func parsePathJSON(raw string) (skillreview.PathDefinition, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return skillreview.PathDefinition{}, fmt.Errorf("empty output")
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end < 0 || end < start {
		return skillreview.PathDefinition{}, fmt.Errorf("no json object found")
	}
	var path skillreview.PathDefinition
	if err := json.Unmarshal([]byte(s[start:end+1]), &path); err != nil {
		return skillreview.PathDefinition{}, err
	}
	if path.PathID == "" || len(path.Nodes) == 0 {
		return skillreview.PathDefinition{}, fmt.Errorf("incomplete path: missing path_id or nodes")
	}
	return path, nil
}
