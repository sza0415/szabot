# reference — 字段召回 `kbcli kb-recall`

> **何时读本文件**：需要用召回结果取数时——无论是①用 `kb_search` 查腾讯内部/全网知识库项目信息、人才库人才信息等元数据字段（metadata 路，需宽字段构造 target），还是②取播放/收入/成本等指标（metric 路，交 `szabot-data-query`）——构造下游参数前必须先执行本流程。

## `kbcli kb-recall`（⚠️ CLI 命令，不是 MCP 工具，禁止当工具名调用）

执行 `kbcli kb-recall --text "..."` 召回**字段/指标**列表。返回可能含 **metadata 段**与 **metric 段**两类（见下），二者消费链路不同：
- **metadata 段** → 决定 target（用哪些字段）+ domain_knowledge（查哪个库），构造 `kb_search` 的 `recall_req_list`；
- **metric 段** → 决定用哪些指标组/字段/来源表/md_doc，交 `szabot-data-query` 用 `mcp_exec_sql` 取数。

## ⛔ 两类返回段（metadata vs metric）—— 先分流再消费

kb-recall 的 `<text>` 结果可能含**两种表格段**，由每段标注的 `domain_knowledge` 区分，**消费链路完全不同，禁止混用**：

| 返回段 | `domain_knowledge`（原样取用，勿写死） | 表格列（schema） | 下游链路 | 用途 |
|---|---|---|---|---|
| **metadata 段** | `影库知识库-ES-0`（电视剧）、`全网知识库-ES-18`（竞品/全网）**等 ES 库**（不止 ES-0） | cname / aliases / short_desc / param_types / enum_values | **`kb_search`（project_search）** | 项目信息/属性/负责人等元数据字段 |
| **metric 段** | `影库知识库-MYSQL-24` | 指标组中文名 / 字段列表 / 组描述 / 数据来源表 / 说明文档(md_doc) | **`szabot-data-query`** → 电视剧播放用 **`kbcli kb-search --sql`**；其他品类用 `mcp_exec_sql` | 播放/收入/成本等**指标**，需 SQL 切片取数 |

> ⚠️ **domain_knowledge 一律原样取自 kb-recall 返回，禁止凭记忆写死为某个固定值**（metadata 段不止 `ES-0`，还有 `全网知识库-ES-18` 等；前缀有"影库知识库/全网知识库"之分，逐字复制返回值即可）。

> 🔎 **怎么识别当前是哪一段（唯一依据 = `domain_knowledge`，非服务端自动感知）**：kb-recall 返回的 `<text>` 里每张表格都带一个 `domain_knowledge` 标签，**由调用方（copilot）逐段解析判断**：
> - 前缀含 **`-MYSQL-24`**（`影库知识库-MYSQL-24`）→ **metric 段**：copilot 把该段的 `字段列表(member_fields)/来源表(source_table)/md_doc` 分流、交 `szabot-data-query` 用 `mcp_exec_sql` 取数；
> - 前缀含 **`-ES-*`**（`影库知识库-ES-0` / `全网知识库-ES-18` 等）→ **metadata 段**：交 `kb_search`。
>
> ⇒ **`szabot-data-query` 拿到的 metric 参数，就是 copilot 从这段 `<text>` 里按 `MYSQL-24` 解析、分流过来的**——data-query 不自己调 kb-recall，也不靠品类目录判断，**只认 `domain_knowledge`**。

**分流规则（返回哪段就走哪段；默认两段都消费、不做取舍，各段按各自的取值铁律处理）**：
1. **metadata 段（ES-0 / ES-18 等）** → 按下方「⚠️ 取值铁律」+「调用流程」构造 `kb_search` 的 `recall_req_list`。
2. **metric 段（MYSQL-24）** → **不要塞进 `kb_search`**（它不是 project_search 的字段库），按下方「⚠️ 取值铁律（metric 段 · MYSQL-24）」交给 `szabot-data-query` 取数。

> 🚦 **metric 段的品类门禁（当前启用范围）**：metric 段目前**只对「电视剧播放」品类启用**——
> - **电视剧播放** → 交 `szabot-data-query`，用 **`kbcli kb-search --domain-knowledge "影库知识库-MYSQL-24" --database <md_doc的db_name> --sql <md_doc的SQL>`** 执行；
> - **其他品类**（电视剧预算 / 制片进度 / 综艺 / 动漫·SzCanvas）→ **忽略 metric 段**，交 `szabot-data-query` 按 `references/<品类>/sql_query.md` + `schema/*.md` 拼 SQL，用 `mcp_exec_sql` 执行。
> - 启用范围与扩品类前置条件详见 `szabot-data-query/SKILL.md §6.3`。

