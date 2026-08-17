```
CREATE TABLE `ads_zongyi_dapan_metrics_di` (
    `imp_date` bigint NOT NULL COMMENT '日期 格式YYYYMMDD',
    `in_vstart_vv_app` bigint COMMENT '播放VV-app端',
    `in_vstart_vv_pc` bigint COMMENT '播放VV-pc端',
    `in_vstart_vv_ott` bigint COMMENT '播放VV-ott端',
    `in_vstart_uv_app` bigint COMMENT '播放UV-app端',
    `in_vstart_uv_pc` bigint COMMENT '播放UV-pc端',
    `in_vstart_uv_ott` bigint COMMENT '播放UV-ott端',
    `in_pdtm_s_app` double COMMENT '播放时长-app端',
    `in_pdtm_s_pc` double COMMENT '播放时长-pc端',
    `in_pdtm_s_ott` double COMMENT '播放时长-ott端',
    `new_hot_in_vstart_vv_app` bigint COMMENT '新热剧-播放VV-app端',
    `new_hot_in_vstart_vv_pc` bigint COMMENT '新热剧-播放VV-pc端',
    `new_hot_in_vstart_vv_ott` bigint COMMENT '新热剧-播放VV-ott端',
    `new_hot_in_pdtm_s_app` double COMMENT '新热剧-播放时长-app端',
    `new_hot_in_pdtm_s_pc` double COMMENT '新热剧-播放时长-pc端',
    `new_hot_in_pdtm_s_ott` double COMMENT '新热剧-播放时长-ott端',
    `in_effective_uv_app` bigint COMMENT '有效UV-app端',
    `in_3s_effective_vv_app` bigint COMMENT '3s有效VV-app端',
    `long_video_in_vstart_vv_dianbo_app` bigint COMMENT '长视频-播放VV-点播-app端',
    `long_video_in_vstart_vv_dianbo_pc` bigint COMMENT '长视频-播放VV-点播-pc端',
    `long_video_in_vstart_vv_dianbo_ott` bigint COMMENT '长视频-播放VV-点播-ott端',
    `long_video_in_vstart_uv_dianbo_app` bigint COMMENT '长视频-播放UV-点播-app端',
    `long_video_in_vstart_uv_dianbo_pc` bigint COMMENT '长视频-播放UV-点播-pc端',
    `long_video_in_vstart_uv_dianbo_ott` bigint COMMENT '长视频-播放UV-点播-ott端',
    `long_video_in_pdtm_s_dianbo_app` double COMMENT '长视频-播放时长-点播-app端',
    `long_video_in_pdtm_s_dianbo_pc` double COMMENT '长视频-播放时长-点播-pc端',
    `long_video_in_pdtm_s_dianbo_ott` double COMMENT '长视频-播放时长-点播-ott端',
    `eco_in_vstart_vv_dianbo_app` bigint COMMENT '生态-播放VV-点播-app端',
    `eco_in_vstart_vv_dianbo_pc` bigint COMMENT '生态-播放VV-点播-pc端',
    `eco_in_vstart_vv_dianbo_ott` bigint COMMENT '生态-播放VV-点播-ott端',
    `eco_in_vstart_uv_dianbo_app` bigint COMMENT '生态-播放UV-点播-app端',
    `eco_in_vstart_uv_dianbo_pc` bigint COMMENT '生态-播放UV-点播-pc端',
    `eco_in_vstart_uv_dianbo_ott` bigint COMMENT '生态-播放UV-点播-ott端',
    `eco_in_pdtm_s_dianbo_app` double COMMENT '生态-播放时长-点播-app端',
    `eco_in_pdtm_s_dianbo_pc` double COMMENT '生态-播放时长-点播-pc端',
    `eco_in_pdtm_s_dianbo_ott` double COMMENT '生态-播放时长-点播-ott端',
    `long_video_hvip_normal_ucnt` bigint COMMENT '长视频-hvip正价驱动开通人数',
    `long_video_svip_normal_ucnt` bigint COMMENT '长视频-ott正价驱动开通人数',
    `long_video_total_value_yuan` double COMMENT '长视频-会员收入，当日总内容价值认定（元）',
    `jingxuanye_in_imp_pv_app` bigint COMMENT '精选页-曝光次数-app端',
    `jingxuanye_in_click_pv_app` bigint COMMENT '精选页-点击次数-app端',
    `jingxuanye_in_click_ctr_app` double COMMENT '精选页-点击ctr-app端',
    `jingxuanye_in_vstart_vv_app` bigint COMMENT '精选页-播放vv-app端',
    `jingxuanye_qiangmudiqu_in_imp_pv_app` bigint COMMENT '精选页-强目的区（焦点图+重磅）-曝光次数-app端',
    `jingxuanye_qiangmudiqu_in_click_pv_app` bigint COMMENT '精选页-强目的区（焦点图+重磅）-点击次数-app端',
    `jingxuanye_qiangmudiqu_in_click_ctr_app` double COMMENT '精选页-强目的区（焦点图+重磅）-点击ctr-app端',
    `jingxuanye_qiangmudiqu_in_vstart_vv_app` bigint COMMENT '精选页-强目的区（焦点图+重磅）-播放vv-app端',
    `jingxuanye_xukanqu_in_imp_pv_app` bigint COMMENT '精选页-续看区（追剧模块）-曝光次数-app端',
    `jingxuanye_xukanqu_in_click_pv_app` bigint COMMENT '精选页-续看区（追剧模块）-点击次数-app端',
    `jingxuanye_xukanqu_in_click_ctr_app` double COMMENT '精选页-续看区（追剧模块）-点击ctr-app端',
    `jingxuanye_xukanqu_in_vstart_vv_app` bigint COMMENT '精选页-续看区（追剧模块）-播放vv-app端',
    `jingxuanye_wumudiqu_in_imp_pv_app` bigint COMMENT '精选页-无目的区（视频卡片流）-曝光次数-app端',
    `jingxuanye_wumudiqu_in_click_pv_app` bigint COMMENT '精选页-无目的区（视频卡片流）-点击次数-app端',
    `jingxuanye_wumudiqu_in_click_ctr_app` double COMMENT '精选页-无目的区（视频卡片流）-点击ctr-app端',
    `jingxuanye_wumudiqu_in_vstart_vv_app` bigint COMMENT '精选页-无目的区（视频卡片流）-播放vv-app端',
    `zongheye_in_imp_pv_app` bigint COMMENT '综合页-曝光次数-app端',
    `zongheye_in_click_pv_app` bigint COMMENT '综合页-点击次数-app端',
    `zongheye_in_click_ctr_app` double COMMENT '综艺页-点击ctr-app端',
    `zongheye_in_vstart_vv_app` bigint COMMENT '综艺页-播放vv-app端',
    `in_imp_pv_app` bigint COMMENT '曝光次数-app端',
    `in_imp_uv_app` bigint COMMENT '曝光人数-app端',
    `in_click_pv_app` bigint COMMENT '点击次数-app端',
    `in_click_uv_app` bigint COMMENT '点击人数-app端'
    PRIMARY KEY (`imp_date`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci COMMENT = '综艺大盘指标日表'
```

