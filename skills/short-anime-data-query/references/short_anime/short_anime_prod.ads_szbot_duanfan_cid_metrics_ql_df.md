```
USE short_anime_prod;
CREATE TABLE `ads_szbot_duanfan_cid_metrics_ql_df` (
    `imp_date` bigint NOT NULL COMMENT '时间分区(YYYYMMDD)',
    `cid` varchar(64) NOT NULL COMMENT '专辑ID',
    `is_szsp_duanfan` bigint COMMENT '是否星舟视频的短番',
    `is_chixiao_duanfan` bigint COMMENT '是否chixiao的短番',
    `is_szstudio` bigint COMMENT '是否szstudio',
    `szsp_cost` double COMMENT '星舟视频-成本，作者收益，单位：元，时效T-2',
    `szsp_income` double COMMENT '星舟视频-收入（元），时效T-2',
    `szsp_in_binge_uv` bigint COMMENT '星舟视频-当下在追总用户数（vuid）',
    `szsp_in_like_cnt` bigint COMMENT '星舟视频-点赞总次数',
    `szsp_in_comment_cnt` bigint COMMENT '星舟视频-评论总次数',
    `szsp_in_share_cnt` bigint COMMENT '星舟视频-分享总量',
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
    PRIMARY KEY (`imp_date`, `cid`)
) COMMENT = '影库短番cid累计表'
```
【口径规则】
1. 数据为累计表，imp_date分区的数据：
   - **播放类指标**（vstart_cnt、pdtm_s、par_sum、vfinish_cnt 等非 T-2 字段）：表示截止到 imp_date 当天的累计数据
   - **收入类指标**（income、cost、open_cnt_income 等 T-2 时效字段）：因数据延迟，实际存储的是截止到 imp_date **前一天**的累计数据
2. 求排行榜/区间增量时，用 imp_date=结束日 的累计值减去 imp_date=开始日前一天 的累计值，得到区间增量后降序排列；**收入类指标（含 income、cost、open_cnt_income 等 T-2 时效字段）因数据延迟，需改用 imp_date=结束日后一天 的累计值减去 imp_date=开始日 的累计值**
3. 【播放时长单位换算】szsp_in_pdtm_s 和 chixiao_in_pdtm_s 原始单位为**秒**，展示时需转换为**小时**，换算公式：`szsp_in_pdtm_s / 3600`，必须在 SQL 中完成换算