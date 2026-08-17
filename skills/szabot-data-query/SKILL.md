---
name: szabot-data-query
namespace: szbot
trust-level: builtin
category: data-query
version: 1.8.0
description: "影库数据查询，覆盖六类场景：①电视剧/长视频/分账长播放指标（VV、UV、曝光、正片播放、热度值、拉新、弹幕、搜索UV、预约、完播率、留存率、弃剧率、预估VV、集均、豆瓣评分、播放转化漏斗等）；②电视剧项目（大盘）预算、预算趋势（四级预算、六级预算、单集成本、集均成本涨跌幅、线上/线下成本、IP费用、剧本、演员、宣发、承制、各组费用等，支持大盘汇总与项目粒度查询）；③综艺项目粒度播放指标（VV、UV、播放时长、热度、首播UV等）；④综艺大盘指标（播放VV/UV、有效UV、曝光点击、收入、正价开通人数等）；⑤动漫/动画（AIGC/SzCanvas）算力、开销、消耗、花费（素材数量、费用、模型消耗、素材类型分布、下载率等）；⑥制片进度和风险（进度、超支、风险等）。"
---

# 影库数据查询技能

查询电视剧/综艺的播放指标与预算数据，以及动漫/动画算力、开销、消耗、花费数据。**取数执行器按品类分流**（见下）。

> 🧭 **双链路 —— 按「品类门禁」分流，不是全局开关**：
>
> | 品类 | 拼 SQL 依据 | 执行器 |
> |---|---|---|
> | **电视剧播放** | kb-recall metric 段（`影库知识库-MYSQL-24`）返回的 **`md_doc` 内 SQL** | **`kbcli kb-search --sql`** |
> | 电视剧预算 / 制片进度 / 综艺 / 动漫·SzCanvas | `references/<品类>/sql_query.md` + `schema/*.md` | `szabot_data_query_svr` · `mcp_exec_sql` |
>
> ⛔ **铁律1（电视剧播放）**：md_doc 可用时**禁止**再翻 `references/dianshiju/` 老 schema 去补/换指标。
> ⛔ **铁律2（其他品类）**：**不要**尝试用 kb-recall metric 段或 `kbcli kb-search` 取数，一律走 `references` + `mcp_exec_sql`。
> 🛟 **紧急兜底**：仅当电视剧播放的 md_doc 不可用（未命中/为空/不含 SQL/执行失败）时，才允许回落 `references/dianshiju/` + `mcp_exec_sql`，且输出须注明「经 references 兜底（md_doc 不可用）」。

> references 对应的完整路径为 `/app/resources/skills/szabot/szabot-data-query/references`

---

## 1. 触发判断

满足以下**任一模式**即触发本 Skill：

**模式 A：明确查询** —— 品类关键词 + 指标关键词同时命中（每列内命中任一即可）：

| 品类关键词 | 指标关键词 |
|-----------|-----------|
| 电视剧、长视频、分账长、某部具体剧名 | 播放VV/UV、曝光、正片播放、热度值、拉新、弹幕、搜索UV、预约、完播率、留存率、弃剧率、预估VV、集均、豆瓣评分、播放时长、播放转化漏斗 |
| 制片进度+风险 | 制片进度、拍摄进度（时间/页数/场次/延期）、预算执行进度、预算超支异常科目、分科目实际使用与总预算、角色完成进度、制片概览/当前状态、超支 |
| 电视剧 + 预算/成本 | 四级/六级预算、单集成本、线上/线下成本、IP费用、演员费用、宣发费用、承制费用、各组费用等（完整列表见品类路由表） |
| 某个具体综艺节目名称 | 播放VV/UV、播放时长、热度、首播UV |
| 综艺大盘、综艺播放、综艺数据（无具体项目名） | 播放VV/UV、有效UV、精选页/综合页曝光点击、收入、正价开通人数 |
| 动漫、动画、AIGC、SzCanvas、算力 | 素材数量、费用、开销、消耗、花费、模型消耗、素材类型分布、下载率 |

**模式 B：模糊查询** —— 品类关键词命中 + 用户表达了数据查询意图但未指明具体指标：
- 触发词示例：播放情况、数据怎么样、表现如何、播得好不好、收视情况、最近怎么样 等泛化表述
- 典型场景："逐玉播放的咋样""庆余年数据怎么样""浪姐最近表现如何"
- 处理方式：触发后按品类路由到对应查询模板，**默认查询该品类的核心指标**（如电视剧默认查 VV/UV/热度，综艺默认查 VV/UV/播放时长）

