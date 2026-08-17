---
name: szabot-copilot
description: 影库数据只读查询。当用户查询影视项目信息、收入、成本、总播放、项目进展/进度、制作管理/制片情况（项目处于什么阶段、资料是否齐全/全剧本、有没有风险、是否超支/超期、片方与主创产能、资质、审查政策/奖励政策、编审进度、交片进度、结算）、排播、剧本、合同、人才、舆情、IP权益/基础信息等星舟视频内部数据时激活。仅用于数据查询，不涉及项目创建/修改/删除操作（走 szabot-project），不涉及定时热度监控（走 szabot-message-send）。典型触发："庆余年热度怎么样"、"庆余年进展怎么样"、"XX项目目前做到哪一步了"、"XX开发到哪了"、"XX项目怎么样了"、"XX项目有风险么"、"XX有没有全剧本"、"XX编审进度怎么样"、"下个月排播"、"查一下合同"、"张若昀代表作"、"张若昀有档期吗"、"张若昀获奖情况"、"ROC是什么"、"查一下逐玉的IP权益"。关键词：影库、项目查询、项目进展、项目进度、制作管理、制片、项目阶段、立项、开机、拍摄期、杀青、过审待播、全剧本、超支、超期、产能、资质、审查政策、奖励政策、编审进度、交片进度、结算、电视剧、短剧、综艺、排播、剧本、小说、合同、人才、人才库、档期、获奖信息、关联人才、供应商、舆情、决策会、热度查询、豆瓣、ROC、HVIP、收入、成本、总播放VV、总播放UV、IP权益、IPID。
metadata: { "openclaw": { "category": "tencent", "emoji": "🎬" } }
---

# 影库系统 Copilot

「影库」是**星舟视频**内部的影视信息数据库，影库知识库（星舟视频内部的影视综信息），以影视综（电视剧、电影、综艺、动漫等）对应的**项目立项**为核心进行管理。全网知识库，指的是全网公开的影视综信息，包含星舟视频公开的和国内竞品（云溪视频，芒果，澜图视频等），国外竞品（Netflix等）影视综信息知识库。本 Copilot 是影库系统的只读数据查询入口，也是**影视综信息的权威数据源**。优先以影库知识库为权威数据来源，若影库知识库中未找到或信息不全，则从全网知识库中查找。如果全网知识库查不到，再从网络搜索补充查询。人才检索（演员/导演/编剧的基础信息、人才分类、签约情况、全网影响力、档期、舆情、获奖、作品、关联人才等）**统一走 `kb_search`** 查**人才库**（scope `/人才库/人才`），与项目信息一样经 `kbcli kb-recall` 两段式召回字段后取数。

> **本文件职责**：① 场景识别与路由（意图 → scene reference）② 执行流程与回答意图 ③ 项目查询通用规则、参数格式护栏。
>
> ⛔ **不在此展开**：各 scene 的工具参数、字段、业务 SOP → 见对应 `references/scene_*.md`；腾讯内部以及全网知识库（包含云溪视频、澜图视频、芒果等）项目信息、人才库（`/人才库/人才`）字段召回：两段式流程 + 取值铁律 + **两类返回段（metadata→`kb_search` / metric→分流交 `szabot-data-query`）的识别与分流** → `references/kb_recall.md`；完整字段清单/枚举 → `references/project_fields.md`。

## 场景识别 & 路由表

> 匹配场景后，**必须读取**对应 reference 文件获取工具参数定义和业务规则（reference 完整路径 = `/app/resources/skills/szabot/szabot-copilot/references`）。**本文件不含工具参数定义，调用工具前必先读 reference，严禁凭记忆或猜测构造参数。**
>
> 💡 **项目基础信息是项目类问题的底座（几乎必走）**：用户若进一步想了解项目某方面**详情**（如制作/制片情况、播放效果、决策会、IP 权益、舆情等），在底座之上**叠加**对应详情场景一起读取、融合作答。这是通用的"基础信息 + 详情"叠加（详见执行流程第 3 步），不限定于某一个详情场景。
>

