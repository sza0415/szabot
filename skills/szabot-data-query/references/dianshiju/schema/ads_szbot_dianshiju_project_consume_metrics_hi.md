```
USE szdw_ads;
CREATE TABLE `ads_szbot_dianshiju_project_consume_metrics_hi` (
    `imp_date` bigint NOT NULL COMMENT '日期 格式yyyymmddhh',
    `pid` bigint NOT NULL COMMENT '项目id',
    `estimated_vv_app_pc_ott` bigint COMMENT '预估vv-三端',
    `estimated_vv_time` varchar(100) COMMENT '预估时间',
    PRIMARY KEY (`imp_date`, `pid`)
) COMMENT = '影库电视剧项目小时级增量表'
```

## 口径规则

1. 当需要取最新数据时，`imp_date` 字段的分区值从维表 `szdw_dim.chuku_progress` 中获取，基于 `table_name = 'ads_szbot_dianshiju_project_consume_metrics_hi'` 条件，取最大的 `imp_hour` 作为分区值 

查询模板
```sql
-- db_name: szdw_ads
SELECT imp_date, pid,
    estimated_vv_app_pc_ott,
    estimated_vv_time
FROM ads_szbot_dianshiju_project_consume_metrics_hi
WHERE pid = {项目ID}
AND imp_date = (
    SELECT MAX(imp_hour) FROM szdw_dim.chuku_progress
    WHERE table_name = 'ads_szbot_dianshiju_project_consume_metrics_hi'
);
```