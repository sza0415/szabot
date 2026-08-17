```
CREATE TABLE `dws_donghua_zhipian_szcanvas_metrics_df` (
    `imp_date` bigint COMMENT '分区日期',
    `data_date` varchar(32) COMMENT '素材日期',
    `pid` bigint COMMENT '项目ID',
    `pname` varchar(255) COMMENT '项目名',
    `studio_name` varchar(255) COMMENT '工作室',
    `studio_group_name` varchar(255) COMMENT '工作室群',
    `content_rating_init_name` varchar(255) COMMENT '立项评级',
    `user_id` bigint COMMENT '用户ID',
    `asset_type` varchar(64) COMMENT '素材类型: video->视频, image->图片, null/空->其他',
    `model_name` varchar(255) COMMENT '模型',
    `has_downloaded` bigint COMMENT '是否被下载过',
    `asset_num` bigint COMMENT '素材数量',
    `asset_cost` double COMMENT '费用（元）'
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_general_ci COMMENT = '动漫制片SzCanvas素材指标表'
```

【口径规则】

1. **三类指标聚合方式**：
   - 素材数（调用次数）：`SUM(asset_num)`
   - 算力成本（费用）：`ROUND(SUM(asset_cost), 2)`
   - 活跃用户数：`COUNT(DISTINCT user_id)`
2. **过滤条件**（按需拼入 WHERE，未指定的条件不加，其余维度同理）：
   - 最新分区 → `imp_date = (SELECT MAX(imp_date) FROM dws_donghua_zhipian_szcanvas_metrics_df)`（必选，禁止写死）
   - 有项目列表（Step 0 取得 ID 后拼入） → `AND pid IN ({项目ID列表})`；未指定项目时省略
   - 指定了日期范围 → `AND data_date >= {开始日期} AND data_date <= {结束日期}`
3. **asset_type**：`video`→视频，`image`→图片，`null`/空→其他；排除其他素材加 `coalesce(asset_type, '') != ''`
4. **has_downloaded**：0 或 1，1 表示已被下载过
5. **工作室维度**：`studio_group_name`（工作室群，默认）或 `studio_name`（工作室）

【查询模板】

### 模板选择路由

| 用户意图 / 关键词 | 命中模板 |
|---|---|
| 总量/总体/概览/默认 | 模板 1 默认查询 |
| 视频/图片/素材类型 | 模板 2 按素材类型 |
| 模型/哪个模型 | 模板 3 按模型 |
| 工作室/团队 | 模板 4 按工作室群 |
| 项目/哪个项目 | 模板 5 按项目 |
| 评级/S级/A级 | 模板 6 按立项评级 |
| 下载/未下载 | 模板 7 按下载状态 |
| 下载率/利用率 | 模板 8 素材下载率 |
| 未明确维度 | 默认模板 1 |

以下模板中 `{过滤条件}` 按口径规则第 2 条拼入 WHERE，imp_date 子查询为必选项，其余按需。

#### 模板 1. 默认查询

```sql
-- db_name: donghua
SELECT
    SUM(asset_num) AS 调用次数,
    ROUND(SUM(asset_cost), 2) AS 算力成本,
    COUNT(DISTINCT user_id) AS 活跃用户数
FROM dws_donghua_zhipian_szcanvas_metrics_df
WHERE {过滤条件};
```

#### 模板 2. 按素材类型汇总

```sql
-- db_name: donghua
SELECT
    asset_type,
    SUM(asset_num) AS 调用次数,
    ROUND(SUM(asset_cost), 2) AS 算力成本,
    COUNT(DISTINCT user_id) AS 活跃用户数
FROM dws_donghua_zhipian_szcanvas_metrics_df
WHERE {过滤条件}
GROUP BY asset_type
ORDER BY 调用次数 DESC;
```

#### 模板 3. 按模型汇总

```sql
-- db_name: donghua
SELECT
    model_name,
    SUM(asset_num) AS 调用次数,
    ROUND(SUM(asset_cost), 2) AS 算力成本,
    COUNT(DISTINCT user_id) AS 活跃用户数
FROM dws_donghua_zhipian_szcanvas_metrics_df
WHERE {过滤条件}
GROUP BY model_name
ORDER BY 算力成本 DESC;
```

#### 模板 4. 按工作室群汇总

```sql
-- db_name: donghua
SELECT
    studio_group_name,
    SUM(asset_num) AS 调用次数,
    ROUND(SUM(asset_cost), 2) AS 算力成本,
    COUNT(DISTINCT user_id) AS 活跃用户数
FROM dws_donghua_zhipian_szcanvas_metrics_df
WHERE {过滤条件}
GROUP BY studio_group_name
ORDER BY 算力成本 DESC;
```

#### 模板 5. 按项目汇总

```sql
-- db_name: donghua
SELECT
    pid,
    pname,
    SUM(asset_num) AS 调用次数,
    ROUND(SUM(asset_cost), 2) AS 算力成本,
    COUNT(DISTINCT user_id) AS 活跃用户数
FROM dws_donghua_zhipian_szcanvas_metrics_df
WHERE {过滤条件}
GROUP BY pid, pname
ORDER BY 算力成本 DESC;
```

#### 模板 6. 按立项评级汇总

```sql
-- db_name: donghua
SELECT
    content_rating_init_name,
    SUM(asset_num) AS 调用次数,
    ROUND(SUM(asset_cost), 2) AS 算力成本,
    COUNT(DISTINCT user_id) AS 活跃用户数
FROM dws_donghua_zhipian_szcanvas_metrics_df
WHERE {过滤条件}
GROUP BY content_rating_init_name
ORDER BY 算力成本 DESC;
```

#### 模板 7. 按下载状态汇总

```sql
-- db_name: donghua
SELECT
    CASE WHEN has_downloaded = 1 THEN '已下载' ELSE '未下载' END AS 下载状态,
    SUM(asset_num) AS 调用次数,
    ROUND(SUM(asset_cost), 2) AS 算力成本,
    COUNT(DISTINCT user_id) AS 活跃用户数
FROM dws_donghua_zhipian_szcanvas_metrics_df
WHERE {过滤条件}
GROUP BY has_downloaded
ORDER BY 调用次数 DESC;
```

#### 模板 8. 素材下载率

```sql
-- db_name: donghua
SELECT
    SUM(asset_num) AS 总调用次数,
    SUM(CASE WHEN has_downloaded = 1 THEN asset_num ELSE 0 END) AS 已下载调用次数,
    ROUND(
        SUM(CASE WHEN has_downloaded = 1 THEN asset_num ELSE 0 END) / NULLIF(SUM(asset_num), 0) * 100,
        2
    ) AS 下载率
FROM dws_donghua_zhipian_szcanvas_metrics_df
WHERE {过滤条件};
```

---

### 展现规则

#### 通用展现规则

| 场景 | 展现形式 |
|---|---|
| 默认汇总 | 按 SQL 字段顺序，直出表格 |
| 按维度分组 | 维度做行标题，指标列对齐 |
| 多项目对比 | 以 `pid` 为主键区分，每个项目占一行，字段按 SQL `SELECT` 顺序 |

#### 指标格式

| 指标 | 格式 |
|---|---|
| 算力成本 | 两位小数，单位「元」 |
| 调用次数 | 整数 |
| 活跃用户数 | 整数 |
| 下载率 | 百分比两位小数 |

#### 翻译与标注

- 素材类型翻译：`video`→视频，`image`→图片，`null`/空→其他
- 项目维度必须回显 `pname` + `pid`
- 输出注明 `data_date` 范围
- 查询结果输出必须注明数据源为「动漫品类的SzCanvas算力看板」
