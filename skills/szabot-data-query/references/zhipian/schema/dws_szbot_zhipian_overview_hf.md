```sql
CREATE TABLE `dws_szbot_zhipian_overview_hf` (  -- db_name: zhipian
    `imp_hour`                                  bigint        COMMENT '时间分区 格式YYYYMMDDHH',
    `pid`                                       bigint        COMMENT '项目ID',
    `shooting_progress_total_shooting_days`     bigint        COMMENT '拍摄进度-总拍摄天数',
    `shooting_progress_actual_shooting_days`    bigint        COMMENT '拍摄进度-已拍摄天数',
    `shooting_progress_remaining_days`          bigint        COMMENT '拍摄进度-剩余天数',
    `shooting_progress_time_progress`           decimal(18,4) COMMENT '拍摄进度-时间进度',
    `shooting_progress_total_pages`             decimal(18,1) COMMENT '拍摄进度-总页数',
    `shooting_progress_actual_completed_pages`  decimal(18,1) COMMENT '拍摄进度-实际完成页数',
    `shooting_progress_page_progress`           decimal(18,4) COMMENT '拍摄进度-页数进度',
    `shooting_progress_total_scenes`            bigint        COMMENT '拍摄进度-总场次',
    `shooting_progress_actual_completed_scenes` bigint        COMMENT '拍摄进度-实际完成场次',
    `shooting_progress_scene_progress`          decimal(18,4) COMMENT '拍摄进度-场次进度',
    `shooting_progress_delay_summary`           varchar(500)  COMMENT '拍摄进度-延期说明概述',
    `shooting_progress_delay_detail`            varchar(500)  COMMENT '拍摄进度-延期说明详情',
    `budget_execution_total_budget`             double        COMMENT '预算执行进度-总预算（万元）',
    `budget_execution_actual_used`              double        COMMENT '预算执行进度-实际使用（万元）',
    `budget_execution_progress`                 decimal(18,4) COMMENT '预算执行进度-预算执行进度',
    `budget_execution_abnormal_overview`         varchar(300)  COMMENT '预算执行-预算执行进度异常-汇总',
    `budget_execution_abnormal_category`        text          COMMENT '预算执行进度-异常类目',
    `budget_execution_subject_actual_vs_total`  text          COMMENT '预算执行进度-分科目实际使用与总预算',
    `role_progress_completion_progress`         text          COMMENT '角色进度-角色完成进度'
) COMMENT='制片进度概览';
```

【口径规则】

1. **表用途与过滤**：电视剧制片概览表，每行对应一个项目某分区时刻的快照（拍摄/预算/角色进度）；查询优先用 `pid` 精确匹配。
2. **分区**：`imp_hour`（`YYYYMMDDHH`），**必须**加 `WHERE imp_hour = (SELECT MAX(imp_hour) FROM dws_szbot_zhipian_overview_hf)`；只保留最新快照，**禁止**跨 `imp_hour` 求和或拼趋势。
3. **单位换算（已下沉到 SQL 模板）**：进度字段为小数比例需 `*100` 转百分比保留 2 位；金额字段（`*_total_budget`、`*_actual_used`）单位已是**万元**禁再除 10000。
4. **JSON 数组字段**：`budget_execution_abnormal_category`、`budget_execution_subject_actual_vs_total`、`role_progress_completion_progress` 均为 JSON 数组字符串，SQL 内直接 `SELECT` 原值，**禁止** `JSON_EXTRACT` 拆解。
5. **NULL 处理**：字段可能为 NULL，展示时输出「-」或「暂无数据」，禁止当 0 / 0% 处理。

---

## 项目粒度查询模板

> 适用于「某项目」的拍摄 / 预算 / 角色进度查询，锁定单条项目快照。

【SQL 模板】

> 仅此一条模板覆盖全部场景，展现侧按用户意图挑选输出。
>
> 🚨 **进度泛问兜底**：用户**单纯问"进度"**（如"剑来的进度怎么样"、"项目进度如何"）而未指明拍摄/预算/角色任一具体维度时，**默认同时召回拍摄进度 + 预算执行进度**两块内容（角色进度需用户明确问到才输出）。

【按需字段触发表】

