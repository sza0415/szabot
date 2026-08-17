```sql
CREATE TABLE `dws_szbot_dianshiju_fee_t_six_project_budget_hf` (
    `imp_hour` bigint NOT NULL COMMENT '时间分区 格式YYYYMMDDHH',
    `studio`                        varchar(255)  COMMENT '工作室',
    `pid`                           bigint        NOT NULL COMMENT '项目ID',
    `project_name`                  varchar(255)  NOT NULL COMMENT '项目名称',
    `project_source`                varchar(255)  COMMENT '数据来源（1=人工上传excel，2=制片管理）',
    `topic_type`                    varchar(255)  COMMENT '题材类型',
    `topic_track`                   varchar(255)  COMMENT '题材赛道',
    `start_year`                    varchar(255)  COMMENT '开机年份',
    `project_rating`                varchar(50)   COMMENT '立项评级',
    `episode_count`                 int           COMMENT '集数',
    `shooting_location`             varchar(255)  COMMENT '拍摄地',
    `shooting_days`                 int           COMMENT '拍摄天数',
    `total_cost`                    decimal(18,2) COMMENT '项目总成本（单位：元）',
    `amount`                        decimal(18,2) COMMENT '线上成本（单位：元）',
    `proportion`                    decimal(5,2)  COMMENT '线上成本占比（百分比）',
    `ip_original`                   decimal(18, 2) COMMENT 'IP费用（单位：元）',
    `script_cost`                   decimal(18,2) COMMENT '剧本费用（单位：元）',
    `top2_actors_cost`              decimal(18,2) COMMENT '演员前二费用（单位：元）',
    `other_actors_cost`             decimal(18,2) COMMENT '其他演员费用（单位：元）',
    `key_crew_cost`                 decimal(18,2) COMMENT '主创费用（单位：元）',
    `promotion_cost`                decimal(18,2) COMMENT '宣发费用（单位：元）',
    `production_fee`                decimal(18,2) COMMENT '承制费（单位：元）',
    `off_line_amount`               decimal(18,2) COMMENT '线下成本（单位：元）',
    `off_line_proportion`           decimal(18,2) COMMENT '线下成本占比（百分比）',
    `production_team_cost`          decimal(18,2) COMMENT '制片组费用（单位：元）',
    `director_team_cost`            decimal(18,2) COMMENT '导演组费用（单位：元）',
    `art_team_cost`                 decimal(18,2) COMMENT '美术组费用（单位：元）',
    `costume_team_cost`             decimal(18,2) COMMENT '服装组费用（单位：元）',
    `makeup_team_cost`              decimal(18,2) COMMENT '化妆组费用（单位：元）',
    `styling_team_cost`             decimal(18,2) COMMENT '造型组费用（单位：元）',
    `set_design_team_cost`          decimal(18,2) COMMENT '置景组费用（单位：元）',
    `props_team_cost`               decimal(18,2) COMMENT '道具组费用（单位：元）',
    `cinematography_team_cost`      decimal(18,2) COMMENT '摄影组费用（单位：元）',
    `lighting_team_cost`            decimal(18,2) COMMENT '灯光组费用（单位：元）',
    `sound_team_cost`               decimal(18,2) COMMENT '录音组费用（单位：元）',
    `editing_team_cost`             decimal(18,2) COMMENT '剪辑组费用（单位：元）',
    `vehicle_team_cost`             decimal(18,2) COMMENT '车辆组费用（单位：元）',
    `action_team_cost`              decimal(18,2) COMMENT '动作组费用（单位：元）',
    `production_equipment_cost`     decimal(18,2) COMMENT '制片组器材费用（单位：元）',
    `art_fee`                       decimal(18,2) COMMENT '美术费（单位：元）',
    `costume_fee`                   decimal(18,2) COMMENT '服装费（单位：元）',
    `makeup_fee`                    decimal(18,2) COMMENT '化妆费（单位：元）',
    `set_design_fee`                decimal(18,2) COMMENT '置景费（单位：元）',
    `props_fee`                     decimal(18,2) COMMENT '道具费（单位：元）',
    `special_effects_fee`           decimal(18,2) COMMENT '特殊效果费用（单位：元）',
    `cinematography_equipment_cost` decimal(18,2) COMMENT '摄影器材费用（单位：元）',
    `lighting_equipment_cost`       decimal(18,2) COMMENT '灯光器材费用（单位：元）',
    `sound_equipment_cost`          decimal(18,2) COMMENT '录音器材费用（单位：元）',
    `editing_equipment_cost`        decimal(18,2) COMMENT '剪辑器材费用（单位：元）',
    `action_equipment_cost`         decimal(18,2) COMMENT '动作器材费用（单位：元）',
    `venue_rent`                    decimal(18,2) COMMENT '场租费用（单位：元）',
    `accommodation_expenses`        decimal(18,2) COMMENT '食宿行办费用（单位：元）',
    `post_production_fee`           decimal(18,2) COMMENT '后期费用（单位：元）',
    `tax`                           decimal(18,2) COMMENT '税金（单位：元）',
    `contingency_fee`               decimal(18,2) COMMENT '不可预见费用（单位：元）',
    KEY `idx_pid` (`pid`)
) COMMENT='项目六级预算表';
```

