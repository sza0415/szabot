```sql
use szstudio;
CREATE TABLE `ads_aigc_szanimate_video_metrics_df` (
    `imp_date` bigint NOT NULL COMMENT '时间分区(YYYYMMDD),【重要：绝对禁止直接给定固定日期！必须通过子查询从 szdw_dim.chuku_progress 表根据 table_name=ads_aigc_szanimate_video_metrics_df 筛选最大 imp_date 作为值】',
    `create_date` varchar(50) COMMENT '数据日期(YYYYMMDD)',
    `profile_id` bigint COMMENT '用户id',
    `corp_id` bigint COMMENT '企业ID',
    `corp_name` varchar(50) COMMENT '企业名',
    `user_name` varchar(50) COMMENT '用户昵称',
    `is_test` bigint COMMENT '是否测试',
    `mode` varchar(50) COMMENT '方式',
    `animation_mode` varchar(50) COMMENT '动效模式',
    `task_source` varchar(50) COMMENT '来源',
    `project_id` bigint COMMENT '项目ID',
    `series_id` bigint COMMENT '剧集ID',
    `project_name` varchar(200) COMMENT '项目名',
    `series_name` varchar(200) COMMENT '剧集名',
    `has_downloaded` bigint COMMENT '是否被下载过',
    `channel_name` varchar(50) COMMENT '用户使用的channel',
    `drive_mode` varchar(50) COMMENT '产品入口',
    `inner_user_name` varchar(50) COMMENT '内部用户名',
    `model_type_1` varchar(50) COMMENT '模型分类1',
    `model_type_2` varchar(50) COMMENT '模型分类2',
    `model_type_3` varchar(50) COMMENT '模型分类3',
    `is_for_gaoqing` bigint COMMENT '是否被4K高清化过',
    `corp_type_name` varchar(50) COMMENT '企业类型',
    `model` varchar(200) COMMENT 'model',
    `task_type_id` varchar(100) COMMENT '任务类型ID',
    `pv` bigint COMMENT '素材数，可以聚合',
    KEY `index` (`imp_date`, `create_date`)
) COMMENT = 'SzStudio生成视频数量'
```

## 【口径规则】

1. 数据为全量表，取数时必须指定imp_date，imp_date从维表szdw_dim.chuku_progress中基于table_name=ads_aigc_szanimate_video_metrics_df取最大的日期
2. 查询时带上数据库，参考szstudio.ads_aigc_szanimate_video_metrics_df
3. 默认is_test为0
4. 数据日期用create_date
5. 求视频数或者视频素材数，直接对pv求和