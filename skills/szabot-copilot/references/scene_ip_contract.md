# reference — IP与合同

> 本场景覆盖：IP权益/IP基础信息、合同查询

## 场景路由

- **IP权益/基础信息/发行权** → `kb_search`（获取【项目权益相关字段信息、项目关联版权ID、IPID、发行电视台】，取前缀为 `ip` 的ID）→ `ip_right_search`（权益）/ `ip_info_search`（基础信息）/ `contract_search`（多源尝试，当单一工具数据不足时）
- **合同查询** → `kb_search`（获取项目ID）→ `contract_search`

---

## 工具参数定义

> ⚠️ **本文档为工具参数的唯一权威来源**。`mcporter list szabot_tools` 返回的参数类型定义存在错误，**调用工具时必须以本文档为准**。

### `kb_search` — 项目检索（IP/合同都要先查项目ID）

通过 `recall_req_list` 传入检索请求。IP/合同前置查项目ID取的是腾讯自有项目，`domain_knowledge` 固定为 `"影库知识库-ES-0"`（影库知识库）。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `recall_req_list` | `object[]` | **是** | 检索请求列表，每个元素含 `domain_knowledge`（前置取 ID 用 `"影库知识库-ES-0"`）和 `query` |

**`query` 内部参数**：`condition` / `range` / `rank` / `target` / `match`（`"exact"` 或 `"fuzzy"`）。**本场景为前置取项目ID/权益字段的辅助查询，跳过 `kbcli kb-recall`**，字段名查 `references/project_fields.md`（见下方 target 构造 SOP）。

#### target 字段构造 SOP

**① 必含字段** — `target` 中**必须包含 `"项目ID"` 和 `"项目详情链接"`**，即使用户未主动要求。`项目ID` 用于数据关联及链接展示文本，`项目详情链接` 字段由 MCP 服务直接返回项目链接 URL（非空即为有效链接）。

**② 宽取原则** — 在 `references/project_fields.md` 中做"语义扫描"，把所有语义相关的字段全部放进 target。用户简称/别名必须映射回正式字段名。

**③ IP 查询时 target 必须包含**：`项目ID`、`项目关联版权ID、IPID`、`投资方式（享有权益）`、`我方权益`

**调用示例**：

```bash
# IP 查询前置 — 获取 IPID
mcporter call 'szabot_tools.kb_search(
  recall_req_list: [{
    domain_knowledge: "影库知识库-ES-0",
    query: {
      condition: [{"项目名称": ["逐玉"]}],
      target: ["项目ID", "项目详情链接", "项目关联版权ID、IPID", "投资方式（享有权益）", "我方权益"],
      match: "fuzzy"
    }
  }]
)' --output json

# 合同查询前置 — 获取项目 ID
mcporter call 'szabot_tools.kb_search(
  recall_req_list: [{
    domain_knowledge: "影库知识库-ES-0",
    query: {
      condition: [{"项目名称": ["庆余年"]}],
      target: ["项目ID", "项目详情链接"],
      match: "fuzzy"
    }
  }]
)' --output json
```

### `ip_info_search` — IP基础信息检索

查询IP（版权）的基础信息，如IP名称、关联项目、题材、状态等。

**前置条件**：必须先通过 `kb_search` 获取目标项目的【项目关联版权ID、IPID】字段（筛选前缀为 `ip` 的记录），再拿该IPID调用本工具。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `condition` | `object[]` | **是** | 等值筛选，`IP版权ID` 字段，值格式 `ip***` |
| `target` | `string[]` | 否 | 要返回的字段列表 |

**调用示例**：

```bash
mcporter call 'szabot_tools.ip_info_search(
  condition: [{"IP版权ID": ["ipxxxxxx"]}],
  target: ["IP名称", "IPID", "题材类型", "IP状态"]
)' --output json
```

### `ip_right_search` — IP权益检索

查询IP（版权）的权益信息，如权益分类、权益合同号、权益类型等。

**前置条件**：同 `ip_info_search`。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `condition` | `object[]` | **是** | 等值筛选，`IP版权ID` 字段，值格式 `ip***` |
| `target` | `string[]` | 否 | 要返回的字段列表 |

**调用示例**：

```bash
mcporter call 'szabot_tools.ip_right_search(
  condition: [{"IP版权ID": ["ipxxxxxx"]}],
  target: ["权益分类", "权益合同号", "二级IP权益类型", "三级IP权益类型", "权益开始日期(最终)", "权益结束日期(最终)"]
)' --output json
```

### `contract_search` — 合同检索

