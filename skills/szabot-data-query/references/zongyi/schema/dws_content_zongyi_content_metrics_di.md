```
CREATE TABLE `dws_content_zongyi_content_metrics_di` (
    `imp_date` bigint NOT NULL COMMENT '日期 格式YYYYMMDD',
    `pid` bigint COMMENT '影库项目ID',
    `content_title` varchar(256) COMMENT '项目名称',
    `content_id` varchar(20) NOT NULL COMMENT '内容ID',
     `in_vstart_vv_all_app` bigint COMMENT '播放VV-直播+点播-app端',
    `in_vstart_vv_all_pc` bigint COMMENT '播放VV-直播+点播-pc端',
    `in_vstart_vv_all_ott` bigint COMMENT '播放VV-直播+点播-ott端',
    `in_vstart_uv_all_app` bigint COMMENT '播放UV-直播+点播-app端',
    `in_vstart_uv_all_pc` bigint COMMENT '播放UV-直播+点播-pc端',
    `in_vstart_uv_all_ott` bigint COMMENT '播放UV-直播+点播-ott端',
    `in_pdtm_ms_all_app` double COMMENT '播放时长(毫秒)-直播+点播-app端',
    `in_pdtm_ms_all_pc` double COMMENT '播放时长(毫秒)-直播+点播-pc端',
    `in_pdtm_ms_all_ott` double COMMENT '播放时长(毫秒)-直播+点播-ott端',
    `hot_value` double COMMENT '热度',
    `in_first_uv_app` bigint COMMENT '首播UV-app端',
     `in_vstart_vv_app` bigint COMMENT '播放VV-app端',
    `in_vstart_vv_pc` bigint COMMENT '播放VV-pc端',
    `in_vstart_vv_ott` bigint COMMENT '播放VV-ott端',
    `in_vstart_uv_app` bigint COMMENT '播放UV-app端',
    `in_vstart_uv_pc` bigint COMMENT '播放UV-pc端',
    `in_vstart_uv_ott` bigint COMMENT '播放UV-ott端',
    `in_pdtm_ms_app` double COMMENT '播放时长(毫秒)-app端',
    `in_pdtm_ms_pc` double COMMENT '播放时长(毫秒)-pc端',
    `in_pdtm_ms_ott` double COMMENT '播放时长(毫秒)-ott端'
    PRIMARY KEY (`imp_date`, `content_id`)
) COMMENT = '综艺项目指标日增量表'
```

【口径规则】

1. 数据为增量日表，查询时必须指定 `imp_date`
2. 🚨 **禁止跨天累计**：所有指标仅代表当日值，禁止跨天 `SUM`；
3. 本表为**项目粒度表**，每行对应一个综艺项目在某天的增量指标；查询单个项目时，需在 WHERE 中指定 `pid`（项目ID，优先使用）或 `content_title`（项目名称，精确匹配，禁止 LIKE）
4. **三端汇总**：如需三端合计，将 `_app`、`_pc`、`_ott` 三个字段相加；展示时去掉端后缀，如「播放VV」= `in_vstart_vv_all_app + in_vstart_vv_all_pc + in_vstart_vv_all_ott`
5. **直播+点播 vs 纯点播**：
   - `in_vstart_vv_*` / `in_vstart_uv_*` / `in_pdtm_ms_*`：纯点播数据
   - `in_vstart_vv_all_*` / `in_vstart_uv_all_*` / `in_pdtm_ms_all_*`：直播+点播合计数据
   - 用户未特别说明时，默认使用直播+点播字段（含 `_all_` 的字段）
6. **指标名展示规则**：展示指标名时，去掉字段 COMMENT 末尾的 `-app端`/`-pc端`/`-ott端` 后缀；三端合计时直接用不带端后缀的名称，如「播放VV」「播放UV」「播放时长」
7. **时间范围**：查询区间时，`imp_date BETWEEN :start_date AND :end_date`；查询最新单日时，`imp_date = (SELECT MAX(imp_date) FROM dws_content_zongyi_content_metrics_di)`
8. 🚨 **时长单位（极易出错）**：`in_pdtm_ms_*` 字段单位为**毫秒（ms）**，**不是秒**；对外展示时必须在 SQL 中完成换算：万小时（÷3600000÷10000）、小时（÷3600000）、分钟（÷60000）、秒（÷1000）；**严禁**将毫秒值当作秒直接展示或换算
9. **热度**：`hot_value` 为该内容当日热度值，可直接展示或排序
