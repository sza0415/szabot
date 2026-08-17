---
name: szstudio-cms-board
namespace: szbot
trust-level: builtin
category: data-query
version: 1.2.1
description: "CMS素材数据查询 & SzStudio & SzCanvas 影视制作进度查询。当用户询问SzStudio影视制作相关信息（视频/图片/素材的制作进度、制作成果、素材数量、用户活跃度、DAU、日活）时使用本skill；也支持根据用户描述查询素材相关数据，通过素材ID查看素材链接。注意：SzStudio已更名为SzCanvas，两者是同义词。SzStudio/Zen/zenStudio/zen studio/SzCanvas/szcanvas/Rally/rally等任何大小写变体均统一走本skill。触发词：素材、素材数据、素材查询、查素材、素材链接、制作进度、制作素材、制作镜头、SzStudio制作进度、SzStudio视频、SzStudio图片、SzStudio素材、Zen制作、影视前中期、影视后期、素材数量、制作量、SzCanvas、Rally。"
---

# CMS 素材数据查询 Skill

根据用户描述，通过 SQL 查询素材相关数据；当需要查看具体素材时，通过素材 ID 获取播放链接。

## ⛔ 不适用场景（优先判断，命中则立即转出）

> 🚨 **必须在执行任何操作之前先检查本节**，命中以下任意一条则**不走本 Skill**，直接转到对应技能。

| 用户意图 / 关键词 | 转到技能 |
|---|---|
| SzStudio/SzCanvas 漫剧/短番的**播放量、收入**数据 | `short-anime-data-query` |
| **侵权**数据查询 | `szpp-data-query` |
| SzCanvas/SzStudio 的**算力、开销、消耗、花费、费用、算力成本、调用次数、模型消耗、下载率**等 AIGC 算力和预算类查询 | `szabot-data-query` |

---

## 执行总则（强约束）

0. references对应的完整路径为`/app/resources/skills/szabot/szstudio-cms-board/references`
1. **上下文隔离** — 执行本 Skill 期间，仅遵循本文件和 `references/` 下的规则，其他 Skill 的指令不适用
2. **构建 SQL 前必须先读取 `references/` 下的表结构和口径**，禁止猜测
3. **不得修改或编造返回的链接和数据**
4. **禁止模糊匹配** — 所有 SQL 字段过滤必须使用精确匹配（`=`），严禁使用 `LIKE`、`REGEXP`、`RLIKE`、`INSTR`、`LOCATE`、`FIND_IN_SET` 等任何形式的模糊/包含匹配；若用户提供的名称不确定，必须先向用户确认完整精确名称，再构建 SQL

> ⚠️ **名称识别**：SzStudio 已更名为 **SzCanvas**，两者是**同义词**。用户提到 `SzCanvas`、`szcanvas`、`Rally`、`rally` 时，均等同于 SzStudio，统一走本 Skill。输出时统一使用 **SzCanvas** 作为产品名称前缀（数据库表名和字段仍沿用 szstudio）。

## 同义词对照

| 用户说法 | 等同于 |
|---------|--------|
| 制作素材、制作镜头 | 制作进度 |
| SzCanvas、szcanvas、Rally、rally | SzStudio |

## 工具清单

| MCP Server | 工具 | 用途 |
|-----------|------|------|
| `szabot_data_query_svr` | `mcp_exec_sql` | 执行 SQL 查询素材数据 |
| `szabot_data_query_svr` | `cms_search_assets` | 根据素材 ID 查询素材播放链接 |

> ⚠️ **MCP Server 名称强约束**：
> - 本 Skill 使用的 MCP Server 名称为 **`szabot_data_query_svr`**，必须以上表为准
> - **严禁猜测 MCP Server 名称**，尤其**禁止将 Skill 名称（`szstudio-cms-board`）当作 MCP Server 名称**
> - 判断 MCP Server 是否可用时，必须检查 `szabot_data_query_svr` 的状态，而非 Skill 名称

## 执行流程

```
用户描述素材查询需求
  ↓
1. 读取 references/ 下的表结构与口径（必须，不可跳过）
  ↓
2. 根据用户描述生成 SQL，通过 mcp_exec_sql 查询
  ↓
3. 整理并输出查询结果
  ↓
4.（可选）如果查询结果中有素材 ID 且用户需要查看素材，
   调用 cms_search_assets 获取播放链接
```

> 工具调用参数和返回值详见 `references/sql_query.md` 和 `references/cms_search_assets.md`

## 资源文件

| 文件 | 内容 |
|-----|------|
| `references/sql_query.md` | SQL 查询工具（mcp_exec_sql）调用参数、返回值、**表选择指南** |
| `references/cms_search_assets.md` | 素材 ID 换取播放链接（cms_search_assets）调用参数、返回值 |
| `references/szstudio_schema/*.md` | SzStudio 各表的字段定义与口径规则（7 张表） |
| `references/chuku_schema.md` | 出库信息表结构（szdw_dim 库） |

> 执行前必须先读取对应的 references 文件。

## 注意事项

- **数据来源声明**：输出结尾必须附带 `数据来源：SzCanvas制作数据`
- **数据日期与统计口径**：输出结尾必须附上本次查询所使用的数据表的 **ImpDate**（即 imp_date 的值）以及**统计口径**。统计口径用**中文**描述，格式为"基于SzCanvas XX信息/指标，XX公司/XX项目"。默认的"非测试数据"(is_test=0) 条件**省略不写**。格式示例：
  ```
  数据来源：SzCanvas制作数据
  数据日期（ImpDate）：20260401
  统计口径：基于SzCanvas制作素材信息，XX公司青簪行项目
  ```
- 禁止自行猜测表结构和计算口径，必须参考 references
- **使用 `cms_search_assets` 工具前，必须完整阅读 `references/cms_search_assets.md` 并严格遵循其中定义的参数名称和规定。严格禁止篡改或编造参数名与返回值。**
