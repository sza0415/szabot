# 电视剧项目预算分析

## 🚨 执行流程总则

**Step 0 项目ID获取** → **Step 1 表路由** → **Step 2 读取 schema** → **Step 3 执行 SQL** → **Step 4 结果展现**，严格按序，**禁止跳步、禁止少步、禁止合并**，五步必须全部执行，违反即作废重来。

---

## Step 0 · 项目ID获取（前置）

🚨 **强制规则**：用户指定了项目名称、管理状态等**表中不存在的字段条件**时，**必须先调用 `szabot-copilot` 技能的 `kb_search` 工具**获取项目ID列表，将返回结果中的「项目ID」字段值直接代入 SQL 模板的 `AND pid IN ({项目ID列表})` 过滤。调用方式遵循 `szabot-copilot` 技能规范。

✅ **以下维度字段可直接在 SQL WHERE 中过滤，无需调用`kb_search`**：工作室、数据来源、题材类型、题材赛道、开机年份、立项评级、集数、拍摄地、拍摄天数

✅ **仅当用户未指定任何项目条件**（即查询全部项目）时，才跳过本步骤，不在 SQL 中加入项目ID过滤条件（其他 WHERE 条件如时间范围等照常保留）。

---

## Step 1 · 表路由选择

### 表选择路由表

| 意图名 | 用户查询需求 | 命中表 | db_name |
|---|---|---|---|
| 四级预算（默认） | 项目总成本、单集成本、线上/线下成本及其明细、预算趋势/浮动 | `dws_szbot_dianshiju_fee_t_four_project_budget_hf` | `dianshiju_fee` |
| 六级预算 | 各组别人员费用（制片组/导演组/美术组/服装组/化妆组/造型组/置景组/道具组/摄影组/灯光组/录音组/剪辑组/车辆组/动作组费用） | `dws_szbot_dianshiju_fee_t_six_project_budget_hf` | `dianshiju_fee` |

🚨 **路由规则**：默认走四级；用户问到各「XX组费用」时走六级。

> **禁止**为了一两个字段无谓地双表 JOIN。
>
> 🚨 两表同义字段命名**不一致**，**必须**以各自 schema 为准，禁止跨表挪用字段名。

### 查询变体识别（不影响选表）

| 变体 | 处理方式 | 命中模板 |
|---|---|---|
| 单/多项目预算明细查询（含对比） | 金额字段 `ROUND(x/10000, 2)` 换算万元，分块展示 | schema 模板 1 |
| 预算趋势 / 预算浮动 / 年度趋势 / 近N年 | 按 `start_year` 聚合 | schema 模板 2 |
| 按维度汇总（赛道/评级/工作室等维度的集均成本统计） | 按指定维度 GROUP BY，指标按需取 | schema 模板 3 |

---

## Step 2 · 读取表 schema（强制）

读取表 schema：`references/dianshiju_fee/schema/{Step 1 命中的每张表名}.md`，文件中包含**字段定义、口径规则、查询模板（模板 1 明细 + 模板 2 年份趋势）**。


🚨 SQL 中每一个字段都必须能在 schema 文件中逐字搜到。**严禁**执行 `DESCRIBE` / `DESC` / `SHOW COLUMNS` / `SHOW CREATE TABLE` / `SHOW TABLES` / 查询 `INFORMATION_SCHEMA` / `SELECT *` 等方式到 DB 反查表结构。若 schema 文件中不存在所需字段，必须直接告知用户"当前 schema 文件中不存在该字段，无法查询"。

🚨 **指标名强约束（最高优先级）**：输出展示的每一个指标名，**必须**与 schema 中对应字段的 `COMMENT` **逐字一致**，禁止改写、美化、翻译、缩写、扩写、同义替换、合并。
- ✅ `项目总成本金额（单位：元）` → 展示为「项目总成本（万元）」（核心名称不变，仅可调整单位说明）
- ❌ 不得编造 schema 中不存在的指标名

---

## Step 3 · 执行 SQL

### MCP 工具调用方式

**MCP**：`szabot_data_query_svr` · `mcp_exec_sql`，入参 `db_name`（预算表统一为 `dianshiju_fee`，禁止猜测）、`sql`。

