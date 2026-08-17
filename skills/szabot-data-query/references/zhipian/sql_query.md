# 制片数据查询流程

## 🚨 执行流程总则

**Step 1 表路由** → **Step 2 读取 schema** → **Step 3 执行 SQL** → **Step 4 结果展现**，严格按序，**禁止跳步、禁止少步、禁止合并**，四步必须全部执行，违反即作废重来。

> 本流程为**与具体表无关的骨架**：所有字段、SQL 模板、展现模板均下沉到各表 schema 文件（`references/zhipian/schema/<表名>.md`），新增表只需新增 schema 文件并在下方路由表追加一行。

---

## Step 1 · 表路由选择

🚨 **前置已就绪**：项目ID 已由前置 `kb_search` 工具注入上下文，可直接代入 SQL 模板的 `{项目ID}` 占位符。**禁止**用 SQL 反查 项目ID。

### 表选择路由表

| 用户查询需求 | 命中表 | db_name |
|---|---|---|
| 制片概览：拍摄进度、预算执行进度、角色进度、进度异常、延期情况、超支科目等 | `dws_szbot_zhipian_overview_hf` | `zhipian` |


---

## Step 2 · 读取 schema（强制）

强制完整读取 Step 1 命中表对应的 schema 文件 `references/zhipian/schema/{Step 1 命中的表名}.md`，该文件包含 **DDL / 口径规则 / SQL 模板 / 展现模板**，**Step 3 / Step 4 全部依赖此文件**。

🚨 SQL 中每一个字段都必须能在 schema 文件 DDL 中逐字搜到。禁止 `SELECT *`、禁止凭经验补字段、禁止用 `DESCRIBE` / `SHOW COLUMNS` / `INFORMATION_SCHEMA` 探测结构。

---

## Step 3 · 执行 SQL

按 schema **【SQL 模板】** 节选用对应模板，代入占位符直接执行。

🚨 禁止改写已下沉到 SQL 的逻辑（百分比换算、单位换算、分区过滤、字段别名等），如需调整请先回到 schema 修改模板而非临场改写。

---

## Step 4 · 结果展现

按 schema **【展现模板】** 节套用对应展现模板输出。不要总结，直接展现相关数据即可。

🚨 JSON 数组字段一律按 schema **【口径规则】** 中关于 JSON 数组的统一条款解析，禁止原样吐文本、禁止排序去重。