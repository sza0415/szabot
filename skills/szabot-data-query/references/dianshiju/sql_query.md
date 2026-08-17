# 电视剧数据查询流程

## 🚨 执行流程总则

**Step 1 表路由** → **Step 2 读取 schema** → **Step 3 执行 SQL** → **Step 4 结果展现**，严格按序，**禁止跳步、禁止少步、禁止合并**，四步必须全部执行，违反即作废重来。

---

## Step 1 · 表路由选择

🚨 **前置已就绪**：项目ID 已由前置 `kb_search` 工具注入上下文，可直接代入 SQL 模板的 `{项目ID}` 占位符。**禁止**用 SQL 反查 pid（如 `SELECT pid FROM dim_... WHERE pname = ...`）。

### 表选择路由表

| 意图名 | 用户查询需求 | 命中表 | db_name | 备注 |
|---|---|---|---|---|
| 项目维度 | 项目维度信息（pid、项目名称、运营期开始/结束日期、豆瓣评分） | `dim_szbot_dianshiju_project_info_hf` | `szdw_ads` | 🚨 仅有 `pid` / `pname` / `operation_start_date` / `operation_end_date` / `douban_score` 五个业务字段 |
| 累计指标 | **触发清单**：凡涉及①播放/播出 + 数据/情况/概览/表现②对比播放③项目对比④运营期播放⑤前N天播放⑥累计VV/UV/完播率/首集留存率/播放时长等任何表达，均命中本行 | `ads_szbot_dianshiju_project_consume_metrics_ql_df` | `szdw_ads` | 🚨 命中本行 → Step 3 必须完整执行「累计查询调用链」①②③，禁止跳步 |
| 增量指标 | 日增量指标（每日播放VV/UV、热度值等） | `ads_szbot_dianshiju_project_consume_metrics_di` | `szdw_ads` | 🚨 默认无需查询 |
| 小时指标 | 小时级指标 / 预估指标（预估VV等） | `ads_szbot_dianshiju_project_consume_metrics_hi` | `szdw_ads` | — |
| 分区进度 | 表最新更新的分区进度 | `chuku_progress` | `szdw_dim` | — |

---

## Step 2 · 读取表 schema（强制）

读取表schema：`references/dianshiju/schema/{Step 1 命中的每张表名}.md` ，文件中有相关查询模板

🚨 SQL 中每一个字段都必须能在 schema 文件中逐字搜到。禁止 `SELECT *`、禁止凭经验补字段、禁止用 `DESCRIBE` / `SHOW COLUMNS` / `INFORMATION_SCHEMA` 探测结构。

---

## Step 3 · 执行 SQL

**MCP**：`szabot_data_query_svr` · `mcp_exec_sql`，入参 `db_name`（以路由表为准，禁止猜测）、`sql`。
```bash
# 使用 --args 参数传递 JSON
mcporter call 'szabot_data_query_svr.mcp_exec_sql' --args '{
  "db_name": "szdw_ads",
  "sql": "SELECT ... FROM ... WHERE ..."
}'
```
**日期格式**：`imp_date` 等为 `yyyyMMdd`（如 `20250315`）；`imp_hour` 为 `yyyyMMddHH`。

所有派生计算（单位换算、天数差、比率等）一律下沉到 SQL，禁止心算。

### 累计查询调用链（命中累计指标时必走，缺一不可）

| 步骤 | 动作 | 模板 | 并行说明 |
|---|---|---|---|
| ① | 取 `tongji_end_date` | 单项目运营期→模板 1a；前 N 天→模板 1b；多项目未指定天数→模板 1c |  |
| ② | 用 `tongji_end_date` 作为 `imp_date` 查累计快照 | 模板 2 | 必须等步骤①返回后执行 |
| ③ | 🚨 **必须完整输出展现模板 1 + 2 + 3 三个独立区块，缺一不可、禁止合并、禁止乱序** | Step 4 | — |

