# 动漫/动画数据查询

## 🚨 执行流程总则

**Step 0 项目ID获取** → **Step 1 表路由** → **Step 2 读取 schema** → **Step 3 执行 SQL** → **Step 4 结果展现**，严格按序，**禁止跳步、禁止少步、禁止合并**，五步必须全部执行，违反即作废重来。

---

## Step 0 · 项目ID获取（前置）

🚨 **强制规则**：用户指定了任何项目条件（项目名称、管理状态等）时，**必须先调用 `szabot-copilot` 技能的 `kb_search` 工具**获取项目ID列表，将返回结果中的「项目ID」字段值直接代入 SQL 模板的 `WHERE pid IN (...)` 过滤。调用方式遵循 `szabot-copilot` 技能规范。

🚫 **禁止**用 SQL 字段条件替代项目过滤。

✅ **仅当用户未指定任何项目条件**（即查询全部项目）时，才跳过本步骤，不在 SQL 中加入项目ID过滤条件（其他 WHERE 条件如时间范围等照常保留）。

---

## Step 1 · 表路由选择

### 表选择路由表

| 意图关键词 | 命中表 | db_name |
|------|--------|---------|
| 算力/开销/消耗/花费/预算/费用/素材数量/模型消耗/下载率/调用次数/活跃用户 | `dws_donghua_zhipian_szcanvas_metrics_df` | `donghua` |

---

## Step 2 · 读取表 schema（强制）

读取表 schema：`references/donghua/schema/{Step 1 命中的表名}.md`，文件中包含**字段定义、口径规则、查询模板**。

🚨 SQL 中每一个字段都必须能在 schema 文件中逐字搜到。**严禁**执行 `DESCRIBE` / `DESC` / `SHOW COLUMNS` / `SHOW CREATE TABLE` / `SHOW TABLES` / 查询 `INFORMATION_SCHEMA` / `SELECT *` 等方式到 DB 反查表结构。若 schema 文件中不存在所需字段，必须直接告知用户"当前 schema 文件中不存在该字段，无法查询"。

---

## Step 3 · 执行 SQL

### MCP 工具调用方式

**MCP**：`szabot_data_query_svr` · `mcp_exec_sql`，入参 `db_name`（本表统一为 `donghua`，禁止猜测）、`sql`。

```bash
# 必须使用 --args 参数传递 JSON，不能用函数式调用
mcporter call 'szabot_data_query_svr.mcp_exec_sql' --args '{
  "db_name": "donghua",
  "sql": "SELECT ... FROM ... WHERE ..."
}'
```

### SQL 模板入口

所有 SQL 模板均在 schema 文件中维护。根据用户问题选择对应模板 → 替换 `{过滤条件}` 占位符 → 执行。

---

## Step 4 · 结果展现

按 schema 文件中「展现规则」章节执行（含通用展现规则、指标格式、翻译与标注）。
