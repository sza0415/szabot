```sql
CREATE TABLE `dws_szbot_dianshiju_fee_t_four_project_budget_hf` (
    `imp_hour` bigint NOT NULL COMMENT '时间分区 格式YYYYMMDDHH',
    `studio`                      varchar(255)  COMMENT '工作室',
    `pid`                         bigint        NOT NULL COMMENT '项目ID',
    `project_name`                varchar(255)  NOT NULL COMMENT '项目名称',
    `project_source`              varchar(255)  COMMENT '数据来源（1=人工上传excel，2=制片管理）',
    `topic_type`                  varchar(255)  COMMENT '题材类型',
    `topic_track`                 varchar(255)  COMMENT '题材赛道',
    `start_year`                  varchar(255)  COMMENT '开机年份',
    `approval_rating`             varchar(255)  COMMENT '立项评级',
    `episode_count`               varchar(255)  COMMENT '集数',
    `shooting_location`           varchar(255)  COMMENT '拍摄地',
    `shooting_days`               int           COMMENT '拍摄天数',
    `total_cost`                  decimal(18,2) COMMENT '项目总成本金额（单位：元）',
    `online_cost_amount`          decimal(18,2) COMMENT '线上总成本（单位：元）',
    `online_cost_ratio`           decimal(10,2) COMMENT '线上成本占（%）',
    `ip_info`                     decimal(18,2) COMMENT 'IP（单位：元）',
    `script_info`                 decimal(18,2) COMMENT '剧本（单位：元）',
    `actor_top2_cost`             decimal(18,2) COMMENT '演员前二（单位：元）',
    `actor_others_cost`           decimal(18,2) COMMENT '演员其他（单位：元）',
    `main_crew_cost`              decimal(18,2) COMMENT '主创费用（单位：元）',
    `promotion_cost`              decimal(18,2) COMMENT '宣发费用（单位：元）',
    `production_fee`              decimal(18,2) COMMENT '项目承制费用（单位：元）',
    `offline_cost_amount`         decimal(18,2) COMMENT '线下总成本（单位：元）',
    `offline_cost_ratio`          decimal(10,2) COMMENT '线下成本占比（%）',
    `other_staff_cost`            decimal(18,2) COMMENT '其他职员费用（单位：元）',
    `production_equipment_cost`   decimal(18,2) COMMENT '制片组器材费用（单位：元）',
    `art_cost`                    decimal(18,2) COMMENT '美术费（单位：元）',
    `costume_cost`                decimal(18,2) COMMENT '服装费（单位：元）',
    `makeup_cost`                 decimal(18,2) COMMENT '化妆费（单位：元）',
    `set_design_cost`             decimal(18,2) COMMENT '置景费（单位：元）',
    `props_cost`                  decimal(18,2) COMMENT '道具费（单位：元）',
    `special_effects_cost`        decimal(18,2) COMMENT '特殊效果费（单位：元）',
    `photography_equipment_cost`  decimal(18,2) COMMENT '摄影器材（单位：元）',
    `lighting_equipment_cost`     decimal(18,2) COMMENT '灯光器材费用（单位：元）',
    `recording_equipment_cost`    decimal(18,2) COMMENT '录音器材费用（单位：元）',
    `editing_equipment_cost`      decimal(18,2) COMMENT '剪辑器材费用（单位：元）',
    `action_equipment_cost`       decimal(18,2) COMMENT '动作器材费用（单位：元）',
    `location_rent_cost`          decimal(18,2) COMMENT '场租费用（单位：元）',
    `accommodation_traffic_cost`  decimal(18,2) COMMENT '食宿行办费用（单位：元）',
    `post_production_cost`        decimal(18,2) COMMENT '后期费用（单位：元）',
    `tax_cost`                    decimal(18,2) COMMENT '税金（单位：元）',
    `contingency_cost`            decimal(18,2) COMMENT '不可预见费用（单位：元）',
    UNIQUE KEY `uk_project_name` (`project_name`),
    KEY `idx_pid` (`pid`)
) COMMENT='项目四级预算表';
```

【口径规则】

1. **过滤条件**（按需拼入 WHERE，未指定的条件不加，其余维度同理）：
   - 最新分区 → `imp_hour = (SELECT MAX(imp_hour) FROM dws_szbot_dianshiju_fee_t_four_project_budget_hf)`（必选，禁止写死）
   - 有项目列表 → `AND pid IN ({项目ID列表})`；未指定项目时省略
   - 指定时间范围 → 用 `start_year`（开机年份），如 `AND start_year BETWEEN {开始年份} AND {结束年份}`；近 N 年用 `AND start_year > YEAR(NOW()) - N`
   - 其他过滤条件一律使用精确匹配（`IN`），禁止使用模糊匹配（`LIKE`）
