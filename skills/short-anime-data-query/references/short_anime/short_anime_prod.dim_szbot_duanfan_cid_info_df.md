```
USE short_anime_prod;
CREATE TABLE `dim_szbot_duanfan_cid_info_df` (
    `imp_date` varchar(32) NOT NULL COMMENT '时间分区(YYYYMMDD)，【重要：绝对禁止直接给定固定日期！必须通过子查询从 szdw_dim.chuku_progress 表根据 table_name=dim_szbot_duanfan_cid_info_df 筛选最大 imp_date 作为值】',
    `cid` varchar(64) NOT NULL COMMENT 'cid',
    `shangjia_time` varchar(64) COMMENT '上架时间',
    `shangjia_date` varchar(32) COMMENT '上架日期',
    `qiangshi_type` varchar(64) COMMENT '强势品类类型',
    `is_duanfan` bigint COMMENT '是否短番',
    `poi_id` text COMMENT '所属兴趣点',
    `is_ecological` bigint COMMENT '是否生态',
    `is_szstudio` bigint COMMENT '是否szstudio',
    `is_selfmade` bigint COMMENT '是否自制',
    `total_vid_cnt` bigint COMMENT 'vid数',
    `total_vid_duration_sum` double COMMENT 'vid视频时长(秒)汇总',
    `valid_vid_cnt` bigint COMMENT '有效vid数，取已上架状态',
    `valid_vid_duration_sum` double COMMENT '有效vid视频时长(秒)汇总，取已上架状态',
    `chixiao_content` varchar(512) COMMENT '短番app内容标记',
    `content_is_duanfan_by_chixiao` bigint COMMENT 'chixiao标记的短番(chixiao_content包含123134934)',
    `checkup_state` varchar(64) COMMENT '专辑状态',
    `is_szsp_duanfan` bigint COMMENT '是否星舟视频的短番',
    `is_chixiao_duanfan` bigint COMMENT '是否chixiao的短番',
    `title` text COMMENT '标题',
    `new_pic_vt` varchar(1024) COMMENT '海报，新专辑竖图',
    `long_video_list` text COMMENT '长视频列表',
    `vcuid` text COMMENT 'vcuid',
    `created_at` datetime NOT NULL COMMENT '创建时间',
    `updated_at` datetime NOT NULL COMMENT '更新时间',
    `video_num` int COMMENT '集数',
    `creater_nick` text COMMENT '账号昵称',
    `micro_series_distributor` json COMMENT '发行方',
    `original_dujia_cid` int COMMENT '独家cid',
    `is_valid` int COMMENT 'cid是否有效',
    `data_type_id` text COMMENT '数据类型ID',
    `is_formal_cid` int COMMENT '是否正式环境cid',
    KEY `idx_cid` (`cid`),
    KEY `idx_imp_date` (`imp_date`),
    KEY `idx_valid` (`imp_date`, `is_valid`, `cid`)
) COMMENT = 'cid信息'
```
【口径规则】
1. 数据为全量维表，取数时必须指定imp_date，imp_date从维表szdw_dim.chuku_progress中基于table_name=dim_szbot_duanfan_cid_info_df取最大的日期
2. 存放的是cid的信息
3. ⛔ **禁止对本表任何字段使用模糊匹配（LIKE）**，所有字段必须精确匹配（使用 `=` 或 `IN`）