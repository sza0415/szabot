---
name: szpp-data-query
namespace: szbot
trust-level: builtin
category: data-query
version: 2.0.0
description: "侵权数据查询（仅限侵权相关）。仅在用户查询涉及版权侵权相关数据时使用，包括发函量、下架量、下架率、防护时长、侵权趋势等指标。触发词：侵权、发函、下架、防护、侵权趋势。⛔ 不适用：影视综项目信息查询（走 szabot-copilot）、非侵权类的任何数据查询"
---

# 侵权数据查询技能

**仅限版权侵权相关**的数据查询，通过 MCP 服务 `szabot_data_query_svr` 的 `mcp_exec_sql` 工具执行 SQL 查询。本 Skill 的使用范围严格限定于侵权场景（发函、下架、防护、侵权趋势等），不处理影视综项目信息查询（那些走 `szabot-copilot`）或其他非侵权类数据。

## 快速开始

```bash
# 验证 MCP 服务
mcporter list szabot_data_query_svr
```

## 正确的调用方式（重要！）

⚠️ **仅支持 `szpp` 和 `szdw_dim` 两个数据库**。禁止使用其他 db_name。

### 步骤1：获取最新 imp_date

```bash
# 使用 db_name="szdw_dim" 查询 chuku_progress 获取最新分区日期
mcporter call 'szabot_data_query_svr.mcp_exec_sql' --args '{
  "db_name": "szdw_dim",
  "sql": "SELECT MAX(imp_hour) as imp_hour FROM chuku_progress WHERE table_name = '\''miniapp_progress_hour'\''"
}'
```

### 步骤2：执行业务查询

```bash
# 使用 db_name="szpp" 查询侵权数据
mcporter call 'szabot_data_query_svr.mcp_exec_sql' --args '{
  "db_name": "szpp",
  "sql": "SELECT * FROM szpp.t_mini_tort_metrics WHERE imp_date = 2026033000 AND platform = '\''ALL'\'' LIMIT 10"
}'
```

## 常见错误与解决

| 错误信息 | 原因 | 解决方法 |
|---------|------|---------|
| `SQL语句不能为空` | JSON 格式问题或引号转义失败 | 使用 `--args` 参数传递 JSON |
| `invalid ExecSQLReq.DbName: value contains invalid strings` | db_name 不在允许列表 | **侵权数据必须用 `"szpp"`，获取分区日期必须用 `"szdw_dim"`** |

### ⚠️ 关键要点

1. **必须使用 `--args` 参数**传递 JSON，不能用函数式调用
2. **db_name 必须是 `"szpp"` 或 `"szdw_dim"`**，其他格式都会报错
3. 查询 `szdw_dim.chuku_progress` 时用 `"szdw_dim"`，查询业务表时用 `"szpp"`

## 执行流程

1. **加载知识库** — 读取 `references/` 下的口径和表结构
2. **生成 SQL** — 基于 `references/sql_templates.md` 构建查询
3. **执行查询** — 通过 `mcp_exec_sql` 执行 SQL

## 触发场景

| 用户意图 | 使用表 |
|---------|-------|
| 发函量/下架量/下架率/防护时长 | `szpp.t_mini_tort_metrics` |
| 侵权趋势/打击量 | `szpp.t_mini_trend_statistic` |
| IP品类/维权等级/监测状态 | `szpp.t_mini_ipinfo` |

## 核心口径

| 指标 | 计算规则 |
|-----|---------|
| 下架率 | `tort_delete_num / letter_num_no_se` |
| imp_date | 从 `szdw_dim.chuku_progress` 取 `MAX(imp_hour)` where `table_name='miniapp_progress_hour'`，**禁止写死日期** |
| 平台字段 | 必须指定 `platform` 和 `platform_type_name`，默认 `ALL` |
| 趋势查询 | 不看 `platform_type_name = '长尾网站'` |

## 资源文件

| 文件 | 内容 |
|-----|------|
| `references/sql_templates.md` | SQL 模板与场景示例 |
| `references/szpp/*.md` | 侵权数据表结构定义 |
| `references/chuku_schema.md` | 数据分区表结构 |

> 构建 SQL 前必须先读取对应的 references 文件。

## 注意事项

- **MCP Server 名称强约束**：本 Skill 使用的 MCP Server 为 **`szabot_data_query_svr`**，必须以此为准。**严禁猜测 MCP Server 名称，尤其禁止将 Skill 名称（`szpp-data-query`）当作 MCP Server 名称**。判断 MCP Server 是否可用时，必须检查 `szabot_data_query_svr` 的状态。
- 必须通过 `mcp_exec_sql` 执行查询
- 禁止自行猜测计算口径，必须参考 references
- 输出结尾附带执行时间及数据来源
