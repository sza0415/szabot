```
USE szdw_ads;
CREATE TABLE `dim_szbot_dianshiju_project_info_hf` (
    `imp_hour` bigint NOT NULL COMMENT '时间分区 格式YYYYMMDDHH',
    `pid` bigint NOT NULL COMMENT '项目ID',
    `pname` varchar(255) COMMENT '项目名称',
    `operation_start_date` varchar(50) COMMENT '运营期开始日期',
    `operation_end_date` varchar(20) COMMENT '运营期结束日期',
    `douban_score` varchar(100) COMMENT '豆瓣评分',
    PRIMARY KEY (`imp_hour`, `pid`)
) COMMENT = '影库电视剧项目信息'
```
【口径规则】
1. 数据为全量表，取数时必须指定imp_hour，imp_hour从维表szdw_dim.chuku_progress中基于table_name=dim_szbot_dianshiju_project_info_hf取最大的imp_hour
2. 支持通过项目ID（pid）或项目名称（pname）精确匹配查询，两者均为精确匹配，不支持模糊查询

查询模板

```sql
-- db_name: szdw_ads
SELECT pid, pname, operation_start_date, operation_end_date, douban_score
FROM dim_szbot_dianshiju_project_info_hf
WHERE imp_hour = (
    SELECT MAX(imp_hour) FROM szdw_dim.chuku_progress
    WHERE table_name = 'dim_szbot_dianshiju_project_info_hf'
)
AND pid = {项目ID};
```
