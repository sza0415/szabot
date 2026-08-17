# SQL 查询素材数据（mcp_exec_sql）

## 调用方式

```bash
# 查询出库信息
mcporter call 'szabot_data_query_svr.mcp_exec_sql' --args '{
  "db_name": "szdw_dim",
  "sql": "SELECT ... FROM ... WHERE ..."
}'

# 查询 SzStudio 信息
mcporter call 'szabot_data_query_svr.mcp_exec_sql' --args '{
  "db_name": "szstudio",
  "sql": "SELECT ... FROM ... WHERE ..."
}'
```

## 参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `sql` | string | 是 | 要执行的 SQL 语句 |
| `db_name` | string | 是 | 数据库名称，仅允许 `szdw_dim` 或 `szstudio`（见下方规则） |
| `max_rows` | int | 否 | 最大返回行数，默认 1000 |

> ⚠️ **db_name 强约束**：
> - 出库相关查询 → `db_name = "szdw_dim"`
> - SzStudio 相关查询 → `db_name = "szstudio"`
> - **禁止使用上述两个以外的任何 db_name 值**，不得编造或猜测

> ⚠️ **禁止通过 mcp_exec_sql 扫描表结构或表列表**（如 `SHOW TABLES`、`DESCRIBE`、`SHOW COLUMNS`、`information_schema` 等），所有表结构信息必须且只能从 `references/szstudio_schema/` 目录或 `references/chuku_schema.md` 中获取。

## 返回结构

```json
{
  "results": [
    "{\"col1\":\"val1\",\"col2\":\"val2\"}"
  ],
  "affected_rows": 0,
  "total_rows": 10,
  "truncated": false,
  "execution_time_ms": 120
}
```

| 字段 | 说明 |
|------|------|
| `results` | 每行数据序列化为一个 JSON 字符串 |
| `total_rows` | 实际返回的行数 |
| `truncated` | 结果是否被截断（超过 max_rows 时为 true） |
| `execution_time_ms` | 执行耗时（毫秒） |

## 数据库与表

| 数据库 | 用途 | 说明 |
|--------|------|------|
| `szdw_dim` | 出库信息查询 | 表结构见 `references/chuku_schema.md` |
| `szstudio` | SzStudio 影视制作信息查询 | 表结构见 `references/szstudio_schema/` 目录 |

> ⚠️ 仅允许使用上述两个数据库，禁止使用其他 db_name。构建 SQL 前必须先读取对应的表结构文件，**禁止通过 SQL 查询表结构或表列表**。

## ⚠️ 禁止模糊匹配

**所有表的所有字段过滤，必须使用精确匹配（`=`），严禁使用 `LIKE`、`REGEXP`、`RLIKE`、`INSTR`、`LOCATE`、`FIND_IN_SET` 等任何形式的模糊/包含匹配。**

> ❌ 禁止：`WHERE project_name LIKE '%青簪行%'`
> ❌ 禁止：`WHERE project_name REGEXP '青簪行'`
> ✅ 正确：`WHERE project_name = '青簪行'`

**若用户提供的名称不确定是否精确，必须先向用户确认完整的精确名称，再构建 SQL，不得擅自使用模糊匹配代替。**

---

## ⚠️ imp_date 强制约束

**所有数据查询的 SQL 都必须在 WHERE 条件中带上 `imp_date`**，不允许省略。

- `imp_date` 代表数据快照日期，不带 `imp_date` 会导致查到多日快照数据叠加，结果严重失真
- 查询前应先从出库表获取目标表的最新 `imp_date`：
  ```sql
  -- db_name: szdw_dim
  SELECT MAX(imp_date) AS latest_imp_date
  FROM chuku_progress
  WHERE table_name = '目标表名'
  ```
- 然后在后续所有查询中加上 `WHERE imp_date = '获取到的值'`
- **禁止**不带 `imp_date` 直接查询任何 szstudio 数据表

