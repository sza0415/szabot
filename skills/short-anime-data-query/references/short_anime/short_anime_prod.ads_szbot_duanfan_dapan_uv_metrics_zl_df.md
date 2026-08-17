```sql
USE short_anime_prod;
CREATE TABLE `ads_szbot_duanfan_dapan_uv_metrics_zl_df` (
    `imp_date` bigint NOT NULL COMMENT '时间分区(YYYYMMDD)',
    `data_date` bigint NOT NULL COMMENT '统计日期(YYYYMMDD)',
    `total_in_vstart_uv` bigint COMMENT '汇总-播放开始设备数',
    `szsp_in_vstart_uv` bigint COMMENT '星舟视频-播放开始设备数',
    `chixiao_in_vstart_uv` bigint COMMENT '赤霄-播放开始设备数',
    `chixiao_in_dau` bigint COMMENT '赤霄-DAU',
    `chixiao_in_dau_retention_rate_d1` double COMMENT '赤霄-DAU-次日留存率',
    `chixiao_in_dau_retention_rate_d7` double COMMENT '赤霄-DAU-7日留存率',
    `chixiao_in_new_user_retention_rate_d1` double COMMENT '赤霄-新增用户-次日留存率',
    `chixiao_in_new_user_retention_rate_d7` double COMMENT '赤霄-新增用户-7日留存率',
    `chixiao_in_old_user_retention_rate_d1` double COMMENT '赤霄-老用户-次日留存率',
    `chixiao_in_old_user_retention_rate_d7` double COMMENT '赤霄-老用户-7日留存率',
    PRIMARY KEY (`imp_date`, `data_date`)
) COMMENT = '短番每日大盘指标-不可聚合'
```

【口径规则】
1. 数据为全量表，取数时必须指定 imp_date，imp_date 从维表 szdw_dim.chuku_progress 中基于 table_name = 'ads_szbot_duanfan_dapan_uv_metrics_zl_df' 取最大的日期
2. data_date 为业务的数据日期，如果求最近 N 天的数据时，默认从昨天开始往前推 N 天
3. 该表为大盘不可聚合表，专门存放 UV、DAU、留存率等不可直接相加的指标
4. UV、DAU、留存率等指标不可直接相加，只能按天展示趋势
5. 该表无 cid、flag、is_szstudio 等维度字段，查询时无需额外过滤条件
6. 该表同时包含星舟视频（szsp_ 前缀）和赤霄（chixiao_ 前缀）指标，如用户未指定端，必须同时返回两端指标，不能只返回其中一端
