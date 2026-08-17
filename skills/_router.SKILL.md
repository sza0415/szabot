---
name: szbot
description: 影视综信息、播放/数据、项目/草稿/评估单、文件、看片、定时热度、专家问答
---

# 影库命名空间路由入口（szbot SKILL）

> **本文件职责**：① 快速路由（意图 → 技能）② 多 Skill 联动 / 强制组合 ③ 跨 skill 消歧与优先级 ④ 公共护栏指针。
>
> ⛔ **不在此展开**：各 skill 的参数、字段、枚举、口径、SQL 模板、ROC 取数、展现模板等**具体业务规则**——它们的**单一真源**是各子 skill 的 `SKILL.md` 与 `references/`，按需 `read`。
>
> **通用约定**：
> - **入口路径** = 本目录下 `<skill-name>/SKILL.md`（如 `szabot-copilot/SKILL.md`）。
> - **先读后调**：调用任何工具前先读对应 `SKILL.md` + `references/`，禁止凭记忆/猜测构造参数。
> - 任何用户输入先当**查询请求**处理；命中路由即走，多命中或歧义见 §2。
> - ⛔ **短词/短语（含像成语/祝福/日常用语的，如"静水流深""主角""来战"）默认是影视综实体（剧名/人名/公司名），必须先查（路由到 `szabot-copilot` 等去搜索），禁止当闲聊/问候直接回复**；不确定也先查，查无再说。
> - 📌 **短词分流（A 具名实体 / B 动作意图）**：短词先判类别——**A 具名实体**（能当作一个具体名字检索：剧名/人名/公司名/IP，如"静水流深""来战"）→ 直接路由查询，查无再说；**B 动作/泛意图**（动作或泛类目、缺具体实体宾语，如"找演员""查数据""看进度""分析一下"）→ 缺可检索实体，**先向用户澄清补全对象再查**，⛔ 不得把动作词当实体名硬查、也不得凭理解直接作答。**判据 = 能否作为一个具体实体名去检索**。

---

## §1 路由速查表（意图 → 技能，按 5 类）

> 命中即走；组合类（多技能）见 §2；歧义见 §3。入口路径 = `/app/resources/skills/szabot/<skill-name>/SKILL.md`。

### A. 数据查询（只读，`data-query`）

| 用户意图关键词 | 技能(skill-name) | 备注 |
|---|---|---|
| 影视综信息/项目/进展/IP/人才/合同/舆情/术语；累计口径（总播放/热度/ROC/收入/会员） | `szabot-copilot`（权威源） | 术语、人才、公司等具体规则见其 SKILL.md / reference |
| 播放数据/正片播放/播放量/播出情况/VV/UV/完播率/热度值/留存/拉新/弃剧/弹幕/预约/预估VV/集均/成本/日增量明细 | `szabot-copilot` + `szabot-data-query` | ⚡组合，见 §2 |
| 后期进度/视效进度/中后期制作 | `szabot-copilot` + `szstudio-cms-board` | ⚡组合，见 §2 |
| 漫剧/短番/赤霄/SzCanvas + 播放/收入 | `short-anime-data-query` | 必须含这些关键词，否则走 copilot |
| SzStudio/SzCanvas/Rally + 制作/素材/进度/视效/制作量/DAU | `szstudio-cms-board` | ⛔ 制作 ≠ 播放（播放走 short-anime） |
| 侵权/下架/发函/下架率/防护/侵权趋势 | `szpp-data-query` | 仅此技能，无回退 |
| 网络搜索/最新信息补充 | `szabot-web-search` | 回退关系见 §2 |
| SzCanvas + 算力/费用/开销/消耗/花费/模型消耗 | `szabot-data-query` | 仅此技能，无回退 |
| 拍摄进度、预算、预算趋势、预算浮动、超支、角色进展、角色进度、风险 | `szabot-data-query` | 仅此技能，无回退 |

> **高频歧义词**：
> - `预算` ≠ `成本`：「预算」指立项预算金额；「成本」指实际发生的消耗费用，两者不互通。
> - `主角` 单独出现或作为查询对象出现时，优先视为星舟视频电视剧项目名，走 `szabot-copilot`；除非有明确语义`XX 的主角`、`男主角`、`女主角`是主演的本义。

> **【取数链路】**（只影响 `szabot-data-query` 内部**用什么依据拼 SQL、由谁执行**，**不改变**本表的技能路由）：
> - **项目ID/剧集分类定位**（所有品类通用）→ **`kbcli kb-search --query`**（ES 段）。
> - **电视剧播放** → **新链路**：kb-recall 命中 metric 段（`影库知识库-MYSQL-24`）→ 取其 `md_doc` 内 SQL → 由 **`kbcli kb-search --sql`** 执行。
> - **其他品类**（电视剧预算 / 制片进度 / 综艺 / 动漫·SzCanvas）→ **老链路**：`references/<品类>/sql_query.md` + `schema/*.md` 拼 SQL → 由 **`mcp_exec_sql`** 执行。
> - 决策树详见 `szabot-data-query/SKILL.md §3.2`。

