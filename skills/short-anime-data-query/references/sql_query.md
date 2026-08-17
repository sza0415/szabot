# 短番/漫剧 SQL 查询指南

## 平台与指标前缀说明

短番/漫剧数据涉及 **2 款 APP**：
- **星舟视频**：指标以 `szsp_` 开头（如 `szsp_in_vstart_cnt`、`szsp_income`）
- **赤霄**：指标以 `chixiao_` 开头（如 `chixiao_in_vstart_cnt`、`chixiao_income`）

> ⚠️ 【重要】如果用户查询时**未明确指定查哪个端**，必须同时返回两个端的指标，不能只返回其中一个，避免误解用户意图。

> ⚠️ 【人均指标分母说明】计算**赤霄（chixiao）人均指标**时（如人均播放VV、人均时长等），分母使用**播放UV**（`chixiao_in_vstart_uv`），而不是 DAU。

> ⚠️ 【完播率计算口径】完播率 = `*_in_par_sum / *_in_vfinish_cnt`（完播率求和 ÷ 播放结束次数），例如星舟视频完播率：`SUM(szsp_in_par_sum) / SUM(szsp_in_vfinish_cnt)`，必须在 SQL 中计算，禁止查出原始值后手动计算。

> ⚠️ 【单位换算规则】如果需要对指标进行单位换算，**必须在 SQL 的 SELECT 中直接计算**，禁止先查出原始值再在返回结果中手动换算，避免计算出错。播放时长字段 `*_in_pdtm_s` 原始单位为**秒**，查询大盘指标时，需转换为**万小时**：`SUM(szsp_in_pdtm_s) / 3600 / 10000 AS 播放时长_万小时`。

## 表选用决策树

