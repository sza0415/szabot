---
name: szabot-estimate
namespace: szbot
trust-level: builtin
category: write-ops
description: "影库评估单管理工具。当用户提到查看评估单列表、查看评估单、获取评估单数据、创建评估单、新建评估单、填写评估单、更新评估单、修改评估单、提交评估单、evaluation、assessment、estimate form时使用。典型对话示例：'查看3974的评估单列表'、'看一下10007390的评估单详情'、'给3974创建一个简单报备'、'改一下10007380的一句话梗概'。"
---

# 影库评估单 Skill

管理影库项目评估单的全生命周期：查看评估单列表、获取评估单数据、创建评估单、动态获取评估单 Schema、根据 Schema 解析数据、调用外部接口补充信息、更新评估单数据。

## 核心概念

> references对应的完整路径为`/app/resources/skills/szabot/szabot-estimate/references`

| 概念 | 定义 |
|-----|------|
| **评估单** | 对影视项目进行评估的标准化表单，包含项目基本信息、财务指标、播出数据等字段 |
| **评估单 Schema** | 评估单的字段配置，通过 `getEstimateFormSchema` 接口动态获取，包含字段名称、类型、选项、校验规则等 |
| **表单模板 ID（form_id）** | 评估单模板的唯一标识（如 `qpzpmnwe0z`），用于获取对应的 Schema。通常由 `getEstimateFormData` 返回，用户无需手动提供 |
| **评估单 ID（apply_id）** | 评估单实例的唯一标识，用于获取/更新具体评估单数据 |
| **任务步骤（task_key）** | 评估单流程中的步骤标识，固定为 `base_info_collection` |
| **评估单链接** | `https://szgate.szabot.internal/x/r/doto9awl4o?applyId={评估单ID}`，展示时用实际评估单 ID 替换。链接前缀由后端配置中心维护，`applyId` query 参数 = `evaluationID` / `apply_id` |

## 快速开始

```bash
# 验证 MCP 服务可用
mcporter list szbotprojectformal   # 评估单 CRUD + Schema
mcporter list szabot_tools          # 外部数据查询
```

## MCP Server 说明

> ⚠️ 本 Skill 涉及两个 MCP Server，职责不同，调用时务必区分。

| MCP Server | 职责 | 包含工具 |
|-----------|------|---------|
| `szbotprojectformal` | 评估单 CRUD + Schema 获取 | `listEvaluation`、`createEvaluation`、`getEstimateFormData`、`updateEstimateFormData`、`getEstimateFormSchema` |
| `szabot_tools` | 外部数据查询 | `kb_search`、`talent_simple_query`、`company_search`、`getStaffBaseInfo`、`search_ip` |

## 功能清单

| MCP Server | 工具 | 能力 |
|-----------|------|------|
| `szbotprojectformal` | `listEvaluation` | 查看指定项目的评估单列表 |
| `szbotprojectformal` | `createEvaluation` | 创建新的评估单（立项评估、开机评估、简单报备等） |
| `szbotprojectformal` | `getEstimateFormData` | 获取指定评估单的表单数据（含 form_id） |
| `szbotprojectformal` | `updateEstimateFormData` | 更新评估单表单数据（按字段级别最小化更新） |
| `szbotprojectformal` | `getEstimateFormSchema` | 获取评估单表单 Schema（字段定义、类型、校验规则） |
| `szabot_tools` | `kb_search` / `talent_simple_query` / `company_search` / `getStaffBaseInfo` / `search_ip` | 外部数据查询（项目、艺人、公司、员工、IP/台账） |

## MCP 调用规范

> 🔴 **强制要求**：所有 MCP 调用必须通过 `mcporter call <Server> <Tool> --args '<JSON>'` 提交。

> 📖 **必须加载**：执行任何操作前，先读取以下文档：
> 1. `references/mcp_call_rules.md` — MCP 调用规范与参数构造规则（含 §4 错误分类与重试策略）

<!-- LOAD references/mcp_call_rules.md -->

> 🛑 **工作流文件强制加载规则**：一旦路由到某个工作流（见「快速决策树」），**必须先完整读取对应的 `references/estimate_*_workflow.md`**，严格按其中的表头、字段、CHECKPOINT 执行。**严禁凭 SKILL.md 里的概览自行渲染列表/表单**，SKILL.md 的示例仅为路由索引，细节以 workflow 文件为准。