### B. 写操作 / 管理（`write-ops`）

| 用户意图关键词 | 技能(skill-name) |
|---|---|
| 创建/修改/删除草稿或项目、关联文件到项目 | `szabot-project` |
| 评估单（查看/创建/填写/更新/提交） | `szabot-estimate` |
| 定时/每隔/cron/自动 + 热度监控（普通一次性热度走 copilot） | `szabot-message-send` |

### C. 文件处理（链式，`file`）

| 用户意图关键词 | 技能(skill-name) | 备注 |
|---|---|---|
| 上传本地文件拿 fid | `szabot-file-uploader` | 原子能力，被其它文件 skill 依赖 |
| 解析/提取文件内容为文本 | `szabot-file-parser` | 依赖 uploader；⛔ 只解析不填写 |
| 模板填写/填表/模板填充/用XX信息填XX文件 | `szabot-file-editor` | ⚡组合，见 §2；⛔ 非 parser |
| 小说分析/剧本分析（须明确说"小说/剧本分析"） | `szabot-novel-analysis` | 依赖 uploader/parser；仅"分析"二字不触发 |

### D. 媒体播放（`media`）

| 用户意图关键词 | 技能(skill-name) |
|---|---|
| 看片/看样片/看成片/看素材/项目视频 | `szabot-play` |

### E. 专家入口（`cli`）

| 用户意图关键词 | 技能(skill-name) | 备注 |
|---|---|---|
| `@expert:{name}`（营销 market / 模式·综艺 variety / 剧本小说 等） | `kbcli` | 按 expert_name 匹配 `--agent-id`，详见其 SKILL.md |

### F. 关联系统

- 星图（媒资，meizi）: 星图媒资素材查询（CID 下的视频/图文/OTT/高清剧照/生态素材）, alias:`skill:meizi-material`(file:/app/resources/skills/meizi-material/SKILL.md)
- szcanvas（AIGC）: AI生图/生视频/无限画布, alias:`skill:szcanvas`(file:/app/resources/skills/szcanvas/SKILL.md)

---

## §2 多 Skill 联动 / 强制组合（⛔ 缺任一 skill 视为执行错误）

> 本节只定义**跨 skill 的协作契约**；各 skill 内部如何取数/展现，见其自身 SKILL.md。

| 触发意图 | 必须的 skill 链（缺一不可） | 协作契约（关键约束） |
|---|---|---|
| 播放数据/播放量/VV/UV/完播率/热度/留存/拉新/弹幕/预约/预估VV | ① `szabot-copilot`（kb_search 取基础数据）→ ② `szabot-data-query` | ⛔ 禁止只调 copilot 就结束；**查询意图必须透传**（指标/时间/口径累计单日趋势/对比维度/项目ID 列表；多项目≥2 一次性全量传入并标"多项目对比"，下游按项目分别展现，**禁合并/求和**）；下游展现模板**原样落地**（区块/指标行/顺序逐字输出，禁改写删减） |
| 后期进度/视效进度 | ① `szabot-copilot`（后期字段）→ ② `szstudio-cms-board`（素材制作进度） | 两者联合，缺一不可 |
| 模板文件填写（短文本待填，如项目名） | ① `szabot-copilot`（kb_search 查全量信息）→ ② `szabot-file-editor` | 短文本待填先经 copilot 查全量，再多字段填写 |
| 文件解析→加工链 | `szabot-file-uploader` → `szabot-file-parser` → {`szabot-file-editor` / `szabot-novel-analysis`} | 后两者依赖前置拿 fid + 解析 |
| 网络搜索补充 | `szabot-copilot` 内 `search_web` ↔ `szabot-web-search` | copilot 工作流内用 `search_web`；独立场景用 `szabot-web-search`；互为回退 |

> ⚡ **预加载优化**：识别到播放数据意图后，在发起 copilot Phase-1 并行调用（kb_search + report_search + search_web）的**同一轮**，同时 `read` `szabot-data-query/SKILL.md` + 其 references/schema，使等待与加载重叠。

---

## §3 公共护栏（跨所有影库 skill 的共性；全局护栏见根 AGENTS.md）

> 已继承 AGENTS.md「全局护栏」：先读后调 / Skill 名 ≠ MCP Server 名 / 禁内置搜索 / 不编造估算 / 失败不放弃 / 职责分工。影库补充：

1. **禁凭记忆构造参数**：字段名、枚举值必须从对应 skill 的 `references/` 读取，禁止凭 `mcporter list` 签名猜测。
2. **禁跨 Skill 混用 MCP**：每个 skill 用各自独立 MCP 服务；⛔ 禁止把 skill 名当 server 名（部分 skill 经脚本而非 mcporter，按其 SKILL.md 执行）。