2. **数据来源**：`project_source` 取值含义：1=人工上传excel，2=制片管理。
3. **成本构成关系**：
   - `total_cost`（项目总成本）= `online_cost_amount`（线上总成本）+ `offline_cost_amount`（线下总成本）
   - **线上总成本**（`online_cost_amount`）由以下明细构成：IP（`ip_info`）、剧本（`script_info`）、演员前二（`actor_top2_cost`）、演员其他（`actor_others_cost`）、主创费用（`main_crew_cost`）、宣发费用（`promotion_cost`）、承制费用（`production_fee`）
   - **线下总成本**（`offline_cost_amount`）由以下明细构成：其他职员费用（`other_staff_cost`）、制片组器材费用（`production_equipment_cost`）、美术费（`art_cost`）、服装费（`costume_cost`）、化妆费（`makeup_cost`）、置景费（`set_design_cost`）、道具费（`props_cost`）、特殊效果费（`special_effects_cost`）、摄影器材（`photography_equipment_cost`）、灯光器材费用（`lighting_equipment_cost`）、录音器材费用（`recording_equipment_cost`）、剪辑器材费用（`editing_equipment_cost`）、动作器材费用（`action_equipment_cost`）、场租费用（`location_rent_cost`）、食宿行办费用（`accommodation_traffic_cost`）、后期费用（`post_production_cost`）、税金（`tax_cost`）、不可预见费用（`contingency_cost`）
