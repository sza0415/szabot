---
name: szabot-file-parser
namespace: szbot
trust-level: builtin
category: file
description: "影库文件解析：将 word（doc/docx）、ppt（ppt/pptx）、pdf 文件内容提取为文本数据，用于结构化信息提取等。当用户提到解析文件、提取文件内容、从文件提取信息、解析word、解析ppt、解析pdf、文件内容识别、读取文件信息、文件结构化时使用。"
---

# Szbot File Parser

## 概述

这个技能用于影库场景下的项目创建前置解析，适用于 `word`、`ppt`、`pdf` 三类文件。流程被拆成两个独立动作：**先通过 `szabot-file-uploader` 技能上传文件获取 `fid`，再执行 `file_parse` 基于 `fid` 创建解析任务并轮询结果**。

内置脚本：

- `scripts/file_parse.py` — 文件解析主脚本

> ⚠️ 文件上传能力由 **`szabot-file-uploader`** 技能提供，本技能不包含上传脚本。

## 触发条件

仅在以下条件同时满足时使用该技能：

- 用户提供的是影库相关文件，或明确说明要上传影库文件
- 文件类型属于 `doc`、`docx`、`ppt`、`pptx`、`pdf`

## 工作流

### 1. 上传文件获取 `fid`（使用 `szabot-file-uploader` 技能）

> ⚠️ 本步骤**必须使用 `szabot-file-uploader` 技能**完成，不要尝试手动调用上传接口。

- 先校验文件类型是否属于可解析类型（`doc`、`docx`、`ppt`、`pptx`、`pdf`）
- 调用 `szabot-file-uploader` 技能上传文件，获取返回的 `fid`
- `fid` 将作为下一步解析任务的输入

### 2. 执行 `file_parse`

- 输入上一步返回的 `fid`
- 调用 `POST /ai/file/createTask` 创建解析任务，返回 `taskId`
- 按固定 **5 秒一次** 的间隔调用 `POST /ai/file/getTask` 轮询任务状态
- 任务通常需要 **1 ~ 3 分钟** 完成，轮询过程中会定期输出等待提示；超过 **5 分钟** 自动终止
- 输出结构化 JSON 解析结果

## 快速使用

在 `szabot-file-parser` 目录下执行：

### 先上传文件（使用 szabot-file-uploader 技能）

```bash
cd {szabot-file-uploader_skill_base_dir}/scripts && python3 upload_file.py /absolute/path/to/file.pdf
```

> 上传成功后会返回 JSON，其中包含 `fid`。

### 再解析文件

```bash
cd {szabot-file-parser_skill_base_dir}/scripts && python3 file_parse.py fid_xxxxxxxx
```

如果需要调整最大轮询次数：

```bash
python3 ./scripts/file_parse.py fid_xxxxxxxx --max-polls 8
```

## 输出约定

- `file_parse` 的标准输出固定输出 JSON，包含 `taskId`、状态和解析结果
- `file_parse` 轮询间隔固定为 **5 秒**，不接受外部传参覆盖
- `file_parse` 总等待时间超过 **5 分钟**（300 秒）后自动终止并返回超时状态
- `file_parse` 任务通常需要 1 ~ 3 分钟完成，轮询期间会定期输出等待提示，大模型应耐心等待而非中断
- `file_parse` 达到预设轮询次数后不会直接失败，而是继续轮询并输出进度日志
- 脚本会把关键过程日志输出到标准错误

## 依赖技能

| 技能 | 用途 | 阶段 |
|------|------|------|
| `szabot-file-uploader` | 上传文件获取 `fid` | Step 1（上传） |

> ⚠️ 执行本技能前，必须确保 `szabot-file-uploader` 可用。上传返回的 `fid` 是唯一合法的文件标识，**严禁使用用户手动提供或凭空编造的 fid**。

## 资源说明

### `scripts/file_parse.py`

负责文件解析这一步，输入 `fid`，调用真实任务接口并轮询结果，适合作为影库项目创建链路里的解析动作。

### `scripts/api_client.py`

公共 HTTP 调用与数据处理工具模块，提供解析任务创建（`createTask`）和轮询（`getTask`）的接口调用能力。
