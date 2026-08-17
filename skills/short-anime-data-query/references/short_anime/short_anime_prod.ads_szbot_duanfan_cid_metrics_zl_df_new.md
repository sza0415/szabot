```sql
USE short_anime_prod;
CREATE TABLE `ads_szbot_duanfan_cid_metrics_zl_df_new` (
    `imp_date` bigint NOT NULL COMMENT '时间分区(YYYYMMDD)',
    `cid` varchar(64) NOT NULL COMMENT '专辑ID',
    `is_szsp_duanfan` bigint COMMENT '是否星舟视频的短番',
    `is_chixiao_duanfan` bigint COMMENT '是否chixiao的短番',
    `data_date` bigint NOT NULL COMMENT '统计日期(YYYYMMDD)',
    `total_in_vstart_uv` bigint COMMENT '汇总-播放开始设备数',
    `szsp_in_vstart_uv` bigint COMMENT '星舟视频-播放开始设备数',
    `szsp_in_vstart_cnt` bigint COMMENT '星舟视频-开始播放次数',
    `szsp_in_par_sum` double COMMENT '星舟视频-完播率求和',
    `szsp_in_vfinish_cnt` bigint COMMENT '星舟视频-播放结束次数',
    `szsp_in_pdtm_s` double COMMENT '星舟视频-播放时长，单位：秒',
    `szsp_hvip_normal_open_cnt` bigint COMMENT '星舟视频-hvip正价开通数，时效T-2',
    `szsp_hvip_discount_open_cnt` bigint COMMENT '星舟视频-hvip低价开通数，时效T-2',
    `szsp_hvip_open_users` bigint COMMENT '星舟视频-hvip开通数，时效T-2',
    `szsp_platform_income_hvip` double COMMENT '星舟视频-hvip开通收入，时效T-2',
    `szsp_svip_normal_open_cnt` bigint COMMENT '星舟视频-svip正价开通数，时效T-2',
    `szsp_svip_discount_open_cnt` bigint COMMENT '星舟视频-svip低价开通数，时效T-2',
    `szsp_svip_open_users` bigint COMMENT '星舟视频-svip开通数，时效T-2',
    `szsp_platform_income_svip` double COMMENT '星舟视频-svip开通收入，时效T-2',
    `szsp_open_users_cnt` bigint COMMENT '星舟视频-会员开通总数，时效T-2',
    `szsp_open_cnt_income` double COMMENT '星舟视频-开通总收入，时效T-2',
    `szsp_is_pay_play_nums` bigint COMMENT '星舟视频-付费播放vv，时效T-2',
    `szsp_paid_cnt_income` double COMMENT '星舟视频-付费vv收入，时效T-2',
    `szsp_silent_recallucnt_income` double COMMENT '星舟视频-沉默回活收入，时效T-2',
    `szsp_platform_income` double COMMENT '星舟视频-平台收入，时效T-2',
    `szsp_paid_playtime` double COMMENT '星舟视频-会员播放时长，单位：分钟，时效T-2',
    `szsp_income` double COMMENT '星舟视频-收入（元），时效T-2',
    `szsp_creator_income` double COMMENT '星舟视频-收益,单位:元，时效T-2',
    `szsp_creator_kou_bu_income` double COMMENT '星舟视频-扣款/补款金额,单位:元，时效T-2',
    `szsp_cost` double COMMENT '星舟视频-成本szsp_creator_income + szsp_creator_kou_bu_income，时效T-2',
    `chixiao_in_vstart_cnt` bigint COMMENT '赤霄-播放开始次数',
    `chixiao_in_vstart_uv` bigint COMMENT '赤霄-播放开始设备数',
    `chixiao_in_par_sum` double COMMENT '赤霄-播放完播率和',
    `chixiao_in_vfinish_cnt` bigint COMMENT '赤霄-播放结束次数',
    `chixiao_in_pdtm_s` double COMMENT '赤霄-播放时长，单位：秒',
    `chixiao_income` double COMMENT '赤霄-总收入，单位：元',
    `chixiao_ad_income` double COMMENT '赤霄-广告收入，单位：元',
    `chixiao_cost` double COMMENT '赤霄-成本，单位：元',
    `video_num` bigint COMMENT '集数（更新集数）',
    `duanyin_out_vstart_cnt` bigint COMMENT '短音-播放量',
    `jutantan_out_vstart_cnt` bigint COMMENT '剧探探-播放量',
    `weview_out_vstart_cnt` bigint COMMENT 'WEVIEW-播放量',
    PRIMARY KEY (`imp_date`, `cid`, `data_date`),
    KEY `idx_imp_date_data_date_cid` (`imp_date`, `data_date`, `cid`),
    KEY `idx_is_chixiao_duanfan` (`is_chixiao_duanfan`)
) COMMENT = '短番每日增量指标'

【口径规则】
1. 数据为全量表，取数时必须指定imp_date，imp_date从维表szdw_dim.chuku_progress中基于table_name=ads_szbot_duanfan_cid_metrics_zl_df_new取最大的日期
2. data_date为业务的数据日期，如果求最近N天的数据时，默认从昨天开始往前推N天
3. 求最近一段时间的大盘指标时，需要按天聚合指标，而不是直接求和
4. 口径别名：开始播放次数、播放量、播放vv是一个概念
5. 每次查询时，需要指定cid，防止查询异常
7. UV等指标，不可以直接相加
9. 该表同时包含星舟视频（szsp_ 前缀）和赤霄（chixiao_ 前缀）指标，如用户未指定端，必须同时返回两端指标，不能只返回其中一端
10. 【播放时长单位换算】szsp_in_pdtm_s 和 chixiao_in_pdtm_s 原始单位为**秒**，展示时需转换为**小时**，换算公式：`szsp_in_pdtm_s / 3600`，必须在 SQL 中完成换算