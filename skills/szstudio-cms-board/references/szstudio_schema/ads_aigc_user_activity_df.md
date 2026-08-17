```sql
use szstudio;
CREATE TABLE `ads_aigc_user_activity_df` (
    `id` bigint NOT NULL COMMENT '自增主键',
    `imp_date` bigint NOT NULL COMMENT '时间分区(YYYYMMDD),【重要：绝对禁止直接给定固定日期！必须通过子查询从 szdw_dim.chuku_progress 表根据 table_name=ads_aigc_user_activity_df 筛选最大 imp_date 作为值】',
    `data_date` varchar(50) COMMENT '数据日期(YYYYMMDD)',
    `app` varchar(50) COMMENT '应用名称',
    `dau` bigint COMMENT '日活跃用户数',
    `pv` bigint COMMENT '活跃事件次数',
    `duration` double COMMENT '总活跃时长(分钟)',
    PRIMARY KEY (`id`, `imp_date`),
    KEY `index` (`imp_date`, `data_date`)
) COMMENT = 'AIGC用户活跃度统计表，全量表'
```

## 【口径规则】

1. 数据为全量表，取数时必须指定imp_date，imp_date从维表szdw_dim.chuku_progress中基于table_name=ads_aigc_user_activity_df取最大的日期
2. 查询时带上数据库，参考szstudio.ads_aigc_user_activity_df
3. 数据日期用data_date；涉及"昨天"、"最近 N 天"等相对日期时，**必须以系统当前自然日为基准推算，与 `imp_date` 无关，禁止用 `imp_date - 1` 代替"昨天"**
4. `pv`、`duration` 可以跨天 SUM 聚合；`dau` **严禁 SUM**，不同天的活跃用户存在重叠，跨天相加会重复计算同一用户导致数值虚高。查询整体概览/汇总时，DAU 只能取单天（最新一天）的值，用 `WHERE data_date = '最新日期'` 过滤后直接展示；如需展示趋势，只能按天列出每天的 dau 值
5. **必须按 app 分开查询，严禁跨 app 聚合**。不同 app 的用户群体相互独立，查询时 WHERE 条件中必须指定具体的 app 值，或在结果中按 app 分组展示，不得将多个 app 的数据合并汇总
6. 查询 SzStudio、zen、SzCanvas 相关指标时，app 字段对应的值为 `szcanvas`，即 `WHERE app = 'szcanvas'`