**❌ 不触发**：
- 无品类关键词、也无法从上下文推断品类的纯指标查询
- DAU 等非播放指标（综艺大盘的收入、正价开通人数除外）
- 动漫/动画播放类指标（播放VV/UV、热度等） → 按剧集分类走对应品类（电视剧/综艺）
- 短番/漫剧/赤霄/SzStudio → `short-anime-data-query`
- 侵权数据 → `szpp-data-query`
- 通用影视综项目信息（不涉及播放/预算指标） → `szabot-copilot`
- 综艺项目累计指标（累计播放量、上线以来总和、总计等口径），可以召回（总播放VV、总播放UV、3s有效播放VV、正片播放VV、正片有效播放UV、正片有效播放市场占有率、周均播放VV、人均播放时长（分钟）等） → `szabot-copilot`

---

## 2. 品类识别与路由

先根据用户意图识别品类。品类结果有两处用途：**① 老链路兜底时定位 `references/<品类>` 目录；② 新链路 kb-recall 的 catalog scope 参考。**

| 品类 | 判定条件 | 兜底目录（references） |
|------|---------|-------|
| 电视剧播放 | 用户提到电视剧/长视频/某部剧名，或 `kbcli kb-search` 返回剧集分类为「电视剧」「短剧-分账长」「短剧-横屏短」 | `references/dianshiju/` |
| 电视剧预算 | 用户提到预算、成本、费用等预算类关键词，且涉及电视剧项目 | `references/dianshiju_fee/` |
| 制片进度 | 用户提到进度、风险、预算执行（进度/异常/超支）、延期情况、制片概览、风险等关键词，且剧集分类为「电视剧」「短剧-分账长」「短剧-横屏短」 | `references/zhipian/` |
| 综艺项目 | 用户提到具体综艺节目名，或 `kbcli kb-search` 返回剧集分类为「综艺」 | `references/zongyi/` |
| 综艺大盘 | 用户提到「综艺大盘/综艺播放/综艺数据」且无具体项目名 | `references/zongyi/` |
| 动漫/动画 | 用户提到「动漫」「动画」「AIGC」「SzCanvas」「算力」关键词，或单独提到「算力」「费用」「开销」「消耗」「花费」+ SzCanvas/SzStudio/Rally 上下文，或 `kbcli kb-search` 返回剧集分类为「动漫」 | `references/donghua/` |

> 🚨 **综艺项目仅支持日增量查询**：仅支持「单日 / 按天分布」的播放VV/UV、播放时长、热度、首播UV等日增量指标；**不支持累计指标**（如累计播放量、上线以来总和、总计等口径）。若用户问累计口径，需直接告知不支持。
>
> 🚨 **动漫/动画仅支持算力、开销、消耗、花费查询**：支持素材数量、费用、模型消耗、素材类型分布、下载率等 AIGC 算力和预算相关指标；**不支持播放类指标**（播放VV/UV/热度等），播放类指标按剧集分类走对应品类路由。

---

## 3. 执行流程（品类门禁 → 执行器分流）

### 3.1 前置：获取项目ID（所有品类通用，走 `kbcli kb-search --query`）

**目的**：为后续 SQL 查询提供 WHERE 条件中的项目过滤值。

**适用范围**：电视剧播放、电视剧预算、综艺项目、动漫/动画（即所有**涉及具体项目名**的查询）。**若用户问的是综艺整体数据而未提及任何具体节目名（如「综艺大盘播放怎么样」「综艺整体数据」），则无需获取项目ID，跳过此步直接进入 3.2。动漫/动画大盘算力查询（未指定项目）同理，跳过此步。**

#### 判断是否需要调用

| 情况 | 处理 |
|------|------|
| 上下文已有**项目ID + 剧集分类** | ✅ 直接使用，跳过调用 |
| 上下文已有**项目ID**，但**缺剧集分类** | 调用 `kbcli kb-search --query` 查回剧集分类（见下方命令 A） |
| 用户只给了项目名称，**无项目ID** | 调用 `kbcli kb-search --query` 获取项目ID和剧集分类（见下方命令 B） |
| 用户未指定任何项目条件（如「查所有项目的六级预算」） | 跳过此步，直接进入 3.2 查全部数据 |

**命令 A：有项目ID，缺剧集分类**

```bash
kbcli kb-search \
  --domain-knowledge "影库知识库-ES-0" \
  --query 'SELECT 剧集分类 WHERE 项目ID = "<上下文中的项目ID>"'
```

**命令 B：只有项目名称，无项目ID**

