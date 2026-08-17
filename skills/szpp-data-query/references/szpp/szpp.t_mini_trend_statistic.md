```
USE szpp;
CREATE TABLE `t_mini_trend_statistic` (
    `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'id',
    `imp_date` bigint NOT NULL COMMENT '时间分区(YYYYMMDDHH)，【重要：绝对禁止直接给定固定日期！必须通过子查询从 szdw_dim.chuku_progress 表根据 table_name=miniapp_progress_hour 筛选最大 imp_hour 作为值】',
    `ip_id` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '版权ID',
    `doc_id` bigint NOT NULL DEFAULT '0' COMMENT 'doc_id',
    `ip_name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '版权名称',
    `platform` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '侵权平台',
    `platform_type_name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '平台类型',
    `company` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '公司',
    `belike_tort_num` bigint NOT NULL DEFAULT '0' COMMENT '疑似侵权量',
    `tort_num` bigint NOT NULL DEFAULT '0' COMMENT '打击量',
    `letter_num` bigint NOT NULL DEFAULT '0' COMMENT '发函量',
    `tort_delete_num` bigint NOT NULL DEFAULT '0' COMMENT '下架量',
    `hour_diff` int NOT NULL DEFAULT '0' COMMENT '小时差',
    `date_diff` int NOT NULL DEFAULT '0' COMMENT '日期差，是指上线后的日期差，方便不同IP进行对比',
    `etl_stamp` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
    `cate_name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT 'IP品类',
    `protect_duration` int NOT NULL DEFAULT '0' COMMENT '介质防护时长',
    PRIMARY KEY (`id`, `imp_date`),
    KEY `idx_ip` (`imp_date`, `ip_id`)
) ENGINE = InnoDB AUTO_INCREMENT = 6513815931 DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci COMMENT = '运营期内作品侵权趋势表'
```

【口径规则】
1. 数据为全量表，取数时必须指定imp_date，imp_date从维表szdw_dim.chuku_progress中基于table_name=miniapp_progress_hour取最大的imp_hour
2. 侵权趋势数据，优先用本表查询
3. 侵权场景、侵权类型和平台类型，是一个意思
4. 求趋势时，默认基于date_diff升序，如果明确指定小时的趋势时，才用hour_diff。
5. 分析趋势数据时，不看platform_type_name = 长尾网站
6. 分IP对比时，将IP数据放在一起对比，不要分开
7. 查询时必须指定IP，可选ip_name、ip_id、doc_id。查询时，需要精确匹配
8. 求下架率时：用下架量/发函量