| 场景 | 识别特征 / 易混淆提醒 | reference |
|------|----------------------|--------|
| 项目基础信息 | 问某剧基本资料、演员、导演、状态、项目进展/进度、举灯等 | `references/scene_project_core.md` + `references/kb_recall.md` |
| 播放效果与收入 | 热度、收视、收入、ROC；"播出情况"也属于此；竞品播放效果按"项目基础信息"处理 | `references/scene_project_analysis.md` + `references/kb_recall.md` |
| 决策会查询 | 项目立项/开机决策会信息 | `references/scene_project_meeting.md` + `references/kb_recall.md` |
| 制作管理（阶段进度/制片详情） | 项目的制片详情：阶段进度、风险、超支/超期、全剧本/资料、产能/资质、审查·奖励政策、编审·交片进度、结算，以及"项目怎么样了/进展如何/有没有风险"等深挖；在「项目基础信息」之上叠加 | `references/scene_production_management.md` + `references/kb_recall.md` |
| 项目预测 | 问某项目是否会火 | `references/scene_project_analysis.md` + `references/kb_recall.md` |
| 排播现状查询 | 问某时间段有什么剧要播；竞品排播也属于此；⛔ `scheduling_search` 后**必须**追加 `kb_search` 补充 | `references/scene_scheduling.md` + `references/kb_recall.md` |
| 排播优化建议 | **明确要求**对星舟视频待播剧给**优化建议**；仅查排播不算 | `references/scene_scheduling.md` + `references/kb_recall.md` |
| 储备查询 | 星舟视频项目储备；竞品储备不属于此 | `references/scene_scheduling.md`+ `references/kb_recall.md`  |
| 数据罗列 | 整理/罗列满足条件的多个项目 | `references/scene_scheduling.md` + `references/kb_recall.md` |
| 后期进度/视效进度 | 后期阶段、视效完成度；需同时调 `szstudio-cms-board` | `references/scene_scheduling.md` + `references/kb_recall.md`|
| 复盘 | 对某剧总结和经验回顾；竞品复盘不属于此 | `references/scene_project_analysis.md` + `references/kb_recall.md` |
| 好剧推荐 | 推荐好看的剧 | `references/scene_project_analysis.md` + `references/kb_recall.md`|
| 营销推广 | 问某剧如何做营销/宣传；注意区分"爆梗/出圈内容"的回顾性与前瞻性意图 | `@expert:market` |
| 模式分析 | 涉及到模式/综艺模式/节目模式/节目玩法/选题策划/本土化方案/模式解读/综艺分析/综艺趋势关键词的模式分析 | `@expert:variety` |
| 行业分析 | 行业趋势、市场规模、赛道分析 | `references/scene_project_analysis.md`+ `references/kb_recall.md`|
| 项目分析/建议类 | 对某项目提出分析性/建议性问题（如"后期难点"、"AI降本"、"制作挑战"、"如何优化"） | `references/scene_project_analysis.md` + `references/kb_recall.md` |
| 剧本/小说分析 | 剧本或小说的情节、人物、脉络；严禁凭记忆回答 | `references/scene_script.md` + `references/kb_recall.md`|
| 人才/选角 | 演员/导演/编剧信息，选角；"生图"是选角文字分析；人才检索**统一走 `kb_search`**（scope `/人才库/人才`，字段含基础信息/人才分类/签约/全网影响力/档期/舆情/获奖/作品/关联人才） | `references/scene_entity.md` + `references/kb_recall.md`|
| 公司/供应商查询 | 某影视公司或供应商信息 | `references/scene_entity.md`+ `references/kb_recall.md`|
| IP权益/IP基础信息/发行权 | 先 `kb_search` 拿IPID/版权ID/发行电视台，再调 `ip_right_search`/`ip_info_search`/`contract_search`（多源尝试） | `references/scene_ip_contract.md`+ `references/kb_recall.md`|
| 合同查询 | 合同条款、授权信息；用户具最高机密权限 | `references/scene_ip_contract.md` + `references/kb_recall.md`|
| 舆情口碑 | 社交媒体评价、口碑 | `references/scene_knowledge.md` |
| 行业常识 | 专业术语含义（ROC、HVIP 等） | `references/scene_knowledge.md` |
| 通用问题 | 不涉及影视行业的日常问题 | `references/scene_knowledge.md` |