## ⚠️ 取值铁律（metadata 段 · ES-*）

1. `domain_knowledge` 和 `query`（condition/range/rank/target）的字段**只能从 kb-recall 返回中取**（全量原样取用，不增不删不换不筛选）。target 必须传入召回表格的**全部**字段，禁止以"与问题无关"为由删减。仅 kb-recall 失败/返回空时才可从 scene reference 或 `project_fields.md` 兜底。
   - ⛔ **仅适用于 metadata 段（ES-0 / ES-18 等 ES 库）**：本条"target 传全部字段"针对 `kb_search`。metric 段（MYSQL-24）不构造 target，按上方分流规则交给 data-query。
2. 需要查多个库时，**新增** `recall_req_list` 元素，不得替换已有的（`recall_req_list` 是数组，要查几个库就传几个元素，每元素一个 `domain_knowledge` + `query`，不限于对比场景）。**注意**：`recall_req_list` 只放 metadata 段的库（ES-*），**禁止**把 `影库知识库-MYSQL-24` 写进 `kb_search` 的 `recall_req_list`。
3. **别混淆两种标识符**：`<catalog>` 的斜杠路径（`/影库知识库/…`）只用于 `--scope`；`kb_search` 的 `domain_knowledge` 取 `<text>` 里标注的库标识（如 `影库知识库-ES-0`），**不要拿 catalog 路径当 domain_knowledge**。

## ⚠️ 取值铁律（metric 段 · MYSQL-24）

> 与 metadata 段同理：**只信 kb-recall 返回，不凭记忆造字段/表名**。

1. **只能从 kb-recall 的 metric 段返回中取**：指标组 `group_id` / `字段列表`(member_fields) / `数据来源表`(source_table) / `说明文档`(md_doc) / `domain_knowledge` 全部**原样取用，不增不删不换不筛选**。禁止凭记忆补指标列、改表名、换 db_name。
2. **md_doc 优先**：拼 SQL 以 md_doc 内的口径切片与可执行 SQL Demo 为准（含 `is_*_accu` 切片、`*_inc` 日增、db_name）。禁止绕过 md_doc 自行推导口径。
3. **失败才回退**：仅当 kb-recall 失败/返回空/未返回 metric 段时，才回退 `szabot-data-query` 老路 reference；**成功返回 metric 段后禁止再翻老 schema 补/换指标**。详见文末统一「Fallback（失败回退）」。

### ❌ 最高频误用：把 metric 段塞进 `kb_search`（务必看反例）

> **症状**：嘴上说"通过 kb-recall 命中 metric 段获取 SQL"，实际却去调 `kb_search` 并把 `MYSQL-24` 当 domain_knowledge 传进去。**这是错的**——metric 段取数**根本不经 `kb_search`**。

```bash
# ❌ 错误（三重违规，严禁）：
mcporter call 'szabot_tools.kb_search(
  recall_req_list: [{
    domain_knowledge: "影库知识库-MYSQL-24",          # ← 违规①：MYSQL-24 不能进 kb_search
    query: { condition: [{"项目ID": ["171"]}],
             target: ["播放VV","收入","日增量","每日"], # ← 违规③：口语字段凭空造，非召回返回
             match: "fuzzy" },
    catalog: "/影库知识库/电视剧/播放指标",             # ← 违规②：catalog 只用于 kb-recall --scope，不是 kb_search 参数
    page_size: 20
  }]
)'
```

**三重错因**：
- ① `kb_search` 是 **project_search（查项目元数据）**，`recall_req_list` **只放 metadata 段（ES-*）**；`影库知识库-MYSQL-24` 属 metric 段，**禁止**写进 `kb_search`。
- ② `catalog` 斜杠路径**只用于 `kbcli kb-recall --scope`**，不是 `kb_search` 的参数。
- ③ `target` 字段必须来自召回返回的 `<text>`，`播放VV/收入/日增量` 是用户口语，**不能凭记忆造**。

