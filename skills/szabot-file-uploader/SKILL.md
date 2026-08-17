---
name: szabot-file-uploader
namespace: szbot
trust-level: builtin
category: file
description: 影库文件系统底层上传工具，仅负责将本地文件上传到影库文件系统并返回 fid（文件唯一标识）。本技能只做上传，不做解析、不做项目关联。触发词：上传文件到影库、获取fid。
---

# Szbot File Uploader

## 概述

这个技能专门负责将用户的本地文件上传到影库文件系统，获取系统生成的 `fid`（文件唯一标识）。`fid` 是后续所有文件操作的必要凭证，包括文件解析（`szabot-file-parser`）和草稿关联文件（`szabot-project`）。

支持**任意类型文件**上传，无文件格式限制。

优先使用内置脚本：

- `scripts/upload_file.py` — 文件上传主脚本
- `scripts/api_client.py` — 公共 HTTP 调用模块

## 触发条件

满足以下**任一**条件时使用该技能：

- 用户明确要求上传文件到影库系统
- 用户提供了文件，且后续需要获取 `fid` 用于影库系统操作
- 其他影库工作流（如文件解析、草稿关联文件）需要先获取 `fid`，但上下文中尚无合法 `fid`
- 用户提到"上传"、"上传文件"、"上传到影库"等关键词

## 不适用场景

以下情况不要触发这个技能：

- 上下文中已有通过上传获取的合法 `fid`，无需重复上传

## 工作流

> 📖 详细步骤参见 [`references/file_upload_workflow.md`](references/file_upload_workflow.md)

**输入**：本地文件路径（绝对路径）
**输出**：JSON（包含 `fid` 和文件元信息）

核心流程：
1. 获取并校验本地文件路径（确保文件存在）
2. 计算文件大小、MD5、mimeType
3. 调用 `PUT /ai/file/upload` 上传文件
4. 返回包含 `fid` 的结构化 JSON 结果

### 快速使用

```bash
cd {skill_base_dir}/scripts && python3 upload_file.py /absolute/path/to/file.pdf
```

> ⚠️ 文件路径必须使用**绝对路径**。

## 全局约束

1. **fid 是系统生成的**：`fid` 由上传接口返回，是系统自动生成的唯一标识。**严禁使用用户口头提供、手动输入或凭空编造的 fid**。
2. **支持任意文件类型**：对文件格式没有限制，任何文件都可以上传到影库文件系统。
3. **上传结果必须保留在上下文中**：上传成功后的 `fid` 和文件元信息必须保存在对话上下文中，供后续工作流（文件解析、草稿关联等）直接使用。

## 输出约定

- 标准输出（stdout）：固定输出 JSON，包含 `fid` 和文件元信息
- 标准错误（stderr）：输出关键过程日志
- 脚本退出码：`0` = 成功，`1` = 文件不存在，`2` = 其他错误



## 与其他技能的衔接

| 下游技能 | 用途 | 使用的字段 |
|---------|------|-----------|
| `szabot-file-parser` | 文件解析，提取结构化内容用于项目创建 | `fid` |
| `szabot-project`（草稿关联文件） | 将文件关联到项目草稿 | `fid` 作为 `file_id` |

> ⚠️ 下游技能中使用的 `file_id` / `fid` **必须来自本技能上传流程的返回结果**，不可使用用户自行提供的 ID。

## 资源说明

### `scripts/upload_file.py`

文件上传主脚本，输入本地文件路径，校验文件后调用上传接口，输出 `fid` 与文件元信息。

### `scripts/api_client.py`

公共 HTTP 调用与数据处理工具模块，提供 MIME 类型推断、MD5 计算、上传接口调用等基础能力。

### `references/file_upload_workflow.md`

文件上传工作流的详细步骤说明，包含完整的 3 步执行流程（获取文件路径 → 执行上传 → 结果处理）。
