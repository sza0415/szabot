---
name: short-anime-data-query
namespace: szbot
trust-level: builtin
category: data-query
version: 2.1.0
description: "漫剧/短番/赤霄/SzStudio/SzCanvas/Rally 的播放数据、收入成本查询技能，支持查询播放VV/UV、时长、DAU、留存率、完播率、收入、成本、会员开通等指标。当用户询问短番、漫剧、赤霄、SzStudio、SzCanvas、Rally（含 zen/szstudio/szcanvas/rally 等大小写变体）的播放情况/播放数据/播放量/收入表现时触发。示例：「某漫剧的播放量」「赤霄DAU怎么样」。"
---

# 漫剧/短番数据查询技能

> ⚠️ **触发前提**：仅当用户明确提到「**短番**」「**漫剧**」「**赤霄**」「**SzStudio**」「**SzCanvas**」「**Rally**」关键词时才使用本 Skill。如果用户只是问某个项目/专辑的播放数据，但没有提到上述关键词，**不要使用本 Skill**，应走 `szabot-copilot`。
>
> ⚠️ **SzStudio / SzCanvas 识别**：SzStudio 已更名为 SzCanvas，两者是**同义词**。用户提到 `zen`、`szstudio`、`Zen`、`SzStudio`、`SzCanvas`、`szcanvas`、`Rally`、`rally` 等任何大小写变体时，均视为 **SzStudio**，对应查询条件 `is_szstudio = 1`。
>
> ⛔ **不适用于 SzStudio 制作类查询**：如果用户问的是 SzStudio 的**素材数量、制作进度、影视后期、视频/图片生成统计、制作量、用户活跃度**等制作侧信息，**不要使用本 Skill**，应走 `szstudio-cms-board`。本 Skill 仅处理漫剧/短番的**播放和收入**指标。

**仅限漫剧、短番、赤霄、SzStudio 业务**的数据查询，通过 MCP 服务 `szabot_data_query_svr` 的 `mcp_exec_sql` 工具执行 SQL 查询。本 Skill 不处理侵权数据查询（走 `szpp-data-query`）或通用影视综项目信息查询（走 `szabot-copilot`）。

## 快速开始

```bash
# 验证 MCP 服务
mcporter list szabot_data_query_svr
```

## 正确的调用方式（重要！）

⚠️ **仅支持 `short_anime_prod` 和 `szdw_dim` 两个数据库**。禁止使用其他 db_name。

### 步骤1：获取最新 imp_date

```bash
# 使用 db_name="szdw_dim" 查询 chuku_progress 获取最新分区日期
mcporter call 'szabot_data_query_svr.mcp_exec_sql' --args '{
  "db_name": "szdw_dim",
  "sql": "SELECT MAX(imp_date) as imp_date FROM chuku_progress WHERE table_name = '\''ads_szbot_duanfan_dapan_metrics_zl_df'\''"
}'
```

### 步骤2：执行业务查询

```bash
# 使用 db_name="short_anime_prod" 查询大盘播放VV汇总（指标可直接 SUM 聚合）
mcporter call 'szabot_data_query_svr.mcp_exec_sql' --args '{
  "db_name": "short_anime_prod",
  "sql": "SELECT SUM(szsp_in_vstart_cnt) AS 星舟视频播放VV汇总, SUM(chixiao_in_vstart_cnt) AS 赤霄播放VV汇总 FROM ads_szbot_duanfan_dapan_metrics_zl_df WHERE imp_date = 20260330 AND data_date >= '\'20260324'\'' AND data_date <= '\'20260330'\'' "
}'
```

## 常见错误与解决

| 错误信息 | 原因 | 解决方法 |
|---------|------|---------|
| `SQL语句不能为空` | JSON 格式问题或引号转义失败 | 使用 `--args` 参数传递 JSON |
| `invalid ExecSQLReq.DbName: value contains invalid strings` | db_name 不在允许列表 | **短番业务必须用 `"short_anime_prod"`，获取分区日期必须用 `"szdw_dim"`** |
| `service codec Unmarshal: invalid character '-' in numeric literal` | SQL 中有语法错误 | 检查 SQL 语句是否有特殊字符问题 |

### ⚠️ 关键要点
0. references对应的完整路径为`/app/resources/skills/szabot/short-anime-data-query/references`
1. **必须使用 `--args` 参数**传递 JSON，不能用函数式调用
2. **db_name 必须是 `"short_anime_prod"` 或 `"szdw_dim"`**，其他格式都会报错
3. 查询 `szdw_dim.chuku_progress` 时用 `"szdw_dim"`，查询业务表时用 `"short_anime_prod"`

## 执行流程

1. **加载知识库** — 读取 `references/` 下的口径和表结构，包括以下资源文件：

| 文件 | 内容 |
|-----|------|
| `references/short_anime/*.md` | 漫剧/短番数据表结构定义 |
| `references/chuku_schema.md` | 数据分区表结构 |
| `references/sql_query.md` | 表选用规则（决策树） + 常见查询模板（含排行榜）, 包含大盘播放、收入、排行榜等完整 SQL 示例 |

> 构建 SQL 前必须先读取对应的 references 文件。

2. **生成 SQL** — 基于口径规则构建查询，禁止自行猜测计算方式
3. **执行查询** — 通过 `mcp_exec_sql` 执行 SQL

## 触发场景

**✅ 使用本 Skill**：用户提到了「短番」「漫剧」「赤霄」「SzStudio」（含 zen/szstudio 等大小写变体）+ 数据查询需求

