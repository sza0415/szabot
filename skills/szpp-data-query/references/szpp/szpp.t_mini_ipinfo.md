```
USE szpp;
CREATE TABLE `t_mini_ipinfo` (
    `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'id',
    `imp_date` bigint NOT NULL COMMENT '时间分区(YYYYMMDDHH)，【重要：绝对禁止直接给定固定日期！必须通过子查询从 szdw_dim.chuku_progress 表根据 table_name=miniapp_progress_hour 筛选最大 imp_hour 作为值】',
    `ip_id` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '版权ID',
    `doc_id` bigint NOT NULL DEFAULT '0' COMMENT 'doc_id',
    `ip_name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '版权名称',
    `hot_level` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '旧维权等级',
    `budget_level` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '维权等级',
    `cate_name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'IP品类',
    `monitor_status` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '监测状态',
    `avg_letter_num_on_operation_period` bigint NOT NULL DEFAULT '0' COMMENT '运营期内日均发函量',
    `etl_stamp` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
    `play_time_start` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '开始播放时间',
    `play_time_end` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '完结时间',
    `hot_level_sort` tinyint NOT NULL DEFAULT '0' COMMENT '剧集等级排序字段',
    `task_type` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '维权任务等级',
    `cate_name_sort` tinyint NOT NULL DEFAULT '0' COMMENT 'IP品类排序字段',
    `accumulate_letter_num_on_operation_period` bigint NOT NULL DEFAULT '0' COMMENT '运营期内累积发函量',
    `hot_value_level` int NOT NULL DEFAULT '0' COMMENT '历史最高热度级别',
    `ahead_play_start_date` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '' COMMENT '超前点播开始时间',
    `main_type` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '' COMMENT '主要类型',
    `secondary_type` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT '' COMMENT '次要类型',
    PRIMARY KEY (`id`, `imp_date`),
    KEY `idx_ip` (`imp_date`, `ip_id`)
) ENGINE = InnoDB AUTO_INCREMENT = 51221632 DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci COMMENT = '小程序外显IP信息'
```
【口径规则】
1. 数据为全量表，取数时必须指定imp_date，imp_date从维表szdw_dim.chuku_progress中基于table_name=miniapp_progress_hour取最大的imp_hour
2. 查询时必须指定IP，可选ip_name、ip_id、doc_id。查询时，需要精确匹配