```
用户查询意图
│
├─ 查询专辑基础信息/分类？
│   └─ ✅ dim_szbot_duanfan_cid_info_df
│       ⛔ 禁止对本表任何字段使用模糊匹配（LIKE），所有字段必须精确匹配（使用 `=` 或 `IN`）
│
├─ 查询某个专辑的累计指标（累计播放量、累计收入等）？
│   └─ ✅ ads_szbot_duanfan_cid_metrics_ql_df（累计表）
│       imp_date 取 szdw_dim.chuku_progress 中 table_name = 'ads_szbot_duanfan_cid_metrics_ql_df' 的 MAX(imp_date)
│       指定具体 cid 查询对应专辑的累计值
│
├─ 排行榜 / Top N / 播放量最多的专辑？
│   └─ ✅ ads_szbot_duanfan_cid_metrics_ql_df（累计表）
│       【普通指标（播放、时长等）】用 imp_date=结束日 的累计值 减去 imp_date=开始日前一天 的累计值 → 得到区间增量后降序排列
│       【收入类指标（income、cost 等 T-2 时效字段）】用 imp_date=结束日后一天 的累计值 减去 imp_date=开始日 的累计值 → 得到区间增量后降序排列
│       ⚠️ 【收入排行榜时效限制】收入类指标 T-2 时效，结束日后一天的数据必须已入库才能查询。若用户查询结束日为昨天，则需要今天的数据，但今天数据尚未入库，**当前无法查询昨天的收入排行**，需提示用户将结束日往前推至少 1 天（即最新可查的收入排行结束日 = 前天）
│       ⚠️ 【结束日上限】结束日不能超过 szdw_dim.chuku_progress 中 table_name = 'ads_szbot_duanfan_cid_metrics_ql_df' 的 MAX(imp_date)
│       ⛔ 禁止用日增量表 zl_df_new 做排行榜
│       ⚠️ 【端拆分】如果用户未指定查赤霄还是星舟视频，必须分别展示两端的 Top N 排行，不能合并或只展示其中一端
│       ⚠️ 【端过滤】查星舟视频排行榜时，WHERE 条件中必须加 is_szsp_duanfan = 1；查赤霄排行榜时，必须加 is_chixiao_duanfan = 1
│       ⚠️ 【强制要求】排行榜结果必须包含专辑标题（title），查询分两步：
│           第一步：先从 ads_szbot_duanfan_cid_metrics_ql_df 查出 Top N 的 cid 列表
│           第二步：再用 cid IN (...) 去 dim_szbot_duanfan_cid_info_df 查询对应专辑标题
│           查询 dim 表时必须指定 imp_date，取 szdw_dim.chuku_progress 中 table_name = 'dim_szbot_duanfan_cid_info_df' 的 MAX(imp_date)
│
├─ 按天趋势 / 播放/收入/DAU/留存/完播率？
│   ├─ 查询每天的【大盘整体指标】（无需按专辑拆分）？
│   │   ⛔ 【强制禁止】大盘整体指标查询，禁止走 ads_szbot_duanfan_cid_metrics_zl_df_new
│   │   ├─ 查询播放VV、时长、完播率、收入、成本等指标？
│   │   │   └─ ✅ ads_szbot_duanfan_dapan_metrics_zl_df（大盘日增量表）
│   │   │       imp_date 取 szdw_dim.chuku_progress 中该表（table_name = 'ads_szbot_duanfan_dapan_metrics_zl_df'）的 MAX(imp_date)
│   │   │       按 data_date 过滤时间范围
│   │   │       ⚠️ 【聚合规则】该表同一 data_date 下存在多行（不同维度组合），查询时必须对指标 SUM 聚合并按 data_date GROUP BY
│   │   │       ⚠️ 【SzStudio 过滤】如果用户指定查询 SzStudio 的数据，需额外加 is_szstudio = 1
│   │   │       ⚠️ 【指标拆分】该表同时包含星舟视频（szsp_前缀）和赤霄（chixiao_前缀）指标，
│   │   │           如用户未指定端，必须同时返回两端指标，不能只返回其中一端
│   │   │       ⛔ 该表无 cid 维度，不可按专辑查询
│   │   │
│   │   └─ 查询 UV、DAU、留存率等指标？
│   │       ├─ 查询全量大盘 UV/DAU/留存（含 SzStudio 大盘）？
│   │       │   └─ ✅ ads_szbot_duanfan_dapan_uv_metrics_zl_df（大盘UV/留存日增量表）
│   │       │       imp_date 取 szdw_dim.chuku_progress 中该表（table_name = 'ads_szbot_duanfan_dapan_uv_metrics_zl_df'）的 MAX(imp_date)
│   │       │       按 data_date 过滤时间范围
│   │       │       ⚠️ 该表无需额外过滤条件，直接查询即为全量大盘数据
│   │       │       ⚠️ 该表指标不可聚合/相加，只能按天展示趋势
│   │       │       ⛔ 该场景禁止走 ads_szbot_duanfan_cid_metrics_zl_df_new
│   │       └─ 查询某个具体专辑的 UV？
│   │           └─ ✅ ads_szbot_duanfan_cid_metrics_zl_df_new，指定具体 cid
│   │
│   └─ 查询【某个具体专辑】的按天指标？
│       └─ ✅ ads_szbot_duanfan_cid_metrics_zl_df_new（专辑日增量表）
│           imp_date 取 szdw_dim.chuku_progress 中该表（table_name = 'ads_szbot_duanfan_cid_metrics_zl_df_new'）的 MAX(imp_date)
│           按 data_date 过滤时间范围
│           ⚠️ 【强制要求】查询该表时，WHERE 条件中必须指定具体 cid（如 cid = '{专辑ID}'），禁止不带 cid 条件查询
│           ⛔ 【SzStudio 未指定具体专辑时禁止走此分支】SzStudio 大盘数据（未指定具体专辑）必须走 ads_szbot_duanfan_dapan_metrics_zl_df，加 is_szstudio = 1
│           ⚠️ 【指标拆分】该表同时包含星舟视频（szsp_前缀）和赤霄（chixiao_前缀）指标，
│               如用户未指定端，必须同时返回两端指标，不能只返回其中一端
```

## 常见查询模板

