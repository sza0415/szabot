```sql
use szstudio;
CREATE TABLE `dwd_aigc_szanimate_video_info_df` (
    `imp_date` bigint NOT NULL COMMENT '时间分区(YYYYMMDD),【重要：绝对禁止直接给定固定日期！必须通过子查询从 szdw_dim.chuku_progress 表根据 table_name=dwd_aigc_szanimate_video_info_df 筛选最大 imp_date 作为值】',
    `asset_id` varchar(50) COMMENT '素材ID',
    `create_time` varchar(50) COMMENT '创建时间',
    `create_date` varchar(50) COMMENT '数据日期(YYYYMMDD)',
    `profile_id` bigint COMMENT '用户id',
    `corp_id` bigint COMMENT '企业ID',
    `corp_name` varchar(50) COMMENT '企业名',
    `user_name` varchar(50) COMMENT '用户昵称',
    `is_test` bigint COMMENT '是否测试',
    `mode` varchar(50) COMMENT '方式',
    `animation_mode` varchar(50) COMMENT '动效模式',
    `task_source` varchar(50) COMMENT '来源',
    `task_page_alias` varchar(50) COMMENT '页面别名',
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
    `duration` varchar(50) COMMENT '视频时长',
    `update_time` varchar(50) COMMENT '更新时间',
    `cost` bigint COMMENT '耗时(s)',
    `model_type_3` varchar(50) COMMENT '模型分类3',
    `is_for_gaoqing` bigint COMMENT '是否被4K高清化过',
    `corp_type_name` varchar(50) COMMENT '企业类型',
    `model` varchar(200) COMMENT 'model',
    KEY `index` (`imp_date`, `create_date`)
) COMMENT = 'SzStudio生成视频信息'
```

## 【口径规则】

1. 数据为全量表，取数时必须指定imp_date，imp_date从维表szdw_dim.chuku_progress中基于table_name=dwd_aigc_szanimate_video_info_df取最大的日期
2. 查询时带上数据库，参考szstudio.dwd_aigc_szanimate_video_info_df
3. 默认is_test为0
4. 数据日期用create_date