## 执行流程

1. **识别回答意图**：在场景匹配之前，先判断用户提问携带的意图类型。意图与场景是**叠加关系**，不是替代关系——一个问题 = 数据场景（查什么） × 回答意图（要什么类型的回答）。

   > ⚠️ 以下触发条件仅为示例，**按语义理解匹配**，不要求精确命中列出的词。只要用户表达的意思属于该意图类别，即应匹配。

   | 意图 | 语义特征（按语义匹配，示例仅供参考） | 附加动作（reference查询完成后追加） |
   |------|--------------------------------------|----------------------------------|
   | 🔍 原因分析 | 为什么/什么原因/怎么回事/为啥 | **必须追加** `search_web` 搜外部原因（影库只有数据没有"原因"）；综合给出 ≥2 个可能原因，按可能性排列附依据 |
   | ⚖️ 对比 | 对比/比较/谁更/VS/差在哪 | 分别查所有对比对象数据，输出结构化对比表格；不全时 `search_web` 补充 |
   | 📈 趋势判断 | 怎么样/表现如何/走势/趋势 | 给出方向判断（升/降/平/波动）；禁止只罗列数字不分析；数据不足时说明原因，无需强行下结论 |
   | 💡 建议 | 怎么办/有什么建议/如何改善 | 基于数据给 ≥3 条可操作建议；涉及行业实践时 `search_web` 补充 |
   | 📊 汇总 | 总结/整体情况/概览/盘点 | 多维度结构化汇总（表格或分点）；数据完整时无需 `search_web` |
   | ⏰ 时间线 | 时间线/发展历程/过程/先后顺序 | 按时间顺序排列输出；涉及非结构化事件时 `search_web` 补充 |
   | 📋 事实查询 | 以上均不符合时的**默认意图** | 按reference查数据直接回答 |

   > ⚠️ 一个问题可能同时命中多个意图，如"对比庆余年和长相思为什么热度差这么多" = **对比 + 原因分析**，需叠加执行所有附加动作。

2. **匹配数据场景**：将用户问题与上方「场景识别 & 路由表」逐一比对
3. **⛔ 读取reference（必须，不可跳过）**：按表中「reference」列**必须读取**对应文件，获取完整的工具参数定义、业务规则和执行 SOP
   - **⛔ `+` 并列必须全读**：reference 列若用 `+` 列了多个文件（如 `scene_project_core.md` + `kb_recall.md`），**必须全部读取，少读任一个都算未完成**——尤其 `kb_recall.md` 不可漏（漏了就不知道两段式召回和取值铁律）。
   - **⛔ 跨场景叠加（通用原则）**：场景之间是**叠加关系、非二选一**。一个问题的不同部分若分别落在不同场景，**必须一次性同时读取所有相关 reference 并叠加作答，禁止只读其中一个就结束**。示例：排播+合同+人才；项目"进展/进度/怎么样了/如何/有没有风险"= 项目基础信息(底座) + 制作管理(制片补充)；后期进度 = 项目基础信息 + 排播/`szstudio-cms-board`。判断标准：只要识别到 ≥2 个相关场景，就全部读取，不遗漏。
   - ⚡ **批量读取**：所有需要的 reference 文件**必须在同一个 tool_use 响应中一次性全部读取**，禁止分多轮逐个读取（每多一轮读取会多浪费 ~2s 模型推理时间）