```sql
-- 获取最新分区日期（大盘表）
SELECT MAX(imp_date) FROM szdw_dim.chuku_progress
WHERE table_name = 'ads_szbot_duanfan_dapan_metrics_zl_df';

-- 大盘每日指标（最近7天，使用大盘表）
-- 该表同一 data_date 下存在多行，必须 SUM 聚合后 GROUP BY data_date
SELECT data_date,
    SUM(szsp_in_vstart_cnt) AS 星舟视频播放VV,
    SUM(chixiao_in_vstart_cnt) AS 赤霄播放VV
FROM short_anime_prod.ads_szbot_duanfan_dapan_metrics_zl_df
WHERE imp_date = (SELECT MAX(imp_date) FROM szdw_dim.chuku_progress
    WHERE table_name = 'ads_szbot_duanfan_dapan_metrics_zl_df')
    AND data_date >= DATE_FORMAT(DATE_SUB(CURDATE(), INTERVAL 8 DAY), '%Y%m%d')
    AND data_date <= DATE_FORMAT(DATE_SUB(CURDATE(), INTERVAL 1 DAY), '%Y%m%d')
GROUP BY data_date
ORDER BY data_date DESC;

-- 大盘每日播放UV（最近7天，使用大盘UV表）
SELECT data_date,
    total_in_vstart_uv AS 汇总播放UV,
    szsp_in_vstart_uv AS 星舟视频播放UV,
    chixiao_in_vstart_uv AS 赤霄播放UV
FROM short_anime_prod.ads_szbot_duanfan_dapan_uv_metrics_zl_df
WHERE imp_date = (SELECT MAX(imp_date) FROM szdw_dim.chuku_progress
    WHERE table_name = 'ads_szbot_duanfan_dapan_uv_metrics_zl_df')
    AND data_date >= DATE_FORMAT(DATE_SUB(CURDATE(), INTERVAL 8 DAY), '%Y%m%d')
    AND data_date <= DATE_FORMAT(DATE_SUB(CURDATE(), INTERVAL 1 DAY), '%Y%m%d')
ORDER BY data_date DESC;

-- 大盘收入数据（T-2 时效：今天能看到的最新数据 = CURDATE() - 2 天）
-- 例如：3月31日查询，收入最新到3月29日，所以 data_date 上界用 INTERVAL 2 DAY
-- 该表同一 data_date 下存在多行，必须 SUM 聚合后 GROUP BY data_date
SELECT data_date,
    SUM(szsp_income) AS 星舟视频收入,
    SUM(chixiao_income) AS 赤霄收入
FROM short_anime_prod.ads_szbot_duanfan_dapan_metrics_zl_df
WHERE imp_date = (SELECT MAX(imp_date) FROM szdw_dim.chuku_progress
    WHERE table_name = 'ads_szbot_duanfan_dapan_metrics_zl_df')
    AND data_date >= DATE_FORMAT(DATE_SUB(CURDATE(), INTERVAL 10 DAY), '%Y%m%d')
    AND data_date <= DATE_FORMAT(DATE_SUB(CURDATE(), INTERVAL 2 DAY), '%Y%m%d')
GROUP BY data_date
ORDER BY data_date DESC;

-- ============================================================
-- 排行榜（某时间段内 Top N，用累计表相减）分两步执行
-- ⚠️ 【关键】普通指标 和 收入类指标 的相减口径不同，必须严格区分：
--   普通指标（播放VV、时长等）：
--     t2.imp_date = {结束日}，t1.imp_date = {开始日前一天}
--   收入类指标（income、cost 等 T-2 时效字段）：
--     t2.imp_date = {结束日后一天}，t1.imp_date = {开始日}
--     ⛔ 收入类指标 t1 绝对不能用 {开始日前一天}，否则会多算一天！
-- ⚠️ 收入类指标 T-2 时效：若结束日为昨天，则需要今天的数据，但今天数据尚未入库，
--    当前无法查询昨天的收入排行，最新可查结束日 = 前天（CURDATE()-2）
-- ============================================================

-- 第一步（普通指标示例）：查出 Top N 的 cid 及区间播放量
-- 结束日由用户指定，但不能超过 szdw_dim.chuku_progress 中 table_name = 'ads_szbot_duanfan_cid_metrics_ql_df' 的 MAX(imp_date)
-- 未指定端时，分别查星舟视频 Top N 和赤霄 Top N（各自独立排序）
-- 星舟视频播放VV Top N（必须加 is_szsp_duanfan = 1）
SELECT
    t2.cid,
    (t2.szsp_in_vstart_cnt - IFNULL(t1.szsp_in_vstart_cnt, 0)) AS 星舟视频区间播放VV
FROM short_anime_prod.ads_szbot_duanfan_cid_metrics_ql_df t2
LEFT JOIN short_anime_prod.ads_szbot_duanfan_cid_metrics_ql_df t1
    ON t1.cid = t2.cid AND t1.imp_date = {开始日前一天} AND t1.is_szsp_duanfan = 1
WHERE t2.imp_date = LEAST({用户指定结束日}, (SELECT MAX(imp_date) FROM szdw_dim.chuku_progress
    WHERE table_name = 'ads_szbot_duanfan_cid_metrics_ql_df'))
    AND t2.is_szsp_duanfan = 1
ORDER BY 星舟视频区间播放VV DESC
LIMIT 10;

-- 赤霄播放VV Top N（必须加 is_chixiao_duanfan = 1）
SELECT
    t2.cid,
    (t2.chixiao_in_vstart_cnt - IFNULL(t1.chixiao_in_vstart_cnt, 0)) AS 赤霄区间播放VV
FROM short_anime_prod.ads_szbot_duanfan_cid_metrics_ql_df t2
LEFT JOIN short_anime_prod.ads_szbot_duanfan_cid_metrics_ql_df t1
    ON t1.cid = t2.cid AND t1.imp_date = {开始日前一天} AND t1.is_chixiao_duanfan = 1
WHERE t2.imp_date = LEAST({用户指定结束日}, (SELECT MAX(imp_date) FROM szdw_dim.chuku_progress
    WHERE table_name = 'ads_szbot_duanfan_cid_metrics_ql_df'))
    AND t2.is_chixiao_duanfan = 1
ORDER BY 赤霄区间播放VV DESC
LIMIT 10;

-- 第一步（收入类指标示例）：查出 Top N 的 cid 及区间收入
-- ⚠️ 收入类指标口径：t2.imp_date = {结束日后一天}，t1.imp_date = {开始日}
-- ⛔ t1.imp_date 绝对不能写成 {开始日前一天}，否则会多算一天的收入！
-- 示例：用户问"前天（20260426）收入排行"→ 开始日=结束日=20260426
--       t2.imp_date = 20260427（结束日后一天），t1.imp_date = 20260426（开始日）
-- 星舟视频收入 Top N（必须加 is_szsp_duanfan = 1）
SELECT
    t2.cid,
    (t2.szsp_income - IFNULL(t1.szsp_income, 0)) AS 星舟视频区间收入
FROM short_anime_prod.ads_szbot_duanfan_cid_metrics_ql_df t2
LEFT JOIN short_anime_prod.ads_szbot_duanfan_cid_metrics_ql_df t1
    ON t1.cid = t2.cid AND t1.imp_date = {开始日} AND t1.is_szsp_duanfan = 1
WHERE t2.imp_date = {结束日后一天}
    AND t2.is_szsp_duanfan = 1
ORDER BY 星舟视频区间收入 DESC
LIMIT 10;

-- 第二步：根据第一步返回的 cid 列表，查询专辑标题
SELECT cid, title
FROM short_anime_prod.dim_szbot_duanfan_cid_info_df
WHERE imp_date = (SELECT MAX(imp_date) FROM szdw_dim.chuku_progress
    WHERE table_name = 'dim_szbot_duanfan_cid_info_df')
    AND cid IN ({第一步返回的cid列表});
```