### 查询模板 1. 取统计结束日期 `tongji_end_date`（命中意图：累计指标）

#### 模板 1a. 单项目运营期累计 — 取 `tongji_end_date`

> `tongji_end_date = LEAST(累计表最新分区, operation_end_date)`

```sql
-- db_name: szdw_ads
SELECT
    operation_start_date,
    operation_end_date,
    LEAST(
        (SELECT MAX(imp_date) FROM szdw_dim.chuku_progress
         WHERE table_name = 'ads_szbot_dianshiju_project_consume_metrics_ql_df'),
        operation_end_date
    ) AS tongji_end_date
FROM dim_szbot_dianshiju_project_info_hf
WHERE pid = {项目ID}
  AND imp_hour = (
      SELECT MAX(imp_hour) FROM szdw_dim.chuku_progress
      WHERE table_name = 'dim_szbot_dianshiju_project_info_hf'
  )
LIMIT 1;
```

#### 模板 1b. 前 N 天累计 — 取 `tongji_end_date`（单项目 / 多项目已指定天数）

> `tongji_end_date = DATE_ADD(operation_start_date, INTERVAL (N-1) DAY)`
> 例：前 7 天 → `INTERVAL 6 DAY`
> 🚨 `operation_start_date` 必须从维表查出，禁止大模型自行填写日期。

```sql
-- db_name: szdw_ads
SELECT
    operation_start_date,
    operation_end_date,
    DATE_FORMAT(DATE_ADD(operation_start_date, INTERVAL ({N} - 1) DAY), '%Y%m%d') AS tongji_end_date
FROM dim_szbot_dianshiju_project_info_hf
WHERE pid = {项目ID}
  AND imp_hour = (
      SELECT MAX(imp_hour) FROM szdw_dim.chuku_progress
      WHERE table_name = 'dim_szbot_dianshiju_project_info_hf'
  )
LIMIT 1;
```

#### 模板 1c. 多项目对比未指定天数 — 对齐口径

> 🚨 必须先对齐到各项目上线天数最小值 N，再各自查第 N 天快照；**禁止**各自取 `LEAST(最新分区, operation_end_date)` 直接查。

**Step 1**：查最新分区
```sql
-- db_name: szdw_dim
SELECT MAX(imp_date) AS latest_imp_date
FROM chuku_progress
WHERE table_name = 'ads_szbot_dianshiju_project_consume_metrics_ql_df';
```

**Step 2**：查各项目上线天数
```sql
-- db_name: szdw_ads
SELECT pid, operation_start_date, operation_end_date,
    DATEDIFF(LEAST({latest_imp_date}, operation_end_date), operation_start_date) + 1 AS shangxian_days
FROM dim_szbot_dianshiju_project_info_hf
WHERE pid IN ({项目ID_1}, {项目ID_2})
  AND imp_hour = (SELECT MAX(imp_hour) FROM szdw_dim.chuku_progress WHERE table_name = 'dim_szbot_dianshiju_project_info_hf');
```

**Step 3**：取 `shangxian_days` 最小值作为 N，代入模板 1b 分别为各项目查出 `tongji_end_date`。

### 模板 2. 查询累计指标

> 🚨 所有累计口径统一从 `ads_szbot_dianshiju_project_consume_metrics_ql_df` 按 `imp_date` 取快照；**禁止**从 `_di` 用 `SUM` 拼凑。
> 🚨 下方字段是默认必查最小集合，一字不得删。用户额外指定的字段必须从 schema 中逐字复制追加，禁止猜测。