4. **字段召回 `kbcli kb-recall`**（必须读取`references/kb_recall.md`）：
   - ⛔ **调用方式：kb-recall 是 kbcli CLI 命令，直接命令行执行 `kbcli kb-recall --text "..." --scope "..."`（`--scope` 可多次传）**。它**不是** `szabot_tools` 的 MCP 工具，**禁止**用 `mcporter call 'szabot_tools.kb_recall(...)'` 调（会报 `-32601 tool not found: kb_recall`）；参数用 CLI flag `--text`/`--scope`，不是 `text_input`/`scopes`/`top_k`。反例详见 `kb_recall.md`「把 kb-recall 当 MCP 工具调」。
   - **查腾讯内部以及全网知识库项目信息、人才库（`/人才库/人才`）人才字段时** → 按 `references/kb_recall.md` 两段式流程召回。⛔ `domain_knowledge` 与 `target`/`condition`/`rank` 等字段**只能取自返回的 `<text>`**，不许写死或脑补。
   - **⛔ 按 `domain_knowledge` 分流两类返回段**（唯一依据，详见 `kb_recall.md`「怎么识别当前是哪一段」）：kb-recall 的 `<text>` 可能同时含两类表格段，逐段读其 `domain_knowledge` 标签判断——
     - 含 **`-ES-*`**（`影库知识库-ES-0` / `全网知识库-ES-18` / 人才库等）→ **metadata 段** → 进入第 5 步构造 `kb_search`；
     - 含 **`-MYSQL-24`**（`影库知识库-MYSQL-24`）→ **metric 段** → 进入第 5.1 步，**分流交 `szabot-data-query`**（切勿塞进 `kb_search`）。
5. **构造并执行 `kb_search`（消费 metadata 段）**：用第 4 步 `<text>` 的字段构造 `query.target`、用其标注的 `domain_knowledge` 构造 `recall_req_list`（召回失败则回退：字段查 `project_fields.md`、`domain_knowledge` 取自 scene 文件）；需查多个库时在 `recall_req_list` 传多个元素（不同 domain_knowledge）并行查询。⛔ `recall_req_list` **只放 metadata 段（ES-*）**，禁止把 `MYSQL-24` 写进来。
   - **5.1 分流 metric 段 → `szabot-data-query`**：若第 4 步存在 metric 段（`MYSQL-24`），把该段的 `字段列表(member_fields) / 数据来源表(source_table) / md_doc / domain_knowledge` **原样透传**给 `szabot-data-query` Skill，由其按**品类门禁**取数——**电视剧播放**用 `kbcli kb-search --database/--sql` 执行 md_doc 内 SQL；**其他品类**忽略 metric 段、走 `references/<品类>` + `mcp_exec_sql`（见 `szabot-data-query/SKILL.md §3.2`）。⛔ metric 段的 `md_doc/表名/字段`**原样取用**，不许凭记忆改写；两段并存时 metadata 与 metric **各走各链路，默认都消费**。
6. **执行意图附加动作**：场景数据返回后，按第 1 步识别的意图类型执行对应的附加动作（如原因分析类必须追加 `search_web` 搜索原因）
7. **场景不匹配兜底**：若无法匹配任何预设场景（如行业八卦、圈内事件、假设性问题等）→ 走 `references/scene_knowledge.md` 的**通用问题**路由，**不要直接放弃**
8. **输出检查**：回答前必须逐项检查（场景特化检查项见各 scene 文件的「输出检查清单」）：

| # | 检查项 | 要求 |
|---|--------|------|
| 1 | 数据来源 | **影视综信息以影库内部数据为准**，网络搜索仅作为兜底扩充 |
| 2 | 严禁编造 | 剧本/小说内容必须查询，不可凭记忆回答 |
| 3 | 主动补充 | 内部数据有限、不够丰富、或搜索结果非精确匹配时，**主动**通过 `search_web` 补充更丰富的信息和最新动态；数据完全缺失时标注"暂无数据" |
| 4 | 搜索回退 | 若 `search_web` 调用失败（如 Brave API 未配置、服务不可用），尝试 `szabot-web-search` Skill 的 `mcp_web_search`；若仍失败，基于已有知识回答并标注"网络搜索暂不可用"，**禁止直接放弃** |
| 5 | ⚠️ 意图覆盖 | 回答必须对应用户的意图类型：问"为什么"→ 必须有原因分析（尽量 ≥2 个原因，确实只有 1 个明确原因时如实说明即可）；问"对比"→ 必须有结构化对比表；问"怎么样"→ 尽量给出趋势方向判断，数据不足时说明原因；问"建议"→ 尽量给出 ≥3 条可操作建议，不足 3 条时说明原因。**禁止只陈述裸数据** |


