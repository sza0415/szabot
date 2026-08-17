```
USE szdw_ads;
CREATE TABLE `ads_szbot_dianshiju_project_consume_metrics_ql_df` (
    `imp_date` bigint NOT NULL COMMENT '日期 格式yyyymmdd',
    `pid` bigint NOT NULL COMMENT '项目id',
    `operation_start_date` varchar(50) COMMENT '运营期开始日期',
    `sxq_sj_completion_rate` varchar(32) COMMENT '首集播放完成度',
    `sxq_sj_retention_rate` varchar(32) COMMENT '首集留存率',
    `sxq_imp_uv` varchar(32) COMMENT '曝光人数-app端',
    `sxq_play_zp_utr` varchar(32) COMMENT '曝光-正片播放UTR-app端',
    `zp_play_uv_app_pc_ott` bigint COMMENT '正片播放UV-三端',
    `sxq_zp_play_uv` varchar(32) COMMENT '正片播放UV-app端',
    `sxq_zp_valid_utr` varchar(32) COMMENT '正片-正片有效播放UTR-app端',
    `sxq_zp_valid_uv` varchar(32) COMMENT '正片有效播放UV-app端',
    `sxq_finish_rate` varchar(32) COMMENT '完播率-app端',
    `sxq_finish_uv` varchar(32) COMMENT '完播人数-app端',
    `sxq_in_vstart_oper_uv_ott` bigint COMMENT '播放UV-三端',
    `sxq_in_vstart_oper_vv_ott` bigint COMMENT '播放VV-三端',
    `history_max_hot_value` varchar(64) COMMENT '历史最高热度值',
    `sxq_zp_play_vv_app_pc_ott` bigint COMMENT '正片VV-三端',
    `sxq_in_vstart_biz_id_oper_nu_app_pc_ott` bigint COMMENT '平台拉新-三端',
    `sxq_in_vstart_type_id_oper_nu_app_pc_ott` bigint COMMENT '品类拉新-三端',
    `sxq_in_imp_uv_app` bigint COMMENT '曝光uv-app端',
    `sxq_in_imp_vv_app` bigint COMMENT '曝光vv-app端',
    `sxq_in_click_ctr_app` double COMMENT '点击ctr-app端',
    `sxq_in_click_utr_app` double COMMENT '点击utr-app端',
    `sxq_first_vid_drop_rate_app_pc_ott` double COMMENT '首集弃剧率-三端',
    `yuyue_keep_appoint_rate_app_pc_ott` double COMMENT '预约履约率-三端',
    `not_auto_yuyue_keep_appoint_rate_app_pc_ott` double COMMENT '预约履约率-剔除弹窗预约-三端',
    `sxq_sucai_num_in_recommend_pool_app` bigint COMMENT '视频素材量-推荐池-app端',
    `sxq_sucai_num_has_imp_app` bigint COMMENT '视频素材量-有曝光-app端',
    `sxq_bullet_interactive_cnt_app` bigint COMMENT '弹幕量-app端',
    `in_imp_vv_position_json_app` varchar(350) COMMENT '曝光vv-分场景-app端',
    `in_imp_uv_position_json_app` varchar(350) COMMENT '曝光uv-分场景-app端',
    `in_play_ctr_position_json_app` varchar(350) COMMENT '播放ctr-分场景-app端',
    `in_play_utr_position_json_app` varchar(350) COMMENT '播放utr-分场景-app端',
    `in_zp_play_ctr_position_json_app` varchar(350) COMMENT '正片播放ctr-分场景-app端',
    `in_zp_play_utr_position_json_app` varchar(350) COMMENT '正片播放utr-分场景-app端',
    `in_pdtm_s_app_pc_ott` double COMMENT '播放时长-三端(单位：秒)',
    `positive_vid_num` bigint COMMENT '集数',
    PRIMARY KEY (`imp_date`, `pid`)

) COMMENT = '电视剧运营期累计消费指标'
```
【口径规则】

1. 数据为累计表，计算了运营期的累计指标，查询时，需要指定imp_date和pid
2. 取累计指标时，imp_date 取该项目的**统计结束日期（`tongji_end_date`）**（参考查询模板 1. 取统计结束日期 `tongji_end_date`）。
3. **正片播放UV**：求正片播放UV时，默认取 `zp_play_uv_app_pc_ott`（正片播放UV-三端）
4. **核心播放指标**（顺序固定，独立成一块展示）：历史最高热度值=`history_max_hot_value`、播放VV=`sxq_in_vstart_oper_vv_ott`、正片VV=`sxq_zp_play_vv_app_pc_ott`、正片有效播放UV=`sxq_zp_valid_uv`、播放时长=`in_pdtm_s_app_pc_ott`（展示为亿分钟，换算：`ROUND(in_pdtm_s_app_pc_ott / 60 / 10000 / 10000, 2)`）、完播率=`sxq_finish_rate`、首集播放完成度=`sxq_sj_completion_rate`、首集弃剧率=`sxq_first_vid_drop_rate_app_pc_ott`
5. **播放转换漏斗**（顺序固定，7个指标独立成一块展示）：曝光人数=`sxq_imp_uv`、曝光-正片播放UTR=`sxq_play_zp_utr`、正片播放UV=`sxq_zp_play_uv`、正片-正片有效播放UTR=`sxq_zp_valid_utr`、正片有效播放UV=`sxq_zp_valid_uv`、完播率=`sxq_finish_rate`、完播人数=`sxq_finish_uv`
6. **集均指标计算**：求集均时，用累计指标除以 `positive_vid_num`（集数）
7. **人均播放时长**：仅在用户明确需要时才查询，**SQL 中直接换算为「分钟」**，写法：`ROUND(in_pdtm_s_app_pc_ott / 60 / sxq_in_vstart_oper_uv_ott, 2) AS pdtm_per_uv_min`，单位标注「分钟/人」