## ⚠️ imp_date 与业务日期的区别

| 字段 | 含义 | 使用方式 |
|------|------|---------|
| `imp_date` | 数据入库快照日期 | 固定取 MAX 值，仅用于分区过滤，不随用户描述的日期变化 |
| `create_date` | 素材实际生成日期 | 用户说"昨天/某天生成的"时，用此字段过滤业务日期 |

**示例**：用户问"昨天生成的视频数"，正确写法：
```sql
WHERE imp_date = '最新imp_date'   -- 固定取最新快照
  AND create_date = '系统当前日期 - 1天'  -- "昨天" = 系统当前自然日减去1天，与 imp_date 无关，禁止用 imp_date - 1 代替
```

> ⚠️ **相对日期强约束**：涉及"昨天"、"最近 N 天"等相对日期时，**必须以系统当前自然日为基准推算**，与 `imp_date` 无关。
> - "昨天" = 系统当前日期 - 1 天
> - "最近 N 天" = 系统当前日期往前推 N 天

## SzStudio 表选择指南

> ⚠️ **选对表是构建正确查询的前提**，请根据以下指南按需选择。

### 1. `ads_aigc_user_activity_df` — 用户活跃度统计表

用于查询 SzStudio 的 **DAU、活跃事件次数（PV）、活跃时长** 等用户活跃度指标。查询 SzStudio、zen、SzCanvas 相关指标时，`app` 字段对应的值为 `szcanvas`，即 `WHERE app = 'szcanvas'`。所有字段过滤均须**精确匹配**（使用 `=`，禁止 `LIKE` 或模糊匹配）。

> ⚠️ **DAU 强约束**：
> - `dau` **严禁 SUM**：不同天的活跃用户存在重叠，跨天相加会重复计算同一用户，导致数值虚高
> - 查询整体概览/汇总时，**DAU 只能取单天（最新一天）的值**，用 `WHERE data_date = '最新日期'` 过滤后直接展示，不得对 dau 求和
> - 如需展示趋势，只能按天列出每天的 dau 值，不能合并加总
> - `pv`、`duration` 可以跨天 SUM 聚合，不受此限制
> - **严禁跨 app 聚合**：不同 app 的用户群体相互独立，查询时必须在 WHERE 中指定具体 app，或按 app 分组展示，不得将多个 app 的数据合并汇总

### 2. `dwd_aigc_szanimate_image_info_df` — 图片信息明细表

SzStudio 生成的**图片信息明细**。所有字段过滤均须**精确匹配**（使用 `=`，禁止 `LIKE` 或模糊匹配）。

> ⚠️ **仅用于查看具体素材内容（asset_id、链接等）**，禁止用此表统计数量/汇总，图片数统计请用 `ads_aigc_szanimate_image_metrics_df`。

### 3. `dwd_aigc_szanimate_video_info_df` — 视频信息明细表

SzStudio 生成的**视频信息明细**。所有字段过滤均须**精确匹配**（使用 `=`，禁止 `LIKE` 或模糊匹配）。

> ⚠️ **仅用于查看具体素材内容（asset_id、链接等）**，禁止用此表统计数量/汇总，视频数统计请用 `ads_aigc_szanimate_video_metrics_df`。

### 4. `ads_aigc_szanimate_image_metrics_df` — 图片指标聚合表

SzStudio 生成图片的**指标聚合表**。当需要统计**日总量、总量**等图片指标时使用此表，通过对 `pv` 字段求和即可。所有字段过滤均须**精确匹配**（使用 `=`，禁止 `LIKE` 或模糊匹配）。

### 5. `ads_aigc_szanimate_video_metrics_df` — 视频指标聚合表

SzStudio 生成视频的**指标聚合表**。当需要统计**日总量、总量**等视频指标时使用此表，通过对 `pv` 字段求和即可。所有字段过滤均须**精确匹配**（使用 `=`，禁止 `LIKE` 或模糊匹配）。

