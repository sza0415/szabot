```
USE szdw_ads;
CREATE TABLE `ads_szbot_dianshiju_project_consume_metrics_di` (
    `imp_date` bigint NOT NULL COMMENT '日期 格式yyyymmdd',
    `pid` bigint NOT NULL COMMENT '项目id',
    `cid` varchar(32) COMMENT '专辑',
    `hot_value` double COMMENT '当日-热度值',
    `in_vstart_vv_app_pc_ott` bigint COMMENT '播放VV-三端',
    `in_vstart_uv_app_pc_ott` bigint COMMENT '播放UV-三端',
    `big_in_vstart_uv_app` bigint COMMENT '品类-播放uv-app端',
    `big_in_imp_uv_app` bigint COMMENT '品类-曝光uv-app端',
    `big_in_imp_vv_app` bigint COMMENT '品类-曝光vv-app端',
    `in_imp_uv_app` bigint COMMENT '曝光uv-app端',
    `in_imp_vv_app` bigint COMMENT '曝光vv-app端',
    `in_click_ctr_app` double COMMENT '点击ctr-app端',
    `in_click_utr_app` double COMMENT '点击utr-app端',
    `in_vstart_biz_id_nu_app_pc_ott` bigint COMMENT '平台拉新-三端',
    `in_vstart_type_id_nu_app_pc_ott` bigint COMMENT '品类拉新-三端',
    `search_uv_app` bigint COMMENT '搜索uv-app端',
    `bullet_interactive_cnt_app` bigint COMMENT '弹幕量-app端',
    `zp_play_vv_app_pc_ott` bigint COMMENT '正片VV-三端',
    `positive_vid_num` bigint COMMENT '集数',
    PRIMARY KEY (`imp_date`, `pid`)
) COMMENT = '电视剧每日消费指标'
```
【口径规则】

1. 数据为增量表，查询时，需要指定imp_date和pid
2. **禁止对指标字段使用 SUM/聚合求和**，每行数据已是当日完整指标，直接取值即可
3. **集均指标计算**：求集均时，用指标除以 `positive_vid_num`（集数）
   - 集均VV = `in_vstart_vv_app_pc_ott` / `positive_vid_num`
   - 集均UV = `in_vstart_uv_app_pc_ott` / `positive_vid_num`
   - 集均正片VV = `zp_play_vv_app_pc_ott` / `positive_vid_num`
   
查询模板

```sql
-- db_name: szdw_ads
SELECT imp_date, pid,
    hot_value,
    in_vstart_vv_app_pc_ott,
    in_vstart_uv_app_pc_ott,
    zp_play_vv_app_pc_ott,
    in_imp_uv_app,
    in_imp_vv_app,
    in_click_ctr_app,
    in_click_utr_app,
    in_vstart_biz_id_nu_app_pc_ott,
    in_vstart_type_id_nu_app_pc_ott,
    search_uv_app,
    bullet_interactive_cnt_app,
    positive_vid_num
FROM ads_szbot_dianshiju_project_consume_metrics_di
WHERE pid = {项目ID}
AND imp_date >= {开始日期}
AND imp_date <= (
    SELECT MAX(imp_date) FROM szdw_dim.chuku_progress
    WHERE table_name = 'ads_szbot_dianshiju_project_consume_metrics_di'
)
ORDER BY imp_date DESC;
```