```sql
-- db_name: szdw_ads
SELECT
    history_max_hot_value,
    sxq_in_vstart_oper_vv_ott,
    sxq_zp_play_vv_app_pc_ott,
    sxq_zp_valid_uv,
    ROUND(in_pdtm_s_app_pc_ott / 60 / 10000 / 10000, 2) AS pdtm_yi_min,
    sxq_finish_rate,
    sxq_sj_completion_rate,
    sxq_first_vid_drop_rate_app_pc_ott,
    sxq_imp_uv,
    sxq_play_zp_utr,
    sxq_zp_play_uv,
    sxq_zp_valid_utr,
    sxq_finish_uv,
    operation_start_date,
    DATEDIFF(imp_date, operation_start_date) + 1 AS shangxian_days,
    ROUND(in_pdtm_s_app_pc_ott / 60 / sxq_in_vstart_oper_uv_ott, 2) AS pdtm_per_uv_min --按需展现
FROM ads_szbot_dianshiju_project_consume_metrics_ql_df
WHERE pid = {项目ID}
  AND imp_date = {tongji_end_date};
```

---

## Step 4 · 结果展现

| 意图名 | 展现形式 |
|---|---|
| 累计指标 | **必须**输出展现模板 1 + 2 + 3 三个独立区块，缺一不可、禁止合并、禁止乱序 |
| 项目维度 / 增量指标 / 小时指标 | 按指标名强约束直出表格 |
| 分区进度 | 直接返回最新分区值 |

🚨 **累计指标**：无论用户怎么问，都必须完整输出展现模板 1 + 2 + 3。用户**主动点名**的非模板字段，在三件套之**前**单独输出独立区块「用户查询指标」（用户问什么先答什么），三件套作为标准上下文紧随其后。

### 指标名强约束（最高优先级，全局生效）

输出的每个指标名必须与 `schema/*.md` 中对应字段的 `COMMENT` **逐字一致**，禁止改写、翻译、缩写、同义替换。

**唯一例外**：展示时必须去掉 COMMENT 末尾的 `-三端`、`-app端` 后缀，其余原样保留。

🚨 **禁止心算换算数值**，如需换算单位必须下沉到 SQL 计算。

### 展现模板 1. 累计核心播放指标（固定 10 行，顺序不可变）

🚨「上线时间」必须取 `operation_start_date` 字段值，禁止用 `imp_date` / `tongji_end_date` 代替。缺任一字段必须重新执行 SQL 补查。

| 指标 | 数值 |
|---|---|
| 上线时间 | {operation_start_date} |
| 上线天数 | {shangxian_days} 天 |
| 历史最高热度值 | {history_max_hot_value} |
| 播放VV | {sxq_in_vstart_oper_vv_ott} |
| 正片VV | {sxq_zp_play_vv_app_pc_ott} |
| 正片有效播放UV | {sxq_zp_valid_uv} |
| 播放时长 | {pdtm_yi_min} 亿分钟 |
| 完播率 | {sxq_finish_rate} |
| 首集播放完成度 | {sxq_sj_completion_rate} |
| 首集弃剧率 | {sxq_first_vid_drop_rate_app_pc_ott} |

多项目对比：「数值」列拆为各项目并列多列。

### 展现模板 2. 累计播放转换漏斗（固定 7 行，顺序不可变）

| 指标 | 数值 |
|---|---|
| 曝光人数 | {sxq_imp_uv} |
| 曝光-正片播放UTR | {sxq_play_zp_utr} |
| 正片播放UV | {sxq_zp_play_uv} |
| 正片-正片有效播放UTR | {sxq_zp_valid_utr} |
| 正片有效播放UV | {sxq_zp_valid_uv} |
| 完播率 | {sxq_finish_rate} |
| 完播人数 | {sxq_finish_uv} |

多项目对比：「数值」列拆为各项目并列多列。

### 展现模板 3. 时间范围（两行独立输出，禁止合并）

- **运营期**：`operation_start_date ~ operation_end_date`
- **统计时间范围**：`operation_start_date ~ LEAST(imp_date, operation_end_date)`

必须给出具体年月日（如 `20250315 ~ 20250630`）。多项目对比时每个项目各自独立列出两行。