【口径规则】

1. **过滤条件**（按需拼入 WHERE，未指定的条件不加，其余维度同理）：
   - 最新分区 → `imp_hour = (SELECT MAX(imp_hour) FROM dws_szbot_dianshiju_fee_t_six_project_budget_hf)`（必选，禁止写死）
   - 有项目列表 → `AND pid IN ({项目ID列表})`；未指定项目时省略
   - 指定时间范围 → 用 `start_year`（开机年份），如 `AND start_year BETWEEN {开始年份} AND {结束年份}`；近 N 年用 `AND start_year > YEAR(NOW()) - N`
   - 其他过滤条件一律使用精确匹配（`IN`），禁止使用模糊匹配（`LIKE`）
2. **数据来源**：`project_source` 取值含义：1=人工上传excel，2=制片管理。
3. **比例字段**：`proportion`、`off_line_proportion` 单位为**百分比（%）**，展示时使用 `CONCAT(字段, "%")` 拼接百分号。
4. **成本构成关系**：
   - `total_cost`（项目总成本）= `amount`（线上成本）+ `off_line_amount`（线下成本）
   - **线上成本**（`amount`）由以下明细构成：IP费用（`ip_original`）、剧本费用（`script_cost`）、演员前二费用（`top2_actors_cost`）、其他演员费用（`other_actors_cost`）、主创费用（`key_crew_cost`）、宣发费用（`promotion_cost`）、承制费（`production_fee`）
   - **线下成本**（`off_line_amount`）由以下明细构成：制片组费用（`production_team_cost`）、导演组费用（`director_team_cost`）、美术组费用（`art_team_cost`）、服装组费用（`costume_team_cost`）、化妆组费用（`makeup_team_cost`）、造型组费用（`styling_team_cost`）、置景组费用（`set_design_team_cost`）、道具组费用（`props_team_cost`）、摄影组费用（`cinematography_team_cost`）、灯光组费用（`lighting_team_cost`）、录音组费用（`sound_team_cost`）、剪辑组费用（`editing_team_cost`）、车辆组费用（`vehicle_team_cost`）、动作组费用（`action_team_cost`）、制片组器材费用（`production_equipment_cost`）、美术费（`art_fee`）、服装费（`costume_fee`）、化妆费（`makeup_fee`）、置景费（`set_design_fee`）、道具费（`props_fee`）、特殊效果费用（`special_effects_fee`）、摄影器材费用（`cinematography_equipment_cost`）、灯光器材费用（`lighting_equipment_cost`）、录音器材费用（`sound_equipment_cost`）、剪辑器材费用（`editing_equipment_cost`）、动作器材费用（`action_equipment_cost`）、场租费用（`venue_rent`）、食宿行办费用（`accommodation_expenses`）、后期费用（`post_production_fee`）、税金（`tax`）、不可预见费用（`contingency_fee`）
