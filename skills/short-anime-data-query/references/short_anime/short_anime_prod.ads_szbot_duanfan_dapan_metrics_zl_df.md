```sql
USE short_anime_prod;
CREATE TABLE `ads_szbot_duanfan_dapan_metrics_zl_df` (
    `imp_date` bigint NOT NULL COMMENT '时间分区(YYYYMMDD)',
    `data_date` bigint NOT NULL COMMENT '数据日期(YYYYMMDD)',
    `is_szsp_duanfan` bigint NOT NULL COMMENT '是否星舟视频的短番',
    `is_chixiao_duanfan` bigint NOT NULL COMMENT '是否chixiao的短番',
    `is_szstudio` bigint NOT NULL COMMENT '是否szstudio',
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
    `szsp_cost` double COMMENT '星舟视频-成本szsp_creator_income + szsp_creator_kou_bu_income，时效T-2',
    `chixiao_in_vstart_cnt` bigint COMMENT '赤霄-播放开始次数',
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
    PRIMARY KEY (`imp_date`, `data_date`, `is_szsp_duanfan`, `is_chixiao_duanfan`, `is_szstudio`),
    KEY `idx_data_date` (`data_date`)
) COMMENT = '每日大盘指标-可聚合'

【口径规则】
1. 数据为全量表，取数时必须指定 imp_date，imp_date 从维表 szdw_dim.chuku_progress 中基于 table_name = 'ads_szbot_duanfan_dapan_metrics_zl_df' 取最大的日期
2. data_date 为业务的数据日期，如果求最近 N 天的数据时，默认从昨天开始往前推 N 天
3. 收入相关指标（szsp_income、chixiao_income 等）时效为 T-2，即今天能看到的最新收入数据为前天
4. 口径别名：开始播放次数、播放量、播放vv 是一个概念
5. 该表为大盘汇总表，不含 cid 维度；如需按专辑查询，请使用 ads_szbot_duanfan_cid_metrics_zl_df_new
6. 如果用户查询 SzStudio 的数据，需额外加 is_szstudio = 1 过滤
7. 该表同时包含星舟视频（szsp_ 前缀）和赤霄（chixiao_ 前缀）指标，如用户未指定端，必须同时返回两端指标，不能只返回其中一端
8. 【聚合规则】该表主键为 (imp_date, data_date, is_szsp_duanfan, is_chixiao_duanfan, is_szstudio)，同一 data_date 下存在多行（不同维度组合）。查询大盘整体数据时，必须对指标进行 SUM 聚合，不能直接 SELECT 单行。例如查询星舟视频短番大盘播放量，应使用 SUM(szsp_in_vstart_cnt) 并按 data_date GROUP BY，而不是直接取某一行的值
9. 【播放时长单位换算 - 仅限本表】查询**本表（大盘表）**的播放时长时，szsp_in_pdtm_s 和 chixiao_in_pdtm_s 原始单位为**秒**，展示时需转换为**万小时**，换算公式：`SUM(szsp_in_pdtm_s) / 3600 / 10000`，必须在 SQL 中完成换算，禁止查出原始秒数后再手动换算。⛔ 此换算规则**不适用于专辑表**（cid_metrics_zl_df_new、cid_metrics_ql_df），专辑表的播放时长直接返回原始秒数。注意：szsp_paid_playtime 原始单位为**分钟**（非秒），是会员播放时长，与 szsp_in_pdtm_s 是不同字段，不要混淆