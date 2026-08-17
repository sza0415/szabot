# reference — 排播与储备

> **本场景覆盖**：排播现状查询、排播优化建议、储备查询、数据罗列、后期进度/视效进度。
>
> ⛔ **必读**：`kb_recall` 的调用方式、两段式流程、取值铁律、Fallback SOP 全部在 `references/kb_recall.md`。**本文件不复述该内容**，构造 `kb_search` 参数前请先读 `kb_recall.md`，否则会出现"把 `kbcli kb-recall`（CLI）当成 `szabot_tools.kb_recall`（MCP）来调"、"跳过 Step 2 直接 Fallback"、"domain_knowledge 凭记忆写死"等已知失败模式。
>
> 🔀 **子场景差异**：`排播现状查询`（辅助补电视剧）走**固定字段**跳过 kb-recall；`储备/数据罗列/排播优化/后期进度`走**宽字段召回**必须先 kb-recall。

## 场景路由

- **排播现状查询** → `scheduling_search` → **[必须]** `kb_search`（补充电视剧数据，排播表可能有遗漏）

> ⛔ **强制规则（MANDATORY）**：`scheduling_search` 调用后**必须**追加 `kb_search` 补充电视剧排播数据，这不是可选步骤。即使 `scheduling_search` 已返回看似完整的数据，仍必须执行 `kb_search` 补充，因为 `scheduling_search` 经常遗漏电视剧。
- **排播优化建议** → `scheduling_search` + `kb_search`（腾讯 + 竞品对比：`recall_req_list` 传影库知识库和全网知识库两个元素并行）→ 按6维度框架分析
- **储备查询** → `kb_search`（管理状态=开发中各状态）
- **数据罗列** → `kb_search`
- **后期进度/视效进度** → `kb_search`（target 包含后期相关字段）+ `szstudio-cms-board`（SzStudio 素材制作量化数据）

---

## 工具参数定义

> ⚠️ **本文档为工具参数的唯一权威来源**。`mcporter list szabot_tools` 返回的参数类型定义存在错误，**调用工具时必须以本文档为准**。

> ℹ️ `kbcli kb-recall` 是本地 CLI（不是 MCP 工具，`mcporter list` 不会列出），调用方式见 `references/kb_recall.md`。

### `kb_search` — 项目检索

影库数据库核心工具，支持多知识库并行。通过 `recall_req_list` 传入一个或多个检索请求。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `recall_req_list` | `object[]` | **是** | 检索请求列表，每个元素含 `domain_knowledge`（库标识字符串）和 `query`（查询参数） |

**`query` 内部参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `condition` | `object[]` | 否 | 等值筛选条件，多个条件之间为 AND 关系。字段名来自 `kbcli kb-recall` 返回，Fallback 时查 `references/project_fields.md` |
| `range` | `object[]` | 否 | 区间筛选 |
| `rank` | `object[]` | 否 | 排序规则 |
| `target` | `string[]` | 否 | 要返回的字段列表 |
| `match` | `string` | 否 | `"exact"` 或 `"fuzzy"`，默认精确匹配 |
| `view` | `object` | 否 | 视图查询（用于热度趋势等） |

#### target 字段构造 SOP（遵循 `references/kb_recall.md` 取值铁律）

**kb-recall 成功时（返回 `<text>`）**：target **必须传入 kb-recall 返回表格中"字段中文名"列的全部值**——不增、不删、不换、不筛选，禁止从本文件示例或默认列表补充/替换字段。

**仅 Fallback 时（kb-recall 失败/返回空）**，按以下 SOP 构造 target：

**① 必含字段** — `target` 中**必须包含 `"项目ID"` 和 `"项目详情链接"`**，即使用户未主动要求。`项目ID` 用于数据关联及链接展示文本，`项目详情链接` 字段由 MCP 服务直接返回项目链接 URL（非空即为有效链接）。

**② 宽取原则** — 除非用户**明确列出**要哪些字段，否则**禁止只返回少量字段**。必须在 `references/project_fields.md` 中做"语义扫描"，把所有语义相关的字段全部放进 target。用户简称/别名必须映射回正式字段名。

**③ 自检口令**：「我是否已经打开 `project_fields.md` 并对关键词做了全文扫描？凡是字段名或说明里出现该关键词 / 近义词的，我是否都已经放进 target 了？」

#### view 参数格式

```
view: {
  "视图名": {
    "param": "<播出后天数，空字符串表示全部>",
    "includes": ["字段1", "字段2"]
  }
}
```

可用视图：版权视图（疑似侵权量趋势）、星舟视频视图（新增预约人数趋势、腾讯最新热度值趋势）、短音视图、猫眼视图、微博视图

**调用示例**：