【口径规则】

1. 数据为增量日表，查询时必须指定 `imp_date`
2. 本表为**大盘表**，无项目维度（无 `pid`/`cid` 字段），直接按日期查询即可，**无需前置查询项目ID**
3. **三端汇总**：如需三端合计，将 `_app`、`_pc`、`_ott` 三个字段相加；展示时去掉端后缀，如「播放VV」= `in_vstart_vv_app + in_vstart_vv_pc + in_vstart_vv_ott`
4. **指标名展示规则**：展示指标名时，去掉字段 COMMENT 末尾的 `-app端`/`-pc端`/`-ott端` 后缀；三端合计时直接用不带端后缀的名称，如「播放VV」「播放UV」「播放时长」
5. **时间范围**：查询区间时，`imp_date BETWEEN :start_date AND :end_date`；查询最新单日时，`imp_date = (SELECT MAX(imp_date) FROM ads_zongyi_dapan_metrics_di)`
6. **时长单位**：`in_pdtm_s_*` 字段单位为秒，对外展示时按需换算为分钟（÷60）或小时（÷3600）
7. **正价开通人数**：用户未指定类型时，正价开通人数 = `long_video_hvip_normal_ucnt`（hvip正价驱动开通人数）+ `long_video_svip_normal_ucnt`（ott正价驱动开通人数）