```
# ✅ 正确：metric 段取数走 kb-recall → md_doc → data-query，不碰 kb_search
① kbcli kb-recall --text "折腰 每日播放VV 收入" --scope "/影库知识库/电视剧/..."
   → 返回 metric 段（domain_knowledge=MYSQL-24，含 md_doc/source_table/member_fields）
② copilot 分流：识别为 metric 段 → 交 szabot-data-query（见 copilot §5.1）
③ szabot-data-query 判品类：
   · 电视剧播放 → 读 md_doc 内可执行 SQL（日增切 *_inc）→ kbcli kb-search --sql 执行
   · 其他品类   → 忽略 metric 段，按 references/<品类> 拼 SQL → mcp_exec_sql 执行
```

> 💡 `kb_search`（domain_knowledge=ES-0）在这类 query 里**只用于先拿"项目ID/剧集分类"**（metadata 段）；一旦要取指标数值，就交给 data-query（电视剧播放用 `kbcli kb-search --sql`，其他品类用 `mcp_exec_sql`），**绝不用 `kb_search` 取指标**。

## ❌ 高频误用：把 kb-recall 当 MCP 工具调（务必看反例）

> **症状**：知道该用 kb-recall 了，却**套用 `mcporter call 'szabot_tools.<tool>(...)'` 模板**去调它——因为本生态里 `kb_search`/`mcp_exec_sql` 都是这么调的，模型惯性把 kb-recall 也当成了 `szabot_tools` 上的 MCP 工具。**这是错的**——kb-recall 是**独立 kbcli CLI 命令**，`szabot_tools` server 上没有这个工具。

```bash
# ❌ 错误（双重违规，严禁）：
mcporter call 'szabot_tools.kb_recall(
  text_input: "庆余年2 每日收入",           # ← 违规②：参数名错，CLI 用 --text 不是 text_input
  scopes: ["/影库知识库/电视剧/播放与收入/收入"], # ← 违规②：CLI 用 --scope 不是 scopes
  top_k: 3
)'
# → MCP error -32601: tool not found: kb_recall   ← szabot_tools 上没有这个工具
```

**双重错因**：
- ① `kb-recall` 是 **kbcli CLI 命令**，**不是** `szabot_tools` 的 MCP 工具，**禁止**用 `mcporter call` 调（调了必报 `-32601 tool not found: kb_recall`）。
- ② 参数是 **CLI flag**：用 `--text` / `--scope`（可多次传 `--scope`），**不是** `text_input` / `scopes` / `top_k`。

```bash
# ✅ 正确：kb-recall 走 kbcli CLI，直接命令行执行，不经 mcporter
kbcli kb-recall --text "庆余年2 每日收入" --scope "/影库知识库/电视剧/预算与成本/详细收入构成"
```

> 💡 **心智模型（一眼区分）**：
> - `kb_search` / `mcp_exec_sql` = **MCP 工具** → `mcporter call 'szabot_tools.kb_search(...)'`；
> - `kb-recall` = **独立 kbcli CLI** → `kbcli kb-recall --text ... --scope ...`。
> 两者调用方式完全不同，**禁止混用模板**。

## 调用流程

> ⛔ **两段式**：首次不带 `--scope` **通常返回 `<catalog>`（中间态，不是结果）**，必须接 Step 2 带 `--scope` 拿到 `<text>` 才算召回完成。最高频错误就是走了 Step 1 拿 catalog 就停、直接跳去 `kb_search`。

| 步骤 | 动作 | 预期返回 |
|------|------|----------|
| Step 1 | 执行 `kbcli kb-recall --text "..."` | ① 返回 `<text>` → 按上方「两类返回段」分流：**metadata 段**构造 `kb_search`、**metric 段**交 `szabot-data-query`（默认两段都消费）；② 返回 `<catalog>` → 进入 Step 2 |
| Step 2 | **必须**执行带 `--scope` 的第二次调用（需指标时 scope 要覆盖 metric catalog，见「scopes 构建规则」） | ① 返回 `<text>` → 同上按段分流消费；② 仍返回 `<catalog>` → 检查错误日志，可选择 A、调整参数后重试，B、跳过 kb-recall 流程，直接走 Fallback |
| Fallback | 失败/返回空/缺某段时 | 按段分别兜底，见文末统一「Fallback（失败回退）」。domain_knowledge / 表名 / 字段一律禁止凭记忆写死 |

**禁止在 Step 1 返回 catalog 时跳过 Step 2 直接走 Fallback**

