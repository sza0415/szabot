```
USE szpp;
CREATE TABLE `t_mini_tort_metrics` (
    `id` bigint UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'id',
    `imp_date` bigint NOT NULL COMMENT '时间分区(YYYYMMDDHH)，【重要：绝对禁止直接给定固定日期！必须通过子查询从 szdw_dim.chuku_progress 表根据 table_name=miniapp_progress_hour 筛选最大 imp_hour 作为值】',
    `ip_id` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '版权ID',
    `doc_id` bigint NOT NULL DEFAULT '0' COMMENT 'doc_id',
    `ip_name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '版权名称',
    `platform` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '侵权平台，必须指定，默认查询指定ALL',
    `platform_type_name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '平台类型，必须指定，默认查询指定ALL',
    `company` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL DEFAULT '' COMMENT '公司',
    `letter_num` bigint NOT NULL DEFAULT '0' COMMENT '发函量',
    `tort_delete_num` bigint NOT NULL DEFAULT '0' COMMENT '下架量,暂时不包含搜索引擎',
    `p95_tort_delete_duration` bigint NOT NULL DEFAULT '0' COMMENT 'p95下架时效',
    `play_vv` bigint NOT NULL DEFAULT '0' COMMENT '侵权播放量',
    `cate_name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci NOT NULL COMMENT 'IP品类',
    `letter_num_no_se` bigint NOT NULL DEFAULT '0' COMMENT '不含搜索引擎发函量',
    `tort_delete_in_48h_num` bigint NOT NULL DEFAULT '0' COMMENT '48小时内下架链接数量，不含搜索引擎',
    `avg_protect_duration` int NOT NULL DEFAULT '0' COMMENT '平均防护时长',
    `svip_avg_protect_duration` int NOT NULL DEFAULT '0' COMMENT 'SVIP平均防护时长',
    `dianying_avg_protect_duration` int NOT NULL DEFAULT '0' COMMENT '点映礼平均防护时长',
    `base_qualify_cnt` int NOT NULL DEFAULT '0' COMMENT '基础达标数',
    `over_qualify_cnt` int NOT NULL DEFAULT '0' COMMENT '冲高达标数',
    `protect_cnt` int NOT NULL DEFAULT '0' COMMENT '介质防护总数',
    `svip_vid_cnt` int NOT NULL DEFAULT '0' COMMENT 'svip防护集数',
    `dianying_vid_cnt` int NOT NULL DEFAULT '0' COMMENT '点映礼防护集数',
    `svip_latest_duration` int NOT NULL DEFAULT '0' COMMENT 'svip最近介质防护时长',
    `etl_stamp` varchar(200) CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci DEFAULT NULL,
    `contribute_valid_clue_nums` int NOT NULL DEFAULT '0' COMMENT 'C端录入-贡献有效线索量',
    `valid_clue_nums_new` int NOT NULL DEFAULT '0' COMMENT 'C端录入-新增有效线索量',
    `crawl_coverage` double NOT NULL DEFAULT '0' COMMENT '抓取覆盖率',
    `all_clue_nums` bigint NOT NULL DEFAULT '0' COMMENT 'C端录入-线索总量',
    `judge_clue_nums` bigint NOT NULL DEFAULT '0' COMMENT 'C端录入-处理中线索量',
    `valid_clue_nums` bigint NOT NULL DEFAULT '0' COMMENT 'C端录入-已处理线索量',
    `valid_clue_nums_letter` bigint NOT NULL DEFAULT '0' COMMENT 'C端录入-发函量',
    `valid_clue_nums_delete_before` bigint NOT NULL DEFAULT '0' COMMENT 'C端录入-发函前下架量',
    PRIMARY KEY (`id`, `imp_date`),
    KEY `idx_ip` (`imp_date`, `ip_id`)
) ENGINE = InnoDB AUTO_INCREMENT = 198926340 DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci COMMENT = '运营期内作品累计统计'
```

【口径规则】
1. 数据为全量表，取数时必须指定imp_date，imp_date从维表szdw_dim.chuku_progress中基于table_name=miniapp_progress_hour取最大的imp_hour
2. 查询时，必须指定platform_type_name和platform，默认查询指定ALL。
3. 分平台类型时，platform_type_name != ALL
4. 分平台时，platform ！= ALL
5. 侵权场景、侵权类型和平台类型，是一个意思
6. 如果从该表算下架率时，用letter_num_no_se作为分母
7. 求发函量时，用letter_num
8. 查询时必须指定IP，可选ip_name、ip_id、doc_id。查询时，需要精确匹配