4. **数据链接**：
   - 项目粒度 → [查看预算](https://zp.szabot.internal/budget-evaluate/four-budget?project_name={项目名称1},{项目名称2},...)，`project_name` 为查询涉及的项目名称，多个用英文逗号分隔；未指定具体项目时不带参数
   - 汇总/概览 → [查看预算](https://zp.szabot.internal/budget-evaluate/overview)

## 常用 SQL 模板

### 模板 1：查询项目四级预算（可指定项目，也可不指定）

> 四级预算查询结果分三块独立展示，依次执行以下三条 SQL。展示时**严格按照 SQL 中的字段顺序**呈现，禁止调整字段的顺序。若查询结果包含多条记录，以 `pid` 作为主键区分，所有记录放在同一表格中，**每个项目占一列**分别展示（字段名作为行标题，项目数据按列排列）。
>
> 回答结束后附上链接：[查看预算](https://zp.szabot.internal/budget-evaluate/four-budget?project_name={项目名称1},{项目名称2},...)，其中 `project_name` 参数值为查询涉及的项目名称，多个项目用英文逗号分隔。若未指定具体项目则不带参数：[查看预算](https://zp.szabot.internal/budget-evaluate/four-budget)

**第一块：四级预算-概览**

```sql
SELECT
    pid                                          AS "项目ID",
    project_name                                    AS "项目名称",
    CASE project_source
        WHEN "1" THEN "人工上传excel"
        WHEN "2" THEN "制片管理"
        ELSE project_source
    END                                             AS "数据来源",
    topic_type                                      AS "题材类型",
    topic_track                                     AS "题材赛道",
    start_year                                      AS "开机年份",
    approval_rating                                 AS "立项评级",
    episode_count                                   AS "集数",
    shooting_location                               AS "拍摄地",
    shooting_days                                   AS "拍摄天数",
    ROUND(total_cost / 10000, 2)                    AS "项目总成本（万元）",
    ROUND(total_cost / episode_count / 10000, 2)    AS "集均成本（万元）",
    ROUND(online_cost_amount / 10000, 2)            AS "线上总成本（万元）",
    CONCAT(online_cost_ratio, "%")                  AS "线上成本占比",
    ROUND(offline_cost_amount / 10000, 2)           AS "线下总成本（万元）",
    CONCAT(offline_cost_ratio, "%")                 AS "线下成本占比"
FROM dws_szbot_dianshiju_fee_t_four_project_budget_hf
WHERE {过滤条件}
ORDER BY pid ASC
```

**第二块：四级预算-线上成本及其明细**

```sql
SELECT
    pid                                          AS "项目ID",
    project_name                                    AS "项目名称",
    ROUND(online_cost_amount / 10000, 2)            AS "线上总成本（万元）",
    ROUND(ip_info / 10000, 2)                       AS "IP（万元）",
    ROUND(script_info / 10000, 2)                   AS "剧本（万元）",
    ROUND(actor_top2_cost / 10000, 2)               AS "演员前二（万元）",
    ROUND(actor_others_cost / 10000, 2)             AS "演员其他（万元）",
    ROUND(main_crew_cost / 10000, 2)                AS "主创费用（万元）",
    ROUND(promotion_cost / 10000, 2)                AS "宣发费用（万元）",
    ROUND(production_fee / 10000, 2)                AS "承制费用（万元）"
FROM dws_szbot_dianshiju_fee_t_four_project_budget_hf
WHERE {过滤条件}
ORDER BY pid ASC
```

**第三块：四级预算-线下成本及其明细**

```sql
SELECT
    pid                                          AS "项目ID",
    project_name                                    AS "项目名称",
    ROUND(offline_cost_amount / 10000, 2)           AS "线下总成本（万元）",
    ROUND(other_staff_cost / 10000, 2)              AS "其他职员费用（万元）",
    ROUND(production_equipment_cost / 10000, 2)     AS "制片组器材费用（万元）",
    ROUND(art_cost / 10000, 2)                      AS "美术费（万元）",
    ROUND(costume_cost / 10000, 2)                  AS "服装费（万元）",
    ROUND(makeup_cost / 10000, 2)                   AS "化妆费（万元）",
    ROUND(set_design_cost / 10000, 2)               AS "置景费（万元）",
    ROUND(props_cost / 10000, 2)                    AS "道具费（万元）",
    ROUND(special_effects_cost / 10000, 2)          AS "特殊效果费（万元）",
    ROUND(photography_equipment_cost / 10000, 2)    AS "摄影器材（万元）",
    ROUND(lighting_equipment_cost / 10000, 2)       AS "灯光器材费用（万元）",
    ROUND(recording_equipment_cost / 10000, 2)      AS "录音器材费用（万元）",
    ROUND(editing_equipment_cost / 10000, 2)        AS "剪辑器材费用（万元）",
    ROUND(action_equipment_cost / 10000, 2)         AS "动作器材费用（万元）",
    ROUND(location_rent_cost / 10000, 2)            AS "场租费用（万元）",
    ROUND(accommodation_traffic_cost / 10000, 2)    AS "食宿行办费用（万元）",
    ROUND(post_production_cost / 10000, 2)          AS "后期费用（万元）",
    ROUND(tax_cost / 10000, 2)                      AS "税金（万元）",
    ROUND(contingency_cost / 10000, 2)              AS "不可预见费用（万元）"
FROM dws_szbot_dianshiju_fee_t_four_project_budget_hf
WHERE {过滤条件}
ORDER BY pid ASC
```

### 模板 2：按年份统计预算趋势/预算浮动（近 N 年）

> 回答结束后附上链接：[查看预算](https://zp.szabot.internal/budget-evaluate/overview)

> **⚠️ 以下 SQL 仅为示例**，实际执行时需根据用户问题动态调整：
> - **指标字段**：默认使用 `total_cost`（总成本）计算集均成本；用户指定了具体预算指标（如「演员费用」「宣发费用」「线上成本」「线下成本」等），则替换为对应字段的聚合（`SUM`），外层 SELECT 同步输出。
> - **时间字段**：使用 `start_year`（开机年份）。
>

```sql
WITH yearly_summary AS (
    SELECT
        start_year AS event_year,
        AVG({指标字段} / episode_count) / 10000            AS cost_per_ep
    FROM dws_szbot_dianshiju_fee_t_four_project_budget_hf
    WHERE {过滤条件}
    GROUP BY 1
)
SELECT
    event_year                                                          AS "年份",
    ROUND(cost_per_ep, 2)                                               AS "{指标名}集均成本（万元）",
    ROUND(
        (cost_per_ep - LAG(cost_per_ep) OVER (ORDER BY event_year ASC))
        / LAG(cost_per_ep) OVER (ORDER BY event_year ASC) * 100, 2
    )                                                                   AS "{指标名}集均成本同比涨跌幅(%)"
FROM yearly_summary
ORDER BY 1 ASC
```

### 模板 3：按维度汇总集均成本统计

> 回答结束后附上链接：[查看预算](https://zp.szabot.internal/budget-evaluate/overview)

> **⚠️ 以下 SQL 仅为示例**，实际执行时需根据用户问题动态调整：
> - **分组维度**：根据用户问题选择分组字段，如赛道、评级、工作室等，可多维度交叉 GROUP BY。
> - **指标字段**：默认使用总成本计算集均成本；用户指定具体指标时按需替换，可选：线上成本、线下成本及其各明细字段。
> - **筛选条件**：按用户指定的赛道、工作室、时间范围等加 WHERE 过滤。

```sql
SELECT
    {分组维度}                                           AS "{维度名}",
    ROUND(SUM({指标字段}) / 10000, 2)                    AS "{指标名}总成本（万元）",
    ROUND(AVG({指标字段} / episode_count) / 10000, 2)      AS "{指标名}集均成本（万元）"
FROM dws_szbot_dianshiju_fee_t_four_project_budget_hf
WHERE {过滤条件}
GROUP BY 1
ORDER BY 2 DESC
```

