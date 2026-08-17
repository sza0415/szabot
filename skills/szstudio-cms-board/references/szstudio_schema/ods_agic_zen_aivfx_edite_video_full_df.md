```sql
use szstudio;
CREATE TABLE `ods_agic_zen_aivfx_edite_video_full_df` (
    `imp_date` bigint NOT NULL COMMENT '时间分区(YYYYMMDD),【重要：绝对禁止直接给定固定日期！必须通过子查询从 szdw_dim.chuku_progress 表根据 table_name=ods_agic_zen_aivfx_edite_video_full_df 筛选最大 imp_date 作为值】',
    `asset_id` varchar(255) COMMENT '视频ID',
    `create_time` varchar(50) COMMENT '创建时间',
    `create_date` varchar(50) COMMENT '数据日期(YYYYMMDD)',
    `task_id` varchar(255) COMMENT '任务id',
    `profile_id` bigint COMMENT '用户id',
    `corp_id` bigint COMMENT '企业ID',
    `corp_name` varchar(255) COMMENT '企业名',
    `user_name` varchar(255) COMMENT '用户昵称',
    `is_test` bigint COMMENT '是否测试',
    `has_downloaded` bigint COMMENT '是否被下载过',
    `drive_mode` varchar(255) COMMENT '产品入口',
    `corp_type_name` varchar(50) COMMENT '企业类型',
    `szbot_project_id` bigint COMMENT '影库项目ID',
    `szbot_project_name` varchar(50) COMMENT '影库项目名称',
    KEY `index` (`imp_date`, `create_date`)
) COMMENT = '影视后期制作素材'
```

## 【口径规则】
1. 数据为全量表，取数时必须指定imp_date，imp_date从维表szdw_dim.chuku_progress中基于table_name=ods_agic_zen_aivfx_edite_video_full_df取最大的日期
2. 默认is_test为0
3. 数据日期用create_date
4. 算数时，基于asset_id去重统计
5. 查询影视后期相关视频数、素材数时，需要查询本表
6. 按项目查询时，直接用 `szbot_project_name` 或 `szbot_project_id` 作为过滤条件；两个字段均需**精确匹配**（使用 `=`，禁止使用 `LIKE` 或模糊匹配）
7. 目前drive_mode有以下这些值
    - 视频编辑
    - 工具箱-影视超分
    - 工具箱-视频生成
    - 后期配音
    - 口型修改
