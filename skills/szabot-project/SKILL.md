---
name: szabot-project
namespace: szbot
trust-level: builtin
category: write-ops
description: "影库项目、草稿、文件的管理工具。当用户提到创建草稿、删除草稿、修改草稿、创建项目、修改项目、草稿文件、项目附件、关联文件到项目时使用。"
---

# 影库项目管理 Skill

管理影库项目草稿的全生命周期（查看、创建、修改、删除草稿及草稿文件管理），更新正式影库项目信息，以及将文件关联到正式项目。

## 核心概念

> references对应的完整路径为`/app/resources/skills/szabot/szabot-project/references`

| 概念 | 定义 | 支持操作 |
|-----|------|---------|
| **草稿** | 项目的草稿状态，仅创建人可见，草稿可以发布为正式项目 | 查看、创建、修改、删除、文件管理（添加/查看/移除） |
| **草稿文件** | 关联到草稿的附件，与草稿生命周期绑定，草稿发布后随之转为项目文件 | 添加、查看、移除 |
| **项目** | 电视剧、综艺、动漫类型的剧集，例如"逐玉"、"魔力歌先生"。项目仅支持在 web 端/小程序端的草稿页面提交创建 | 查询、修改、添加文件 |
| **项目文件** | 电视剧项目制作过程产生的文件，包括原著小说、剧本、项目策划案、服化造方案等 | 添加 |

## 快速开始

```bash
# 验证环境变量
env | grep TAI_IT_TOKEN

# 验证 MCP 服务
mcporter list szbotprojectdraft
mcporter list szbotprojectformal
mcporter list szbotprojectfile
mcporter list szabot_tools
```

## 功能清单

| MCP Server | 能力 |
|-----------|------|
| `szbotprojectdraft` | 草稿 CRUD、草稿文件管理（添加/查看/移除）、艺人/公司/IP 查询 |
| `szbotprojectformal` | 正式项目搜索、部分更新 |
| `szbotprojectfile` | 正式项目文件关联（`CreateProjectFile`） |
| `szabot_tools` | 项目检索（`kb_search`）、艺人查询（`talent_simple_query`）、台账/IP 查询（`search_ip`）、员工信息查询（`getStaffBaseInfo`）、公司查询（`company_search`） |

支持的内容类型：`2` — 电视剧


## MCP 调用规范

> 🔴 **强制要求**：所有 MCP 调用必须通过 `mcporter` CLI 的 `--args` 参数提交，**严禁**使用 `mcp_call_tool` 等其他方式。
>
> ```bash
> mcporter call <server>.<tool> --args '<JSON参数>'
> ```

> 📖 **必须加载**：执行任何 MCP 调用前，先读取以下文档：
> 1. `references/mcporter_examples.md` — 完整调用示例和规范
> 2. `references/mcp_param_rules.md` — **参数构造规范与自检清单**（枚举值、序列化规则、类型约束、常见错误）

<!-- LOAD references/mcporter_examples.md -->
<!-- LOAD references/mcp_param_rules.md -->

## 意图识别

| 用户意图关键词 | 工作流 |
|--------------|--------|
| 查看、列表、详情、有哪些草稿 | `references/draft_view_workflow.md` |
| 创建、新建、录入、提交草稿 | `references/draft_create_workflow.md` |
| 修改草稿、更新草稿、改一下草稿 | `references/draft_update_workflow.md` |
| 删除草稿、废弃、不要了 | `references/draft_delete_workflow.md` |
| 草稿文件操作（见下方子分发表） | ↓ |
| 修改项目、更新项目、项目XX改成YY | `references/project_update_workflow.md` |
| 给项目添加文件、关联文件到项目、上传文件到项目、项目附件 | `references/project_add_files_workflow.md` |
| 创建项目、新建项目、创建影库项目 | ⛔ 不支持直接创建项目，提示用户：项目仅支持在 web 端/小程序端通过草稿提交创建，引导用户改为**创建草稿** |

**草稿文件操作子分发：**

| 子操作关键词 | 工作流 |
|------------|--------|
| 添加文件、关联文件、上传文件到草稿 | `references/draft_add_files_workflow.md` |
| 查看草稿文件、文件列表、草稿附件 | `references/draft_view_files_workflow.md` |
| 移除文件、删除草稿文件、取消文件关联 | `references/draft_remove_files_workflow.md` |

**歧义消解规则：**
- 用户仅说"修改/更新"但未明确是"草稿"还是"项目" → 询问确认
- 用户仅说"删除/移除"但未明确是"删除草稿"还是"移除草稿文件" → 询问确认

## 不适用场景

⛔ 代码编写、文件操作、通用问答、其他项目管理系统（TAPD、Jira、飞书项目等）
