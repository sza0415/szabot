# reference — 轻量问答

> 本场景覆盖：行业常识、舆情口碑、通用问题
> **零 `kb_search` 依赖** — 三个场景均不需要调用 `kb_search`

## 场景路由

- **行业常识** → `common_sense`
- **舆情口碑** → 并行调用 `report_search` + `search_web`（两者入参均来自项目名称，无依赖，同时发起）
- **通用问题** → `search_web` 或直接回答

## 核心规则

- **行业常识**：直接调用 `common_sense`，不需要 `kb_search`
- **舆情口碑**：并行调用 `report_search` + `search_web`，两者入参均为项目名称关键词，无依赖
- **通用问题**：优先 `search_web`，无需调用影库内部工具；如 `search_web` 调用失败，尝试 `szabot-web-search` Skill 的 `mcp_web_search`；若仍失败，基于已有知识回答并标注"网络搜索暂不可用"

---

## 工具参数定义

> ⚠️ **本文档为工具参数的唯一权威来源**。`mcporter list szabot_tools` 返回的参数类型定义存在错误，**调用工具时必须以本文档为准**。

### `common_sense` — 常识知识库（HVIP、ROC、完播率等）

查询影视行业术语定义。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `query` | `string` | **是** | 要查询的常识问题，建议单跳问句 |

**调用示例**：

```bash
mcporter call 'szabot_tools.common_sense(
  query: "ROC指标是什么意思"
)' --output json
```

### `report_search` — 舆情报告检索

查询项目的舆情分析报告，包含微博、小红书、短音用户的真实评价分析。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `keyword` | `string` | **是** | 项目名称关键词，不加书名号 |

> ⚠️ report_search 数据覆盖率有限，常返回空。Fallback → `search_web(keyword: "项目名 口碑/舆情")`

**调用示例**：

```bash
mcporter call 'szabot_tools.report_search(
  keyword: "庆余年"
)' --output json
```

### `search_web` — 网络搜索

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `keyword` | `string` | **是** | 搜索关键词 |
| `limit` | `number` | 否 | 返回结果数量，默认 10 |

> ⚠️ **常见错误**：参数名是 `keyword`，**不是** `query`！

**调用示例**：

```bash
mcporter call 'szabot_tools.search_web(
  keyword: "庆余年 口碑 舆情"
)' --output json

mcporter call 'szabot_tools.search_web(
  keyword: "逐玉 盗版 侵权",
  limit: 5
)' --output json
```

---

## Gotchas

- `report_search` 数据覆盖率有限，很多项目没有舆情报告，返回空是常态。Fallback → `search_web(keyword: "项目名 口碑/舆情")`
- `search_web` 的参数名是 `keyword`，**不是** `query`
- 若 `search_web` 调用失败（如 Brave API 未配置、服务不可用），尝试 `szabot-web-search` Skill 的 `mcp_web_search`；若仍失败，基于已有知识回答并标注"网络搜索暂不可用"，**禁止直接放弃**