```bash
kbcli kb-search \
  --domain-knowledge "影库知识库-ES-0" \
  --query 'SELECT 项目ID, 项目名称, 剧集分类 WHERE 项目名称 = "<用户提到的剧名/综艺名>"'
```

**`--query` 语法**：`SELECT <字段,字段> [WHERE 字段 = "值" AND 字段 IN ("值1","值2")]`
- `SELECT` 后 = `target`（取字段）；`WHERE` 后 = `condition`（过滤条件），只支持 `=` 和 `IN`，用 `AND` 连接，**值必须带引号**。
- ⚠️ `match` 固定为 `fuzzy`（CLI 不可指定 `exact`）。项目ID 是唯一标识，fuzzy 亦可命中；若发现多召，用返回的项目名称二次确认。
- ⚠️ `--query` 与 `--database/--sql` **互斥**，不可同时传；`--query` 值须压成一行。
- ⚠️ 一次只能查一个 `--domain-knowledge`；需查多库请分多次调用。

> ⚠️ `SELECT` 只取定位所需字段，播放/预算数据由后续 SQL 查询获取，**禁止在 `SELECT` 中添加播放指标字段**。
> ⚠️ 此处的 `kb-search`（`--query` 模式，`domain_knowledge=ES-0`）只用于**定位项目ID/剧集分类**（metadata 段）；取指标用的是 `--database/--sql` 模式（metric 段），两者是两回事，别混用。

#### SQL 过滤字段选择

拿到项目ID后，SQL `WHERE` 子句的过滤字段：**新链路以 md_doc 内 SQL Demo 的过滤列为准；老链路以对应品类 `sql_query.md` / `schema/*.md` 为准**。
- **优先用项目ID过滤**（精确、唯一）
- **仅有项目名称时**退化为名称字段过滤（要求精确匹配，禁止 LIKE）

### 3.2 取数主流程（决策树：品类门禁 → 执行器选择）

```
拿到项目ID后，拼 SQL 前先判断走哪条链路：

品类 == 电视剧播放？
 ├─ 否（电视剧预算/制片进度/综艺/动漫SzCanvas）
 │    └─→ 【老链路 · §3.4】references + mcp_exec_sql（不要碰 metric 段/kbcli）
 └─ 是
      └─ kb-recall 命中 metric 段(影库知识库-MYSQL-24) 且 md_doc 含可执行 SQL？
           ├─ 是   → 【新链路 · §3.3】md_doc SQL + kbcli kb-search --sql
           └─ 否   → 【紧急兜底 · §3.4】references/dianshiju + mcp_exec_sql
                     （输出须注明「经 references 兜底（md_doc 不可用）」）
```

- ⛔ **铁律**：电视剧播放且 md_doc 可用时，禁止再翻 `references/dianshiju/` 老 schema 补/换指标。
- ⛔ **反向铁律**：非电视剧播放品类，禁止用 metric 段/`kbcli kb-search` 取数。
- 自检口令：**"我查的是哪个品类？电视剧播放才走 md_doc + kbcli；其余一律 references + mcp_exec_sql。"**

### 3.3 【新链路 · 仅电视剧播放】metric md_doc + kbcli kb-search

1. **取 md_doc**：直接采用 kb-recall metric 段返回的 `md_doc` 内**可执行 SQL Demo**，绕过 `references/dianshiju/`。
2. **db_name**：取 md_doc 首部注释 `-- db_name: xxx`（通常为 `szbot`，画像组 `produce`），作为 `--database` 传入；**禁止**沿用老路 `szdw_ads`。
3. **执行器**：用 **`kbcli kb-search`**（不是 `mcp_exec_sql`）：

```bash
kbcli kb-search \
  --domain-knowledge "影库知识库-MYSQL-24" \
  --database "<md_doc 的 db_name>" \
  --sql "<md_doc 的可执行 SQL，压成一行>"
```

- ⚠️ `--database` 与 `--sql` **必须同时传**（缺一报 `InvalidQuery`）；且**不能**与 `--query` 同时使用。
- ⚠️ `--sql` 值必须是**单行**（不能含换行），否则 shell 会截断命令。
- ⚠️ 返回结果包在 `<text>...</text>` 内。

4. **口径切片**：累计按用户口径词切 `is_operation_accu` / `is_latest_accu` / `is_value_accu`；日增用 `*_inc`；**勿对 `is_*_accu` 叠加求和**。
5. **字段范围**：`member_fields / source_table / md_doc` 原样取用，不增不删不换不筛选，禁止凭记忆补指标列/改表名。
6. metric 段（`--database/--sql` 模式）与 metadata 段（`--query` 模式）**分开调用**，禁止把 metric 的表/SQL 塞进 `--query`，也禁止把 metadata 字段塞进 `--sql`（见 `kb_recall.md` 分流规则）。

