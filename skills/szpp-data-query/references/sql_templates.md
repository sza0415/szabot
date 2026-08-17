# SQL 模板与场景示例

## 一、SQL 模板

### 获取最新 imp_date（所有 szpp 表通用）

```sql
SELECT MAX(imp_hour) FROM szdw_dim.chuku_progress WHERE table_name = 'miniapp_progress_hour';
```

### 查询某IP的累计侵权统计

```sql
SELECT
    ip_name AS 版权名称,
    letter_num AS 发函量,
    tort_delete_num AS 下架量,
    letter_num_no_se AS 不含搜索引擎发函量,
    tort_delete_in_48h_num AS 48小时内下架量,
    avg_protect_duration AS 平均防护时长,
    play_vv AS 侵权播放量
FROM szpp.t_mini_tort_metrics
WHERE imp_date = (
    SELECT MAX(imp_hour) FROM szdw_dim.chuku_progress WHERE table_name = 'miniapp_progress_hour'
)
AND ip_name = '{IP名称}'
AND platform = 'ALL'
AND platform_type_name = 'ALL';
```

### 查询某IP的侵权趋势

```sql
SELECT
    date_diff AS 上线天数,
    belike_tort_num AS 疑似侵权量,
    tort_num AS 打击量,
    letter_num AS 发函量,
    tort_delete_num AS 下架量
FROM szpp.t_mini_trend_statistic
WHERE imp_date = (
    SELECT MAX(imp_hour) FROM szdw_dim.chuku_progress WHERE table_name = 'miniapp_progress_hour'
)
AND ip_name = '{IP名称}'
AND platform = 'ALL'
AND platform_type_name = 'ALL'
AND platform_type_name != '长尾网站'
ORDER BY date_diff ASC;
```

### 查询IP基础信息

```sql
SELECT
    ip_name AS 版权名称,
    cate_name AS IP品类,
    budget_level AS 维权等级,
    monitor_status AS 监测状态,
    play_time_start AS 开始播放时间,
    play_time_end AS 完结时间,
    avg_letter_num_on_operation_period AS 运营期日均发函量,
    accumulate_letter_num_on_operation_period AS 运营期累积发函量
FROM szpp.t_mini_ipinfo
WHERE imp_date = (
    SELECT MAX(imp_hour) FROM szdw_dim.chuku_progress WHERE table_name = 'miniapp_progress_hour'
)
AND ip_name = '{IP名称}';
```

---

## 二、常见查询场景

### 场景1：某IP的发函和下架数据

**用户问**："查询XX剧的发函量和下架量"

**处理流程**：
1. 识别：侵权累计统计 → 使用 `szpp.t_mini_tort_metrics`
2. 识别：需要指定 ip_name，platform/platform_type_name 默认 ALL
3. 构建 SQL：

```sql
SELECT
    ip_name AS 版权名称,
    letter_num AS 发函量,
    tort_delete_num AS 下架量,
    letter_num_no_se AS 不含搜索引擎发函量,
    ROUND(tort_delete_num / NULLIF(letter_num_no_se, 0) * 100, 2) AS 下架率
FROM szpp.t_mini_tort_metrics
WHERE imp_date = (
    SELECT MAX(imp_hour) FROM szdw_dim.chuku_progress WHERE table_name = 'miniapp_progress_hour'
)
AND ip_name = 'XX剧'
AND platform = 'ALL'
AND platform_type_name = 'ALL';
```

### 场景2：某IP的侵权趋势

**用户问**："XX剧的侵权趋势怎样"

**处理流程**：
1. 识别：侵权趋势 → 使用 `szpp.t_mini_trend_statistic`
2. 识别：默认按 date_diff 升序，排除长尾网站
3. 构建 SQL：

```sql
SELECT
    date_diff AS 上线天数,
    belike_tort_num AS 疑似侵权量,
    tort_num AS 打击量,
    letter_num AS 发函量,
    tort_delete_num AS 下架量
FROM szpp.t_mini_trend_statistic
WHERE imp_date = (
    SELECT MAX(imp_hour) FROM szdw_dim.chuku_progress WHERE table_name = 'miniapp_progress_hour'
)
AND ip_name = 'XX剧'
AND platform = 'ALL'
AND platform_type_name != '长尾网站'
ORDER BY date_diff ASC;
```

### 场景3：分平台侵权对比

**用户问**："XX剧在各平台的下架量对比"

**处理流程**：
1. 识别：分平台 → `platform != 'ALL'`
2. 构建 SQL：

```sql
SELECT
    platform AS 平台,
    letter_num AS 发函量,
    tort_delete_num AS 下架量,
    ROUND(tort_delete_num / NULLIF(letter_num_no_se, 0) * 100, 2) AS 下架率
FROM szpp.t_mini_tort_metrics
WHERE imp_date = (
    SELECT MAX(imp_hour) FROM szdw_dim.chuku_progress WHERE table_name = 'miniapp_progress_hour'
)
AND ip_name = 'XX剧'
AND platform != 'ALL'
AND platform_type_name = 'ALL'
ORDER BY letter_num DESC;
```

### 场景4：多IP横向对比

**用户问**："对比A剧和B剧的发函量和下架率"

**处理流程**：
1. 识别：多 IP 对比 → 放在同一查询中
2. 构建 SQL：

```sql
SELECT
    ip_name AS 版权名称,
    letter_num AS 发函量,
    tort_delete_num AS 下架量,
    ROUND(tort_delete_num / NULLIF(letter_num_no_se, 0) * 100, 2) AS 下架率,
    avg_protect_duration AS 平均防护时长
FROM szpp.t_mini_tort_metrics
WHERE imp_date = (
    SELECT MAX(imp_hour) FROM szdw_dim.chuku_progress WHERE table_name = 'miniapp_progress_hour'
)
AND ip_name IN ('A剧', 'B剧')
AND platform = 'ALL'
AND platform_type_name = 'ALL';
```

---

## 三、结果展示规范

### 数据格式化

- 数值类型添加千位分隔符
- 百分比保留2位小数
- 时长类型标注单位（小时/天）

### 结果展示示例

```markdown
## 查询结果

### 某IP侵权累计统计

| 版权名称 | 发函量 | 下架量 | 下架率 | 平均防护时长 |
|----------|--------|--------|--------|-------------|
| XX剧 | 12,345 | 10,876 | 88.1% | 48小时 |

**数据说明**：
- 下架率 = 下架量 / 不含搜索引擎发函量
- 数据来源：szpp.t_mini_tort_metrics
```

### IP 对比展示

- 多 IP 对比时放在同一表格中