```bash
# 必须使用 --args 参数传递 JSON，不能用函数式调用
mcporter call 'szabot_data_query_svr.mcp_exec_sql' --args '{
  "db_name": "dianshiju_fee",
  "sql": "SELECT ... FROM ... WHERE ..."
}'
```

### 强制 SQL 规则（每条 SQL 都必须满足）

| 规则 | 内容 |
|---|---|
| 🚨 分区过滤 | 每条 SQL **必须**加 `WHERE imp_hour = (SELECT MAX(imp_hour) FROM <当前表名>)` 取最新分区，禁止全表扫描 |
| 🚨 项目过滤 | Step 0 取得 ID 后用 `AND pid IN ({项目ID列表})`；未指定项目时省略本条件 |
| 🚨 金额换算 | 默认换算为**万元**（`ROUND(x/10000, 2)`），用户指定单位以用户为准；**必须在 SQL 中完成**，禁止结果二次心算，展示注明单位 |
| 🚨 比例字段 | 比例字段（四级：`online_cost_ratio` / `offline_cost_ratio`；六级：`proportion` / `off_line_proportion`）单位为百分比（%），用 `CONCAT(字段, "%")` 拼接展示 |
| 🚨 NULL 处理 | NULL 展示为「暂无数据」，禁止当 0、禁止 `COALESCE(field, 0)` |
| 🚨 隐式过滤禁止 | 禁止自加 `field > 0` / `IS NOT NULL` / `!= 0`（除非用户明确要求） |
| 🚨 数据来源 | 用户指定来源时加 `AND project_source = "2"`（制片管理）或 `"1"`（人工上传 excel） |
| 🚨 集均成本计算 | 集均成本（即单集成本），表中无该字段，需实时计算：单项目用 `ROUND({指标字段} / episode_count / 10000, 2)`；汇总用 `ROUND(AVG({指标字段} / episode_count) / 10000, 2)`（各项目集均的平均值）。`{指标字段}` 默认为 `total_cost`，用户指定具体费用时替换为对应字段（如 `actor_cost`、`publicity_cost` 等） |
| 🚨 时间过滤 | 时间字段统一用 `start_year`（开机年份）；近 N 年：`start_year > YEAR(NOW()) - N`（**大于号**，不是 `>=`）；具体范围：`start_year BETWEEN "2020" AND "2026"` |

### SQL 模板入口

所有 SQL 模板均在各表 schema 文件中维护（按查询场景分组）。模板可能包含多个子区块（如概览/明细、汇总/明细等），具体形状以 schema 文件当前定义为准。

🚨 **按需查询**：严格按用户问题涉及的指标执行，不要随意拓展指标，不要更改查询方式。模板内的多个区块按**用户问题命中范围**执行：用户问全量 → 全部子块都跑；只问其中某一类字段 → 仅跑对应子块。**已选定执行的多个子块必须独立展示，按 schema 模板中子块的书写顺序输出，禁止合并、禁止漏块**。

---

## Step 4 · 结果展现

🚨 回答最后一行固定输出：`[查看预算](https://zp.szabot.internal/budget-evaluate/overview)`，schema 有更精确链接时替换。

### 通用展现规则

| 场景 | 展现形式 |
|---|---|
| 单项目明细 | 按 SQL 字段顺序，竖排键值对或单行表格 |
| 多项目对比 | 以 `pid` 为主键区分，**每个项目占一列**，字段名作为行标题，禁止合并 |
| 多子块模板 | 一个模板包含多个子区块时，**只输出与用户问题命中的子块**；多块同时命中时各自独立成区块，按 schema 模板中子块的书写顺序排列，区块内字段顺序仍按 SQL `SELECT` 顺序，禁止合并 |

### 字段顺序与命名

🚨 展示字段顺序**严格**按 SQL 中 `SELECT` 的字段顺序呈现，禁止调整顺序。
🚨 字段中文名（即 SQL 里的 `AS "..."`) 与 schema 字段 `COMMENT` 保持核心名称一致。

> 展示单位与精度规则与 SQL 内换算一致，详见 Step 3「强制 SQL 规则」中的金额换算与比例字段两条。