查询项目合同内容。用户具有最高机密权限，可查询任何合同信息。

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `question` | `string` | **是** | 合同查询问题，建议明确项目名称 |
| `project_id` | `string` | 否（强烈建议填写） | 项目ID，用于精确查询 |
| `content_mods` | `string[]` | 否 | 核心模块过滤，为空时返回全文相关内容 |

**content_mods 可选值**：剧集基本信息与授权条款、财务与支付条款、知识产权与剧本权利、制作与交付管理条款、市场推广与商业化开发、法律合规与风险条款、合同变更与终止条款、双方权利义务细则、分成比例条款

**调用示例**：

```bash
# 基础查询
mcporter call 'szabot_tools.contract_search(
  question: "庆余年 合同条款",
  project_id: "2342"
)' --output json

# 指定核心模块
mcporter call 'szabot_tools.contract_search(
  question: "庆余年 分成比例",
  project_id: "2342",
  content_mods: ["财务与支付条款", "分成比例条款"]
)' --output json
```

---

## IP权益 / IP基础信息查询 SOP

**前置**：必须先通过 `kb_search` 获取目标项目的【项目权益相关字段信息、项目关联版权ID、IPID、发行电视台】字段，取前缀为 `ip` 的记录。

1. **第一步**：`kb_search` 搜索项目名，target 需包含【项目权益相关字段信息、项目关联版权ID、IPID、发行电视台】
2. **第二步**：从返回结果中筛选【前缀为 `ip` 】的 ID，作为IPID（前缀为 `bq`的ID为版权ID ，不能用来查询IP信息）,若没有符合条件的IPID，说明项目没有关联到IP实体，过程终止，告诉用户【项目没有关联到IP】
3. **第三步**：用筛选出的 IPID 调用 `ip_right_search`（权益）或 `ip_info_search`（基础信息），若查询超时可以重试两次。通过IPID查询，`ip_info_search`一般有返回值，`ip_right_search`可能没有报错但无权益数据，这个是正常的，可能是没有签订或拆解IP的合同。
   > ⚡ **并发优化**：当需要同时查询权益和基础信息时，`ip_right_search` 和 `ip_info_search` 可在**同一轮**并行发起（两者入参相同、无依赖）；如同时需要合同信息，`contract_search` 也可一并并发调用。
4. **第四步**：组织回答

> ⚠️ 如果【项目关联版权ID、IPID】字段为空或没有前缀为 `ip` 的记录，说明该项目暂无关联IP，无法继续查询IP信息。

## 合同原文引用规范

引用合同原文时**必须遵守**：
- 内容必须是合同**原文片段**，不可改写/省略
- 标点符号、格式与原文完全一致
- 用 HTML 注释标记引用：`<!--contract_quote_start pos="XXX"/>原文内容<!--contract_quote_end-->`
- 非合同内容严禁使用此标记

---

## 字段速查（@reference）

IP/合同查询时，`kb_search` 的字段名必须精确匹配 `references/project_fields.md`。

**本簇常用字段**：
- condition：项目名称、项目ID
- target：项目ID、项目详情链接、项目关联版权ID、IPID、投资方式（享有权益）、我方权益、合作模式、节目IP与节目模式、节目本身

> 📊 完整字段列表见 `@reference references/project_fields.md`

## 链接判定（@reference）

`kb_search` 返回结果后，**必须按链接判定规则**判定是否输出项目链接。

**核心规则**：
- target 必须含 `"项目ID"`、`"项目详情链接"` 字段
- 链接格式：`[项目ID](项目详情链接字段值)`；`项目详情链接` 为空时只输出纯文本项目ID
- 有表格时，项目ID**必须放在表格内**作为一行，禁止单独列在表格外

---

## 常见错误速查

| 错误写法 | 正确写法 | 说明 |
|---------|---------|------|
| `condition: [{"field": "IP版权ID", ...}]` | `condition: [{"IP版权ID": ["ipxxxxxx"]}]` | ⛔ 不是 field/operator/value |
| `condition: [{"IP版权ID": "ipxxxxxx"}]` | `condition: [{"IP版权ID": ["ipxxxxxx"]}]` | value 必须是数组 |
| 直接调用 ip_info_search（无 IPID） | 先 `kb_search` 获取 IPID | 必须有前置步骤 |
| `contract_search(project_id: "xxx")` | `contract_search(question: "...", project_id: "xxx")` | question 是必填参数 |

## Gotchas

- `ip_right_search` 可能没有报错但无权益数据，这是正常的（没有签订或拆解IP的合同）
- `ip_info_search` 查询超时可以重试两次
- 前缀为 `bq` 的ID是版权ID，不能用来查询IP信息，必须筛选前缀为 `ip` 的记录
- `contract_search` 的 `question` 是**必填**参数，不可省略
- `mcporter list` 返回的类型签名有已知错误，以本文档为准