> ⚠️ **命中 ≠ 可用**：只有 `md_doc` **存在且含可执行 SQL** 才算新链路可用。若 metric 段命中但 `md_doc` 为空/拉取失败/不含 SQL → **视为未可用，回退 §3.4 紧急兜底**；**禁止**用 `member_fields`+`source_table` 裸拼 SQL（缺口径切片，无法保证 `is_*_accu` 正确），输出注明"经 references 兜底（metric md_doc 暂不可用）"。

### 3.4 【老链路 · 其他品类 + 电视剧播放紧急兜底】references/schema + mcp_exec_sql

**适用**：① 电视剧预算 / 制片进度 / 综艺 / 动漫·SzCanvas（**常规路径**）；② 电视剧播放的 md_doc 不可用时（**紧急兜底**）。

1. **加载知识库**：在「§2 品类识别与路由」查到品类对应目录，读取该目录下的 `sql_query.md`
2. **生成 SQL**：严格按 `sql_query.md` 中的流程执行（含读取 schema、拼写 SQL），**禁止自行猜测计算方式**
3. **执行查询**：通过 `mcp_exec_sql` 执行（`db_name` 等调用参数见对应品类 `sql_query.md`）

⚡ **批量读取**：`sql_query.md` + 所有需要的 `schema/*.md` 文件**必须在同一个 tool_use 响应中一次性全部读取**，禁止分多轮逐个读取（每多一轮读取浪费 ~2s 模型推理时间）。

### 3.5 ⛔ 执行 SQL 之前

- **新链路（电视剧播放）**：必须先**完整读完** metric 段 md_doc 的 SQL 契约与 db_name，再拼 SQL 交 `kbcli kb-search`。
- **老链路**：必须先**完整读完** 3.4 加载的 `sql_query.md` 与对应 `schema/*.md`，再拼第一条 SQL 交 `mcp_exec_sql`。
- 两条链路共通：禁止凭印象/记忆动手，禁止"边猜边查"。

---

## 4. 口径与计算规则

### 口径别名

| 用户可能的说法 | 等价含义 |
|-------------|---------|
| 上新期 | 运营期 |
| 播放VV | 播放量 |
| UV | 人数（如播放UV = 播放人数） |
| 移动端 | app端 |
| 正价开通人数 | 正价驱动开通人数 |
| 费用 / 费 / 成本 | 含义相同（如演员费用 = 演员成本） |

### 计算红线

1. **指标口径**：**新链路以 md_doc 内 SQL Demo/说明为准；兜底以 `schema/*.md` 字段 COMMENT 为准**，两者都禁止自行推导。
2. 🚨 **日期补全**：用户说「4月28日」未指定年份时，以**当前系统时间年份**补全，严禁用训练截止年份
3. 🚨 **「昨天」处理**：以系统日期 - 1 推算具体日期写入 `WHERE imp_date = 'YYYYMMDD'`，严禁用 `MAX(imp_date)` 代替
4. 🚨 **输出年份一致**：回答中描述的日期年份必须与 SQL 实际查询的日期分区完全一致
5. **时间聚合**：只能求平均，不能求和展现（电视剧多日趋势除外）
6. 🚨 **计算下沉**：单位换算、加和、比率、占比、日期差等**所有派生计算**必须在 SQL `SELECT` 中完成，严禁大模型心算

---

## 5. 强约束（红线）

| # | 约束项 | 规则 |
|---|-------|------|
| 1 | **执行器（按品类）** | **电视剧播放**用 `kbcli kb-search --domain-knowledge/--database/--sql`；**其他品类**用 `szabot_data_query_svr` 的 `mcp_exec_sql`。⛔ 不可互换 |
| 2 | **取数来源（按品类）** | **电视剧播放**以 metric 段 `md_doc` 为准（md_doc 不可用才兜底 `references/dianshiju`，须注明）；**其他品类**一律 `references/<品类>` |
| 3 | **指标名** | 与来源逐字一致：电视剧播放以 md_doc 列名为准，其他以 `schema` 字段 COMMENT 为准（仅允许去掉末尾 `-三端`/`-app端`/`-pc端`/`-ott端` 后缀），禁止翻译/美化/缩写/编造 |
| 4 | **字段来源** | SQL 所有字段必须来自 md_doc（电视剧播放）或 `schema/*.md`（其他品类），严禁用 `DESCRIBE`/`SHOW COLUMNS`/`INFORMATION_SCHEMA`/`SELECT *` 反查 |
| 5 | **输出前自检** | 老链路按对应品类 `sql_query.md` 末尾的「输出前自检清单」逐条核对；新链路核对 md_doc 契约，任何一条不过立即修正 |
| 6 | **数据溯源** | 输出结尾附带执行时间及数据来源 |