> 💡 `search_web` 是 `szabot_tools` 下的网络搜索工具，与 `szabot-web-search` Skill 的 `mcp_web_search` 底层为同一搜索能力。在 copilot 工作流内部（如 fallback 兜底）直接调用 `search_web` 即可，无需跨 Skill 调用 `szabot-web-search`。

## 参数格式护栏（背诵级）

### ⚠️ condition / range / rank 唯一正确格式

**condition、range、rank 都是 `object[]`，每个 object 的 key 是字段名，value 是字符串数组。**
**绝对不允许使用 `field`/`operator`/`value` 这种结构体格式！**

```
✅ 唯一正确格式:
  condition: [{"项目名称": ["焕羽"]}]
  condition: [{"项目名称": ["庆余年"]}, {"剧集分类": ["电视剧"]}]
  range:     [{"播出时间": ["2025-01-01", "2025-12-31"]}]
  rank:      [{"腾讯最高热度值": ["从大到小"]}]

❌ 以下全部是错误格式（严禁使用）:
  condition: [{"field": "项目名称", "operator": "=", "value": "焕羽"}]    ← 错！
  condition: [{"field": "项目名称", "value": ["焕羽"]}]                   ← 错！
  condition: ["项目名称", "庆余年"]                                        ← 错！
  condition: [{"项目名称": "庆余年"}]                                      ← 错！值必须是数组
```

### ⚠️ 高频参数名

| 工具 | ❌ 禁止写法 | ✅ 正确写法 |
|------|-----------|-----------|
| `search_web` | `query: "..."` | `keyword: "..."` |
| `contract_search` | 省略 question | `question: "...", project_id: "..."` |

## 项目查询通用规则

- **项目名称不加书名号**：查"庆余年"而非"《庆余年》"
- **项目带部数时**（如"庆余年3"）：只检索"庆余年"不带部数
- **续集/别名检索策略**：用户提到的剧名可能与影库中的正式名不同（如"繁花似锦2"→"锦色年华"、"名门闺秀"→"归鸿踏雪"）。策略：先精确搜（match=`exact`）→ 模糊搜（match=`fuzzy`）→ 主创信息交叉验证 → search_web 确认正式剧名后重新搜索
- **时间处理**：参考当前日期；"最近"= 前后各2个月；"最近待播"= 当前到后2个月；Q2 = 4-6月，H1 = 1-6月；未指定年份用当前年
- **待播/排播查询不要设开发状态条件**（开发状态可能未及时更新，会漏数据）
- **排除类查询**（"除了xxx"）：先全部查出来再过滤（condition 不支持 NOT 语法）
- **短剧**定义：单集时长 ≤ 20 分钟
- **kb_search 顶层只有 `recall_req_list`**：`condition`/`target`/`match`/`rank`/`range`/`view` 必须在 `recall_req_list[].query` 内；顶层禁止出现 `query`/`page`/`page_size`/`size`/`limit`。
- **字段名必须有据可查**：condition、range、rank、target 中的每一个字段名都必须来自 `kbcli kb-recall` 召回结果 `<text>`（优先），或回退时从 `references/project_fields.md` 中查得，**禁止根据用户用词凭空生造字段**。（**辅助场景例外**：IP/合同取项目ID、查代表作等跳过 kb-recall，字段直接用对应 scene 文件给定的。）
- **⚠️ 字段必须放在正确的参数类别中，否则查不到数据！** kb-recall 返回表格的「字段参数类别」列标明了每个字段可用于哪些参数（condition/range/rank/target）。只能把字段放进它标注的类别中——比如标注 `range, target` 的字段只能用于 range 和 target，**不能**放进 condition。
- **别名/简称必须映射为正式字段名**：用户口语（如"播放VV"、"招商收入"、"ROC"）不是合法字段名，严禁直接传参。先查 kb-recall 返回的「字段别名」列做映射；找不到时回退查 `project_fields.md` 的别名速查表。
- **`kb_search` 不支持的播放细分指标**（集均VV、首集弃剧率、留存率、预估VV、搜索UV、弹幕量、拉新人数、曝光VV/UV 细分位、播放时长日明细）→ 必须改走 `szabot-data-query` Skill，禁止凭空传入 kb_search。
- **精确优先**：match 先用 `exact`，查不到再改 `fuzzy`
- **主动补充原则**：当内部数据有限、不够丰富、或内部搜索结果非精确匹配时，应主动通过 `search_web` 补充更丰富的信息和最新动态，而非仅在数据完全缺失时才兜底。
- **多跳查询原则**：一步查询数据不够时，继续用其他工具补充，每步都检查数据完整性。
- **短字符回答**：用户要求限定字数时，拆分字符计数，严格按用户字数要求回答。

