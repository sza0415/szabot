# reference — 剧本/小说分析

## 场景路由

- **剧本/小说分析** → `script_search`（优先）或 `novel_search`

## 核心规则

- **严禁凭已有知识回答剧本/小说相关问题，必须查询**
- `novel_id` 当前无直接获取途径，**优先使用 `script_search`**
- 如确需小说内容，先通过 `kb_search` 获取项目关联信息，或用 `search_web` 补充

---

## 工具参数定义

> ⚠️ **本文档为工具参数的唯一权威来源**。`mcporter list szabot_tools` 返回的参数类型定义存在错误，**调用工具时必须以本文档为准**。

### `script_search` — 剧本检索

查询剧本的脉络、情节、人物、戏份等内容。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `question` | `string` | **是** | 剧本查询问题，建议包含项目名称 |

**调用示例**：

```bash
mcporter call 'szabot_tools.script_search(
  question: "庆余年 的剧本是怎么样的"
)' --output json
```

### `novel_search` — 小说检索

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `novel_id` | `string` | **是** | 小说ID |
| `question` | `string` | **是** | 查询问题 |

> ⚠️ `novel_id` 当前无直接获取途径，**优先使用 `script_search`**。

示例：`novel_search(novel_id: "12345", question: "主角是谁")`

---

## Gotchas

- `novel_search` 需要 `novel_id`，当前无直接获取途径。优先用 `script_search`
- 如确需小说内容：先通过 `kb_search` 获取项目关联信息，或用 `search_web` 补充
- **严禁凭已有知识回答**，即使对热门剧集也必须通过工具查询