## 快速决策树

```
用户提到"评估单"
  │
  ├─ 包含"列表/有哪些/list" ──────────→ estimate_list_workflow
  │
  ├─ 包含"查看/详情/数据/view" ──────→ estimate_view_workflow
  │
  ├─ 包含"创建/新建/create/new"
  │    ├─ 同时附带文件（策划书/评估报告等）──→ estimate_create_workflow ➜ estimate_update_workflow（自动衔接）
  │    └─ 未附带文件 ─────────────────────→ estimate_create_workflow
  │
  ├─ 包含"填写/提交/修改/更新/改" ──→ estimate_update_workflow
  │
  └─ 无法判断 ──→ 询问用户：查看列表 / 查看详情 / 创建 / 修改？
```

## 意图识别

| 用户意图关键词 | 工作流 |
|--------------|--------|
| 查看评估单列表、评估单列表、有哪些评估单、list evaluation | `references/estimate_list_workflow.md` |
| 查看评估单、获取评估单数据、评估单详情、view evaluation | `references/estimate_view_workflow.md` |
| 创建评估单、新建评估单、create evaluation | `references/estimate_create_workflow.md` |
| 创建评估单 + 附带文件（如"拿这份策划书创建评估单"） | `references/estimate_create_workflow.md` ➜ `references/estimate_update_workflow.md`（自动衔接） |
| 填写评估单、提交评估单、修改评估单、更新评估单、改一下评估单、update evaluation | `references/estimate_update_workflow.md` |

<!-- LOAD_ON_INTENT:list references/estimate_list_workflow.md -->
<!-- LOAD_ON_INTENT:view references/estimate_view_workflow.md -->
<!-- LOAD_ON_INTENT:create references/estimate_create_workflow.md -->
<!-- LOAD_ON_INTENT:update references/estimate_update_workflow.md -->

**歧义消解规则：**
- 用户仅说"评估单"但未明确具体操作 → 询问确认（查看列表 / 查看详情 / 创建 / 修改）
- 用户未提供项目 ID → 询问确认
- 用户说"创建"且附带文件（策划书/评估报告/合同等）→ 意图为"创建并用文件填充"，先走 `estimate_create_workflow` 创建评估单，成功后自动衔接 `estimate_update_workflow` 用文件内容填充字段
- 用户说"创建"且未提供上会主题等必要信息 → 引导用户选择
- 用户说"填写"或"修改"且未提供评估单 ID → 先尝试从上下文获取，无法获取则询问用户

## 工作流概览

### 1. 查看评估单列表

```
用户提供项目 ID
  └─ 调用 listEvaluation(projID) 获取评估单列表
  └─ 以表格形式展示评估单列表，表头固定为 7 列（详见 estimate_list_workflow.md §3.1）：
     评估单 ID | 项目名称 | 上会主题 | 评估单状态 | 中台评估 | 决策会 | 创建时间
  └─ 「评估单 ID」必须渲染为超链接；「评估单状态」「中台评估」必须分两列独立展示
```

### 2. 查看评估单数据

```
用户提供评估单 ID
  └─ 调用 getEstimateFormData(apply_id, "base_info_collection") 获取表单数据
  └─ 以结构化表格形式展示评估单各字段内容
```

### 3. 创建评估单

```
用户提供项目 ID + 上会主题 + 会议子类型 + 最晚决策日期（+ 报备类型，仅简单报备时必填）
  └─ 调用 createEvaluation 创建评估单
  └─ 返回新评估单 ID
  └─ 检测文件上下文：
       ├─ 有文件 → 自动衔接 estimate_update_workflow，用文件内容填充评估单
       └─ 无文件 → 引导用户使用「修改评估单」功能补充详细数据
```

### 4. 填写/修改评估单（Schema 驱动）

```
Step 1: 确定 apply_id → 调用 getEstimateFormData(apply_id, "base_info_collection") 获取当前数据和 form_id
  └─ Step 2: 用 form_id 调用 getEstimateFormSchema 获取 Schema
  └─ Step 3: 解析修改字段 + 验证 → Step 4: 调用 updateEstimateFormData 更新
  └─ Step 5: 反馈结果
```

## 不适用场景