> 以上为速览，老链路完整规则详见各品类 `sql_query.md` 顶部的「指标名强约束」和「字段来源强约束」区。

---

<!-- ============================================================= -->
<!-- metric 桶（MYSQL-24）补充说明 —— 当前仅电视剧播放启用            -->
<!-- ============================================================= -->

## 6. metric 桶（影库知识库-MYSQL-24）补充说明

> **背景**：统一召回服务上线 metric 桶（`domain_knowledge=影库知识库-MYSQL-24`），指标组定义在无极表 `t_mertic_group`。kb-recall 返回的 metric 段带 `字段列表(member_fields) / 数据来源表(source_table) / md_doc`。
> **当前启用范围 = 仅「电视剧播放」品类**，执行器为 `kbcli kb-search --sql`；其他品类不使用 metric 段。取数决策见 §3.2。

### 6.1 新链路（md_doc）与老链路（references）是两套表和口径

| 维度 | 老链路 references | 新链路 md_doc（MYSQL-24） |
|---|---|---|
| 适用品类 | 电视剧预算/制片/综艺/动漫（常规）+ 电视剧播放（紧急兜底） | **仅电视剧播放** |
| 执行器 | `mcp_exec_sql` | **`kbcli kb-search --database/--sql`** |
| 来源表 | `szdw_ads.ads_szbot_dianshiju_project_consume_metrics_ql_df` 等 | `szbot.dws_szbot_dianshiju_project_metrics_df` 等（少数 `produce.` 如画像） |
| db_name | `szdw_ads` / `szdw_dim` / `dianshiju_fee` | `szbot`（画像组=`produce`），以 md_doc 首部注释为准 |
| 累计取法 | 先算 `tongji_end_date=LEAST(最新分区, operation_end_date)` 再取快照（只有"运营期累计"） | 直接切片 `is_operation_accu` / `is_latest_accu` / `is_value_accu`，另有 `*_inc` 日增 |
| 字段风格 | `sxq_*`、`history_max_hot_value` | `total_income`、`member_related_income` 等语义列名 |
| SQL 来源 | 品类 `sql_query.md` + `schema/*.md` | **md_doc 内已含完整可执行 SQL Demo** |

### 6.2 catalog 树共用（scope 注意）

- metric 桶与 metadata 桶**共用同一棵 catalog 树**，靠 `domain_knowledge`（`影库知识库-MYSQL-24` vs `影库知识库-ES-*` 等 ES 库）区分，**不是靠独立路径/目录**。metric 组挂在与 metadata 同名的 L1/L2 tag 下（如 `dsj_base_info`/`dsj_dim` 挂 `/影库知识库/电视剧/项目信息/项目基础信息`）。
- 故上游 kb-recall 的 `--scope` 收窄会连同该 tag 下的 metric 一并过滤，不确定时放宽到品类层或全域召回（详见 `kb_recall.md`「scopes 构建规则」）。

### 6.3 新链路启用范围（当前状态）

| 品类 | 进无极表状态 | 当前取数方式 |
|---|---|---|
| **电视剧播放**（`dsj_*` 19 组） | ✅ 已入表 | ✅ **新链路已启用**：md_doc + `kbcli kb-search --sql`（md_doc 不可用才兜底 `references/dianshiju`） |
| 电视剧预算（四级/六级预算） | ❓ **未确认入表** | ⛔ 新链路**未启用**：走 `references/dianshiju_fee` + `mcp_exec_sql` |
| SzCanvas 算力/费用/消耗 | ⏳ 待补录 | ⛔ 走 `references/donghua` + `mcp_exec_sql` |
| 制片进度/超支/风险 | ⏳ 待补录 | ⛔ 走 `references/zhipian` + `mcp_exec_sql` |
| 综艺项目/大盘 | ⏳ 待补录 | ⛔ 走 `references/zongyi` + `mcp_exec_sql` |

> 📌 **扩品类前置条件**：某品类要启用新链路，须先验证 kb-recall 能召回该品类 metric 段且 md_doc 含可执行 SQL（验证命令见 `kb_recall.md`），确认后再改 §3.2 决策树的品类门禁。