**⚠️ kb-recall 成功（返回 `<text>`）后的禁令**：
- **metadata 段**：禁止再读 `project_fields.md` 或任何 scene reference 补充/替换字段。target **必须包含**返回表格"字段中文名"列的**全部**值——不增、不删、不换、不筛选；即使认为"与问题无关"也必须原样全部传入。`project_fields.md` 仅在 Fallback 时才可读取。
- **metric 段**：禁止再翻 `szabot-data-query` 老路 `schema/*.md` 去补/换指标；`group_id / 字段列表 / 数据来源表 / md_doc` 原样取用，拼 SQL 以 md_doc 为准（详见上方「取值铁律（metric 段 · MYSQL-24）」）。老路 schema 仅在 Fallback 时才可读取。

## scopes 构建规则

**catalog 路径结构**（四层）：`/{知识库层}/{品类}/{L1tag}/{L2tag}`
- **L1 知识库层**（如 `影库知识库`）：**含影库 + 竞品（全网）数据**；
- **L2 品类**：`电视剧` / `综艺` / …；
- **L3 = L1tag**（我们定义的标签）：如 `项目信息` / `播放及用户分析` / `预算与成本` / `大盘数据`；
- **L4 = L2tag**：如 `项目基础信息` / `播放表现` / `详细收入构成` / `ROC`。
- 真实示例：`/影库知识库/电视剧/项目信息/项目基础信息`。

**规则**：
- **输入**：从 Step 1 返回的 `<catalog>` 表格第一列选择相关完整路径，作为 `--scope` 传入；需要多个范围时重复传多个 `--scope`。
- 父子包含：路径越短覆盖越大（含其下级全部字段），越长越精确。
- ⚠️ **metric 与 metadata 共用同一棵 catalog 树**，靠 `domain_knowledge`（ES vs MYSQL-24）区分，**不是靠单独路径**。例如 `dsj_base_info`/`dsj_dim`（metric 组）就挂在 `/影库知识库/电视剧/项目信息/项目基础信息`，与 metadata 项目字段同路径。
- 🚨 **scope 收窄陷阱**：scope 按 L1/L2 tag 过滤，**scope 太窄会漏掉别的 tag 下的组（无论 metadata 还是 metric）**。例：只 scope `项目信息` 会召不回 `预算与成本/详细收入构成` 下的收入指标。**要取某类指标，scope 必须覆盖到它所在的 L1/L2 tag**；不确定时可放宽到品类层 `/影库知识库/电视剧` 或不加 `--scope`（全域召回）。
- 常用 L1tag → 覆盖内容（电视剧）：
  - `项目信息/项目基础信息` → 项目属性 + base_info/dim
  - `播放及用户分析/播放表现` → 播放/热度/拉新/曝光/转化/时长等指标 + 用户画像
  - `预算与成本/详细收入构成` → 收入总览/会员/广告收入；`预算与成本/ROC` → ROC；`预算与成本/制作·采购成本`「其他费用」→ 成本
  - `大盘数据/品类大盘数据` → 品类大盘
- 示例：
  ```bash
  # 查项目基础信息 + 收入：两个 L1/L2 都要覆盖，否则漏收入
  --scope "/影库知识库/电视剧/项目信息/项目基础信息" \
    --scope "/影库知识库/电视剧/预算与成本/详细收入构成"

  # 查播放 + 项目属性 + 人才：人才库是独立 L1，需单独加 scope
  --scope "/影库知识库/电视剧/播放及用户分析" \
    --scope "/影库知识库/电视剧/项目信息/项目基础信息" \
    --scope "/人才库/人才"
  ```

## Fallback（失败回退）—— 唯一权威口径

**⛔ 触发条件（只有满足才允许兜底）**：`kbcli kb-recall` 命令不存在 / 报错 / 超时 / 返回空 / 未返回对应段。
> **成功返回 `<text>` 后严禁兜底补数据**：metadata 段已回就不许再读 `project_fields.md`；metric 段已回就不许再翻老 `schema/*.md`。

**按段分别回退**：
- **metadata 段** → 回退读 `references/project_fields.md` 做语义扫描构造 target；
- **metric 段** → 回退 `szabot-data-query` 老路（对应品类 `references/<品类>/sql_query.md` + `schema/*.md`）按原有流程取数。

**自检口令**：
- metadata：**"我的 target 字段，是来自 kbcli kb-recall 还是 project_fields.md？若都没有，立即停下先去拿。"**
- metric：**"我的指标列/表名/口径，是来自 kb-recall 的 metric 段，还是老路 schema？kb-recall 已返回就不许再翻老 schema。"**