### 6. `ods_agic_zen_aivfx_edite_video_full_df` — 影视后期视频总表

**仅当用户明确指出「影视后期」时使用**，直接用 `szbot_project_name` 或 `szbot_project_id` 过滤。所有字段过滤均须**精确匹配**（使用 `=`，禁止 `LIKE` 或模糊匹配）。

### 7. `dwd_aigc_zen_zhongqi_df` — 影视前中期素材表

**仅当用户明确指出「影视前中期」时使用**，直接用 `szbot_project_name` 或 `szbot_project_id` 过滤。所有字段过滤均须**精确匹配**（使用 `=`，禁止 `LIKE` 或模糊匹配）。

### 表选择决策树

```
用户想查什么？
  │
  ├─ SzStudio 整体概览 / 整体制作数据（含"整体数据"、"整体情况"、"概览"等模糊表述）
  │   ⚠️ 此分支优先匹配，不得被"用户活跃度"分支抢先处理
  │   ⚠️ 禁止查询 DAU，禁止自行扩展查询项目数、用户数等聚合表中不存在的指标
  │   └─ 仅查询有明确定义的指标：
  │       ├─ 短番视频总数 → ads_aigc_szanimate_video_metrics_df（SUM(pv)）
  │       ├─ 短番图片总数 → ads_aigc_szanimate_image_metrics_df（SUM(pv)）
  │       ├─ 影视前中期素材数 → dwd_aigc_zen_zhongqi_df（COUNT(DISTINCT asset_id)）
  │       └─ 影视后期素材数 → ods_agic_zen_aivfx_edite_video_full_df（COUNT(DISTINCT asset_id)）
  │
  ├─ 用户活跃度（DAU/PV/时长）【仅当用户明确询问活跃度/DAU/日活/使用时长时才走此分支】
  │   └─ → ads_aigc_user_activity_df（SzStudio/zen/SzCanvas 指标：WHERE app = 'szcanvas'）
  │
  ├─ 影视后期素材/制作进度
  │   └─ → ods_agic_zen_aivfx_edite_video_full_df
  │
  ├─ 影视前中期素材/制作进度
  │   └─ → dwd_aigc_zen_zhongqi_df
  │
  ├─ SzStudio 短番视频/图片（含 szcanvas/zen/SzCanvas 等别名，**统计数量时禁止用明细表**）
  │   │
  │   ├─ 统计数量（视频数/图片数，无论是否按项目过滤）
  │   │   ├─ 视频数 → ads_aigc_szanimate_video_metrics_df（SUM(pv)）
  │   │   └─ 图片数 → ads_aigc_szanimate_image_metrics_df（SUM(pv)）
  │   │
  │   ├─ 查看具体素材内容/明细（asset_id、链接等）
  │   │   ├─ 视频明细 → dwd_aigc_szanimate_video_info_df
  │   │   └─ 图片明细 → dwd_aigc_szanimate_image_info_df
  │   │
  │   └─ 不确定 → 优先用聚合表，禁止用明细表做统计
  └─ 不确定 → 优先用聚合表，避免全表扫描明细表
```

## 项目名称/ID 查询方式

> ⚠️ **仅适用于影视制作（前中期/后期）场景**。SzStudio 短番视频/图片查询（`ads_aigc_szanimate_video_metrics_df`、`ads_aigc_szanimate_image_metrics_df`、`dwd_aigc_szanimate_video_info_df`、`dwd_aigc_szanimate_image_info_df`）直接用 `project_name = '项目名'` 精确匹配即可，**禁止**为此类查询加载 `szabot-copilot`。

当用户询问**影视制作（前中期/后期）**且提到某个**项目名称**或**影库项目ID**时：

1. 加载 `szabot-copilot` Skill，**仅查项目基础信息**（项目名称、项目ID、品类、集数、状态等），查完**立即回到本 Skill**，不要继续执行 szabot-copilot 的后续流程
2. 直接用 `szbot_project_name` 或 `szbot_project_id` 作为 WHERE 条件查询制作数据
3. 将项目基础信息 + 制作数据一并展示给用户