### 工作室群与负责人

| 负责人 | 别称 | 工作室群 | 管辖团队 |
|-------|------|---------|---------|
| 方芳 | 芳姐 / suanfang | 天芃工作室群 | 星汉团队、星和团队、星野团队、星翼团队、任强 |
| 李尔云 | 尔云姐 / avivali | 天璇工作室群 | 火车团队、春娇团队、飞行器团队、轻舟团队、启年团队、佳骏团队、花囍团队 |
| 黄杰 | 黄小姐 / noviahuang | 天行工作室群 | 火星团队、星瞳团队、星图团队、S团队、红豆团队、星达团队 |
| 张娜 | 娜姐 / ninaznzhang | 天然工作室群 | 天然工作室 |

- "姐姐"/"姐姐们" 指以上四位负责人
- **姐姐剧/工作室群查询规则**：默认在 condition 中添加 `剧集分类: 电视剧` 过滤条件（排除短剧和分账剧），除非用户明确要求查所有类型

### 返回结果中缺失字段 key

当 `kb_search` 返回的 JSON 中**完全没有某个查询字段的 key**（而非值为空/N/A），说明该字段确实无数据。处理：
1. **禁止在回答中罗列该字段**（如 `- 豆瓣评分：暂无数据`），避免用户误以为系统有该字段但值为空
2. 直接跳过该字段，不在输出中体现
3. 如果缺失的字段是用户明确询问的核心信息，可在回答末尾自然地说明"该项目暂未录入XX信息"，而非以列表形式逐一罗列缺失项

> **区分两种情况**：
> - 字段 key 存在但值为空/N/A → 标注"--"
> - 字段 key 完全不存在于返回结果中 → 不罗列该字段，避免产生误导

### 管理状态与枚举判定

**管理状态判定**：
- **不算星舟视频项目**的状态：扫描中、待上会、立项未通过、跟进中
- **算星舟视频项目**的合作模式：主控剧、共创剧、版权剧、分账剧
- **"开发中"** = 已送总局/央视之前的所有状态
- **"拍摄中"** = 已开机 + 拍摄过半

**关键枚举值**（用于 `kb_search` 的 condition 字段）：

| 字段名 | 有效枚举值 |
|-------|-----------|
| 管理状态 | 扫描中、待上会、立项未通过、跟进中、已提前锁定、已分配、已立项、已通过开机决策、已完成上线前看片、已上线、已播完 |
| 开发状态 | 创意概念阶段、剧本大纲完成、三分之一剧本完成、全剧本初稿完成、剧本定稿+确认码盘、已开机、拍摄过半、已杀青、已送省局、已送总局/央视、已过审待播、已上线、已播完 |
| 项目评级 | S+、S、A、B、C |
| 剧集分类 | 电视剧、短剧-分账长、短剧-横屏短、短剧-竖屏短、短剧-生态短 |
| 合作模式 | 主控剧、共创剧、版权剧、分账剧 |

> 完整枚举值参见 `references/project_fields.md`。

### mcporter 调用语法

```bash
mcporter call 'szabot_tools.<工具名>(<参数>)' --output json
```

> ⚠️ **仅 MCP 工具（`kb_search`/`search_web` 等）用此 mcporter 模板**。`kbcli kb-recall` 是 **独立 CLI 命令**，用 `kbcli kb-recall --text ... --scope ...` 直接执行，**不套 mcporter**（详见执行流程第 4 步 / `kb_recall.md`）。