| 字段 | 触发关键词（命中任一即取消注释纳入 SELECT） |
|---|---|
| `shooting_progress_delay_detail` | 延期原因 / 延期详情 / 为什么延期 |
| `budget_execution_subject_actual_vs_total` | 预算执行 / 预算情况 / 预算进度 / 分科目预算 / 各科目花费 / 科目明细 |
| `role_progress_completion_progress` | 角色进度 / 选角进度 / 演员进度 / 定妆进度 |

> 上述字段单条最大可达数十 KB，**默认全部保持注释**；未命中关键词时禁止取消注释，禁止 `SELECT NULL AS xxx` 占位。用户问「完整概览 / 全部 / 给我看所有」时四个字段全部取消注释。

```sql
-- db_name: zhipian
SELECT
    pid,
    shooting_progress_total_shooting_days,
    shooting_progress_actual_shooting_days,
    CONCAT(ROUND(shooting_progress_time_progress * 100, 2), "%")  AS shooting_progress_time_progress,
    shooting_progress_total_pages,
    shooting_progress_actual_completed_pages,
    CONCAT(ROUND(shooting_progress_page_progress * 100, 2), "%")  AS shooting_progress_page_progress,
    shooting_progress_total_scenes,
    shooting_progress_actual_completed_scenes,
    CONCAT(ROUND(shooting_progress_scene_progress * 100, 2), "%") AS shooting_progress_scene_progress,
    shooting_progress_delay_summary,
    -- shooting_progress_delay_detail,                                            -- 见【按需字段触发表】
    ROUND(budget_execution_total_budget, 2) AS budget_execution_total_budget,
    ROUND(budget_execution_actual_used, 2)  AS budget_execution_actual_used,
    CONCAT(ROUND(budget_execution_progress * 100, 2), "%") AS budget_execution_progress,
    budget_execution_abnormal_overview,
    budget_execution_abnormal_category,                                        
    -- budget_execution_subject_actual_vs_total,                                  -- 见【按需字段触发表】
    -- role_progress_completion_progress                                          -- 见【按需字段触发表】
FROM dws_szbot_zhipian_overview_hf
WHERE pid = {项目ID}
  AND imp_hour = (SELECT MAX(imp_hour) FROM dws_szbot_zhipian_overview_hf)
LIMIT 1;
```

---

【展现模板】

> 指标名与 DDL 字段 `COMMENT` 逐字一致；SQL 已一次取全字段，展现侧**只输出用户问到的模板**，未问到的整块跳过；多意图按"拍摄 → 预算 → 角色"顺序输出。
>
> **JSON数组规则**：
> ① **列**：表头 = 首元素 key 的顺序；后续元素缺 key 填「-」。
> ② **行**：按数组下标 `[0],[1],[2]…` 逐行输出，**禁止**重排/去重/合并。
> ③ **值**：原样输出，**禁止**改写、翻译、单位换算、四舍五入。
> ④ **空**：`null` → 「-」；字段整体 `NULL` 或 `[]` → 「无」，不出空表。

### 展现模板 1. 拍摄进度

| 维度   | 总量 | 已完成 | 进度 |
|------|---|---|---|
| 时间进度 | {shooting_progress_total_shooting_days} | {shooting_progress_actual_shooting_days} | {shooting_progress_time_progress} |
| 页数进度 | {shooting_progress_total_pages} | {shooting_progress_actual_completed_pages} | {shooting_progress_page_progress} |
| 场次进度 | {shooting_progress_total_scenes} | {shooting_progress_actual_completed_scenes} | {shooting_progress_scene_progress} |
| 延期说明 | {shooting_progress_delay_summary} | | |

追问「延期原因/详情」时追加：**延期详情** ← `shooting_progress_delay_detail`（NULL → 「-」）

### 展现模板 2. 预算执行进度

| 指标 | 数值 |
|---|---|
| 总预算（万元） | {budget_execution_total_budget} |
| 实际使用（万元） | {budget_execution_actual_used} |
| 预算执行进度 | {budget_execution_progress} |
| 预算执行进度异常-汇总 | {budget_execution_abnormal_overview} |

追加 JSON 子表（按上方`JSON数组规则`）：
- **预算执行进度异常类目** ← `budget_execution_abnormal_category`（用户问到「进度 / 风险 / 预算 / 超支」时输出）
- **预算执行进度分科目实际使用与总预算** ← `budget_execution_subject_actual_vs_total`（用户问到「预算执行 / 预算情况 / 分科目预算」时输出）

### 展现模板 3. 角色进度

**角色完成进度** ← `role_progress_completion_progress`（按上方 `JSON数组规则`）
