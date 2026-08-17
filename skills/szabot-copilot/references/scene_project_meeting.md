# reference — 决策会查询

> 本场景覆盖：决策会查询（立项/开机决策会信息）

## 场景路由

- **决策会查询** → `decision_making_meeting_search` → `kb_search`（补链接 + 补剧集分类）

---

## 工具参数定义

> ⚠️ **本文档为工具参数的唯一权威来源**。`mcporter list szabot_tools` 返回的参数类型定义存在错误，**调用工具时必须以本文档为准**。

### `decision_making_meeting_search` — 决策会检索

查询项目的立项/开机决策会信息。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `condition` | `object[]` | 否 | 等值筛选（如项目名称） |
| `range` | `object[]` | 否 | 区间筛选（如会议日期） |
| `rank` | `object[]` | 否 | 排序规则 |
| `target` | `string[]` | 否 | 要返回的字段列表 |
| `match` | `string` | 否 | `"exact"` 或 `"fuzzy"` |

可用字段：会议日期、项目ID、项目名称、上会主题、会议状态、工作室、会议类型、ROC

> ⚠️ **target 必含字段**：每次调用时 target 中**必须包含 `"项目ID"`、`"ROC"` 和 `"阐述页链接"`**。`项目ID` 用于追加调用 `kb_search` 获取项目链接。`阐述页链接` 由 MCP 服务直接返回完整 URL（非空即为有效链接）。决策会接口返回的 ROC 字段为**评估单ID**。

**调用示例**：

```bash
# 查某项目决策会
mcporter call 'szabot_tools.decision_making_meeting_search(
  condition: [{"项目名称": ["庆余年"]}],
  target: ["会议日期", "项目名称", "项目ID", "上会主题", "会议状态", "工作室", "ROC", "阐述页链接"],
  match: "fuzzy"
)' --output json

# 查未来1个月决策会
mcporter call 'szabot_tools.decision_making_meeting_search(
  range: [{"会议日期": ["2026-03-25", "2026-04-25"]}],
  target: ["会议日期", "项目名称", "项目ID", "上会主题", "会议状态", "工作室", "ROC", "阐述页链接"]
)' --output json
```

### 追加调用 `kb_search`

决策会返回结果后，**必须追加调用 `kb_search`**，以 `项目ID` 为条件，target 中**必须包含 `"项目详情链接"`**，用于项目链接判定。

```bash
mcporter call 'szabot_tools.kb_search(
  recall_req_list: [{
    domain_knowledge: "影库知识库-ES-0",  // 决策会为腾讯上会项目，固定查影库知识库
    query: {
      condition: [{"项目ID": ["2142"]}],
      target: ["项目ID", "项目详情链接", "剧集分类"],
      match: "exact"
    }
  }]
)' --output json
```

> `kb_search` 的完整参数定义见 `references/scene_project_core.md`。

---

## 链接判定

### 项目链接

- 追加调用 `kb_search` 时 target 必须含 `"项目ID"`、`"项目详情链接"`
- 链接格式：`[项目ID](项目详情链接字段值)`；`项目详情链接` 为空时只输出纯文本项目ID
- 有表格时，项目ID**必须放在表格内**作为一行，禁止单独列在表格外

### 阐述页链接（仅决策会场景）

- target 必须含 `"ROC"`、`"阐述页链接"` 字段
- 输出格式：`[ROC值](阐述页链接字段值)`，即超链接挂在ROC（评估单ID）上；`阐述页链接` 为空时只输出纯文本ROC值，不输出超链接

---

## 输出检查清单

> 通用检查项（数据来源、严禁编造、主动补充、搜索回退）见 SKILL.md 执行流程第 5 步，以下为本场景特化检查项：

| # | 检查项 | 要求 |
|---|--------|------|
| 1 | ⛔ **项目链接** | `项目详情链接` 非空则展示，`[项目ID](项目详情链接)` 格式 |
| 2 | ⛔ **阐述页链接** | `阐述页链接` 非空则展示，`[ROC值](阐述页链接)` 格式 |

## 常见错误速查

| 错误写法 | 正确写法 | 说明 |
|---------|---------|------|
| `condition: [{"field": "项目名称", ...}]` | `condition: [{"项目名称": ["庆余年"]}]` | ⛔ **最高频错误！** |
| `condition: [{"项目名称": "庆余年"}]` | `condition: [{"项目名称": ["庆余年"]}]` | value 必须是数组 |

## Gotchas

- **condition 格式**：模型极容易用 `field/operator/value` 结构，这是**完全错误**的！正确格式是 `{字段名: [值数组]}`
- `mcporter list` 返回的类型签名有已知错误，以本文档为准
- 阐述页链接：超链接挂在 ROC（评估单ID）上，为空则只输出纯文本 ROC 值
