```sql
use szstudio;
CREATE TABLE `dwd_aigc_szanimate_image_info_df` (
    `imp_date` bigint NOT NULL COMMENT '时间分区(YYYYMMDD),【重要：绝对禁止直接给定固定日期！必须通过子查询从 szdw_dim.chuku_progress 表根据 table_name=dwd_aigc_szanimate_image_info_df 筛选最大 imp_date 作为值】',
    `asset_id` varchar(50) COMMENT '图片ID',
    `create_time` varchar(50) COMMENT '创建时间',
    `create_date` varchar(20) COMMENT '数据日期(YYYYMMDD)',
    `profile_id` bigint COMMENT '用户id',
    `model` varchar(100) COMMENT '背景模型',
    `task_type_id` varchar(50) COMMENT '具体工作类型id',
    `mode` varchar(50) COMMENT '出图方式',
    `channel` varchar(50) COMMENT '平台',
    `menu` varchar(50) COMMENT '菜单',
    `image_mode_level1` varchar(50) COMMENT '出图模式一级',
    `image_mode_level2` varchar(50) COMMENT '出图模式二级',
    `corp_id` bigint COMMENT '企业ID',
    `corp_name` varchar(50) COMMENT '企业名',
    `corp_type` bigint COMMENT '0 商务 1 工作室',
    `user_name` varchar(50) COMMENT '用户昵称',
    `lora_model_name` varchar(100) COMMENT 'lora模型名',
    `lora_train_model_name` varchar(100) COMMENT '训练用的模型名称',
    `lora_role_id` bigint COMMENT '外部 ID',
    `lora_role_name` varchar(100) COMMENT '角色名',
    `is_test` bigint COMMENT '是否测试',
    `model_name` varchar(100) COMMENT '背景模型名',
    `biz_data` text COMMENT 'biz_data',
    `upload_source` varchar(50) COMMENT '素材来源',
    `has_downloaded` bigint COMMENT '是否被下载过',
    `project_id` bigint COMMENT '项目ID',
    `series_id` bigint COMMENT '剧集ID',
    `project_name` varchar(100) COMMENT '项目名',
    `series_name` varchar(200) COMMENT '剧集名',
    `model_type` varchar(50) COMMENT '模式类型',
    `channel_name` varchar(50) COMMENT '用户使用的channel',
    `ckpt_type` varchar(50) COMMENT '模型类型',
    `model_name_type` varchar(50) COMMENT '背景模型分类',
    `inner_user_name` varchar(50) COMMENT '内部用户名',
    `corp_type_name` varchar(50) COMMENT '企业类型',
    KEY `index` (`imp_date`, `create_date`)
) COMMENT = 'SzStudio生成图片信息'
```
## 【口径规则】

1. 数据为全量表，取数时必须指定imp_date，imp_date从维表szdw_dim.chuku_progress中基于table_name=dwd_aigc_szanimate_image_info_df取最大的日期
2. 查询时带上数据库，参考szstudio.dwd_aigc_szanimate_image_info_df
3. 默认is_test为0
4. 数据日期用create_date