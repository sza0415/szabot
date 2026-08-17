```sql
use szstudio;
CREATE TABLE `dwd_aigc_zen_zhongqi_df` (
    `imp_date` bigint NOT NULL COMMENT '分区时间(YYYYMMDD),【重要：绝对禁止直接给定固定日期！必须通过子查询从 szdw_dim.chuku_progress 表根据 table_name=dwd_aigc_zen_zhongqi_df 筛选最大 imp_date 作为值】',
    `create_date` varchar(100) COMMENT '创建日期(YYYYMMDD)',
    `create_time` varchar(100) COMMENT '创建时间',
    `asset_id` varchar(100) COMMENT '素材ID',
    `user_id` varchar(100) COMMENT '用户ID',
    `corp_id` varchar(100) COMMENT '企业ID',
    `corp_name` varchar(100) COMMENT '企业名',
    `szbot_project_id` varchar(100) COMMENT '影库项目ID',
    `szbot_project_name` varchar(100) COMMENT '影库项目名称',
    KEY `date` (`create_date`)
) ENGINE = InnoDB COMMENT = '影视前中期制作素材'
```

## 【口径规则】
1. 数据为全量表，取数时必须指定imp_date，imp_date从维表szdw_dim.chuku_progress中基于table_name=dwd_aigc_zen_zhongqi_df取最大的日期
2. 数据日期用create_date
3. 算数时，基于asset_id去重统计
4. 查询影视前中期相关素材数时，需要查询本表
5. 按项目查询时，直接用 `szbot_project_name` 或 `szbot_project_id` 作为过滤条件；两个字段均需**精确匹配**（使用 `=`，禁止使用 `LIKE` 或模糊匹配）
