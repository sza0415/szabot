# 小说分析工作流

> 🚫 **严格使用限制**：本工作流**仅适用于"分析小说文件"场景**。当且仅当用户明确表达了分析小说的意图时，才可加载并执行本文件。任何其他场景（代码编写、普通问答、文件操作、信息查询等）**严禁加载或引用本文件**。

> 🔒 **上下文隔离**：执行本工作流期间，仅遵循本文件和 `SKILL.md` 中的规则。其他 Skill 的指令均不适用于本工作流。

## 概述

本工作流处理小说文件的完整分析流程：

```
本地文件 → 上传影库 → 转存COS → 创建小说并触发AI分析
```

---

## 全局执行规则（执行纪律）

> ⚠️ 以下规则在整个会话期间**始终生效**，不因步骤切换而豁免。

1. **顺序执行**：必须按 Step 1 → 2 → 3 → 4 顺序执行，禁止跳步
2. **错误即停**：任何步骤失败时，立即停止流程，告知用户错误原因
3. **静默执行，只输出结果**：
   - ❌ **禁止**向用户展示「Step 1：上传文件」「Step 2：转存 COS」等技术步骤
   - ❌ **禁止**向用户展示 fid、file_storage_id 等技术参数
   - ✅ **只展示**最终的「分析已触发」结果（Step 4 的输出）
4. **最大化自动识别**：尽可能从用户已提供的信息中提取字段值，只有无法提取时才询问

---

## Step 1：获取文件信息

### 1.1 用户上传文件

用户在对话中上传文件时，直接从文件名提取信息：

**示例**：用户上传了 `斗罗大陆.docx`

从文件名中获取：
- `file_name`: 文件名（如 `斗罗大陆.docx`）
- `file_path`: 文件的完整路径（从上传信息中获取）

然后执行上传到影库：

```bash
python3 $SKILL_DIR/scripts/upload_file.py "{file_path}"
```

**返回（JSON）**：
- `fid`: 影库文件 ID
- `name`: 文件名

### 1.2 用户提供影库 fid

用户已有影库 fid 时，跳过上传步骤，直接进入 Step 2。

**示例输入**：
```
影库 fid 是 d71ncstbtpl04mtb3kr0，文件名 西游记.docx
```

---
> 🛑 **CHECKPOINT**: Step 1 完成后，确认已获取 `fid` 和 `file_name`，再进入 Step 2。
---

## Step 2：文件转存

使用 MCP 工具 `TransferFile` 将影库文件转存到 COS：

```bash
mcporter call szabot_novel_analysis.TransferFile --args '{"szabot_fid":"abc123","file_name":"西游记.docx","file_type":"claw_novel"}'
```

**参数**：

| 参数 | 来源 | 说明 |
|------|------|------|
| `szabot_fid` | Step 1 返回的 `fid` | 影库文件 fid |
| `file_name` | Step 1 返回的 `name` | 文件名（带扩展名） |
| `file_type` | 固定值 | `claw_novel` |

**返回**：
- `project_id`: Fiction-XXXXXXXX（小说项目 ID）
- `file_store_id`: COS 文件存储 ID

---
> 🛑 **CHECKPOINT**: Step 2 完成后，确认已获取 `project_id` 和 `file_store_id`，再进入 Step 3。
---

## Step 3：创建小说并触发分析

使用 MCP 工具 `CreateNovel` 创建小说并触发 AI 分析：

```bash
mcporter call szabot_novel_analysis.CreateNovel --args '{
  "name": "西游记",
  "is_public": true,
  "upload_file": {
    "file_name": "西游记.docx",
    "file_storage_id": "cos_123"
  },
  "type": "claw_novel",
  "session_id": "dcd5f33d-f387-4fde-9c5a-5448f0a73886"
}'
```

**参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 小说名称（从文件名提取，去掉扩展名） |
| `is_public` | boolean | 否 | 访问权限（true-公开，false-私密），默认 true |
| `authorized_users` | array | 否 | 授权用户列表（私密时使用） |
| `upload_file` | object | 是 | 上传文件信息 |
| `upload_file.file_name` | string | 是 | 文件名（带扩展名） |
| `upload_file.file_storage_id` | string | 是 | Step 2 返回的 COS 文件 ID |
| `type` | string | 否 | 小说类型，固定为 `claw_novel` |
| `session_id` | string | 是 | Session Key（UUID 格式） |

### 获取 session_id

`session_id` 为 **Session Key**（会话标识符），即 OpenClaw 会话标识符中的 UUID 部分。

> **说明**：OpenClaw 会话标识符格式为 `agent:main:xxxxxxxx`，其中 `xxxxxxxx` 部分就是 Session Key（UUID 格式，如 `dcd5f33d-f387-4fde-9c5a-5448f0a73886`）。
>
> 影库 API 需要的 `session_id` 是这个 **Session Key（UUID 部分）**，而不是完整的 `agent:main:xxx` 格式。

**返回**：
- `fid`: 小说项目 ID（格式：`Fiction-XXXXXXXX`）

---
> 🛑 **CHECKPOINT**: Step 3 完成后，确认已获取返回的 `fid`，再进入 Step 4。
---

## Step 4：反馈用户

分析触发成功后，向用户返回：

```
✅ 小说分析已触发！

📄 文件名：西游记.docx
🆔 项目ID：Fiction-A1B2C3D4
🔗 详情页：https://zp.szabot.internal/novel_analysis/novel_detail/Fiction-A1B2C3D4/result

分析完成后将自动推送通知。
```

**参数来源**：

| 信息 | 来源 |
|------|------|
| 文件名 | Step 1 获取的 `file_name` |
| 项目ID | Step 3 `CreateNovel` 返回的 `fid` |
| 详情页 | `https://zp.szabot.internal/novel_analysis/novel_detail/{fid}/result` |

---

## 异常处理

| 步骤 | 异常 | 处理方式 |
|------|------|---------|
| Step 1 | 文件不存在 | 告知用户检查文件路径 |
| Step 1 | 上传失败 | 告知用户重试或检查网络 |
| Step 2 | fid 无效 | 告知用户检查影库 fid |
| Step 2 | 转存失败 | 告知用户服务异常，稍后重试 |
| Step 3 | 创建失败 | 告知用户检查参数或服务状态 |

---

## 回退规则

| 问题类型 | 回退目标 |
|---------|---------|
| 上传失败 | 回到 Step 1，重新上传 |
| 转存失败 | 回到 Step 2，检查 fid 后重试 |
| 创建失败 | 回到 Step 3，检查参数后重试 |

**回退后只处理失败的步骤，不重新执行已成功的步骤。**