5. **数据链接**：
   - 项目粒度 → [查看预算](https://zp.szabot.internal/budget-evaluate/six-budget?project_name={项目名称1},{项目名称2},...)，`project_name` 为查询涉及的项目名称，多个用英文逗号分隔；未指定具体项目时不带参数
   - 汇总/概览 → [查看预算](https://zp.szabot.internal/budget-evaluate/overview)


## 常用 SQL 模板

### 模板 1：查询项目六级预算（可指定项目，也可不指定）

> 仅查询六级独有字段（各组别人员费用）时使用本表，其它字段优先走四级预算表。结果分三块依次执行，按 SQL 字段顺序展示；多记录时以 `pid` 区分，每个项目占一列，字段名作为行标题。
>
> 回答结束后附上链接：[查看预算](https://zp.szabot.internal/budget-evaluate/six-budget?project_name={项目名称1},{项目名称2},...)，其中 `project_name` 参数值为查询涉及的项目名称，多个项目用英文逗号分隔。若未指定具体项目则不带参数：[查看预算](https://zp.szabot.internal/budget-evaluate/six-budget)

**第一块：六级预算-概览**

```sql
SELECT
    studio                                              AS "工作室",
    pid                                              AS "项目ID",
    project_name                                        AS "项目名称",
    CASE project_source
        WHEN "1" THEN "人工上传excel"
        WHEN "2" THEN "制片管理"
        ELSE project_source
    END                                                 AS "数据来源",
    topic_type                                          AS "题材类型",
    topic_track                                         AS "题材赛道",
    start_year                                          AS "开机年份",
    project_rating                                      AS "立项评级",
    episode_count                                       AS "集数",
    shooting_location                                   AS "拍摄地",
    shooting_days                                       AS "拍摄天数",
    ROUND(total_cost / 10000, 2)                        AS "项目总成本（万元）",
    ROUND(total_cost / episode_count / 10000, 2)        AS "集均成本（万元）",
    ROUND(amount / 10000, 2)                            AS "线上成本（万元）",
    CONCAT(proportion, "%")                             AS "线上成本占比",
    ROUND(off_line_amount / 10000, 2)                   AS "线下成本（万元）",
    CONCAT(off_line_proportion, "%")                    AS "线下成本占比"
FROM dws_szbot_dianshiju_fee_t_six_project_budget_hf
WHERE {过滤条件}
ORDER BY pid ASC
```

**第二块：六级预算-线上成本及其明细**

```sql
SELECT
    pid                                              AS "项目ID",
    project_name                                        AS "项目名称",
    ROUND(amount / 10000, 2)                            AS "线上成本（万元）",
    ROUND(ip_original / 10000, 2)                       AS "IP费用（万元）",
    ROUND(script_cost / 10000, 2)                       AS "剧本费用（万元）",
    ROUND(top2_actors_cost / 10000, 2)                  AS "演员前二费用（万元）",
    ROUND(other_actors_cost / 10000, 2)                 AS "其他演员费用（万元）",
    ROUND(key_crew_cost / 10000, 2)                     AS "主创费用（万元）",
    ROUND(promotion_cost / 10000, 2)                    AS "宣发费用（万元）",
    ROUND(production_fee / 10000, 2)                    AS "承制费（万元）"
FROM dws_szbot_dianshiju_fee_t_six_project_budget_hf
WHERE {过滤条件}
ORDER BY pid ASC
```

**第三块：六级预算-线下成本及其明细**

```sql
SELECT
    pid                                              AS "项目ID",
    project_name                                        AS "项目名称",
    ROUND(off_line_amount / 10000, 2)                   AS "线下成本（万元）",
    ROUND(production_team_cost / 10000, 2)              AS "制片组费用（万元）",
    ROUND(director_team_cost / 10000, 2)                AS "导演组费用（万元）",
    ROUND(art_team_cost / 10000, 2)                     AS "美术组费用（万元）",
    ROUND(costume_team_cost / 10000, 2)                 AS "服装组费用（万元）",
    ROUND(makeup_team_cost / 10000, 2)                  AS "化妆组费用（万元）",
    ROUND(styling_team_cost / 10000, 2)                 AS "造型组费用（万元）",
    ROUND(set_design_team_cost / 10000, 2)              AS "置景组费用（万元）",
    ROUND(props_team_cost / 10000, 2)                   AS "道具组费用（万元）",
    ROUND(cinematography_team_cost / 10000, 2)          AS "摄影组费用（万元）",
    ROUND(lighting_team_cost / 10000, 2)                AS "灯光组费用（万元）",
    ROUND(sound_team_cost / 10000, 2)                   AS "录音组费用（万元）",
    ROUND(editing_team_cost / 10000, 2)                 AS "剪辑组费用（万元）",
    ROUND(vehicle_team_cost / 10000, 2)                 AS "车辆组费用（万元）",
    ROUND(action_team_cost / 10000, 2)                  AS "动作组费用（万元）",
    ROUND(production_equipment_cost / 10000, 2)         AS "制片组器材费用（万元）",
    ROUND(art_fee / 10000, 2)                           AS "美术费（万元）",
    ROUND(costume_fee / 10000, 2)                       AS "服装费（万元）",
    ROUND(makeup_fee / 10000, 2)                        AS "化妆费（万元）",
    ROUND(set_design_fee / 10000, 2)                    AS "置景费（万元）",
    ROUND(props_fee / 10000, 2)                         AS "道具费（万元）",
    ROUND(special_effects_fee / 10000, 2)               AS "特殊效果费用（万元）",
    ROUND(cinematography_equipment_cost / 10000, 2)     AS "摄影器材费用（万元）",
    ROUND(lighting_equipment_cost / 10000, 2)           AS "灯光器材费用（万元）",
    ROUND(sound_equipment_cost / 10000, 2)              AS "录音器材费用（万元）",
    ROUND(editing_equipment_cost / 10000, 2)            AS "剪辑器材费用（万元）",
    ROUND(action_equipment_cost / 10000, 2)             AS "动作器材费用（万元）",
    ROUND(venue_rent / 10000, 2)                        AS "场租费用（万元）",
    ROUND(accommodation_expenses / 10000, 2)            AS "食宿行办费用（万元）",
    ROUND(post_production_fee / 10000, 2)               AS "后期费用（万元）",
    ROUND(tax / 10000, 2)                               AS "税金（万元）",
    ROUND(contingency_fee / 10000, 2)                   AS "不可预见费用（万元）"
FROM dws_szbot_dianshiju_fee_t_six_project_budget_hf
WHERE {过滤条件}
ORDER BY pid ASC
```

### 模板 2：按年份统计预算趋势/预算浮动（近 N 年）

> 回答结束后附上链接：[查看预算](https://zp.szabot.internal/budget-evaluate/overview)

> **⚠️ 以下 SQL 仅为示例**，实际执行时需根据用户问题动态调整：
> - **指标字段**：默认使用 `total_cost`（总成本）计算集均成本；用户指定了具体预算指标（如「演员费用」「宣发费用」「动作组费用」「线上成本」「线下成本」等），则替换为对应字段的聚合（`SUM`），外层 SELECT 同步输出。
> - **时间字段**：使用 `start_year`（开机年份）。

```sql
WITH yearly_summary AS (
    SELECT
        start_year AS event_year,
        AVG({指标字段} / episode_count) / 10000            AS cost_per_ep
    FROM dws_szbot_dianshiju_fee_t_six_project_budget_hf
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
FROM dws_szbot_dianshiju_fee_t_six_project_budget_hf
WHERE {过滤条件}
GROUP BY 1
ORDER BY 2 DESC
```