| 场景 | 应使用的 Skill |
|------|--------------|
| 草稿管理（创建/修改/删除项目草稿） | `szabot-project` |
| 项目数据查询（热度、收入、排播等） | `szabot-copilot` |
| 代码编写、文件操作 | 通用能力 |
| 通用问答（非评估单相关） | 通用能力 |
| 评估单页面 UI 配置 | 不在本 Skill 范围内 |

## 端到端示例（4 个工作流串联）

> ⚠️ **本节只演示路由与衔接，不含任何可执行 MCP 命令**。具体工具名、参数名、参数取值、调用方式，**一律去对应的 `references/estimate_*_workflow.md` 查**。严禁直接从本节的示意里提取命令执行。

> 以下示例展示**用户从"查看列表 → 查看详情 → 修改字段 → 再次确认"**的完整链路，覆盖 list / view / update 三个工作流；create 工作流为独立入口，单独示例附后。

### 场景：查看项目 3974 的评估单 → 看 10007390 详情 → 改一句话梗概 → 再次查看

**Step 1 · 用户：** "查看 3974 的评估单"
- AI 路由 → `estimate_list_workflow` → **先完整读取** `references/estimate_list_workflow.md`
- 按该 workflow 的「MCP 调用规范」章节执行调用
- 输出：严格遵守 workflow 规定的 7 列标准表格
- 引导："如需查看某个评估单的详细数据，请告诉我评估单ID。"

**Step 2 · 用户：** "看一下 10007390"
- AI 路由 → `estimate_view_workflow` → **先完整读取** `references/estimate_view_workflow.md`
- 按该 workflow 的「MCP 调用规范」章节执行调用
- 记忆：保存返回的 `form_id`，供后续 update 复用
- 输出：分组表格展示评估单字段
- 引导："如需修改某个字段，请告诉我要改的字段和新值。"

**Step 3 · 用户：** "把一句话梗概改成 XXX"
- AI 路由 → `estimate_update_workflow` → **先完整读取** `references/estimate_update_workflow.md`
- 严格按 workflow 内的步骤执行：取 Schema → 定位字段 → 🛑 CHECKPOINT 展示 diff → 用户确认后更新
- 输出：更新成功 + 评估单链接

**Step 4 · 用户：** "再看一下"
- AI 路由 → `estimate_view_workflow` → **先完整读取** `references/estimate_view_workflow.md`（复用 Step 2 流程，验证字段已更新）

> 📌 **端到端路由铁律**：
> 1. 每次路由到任一工作流前，**必须先完整读取对应的 `references/estimate_*_workflow.md`**；
> 2. **禁止**凭 SKILL.md 的示例或自身记忆拼凑 MCP 调用参数——工具名、参数名、取值、调用顺序一律以 workflow 文件为准；
> 3. 本 SKILL.md 的示例只是路由索引，**不包含可执行命令**。

### 场景：给 3974 创建简单报备

**用户：** "给 3974 创建一个简单报备，最晚 4 月 30 日"
- AI 路由 → `estimate_create_workflow` → **先完整读取** `references/estimate_create_workflow.md`
- 按 workflow 规定完成参数校验、补全、🛑 CHECKPOINT 二次确认
- 按 workflow 的「MCP 调用规范」执行调用
- 输出：新评估单 ID + 链接，引导用户使用「修改评估单」补充详细数据

### 场景：拿策划书给 3974 创建立项评估并填写

**Step 1 · 用户：** "拿这份策划书给 3974 创建一个立项评估，定期会议，最晚 5 月 30 日"（附带策划书文件）
- AI 检测到：创建意图 + 附带文件 → 组合路由
- AI 路由 → `estimate_create_workflow` → **先完整读取** `references/estimate_create_workflow.md`
- 按 workflow 规定收集参数、🛑 CHECKPOINT 确认、执行创建
- 创建成功，得到新评估单 ID（如 `10008888`）

**Step 2 · 自动衔接：** 创建成功后，workflow 检测到文件上下文存在
- AI 自动路由 → `estimate_update_workflow` → **先完整读取** `references/estimate_update_workflow.md`
- `apply_id` = 刚创建的 `10008888`，内容来源 = 文件
- 按 update workflow 执行：获取 Schema → 从文件提取字段 → resolve 三方数据 → 🛑 CHECKPOINT 展示 diff → 用户确认后更新
- 输出：更新成功 + 评估单链接