## 影视制作进度查询（前中期 + 后期）

当用户询问某项目的**影视制作进度**时，**必须同时查询前中期和后期两张表**，分别展示：

| 阶段 | 表 | 说明 |
|------|-----|------|
| 影视前中期 | `dwd_aigc_zen_zhongqi_df` | 前中期制作素材 |
| 影视后期 | `ods_agic_zen_aivfx_edite_video_full_df` | 后期生成视频 |

> ⚠️ **禁止只查其中一张表**，两张表必须同时查询，分别展示前中期和后期数据。

### 端到端示例

**用户问**：青簪行的影视制作进度怎么样？

#### Step 1：加载 szabot-copilot，查询项目基础信息（项目名称、项目ID、品类、集数、状态等），查完立即回到本 Skill

#### Step 2：分别获取前中期和后期表的最新 imp_date

```sql
-- db_name: szdw_dim（前中期表）
SELECT MAX(imp_date) AS latest_imp_date
FROM chuku_progress
WHERE table_name = 'dwd_aigc_zen_zhongqi_df'
```

```sql
-- db_name: szdw_dim（后期表）
SELECT MAX(imp_date) AS latest_imp_date
FROM chuku_progress
WHERE table_name = 'ods_agic_zen_aivfx_edite_video_full_df'
```

#### Step 3：查询【前中期】素材总量与每天增量

```sql
-- db_name: szstudio
SELECT
  COUNT(DISTINCT asset_id) AS total_assets
FROM dwd_aigc_zen_zhongqi_df
WHERE imp_date = '20260401'
  AND szbot_project_name = '青簪行'
```

```sql
-- db_name: szstudio
SELECT
  create_date,
  COUNT(DISTINCT asset_id) AS daily_new_assets
FROM dwd_aigc_zen_zhongqi_df
WHERE imp_date = '20260401'
  AND szbot_project_name = '青簪行'
GROUP BY create_date
ORDER BY create_date DESC
LIMIT 7
```

#### Step 4：查询【后期】视频总量、被下载总量与每天增量

```sql
-- db_name: szstudio
SELECT
  COUNT(DISTINCT asset_id) AS total_videos,
  COUNT(DISTINCT CASE WHEN has_downloaded = 1 THEN asset_id END) AS total_downloaded
FROM ods_agic_zen_aivfx_edite_video_full_df
WHERE imp_date = '20260401'
  AND szbot_project_name = '青簪行'
  AND is_test = 0
```

```sql
-- db_name: szstudio
SELECT
  create_date,
  COUNT(DISTINCT asset_id) AS daily_new_videos,
  COUNT(DISTINCT CASE WHEN has_downloaded = 1 THEN asset_id END) AS daily_new_downloaded
FROM ods_agic_zen_aivfx_edite_video_full_df
WHERE imp_date = '20260401'
  AND szbot_project_name = '青簪行'
  AND is_test = 0
GROUP BY create_date
ORDER BY create_date DESC
LIMIT 7
```

#### Step 5：分别获取前中期和后期最近 5 个素材的 asset_id

```sql
-- db_name: szstudio（前中期）
SELECT DISTINCT asset_id, create_time
FROM dwd_aigc_zen_zhongqi_df
WHERE imp_date = '20260401'
  AND szbot_project_name = '青簪行'
ORDER BY create_time DESC
LIMIT 5
```

```sql
-- db_name: szstudio（后期）
SELECT DISTINCT asset_id, create_time
FROM ods_agic_zen_aivfx_edite_video_full_df
WHERE imp_date = '20260401'
  AND szbot_project_name = '青簪行'
  AND is_test = 0
ORDER BY create_time DESC
LIMIT 5
```

#### Step 6：用 asset_id 获取播放链接