```bash
# 储备查询
mcporter call 'szabot_tools.kb_search(
  recall_req_list: [{
    domain_knowledge: "影库知识库-ES-0",
    query: {
      condition: [{"管理状态": ["已提前锁定", "已分配", "已立项", "已通过开机决策"]}, {"剧集分类": ["电视剧"]}],
      target: ["项目ID", "项目详情链接", "管理状态", "开发状态", "项目评级", "所属工作室", "主要演员", "导演"],
      match: "exact"
    }
  }]
)' --output json

# 数据罗列
mcporter call 'szabot_tools.kb_search(
  recall_req_list: [{
    domain_knowledge: "影库知识库-ES-0",
    query: {
      condition: [{"管理状态": ["已上线", "已播完"]}, {"剧集分类": ["电视剧"]}],
      range: [{"播出时间": ["2025-01-01", "2025-12-31"]}],
      rank: [{"腾讯最高热度值": ["从大到小"]}],
      target: ["项目ID", "项目详情链接", "播出时间", "腾讯最高热度值", "豆瓣评分", "管理状态"],
      match: "exact"
    }
  }]
)' --output json

# 后期进度查询
mcporter call 'szabot_tools.kb_search(
  recall_req_list: [{
    domain_knowledge: "影库知识库-ES-0",
    query: {
      condition: [{"项目名称": ["某剧"]}],
      target: ["项目ID", "项目详情链接", "当前后期阶段", "后期制作是否符合预期", "视效完成度", "预计后期时间周期(天)", "后期制作的问题和解决方法", "后期进展描述"],
      match: "fuzzy"
    }
  }]
)' --output json

# 排播优化建议：腾讯 + 竞品并行
mcporter call 'szabot_tools.kb_search(
  recall_req_list: [
    {
      domain_knowledge: "影库知识库-ES-0",
      query: {
        condition: [{"剧集分类": ["电视剧"]}],
        range: [{"播出时间": ["2026-03-01", "2026-03-31"]}],
        target: ["项目ID", "项目详情链接", "项目评级", "题材赛道", "主要演员", "播出时间"],
        match: "exact"
      }
    },
    {
      domain_knowledge: "全网知识库-ES-23",
      query: {
        range: [{"播出时间": ["2026-03-01", "2026-03-31"]}],
        target: ["项目名称", "项目详情链接", "项目评级", "题材赛道", "播出时间"],
        match: "exact"
      }
    }
  ]
)' --output json
```

### `scheduling_search` — 排播检索

查找指定时间范围内星舟视频和竞品的项目排播信息。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `start_date` | `string` | 是 | 起始日期，格式 `yyyy-mm-dd` |
| `end_date` | `string` | 否 | 截止日期，格式 `yyyy-mm-dd`。只有 start_date 表示查 `开播时间 > start_date` 的剧 |

> ⚠️ 可能只返回电影，电视剧排播需额外 `kb_search` 补充。

**调用示例**：

```bash
# 查某月排播
mcporter call 'szabot_tools.scheduling_search(
  start_date: "2026-03-01",
  end_date: "2026-04-01"
)' --output json

# 查某日期之后的排播
mcporter call 'szabot_tools.scheduling_search(
  start_date: "2026-03-25"
)' --output json
```

---

## 业务规则

### 排播优化建议（6维度）

仅在用户**明确要求排播优化建议**时使用：

1. **内容组合策略**：每月是否有 S+/S 级项目？是否有高主创配置？
2. **赛道与圈层**：每月至少1部"爱"赛道、各1部"燃"/"智"赛道、1部男性题材；每月最多2部古装
3. **创新实验田**：X剧场每季度1部？有无板凳单元？
4. **竞品防御**：同期竞品 S+/S/A 级项目和赛道差异
5. **播出类型**：独播 vs 联播数量
6. **成本管理**：月均 7~8 亿，年度 95~100 亿（无权限时说明并跳过）

### 数据罗列规范

1. 先总结符合条件的**项目数量**
2. 用 **markdown 表格**罗列所有项目
3. 结尾**不要**做总结
4. 如果目标字段返回为空/N/A，明确说明数据缺失，尝试 search_web 补充或标注"该字段暂无数据"

---

## 字段速查（@reference）

构造 condition/range/rank/target 时，字段名优先取自 `kbcli kb-recall` 返回；仅 Fallback 时精确匹配 `references/project_fields.md`。

> 📊 完整字段列表见 `@reference references/project_fields.md`

## 链接判定（@reference）

`kb_search` 返回结果后（储备查询/数据罗列需要项目链接），**必须按链接判定规则**判定。

**核心规则**：
- target 必须含 `"项目ID"`、`"项目详情链接"` 字段
- 链接格式：`[项目ID](项目详情链接字段值)`；`项目详情链接` 为空时只输出纯文本项目ID
- 有表格时，项目ID**必须放在表格内**作为一行，禁止单独列在表格外

---

## 常见错误速查

| 错误写法 | 正确写法 | 说明 |
|---------|---------|------|
| `condition: [{"field": "项目名称", ...}]` | `condition: [{"项目名称": ["庆余年"]}]` | ⛔ 不是 field/operator/value |
| `condition: [{"项目名称": "庆余年"}]` | `condition: [{"项目名称": ["庆余年"]}]` | value 必须是数组 |
| `rank: ["腾讯最高热度值", "从大到小"]` | `rank: [{"腾讯最高热度值": ["从大到小"]}]` | rank 是 object[] |
| `range: ["播出时间", "2025-01-01", ""]` | `range: [{"播出时间": ["2025-01-01", ""]}]` | range 是 object[] |

## Gotchas

- `scheduling_search` 可能只返回电影，电视剧排播需用 `kb_search`（condition: 管理状态=已定档/已上线）补充
- range 日期过滤不可靠，用管理状态 + 客户端二次过滤
- 待播/排播查询**不要**在 condition 中设开发状态条件（会漏数据）
- 开发状态可能未及时更新，用"管理状态"代替
- 关键字段可能为空（项目评级、首播时间、热度等），处理：回答中明确说明数据缺失，尝试 `search_web` 补充
- 后期进度/视效进度查询需同时调用 `szstudio-cms-board` 获取 SzStudio 素材制作数据（两个 Skill 联合使用）