> 💡 当用户提到 SzStudio/Workrally 时，查询条件需加 `is_szstudio = 1`。

**❌ 不使用本 Skill**：
- 用户只提到某个项目名/专辑名的播放量、收入等，但未提及短番/漫剧/赤霄/SzStudio → 走 `szabot-copilot`
- 用户问 SzStudio/Workrally 的**素材数量、制作进度、影视后期、视频/图片生成统计、制作量、用户活跃度** → 走 `szstudio-cms-board`


## 核心口径

> ⛔ **口径铁律**：下表中列出的指标和计算规则是唯一允许使用的口径。**任何未在本表或 references 文档中明确给出计算公式的衍生指标，一律不得输出**。只能直接返回原始字段值，不得自行组合字段推导新指标。

| 指标 | 计算规则 |
|-----|---------|
| 播放VV（播放量） | `*_vstart_cnt` 字段 |
| 播放UV | `*_vstart_uv` 字段 |
| 完播率 | `*_in_par_sum / *_in_vfinish_cnt`（完播率求和 ÷ 播放结束次数），必须在 SQL 中计算 |
| ROC | `收入 / 成本`（对应字段：`*_income / *_cost`），必须在 SQL 中计算 |
| 赤霄人均播放 | 分母用播放UV，不是DAU |
| 星舟视频 / 主站三端 | 两者是**同义词**，用户说"主站三端"等同于"星舟视频"，对应 `szsp_` 前缀指标 |
| 大盘数据 | 播放VV、时长、完播率、收入、成本等指标走 `ads_szbot_duanfan_dapan_metrics_zl_df`；UV、DAU、留存率等指标走 `ads_szbot_duanfan_dapan_uv_metrics_zl_df`；⛔ **禁止走 `ads_szbot_duanfan_cid_metrics_zl_df_new`** |
| imp_date | 从 `szdw_dim.chuku_progress` 取 `MAX(imp_date)` where `table_name='{表名}'`，**禁止写死日期** |
| 数据时效 | data_date 代表数据所属的自然日，用户说"昨天"就直接用昨天的日期，**不要再额外减天**。数据时效仅用于约束 data_date 可查询的**上界**：播放指标上界 `CURDATE() - 1`（今天能查到的最新数据是昨天的）；收入/成本/会员开通上界 `CURDATE() - 2`（今天能查到的最新数据是前天的）。如果用户请求的日期超过上界，应提示"该日期数据尚未出库"。示例：4月28日查"昨天播放量"→ data_date='20260427'；4月28日查"昨天收入"→ 收入上界为4月26日，昨天(4月27日)超过上界，应提示数据尚未出库 |
| 时间聚合 | `ads_szbot_duanfan_dapan_metrics_zl_df` 的指标**可以直接 SUM 聚合**，求时间段汇总时直接对各天求和即可；`ads_szbot_duanfan_dapan_uv_metrics_zl_df` 的指标**不可聚合**，只能按天展示趋势 |
| 排行榜累计相减口径 | 使用 `ads_szbot_duanfan_cid_metrics_ql_df` 累计表相减得区间增量，但**普通指标**和**收入类指标**算法不同：① **普通指标**（播放VV、时长等）：`imp_date=结束日` 累计值 **减去** `imp_date=开始日前一天` 累计值；② **收入类指标**（`*_income`、`*_cost` 等 T-2 字段）：`imp_date=结束日后一天` 累计值 **减去** `imp_date=开始日` 累计值。⚠️ **收入排行榜时效限制**：若用户查询结束日为昨天，则需要今天的数据，但今天尚未入库，**当前无法查询昨天的收入排行**，必须提示用户：最新可查的收入排行结束日 = 前天（`CURDATE() - 2`）|


## 注意事项

- **MCP Server 名称强约束**：本 Skill 使用的 MCP Server 为 **`szabot_data_query_svr`**，必须以此为准。**严禁猜测 MCP Server 名称，尤其禁止将 Skill 名称（`short-anime-data-query`）当作 MCP Server 名称**。判断 MCP Server 是否可用时，必须检查 `szabot_data_query_svr` 的状态。
- ⚠️ **禁止通过 mcp_exec_sql 扫描表结构或表列表**（如 `SHOW TABLES`、`DESCRIBE`、`SHOW COLUMNS`、`information_schema` 等），所有表结构信息必须且只能从 `references/short_anime/` 目录或 `references/chuku_schema.md` 中获取。
- 必须通过 `mcp_exec_sql` 执行查询
- ⛔ **严禁自造口径**：只能使用 references 文档中**明确定义了计算公式**的指标。如果某个指标在 references 中只有字段名和注释，但**没有给出明确的计算公式**，则**禁止自行推测、组合或编造计算方式**。遇到这种情况，应直接返回原始字段值，并注明「该指标的计算口径未在文档中明确定义，建议与数据口径负责人确认」。
- IP对比时，最好放在一个表格进行分析，不要分开对比
- ⚠️ **单位换算必须在 SQL 中完成**：如果需要对指标进行单位换算，必须在 SQL 的 SELECT 中直接计算，**禁止先查出原始值再在返回结果中手动换算**，避免计算出错。⛔ **播放时长万小时换算仅限大盘表**：仅查询 `ads_szbot_duanfan_dapan_metrics_zl_df`（大盘日增量表）时，才将 `*_in_pdtm_s`（秒）转换为万小时（`SUM(*_in_pdtm_s) / 3600 / 10000`）；
- 输出结尾附带执行时间及数据来源