将 Step 5 拿到的前中期和后期 asset_id 列表分别传给 `cms_search_assets`，获取链接后展示给用户。

---

**输出示例**：

> **青簪行** 影视制作进度（截至 2026-04-01）：
>
> **项目基础信息**（来源：影库）：
> - 品类：电视剧 | 集数：40集 | 状态：制作中
>
> ---
>
> ### 🎬 影视前中期
>
> **影视前中期总量概览**：
> - 影视前中期素材总量：**86** 个
>
> **近 7 天每日增量**：
>
> | 日期 | 新增影视前中期素材 |
> |------|----------|
> | 2026-04-01 | 6        |
> | 2026-03-31 | 10       |
> | ... | ...      |
>
> **最近 5 个影视前中期素材**：
>
> | 素材ID | 创建时间 | 链接 |
> |--------|---------|------|
> | asset_001 | 2026-04-01 18:30 | [点击查看](url) |
> | ... | ... | ... |
>
> ---
>
> ### 🎞️ 影视后期
>
> **影视后期总量概览**：
> - 影视后期总量：**128** 个
> - 影视后期已被下载：**96** 个
>
> **近 7 天每日增量**：
>
> | 日期 | 影视后期视频 | 影视后期新增下载 |
> |------|---------|------|
> | 2026-04-01 | 8 | 5 |
> | 2026-03-31 | 12 | 9 |
> | ... | ... | ... |
>
> **最近 5 个影视后期素材**：
>
> | 素材ID | 创建时间 | 链接 |
> |--------|---------|------|
> | asset_001 | 2026-04-01 18:30 | [点击查看](url) |
> | ... | ... | ... |
>
> 数据来源：szcanvas影视制作数据
> 数据日期（ImpDate）：前中期 20260401 / 后期 20260401
> 统计口径：基于影视前中期素材和后期素材信息，青簪行项目

---

## 影视制作后期进度查询

当用户询问某项目的**影视制作后期进度**时，使用 `ods_agic_zen_aivfx_edite_video_full_df` 表查询。

> ⚠️ 执行前必须先按照上方「**项目名称/ID 查询方式**」获取项目基础信息（szbot_project_id、szbot_project_name 等），再用于后续 SQL 过滤。

### 进度指标定义

| 指标 | 含义 | 计算方式 |
|------|------|---------|
| 视频总量 | 该项目累计生成的视频数 | `COUNT(DISTINCT asset_id)` |
| 被下载总量 | 该项目已被下载的视频数 | `COUNT(DISTINCT asset_id) WHERE has_downloaded = 1` |
| 视频每天增量 | 按 `create_date` 分组的每日新增视频数 | `COUNT(DISTINCT asset_id) GROUP BY create_date` |
| 被下载每天增量 | 按 `create_date` 分组的每日新增被下载视频数 | `COUNT(DISTINCT asset_id) WHERE has_downloaded = 1 GROUP BY create_date` |

> 默认给出最近 **5** 个素材的 asset_id 及其播放链接。

---

> 完整端到端示例见上方「**影视制作进度查询（前中期 + 后期）**」章节。

---

## 影视制作前中期进度查询

当用户询问某项目的**影视制作前中期进度**时，使用 `dwd_aigc_zen_zhongqi_df` 表查询。

> ⚠️ 执行前必须先按照上方「**项目名称/ID 查询方式**」获取项目基础信息（szbot_project_id、szbot_project_name 等），再用于后续 SQL 过滤。

### 进度指标定义

| 指标 | 含义 | 计算方式 |
|------|------|---------|
| 素材总量 | 该项目累计生成的前中期素材数 | `COUNT(DISTINCT asset_id)` |
| 素材每天增量 | 按 `create_date` 分组的每日新增素材数 | `COUNT(DISTINCT asset_id) GROUP BY create_date` |

> 默认给出最近 **5** 个素材的 asset_id 及其播放链接。

---

> 完整端到端示例见上方「**影视制作进度查询（前中期 + 后期）**」章节。