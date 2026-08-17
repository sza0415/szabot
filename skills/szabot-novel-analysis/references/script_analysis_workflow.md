# 剧本分析工作流

> 🚫 **严格使用限制**：本工作流**仅适用于"分析剧本文件"场景**。当且仅当用户明确表达了分析剧本的意图时，才可加载并执行本文件。任何其他场景（代码编写、普通问答、文件操作、信息查询等）**严禁加载或引用本文件**。

> 🔒 **上下文隔离**：执行本工作流期间，仅遵循本文件和 `SKILL.md` 中的规则。其他 Skill 的指令均不适用于本工作流。

---

## ⛔ 禁止事项（必须遵守）

1. **禁止上传后直接调用 file_parse** — 必须先执行：转存COS 
2. **禁止向用户展示 file_parse 结果** — AI摘要/剧情概要仅用于内部预测赛道，**绝对不能输出给用户**
3. **禁止混淆 szabot-file-parser SKILL** — 对于本skills来说是辅助获取赛道，不是剧本分析
4. **禁止向用户展示 Step 执行过程** — 不要输出「Step 1：上传文件」「Step 2：转存 COS」等技术细节
5. **禁止向用户展示技术参数** — fid、file_storage_id、版本名称、项目ID 等内部参数不展示
6. **禁止展示「剧情概要」「摘要」「内部参考」等内容** — 这些是 file_parse 返回的内部数据
7. **禁止展示字数统计** — 不要输出「字数统计：7087 字」，也不要写「长剧（字数≥6000）」，直接写「长剧」或「短剧」
8. **多项目必须分别触发** — 用户选择「分别分析」时，N 个项目必须调用 N 次 `AddScriptInfo`，不能只调用一次
9. **合并项目时禁止硬编码参数** — `belong` 和 `public` 必须从 Step 4 检索结果中获取，禁止硬编码 `public: false`
10. **session_id 不能为空** — 每次调用 `AddScriptInfo` 都必须传有效的 `session_id`（当前会话的 UUID），多项目分别触发时使用同一个 `session_id`
11. **禁止使用数字选项** — 所有用户选择场景**严禁使用数字列表**（如「1. xxx」「2. xxx」「请回复数字选择」），**必须使用文字选项**（如「新建」「合并」「使用建议名称」等），用户回复**文字**而非数字。违反此规则将导致工作流失败！
12. **重名场景必须使用带日期的项目名** — 当 Step 5.1 用户选择「使用建议名称」新建时，`proj_name` **必须**使用 `原项目名_YYYYMMDD` 格式（当前日期），禁止使用原项目名

## 概述

本工作流处理剧本文件的完整分析流程：

```
[Step 1] 上传文件 → 获取 fid + file_name
           ↓
[Step 2] 转存COS → 获取 file_storage_id, dir_id
           ↓
[Step 3] 从文件名提取项目名（如「豪门继承人第1集」→「豪门继承人」）
           ↓
[Step 4] 检索影库项目（同时查长剧+短剧）
           │
           ├── 检索到影库项目 → 询问用户 ─┬─ 合并 → [Step 7]
           │                          │
           │                          └─ 新建 ─┐
           │                                   ↓
           └── 未检索到 ───────────────→ [Step 5]

[Step 5] 检查无项目剧本重名（仅新建时执行）
           │
           ├── 检索到同名无项目剧本 → 询问用户 ─┬─ 合并 → [Step 7]
           │                              │
           │                              └─ 新建 ─┐
           │                                       ↓
           └── 未检索到同名无项目剧本 ─────────────────────→ [Step 6]

[Step 6] 分析内容并确认（仅新建时）
           │
           ├── 短剧（<6000字）→ 用户确认 → [Step 7]
           │
           └── 长剧（≥6000字）→ 预测赛道 + 用户确认 → [Step 7]

[Step 7] 触发剧本分析（AddScriptInfo）
           ↓
[Step 8] 反馈用户
```

**关键流程说明**：
- **合并到已有项目**：直接使用项目的 `belong`、`public`，无需分析字数和赛道
- **新建项目 + 短剧**：统计字数后直接触发，不需要赛道
- **新建项目 + 长剧**：调用 `file_parse` 预测赛道，用户确认后触发

**与小说分析的主要区别**：
- 文件转存时 `file_type` 为 `claw_script`（而非 `claw_novel`）
- 使用 `AddScriptInfo` 触发分析（而非 `TriggerNovelAnalysis`）
- 增加了 Step 5 检查无项目剧本重名（仅新建时执行）

---

## 全局执行规则

> ⚠️ 以下规则在整个会话期间**始终生效**。

1. **顺序执行**：按 Step 1 → 2 → 3 → 4 → (5) → (6) → 7 → 8 顺序执行
   - 合并已有项目：Step 4 后直接进入 Step 7
   - 新建项目：Step 4 → 5 → 6 → 7 → 8
2. **错误即停**：任何步骤失败时，立即停止流程，告知用户错误原因
3. **静默执行**：不向用户展示 Step 编号、技术参数（fid、file_storage_id 等）、file_parse 摘要
4. **最大化自动识别**：尽可能从用户已提供的信息中提取字段值

---

## Step 1：获取文件信息

### 1.1 用户上传文件

用户在对话中上传文件时，直接从文件名提取信息：

**示例**：用户上传了 `豪门继承人第1集.docx`

从文件名中获取：
- `file_name`: 文件名（如 `豪门继承人第1集.docx`）
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
影库 fid 是 d71ncstbtpl04mtb3kr0，文件名 豪门继承人第1集.docx
```

---
> 🛑 **CHECKPOINT**: Step 1 完成后，确认已获取 `fid` 和 `file_name`，再进入 Step 2。
---

## Step 2：文件转存

使用 MCP 工具 `TransferFile` 将影库文件转存到 COS：

```bash
mcporter call szabot_novel_analysis.TransferFile --args '{"szabot_fid":"abc123","file_name":"豪门继承人第1集.docx","file_type":"claw_script"}'
```

**参数**：

| 参数 | 来源 | 说明 |
|------|------|------|
| `szabot_fid` | Step 1 返回的 `fid` | 影库文件 fid |
| `file_name` | Step 1 返回的 `name` | 文件名（带扩展名） |
| `file_type` | 固定值 | `claw_script`（剧本分析专用） |

**返回**：
- `file_store_id`: COS 文件存储 ID
- `dir_id`: 目录 ID（用于 AddScriptInfo 调用）

---
> 🛑 **CHECKPOINT**: Step 2 完成后，确认已获取 `file_store_id` 和 `dir_id`，再进入 Step 3。
---

## Step 3：提取项目名称

从文件名中提取项目名称，按以下规则匹配（优先级从高到低）：

| 文件名模式 | 提取结果 | 示例 |
|-----------|---------|------|
| `xxx第N集` | `xxx` | `豪门继承人第1集.docx` → `豪门继承人` |
| `xxx 前X集` | `xxx` | `jone创建一个项目again 前三集(4).docx` → `jone创建一个项目again` |
| `xxx前X集` | `xxx` | `豪门继承人前三集.docx` → `豪门继承人` |
| `xxxN`（N为数字） | `xxx` | `豪门继承人1.docx` → `豪门继承人` |
| `xxx全集定稿` | `xxx` | `豪门继承人全集定稿.docx` → `豪门继承人` |
| `xxx全集` | `xxx` | `豪门继承人全集.docx` → `豪门继承人` |
| `xx一卡` | `xx` | `豪门继承人一卡.docx` → `豪门继承人` |

> ⚠️ **提取注意事项**：
> - 先移除文件扩展名（`.docx`、`.doc`、`.pdf`、`.txt`）
> - 移除括号及其内容：`(N)`、`（N）`、`(数字)`、`（数字）`
> - 按优先级依次匹配上述模式
> - `前X集` 中的 X 可以是数字（1、2、3）或中文数字（一、二、三）

**提取失败时**：询问用户手动输入项目名称

```
无法从文件名「{file_name}」中自动提取项目名称，请输入项目名称：
```

---
> 🛑 **CHECKPOINT**: Step 3 完成后，确认已获取 `project_name`，再进入 Step 4。
---

## Step 4：检索影库项目

使用 MCP 工具 `GetProjList` 检索影库项目，需要分别查询长剧和短剧：

### 4.0 调用方式

**查询长剧**（belong="1"）：

```bash
mcporter call szabot_novel_analysis.GetProjList --args '{"keyword":"豪门继承人","belong":"1","proj_type":"PROJ_APPROVED","index":0,"page":10,"search_pattern":"FUZZY_MATCH"}'
```

**查询短剧**（belong="4"）：

```bash
mcporter call szabot_novel_analysis.GetProjList --args '{"keyword":"豪门继承人","belong":"4","proj_type":"PROJ_APPROVED","index":0,"page":10,"search_pattern":"FUZZY_MATCH"}'
```

**参数说明**：

| 参数 | 值 | 说明 |
|------|------|------|
| `keyword` | Step 3 提取的项目名 | 搜索关键字（项目名称） |
| `belong` | `"1"` 或 `"4"` | 长剧/短剧类型，**两个都要查** |
| `proj_type` | `"PROJ_APPROVED"` | 项目类型（已立项项目） |
| `index` | `0` | 第几页，从0开始 |
| `page` | `10` | 每页条数 |
| `search_pattern` | `"FUZZY_MATCH"` | 模糊匹配 |

**返回字段**：

| 字段 | 说明 |
|------|------|
| `proj_id` | 项目ID |
| `proj_name` | 项目名称 |
| `belong` | 剧本类型（1=长剧，4=短剧） |
| `producer` | 制片人 |
| `public` | 是否公开 |

**返回结果处理**：

### 4.1 检索到项目（合并长剧+短剧结果）

向用户展示检索结果并让用户选择：

```
检索到以下影库项目：

| 项目ID | 项目名称 | 剧本类型 | 制片人 |
|--------|---------|---------|--------|
| PID-abc123 | 豪门继承人 | 长剧 | 张三 |
| PID-def456 | 豪门继承人之恋 | 短剧 | 李四 |

请选择要合并的项目，或新建项目：
- **新建** - 创建新的无项目剧本
- **豪门继承人** - 合并到「豪门继承人」（长剧，制片人：张三）
- **豪门继承人之恋** - 合并到「豪门继承人之恋」（短剧，制片人：李四）

请回复项目名称或「新建」：
```

**用户输入处理**：
- 用户回复「新建」→ 进入 **Step 5** 检查无项目剧本是否有重名
- 用户回复项目名称（如「豪门继承人」）→ 记录对应的 `proj_id`、`belong`、`producer`、`public`，**直接进入 Step 7**（跳过 Step 6）

### 4.2 未检索到已立项影库项目

当长剧和短剧查询都无结果时，进入 **Step 5** 检查无项目剧本重名。

---
> 🛑 **CHECKPOINT**: Step 4 完成后，根据结果决定下一步：
> - 检索到已立项影库项目且用户选择合并 → 直接进入 Step 7
> - 用户选择新建/未检索到项目 → 进入 Step 5
---

## Step 5：检查无项目剧本重名（新建/未检索到项目时执行）

> ⚠️ **仅当用户选择新建（Step 4 检索到项目但选择新建）或未检索到已立项影库项目时执行此步骤**

查询是否存在同名的无项目剧本：

```bash
mcporter call szabot_novel_analysis.GetProjList --args '{"keyword":"豪门继承人","proj_type":"PROJ_NOT_APPROVED","index":0,"page":10,"search_pattern":"FUZZY_MATCH"}'
```

**结果处理**：

### 5.1 检索到同名无项目剧本

> ⚠️ **严禁使用数字选项**：不能写「1. xxx」「2. xxx」或「请回复数字选择」，必须用文字选项！

向用户展示重名提醒并让用户确认：

```
⚠️ 发现已存在同名的「无项目剧本」：

| 项目ID | 项目名称 | 剧本类型 | 创建人 |
|--------|---------|---------|--------|
| PID-xxx123 | 豪门继承人 | 长剧 | 王五 |

为避免重名，建议使用带日期的项目名：「豪门继承人_20260402」

请选择：
- **使用建议名称** - 项目名：「豪门继承人_20260402」
- **合并** - 合并到「豪门继承人」（无项目剧本，创建人：王五）

请回复「使用建议名称」或「合并」：
```

> ⚠️ **重名场景命名规则（关键）**：
> - 用户选择「使用建议名称」时，`proj_name` 和 `script_version_name` **都必须**使用 `原项目名_YYYYMMDD` 格式（当前日期）
> - 例如：`proj_name = "豪门继承人_20260402"`，`script_version_name = "豪门继承人_20260402"`
> - **禁止**在重名场景下使用原项目名（如 `"豪门继承人"`）作为 `proj_name`

**用户输入处理**：
- 用户回复「使用建议名称」或「新建」或「1」→ 将 `proj_name` **更新为带日期的名称**（如 `豪门继承人_20260402`），进入 **Step 6**
- 用户回复「合并」或「2」或项目名称 → 记录对应的 `proj_id`、`belong`、`producer`、`public`，**直接进入 Step 7**

### 5.2 未检索到同名无项目剧本

告知用户：

```
未检索到名为「{project_name}」的影库项目，将为您创建无项目剧本。
```

进入 **Step 6**（分析内容并确认）。

---
> 🛑 **CHECKPOINT**: Step 5 完成后，根据结果决定下一步：
> - 用户选择合并到无项目剧本 → 直接进入 Step 7
> - 未检索到同名或用户选择使用建议名称 → 进入 Step 6
---

## Step 6：分析内容并确认（仅新建时执行）

> ⚠️ **仅当 Step 4 未检索到项目或用户选择新建时才执行此步骤**
> 如果用户选择合并到已有项目，直接跳到 Step 7。

本步骤需要完成：
1. **统计字数** → 判断长剧/短剧类型（`belong`）
2. **分析题材**（仅长剧）→ 预测赛道（`topic_track2`）

> ⚠️ **短剧不需要赛道**：字数 < 6000 时，只需设置 `belong="4"`，不需要 `topic_track2` 字段

### 6.1 统计字数（判断长剧/短剧）

**调用 MCP 工具 `CountWordsFromCOS` 统计字数**：

```bash
mcporter call szabot_novel_analysis.CountWordsFromCOS --args '{"szabot_fid":"abc123","file_name":"豪门继承人第1集.docx"}'
```

**参数**：

| 参数 | 来源 | 说明 |
|------|------|------|
| `szabot_fid` | Step 1 返回的 `fid` | 影库文件 fid |
| `file_name` | Step 1 返回的 `name` | 文件名（用于判断文件类型） |

**返回示例**：
```json
{
  "file_name": "豪门继承人第1集.docx",
  "episode_count": 3,
  "avg_char_count": 6200,
  "episodes": [
    {"episode": "第一集", "episode_num": 1, "chinese_char_count": 6500},
    {"episode": "第二集", "episode_num": 2, "chinese_char_count": 6200},
    {"episode": "第三集", "episode_num": 3, "chinese_char_count": 5900}
  ]
}
```

**字段说明**：
- `avg_char_count`：集均字数，用于判断长短剧
- `episode_count`：总集数
- `episodes`：每集字数详情

**长短剧判断规则**（基于 `avg_char_count`）：
- `avg_char_count >= 6000` → 长剧（`belong = "1"`）
- `avg_char_count < 6000` → 短剧（`belong = "4"`）

**后续流程**：
- `belong = "1"`（长剧）→ 继续执行 6.2 预测赛道
- `belong = "4"`（短剧）→ **跳过 6.2 和 6.3**，直接进入 Step 7

### 6.2 调用 file_parse 获取 AI 摘要（仅长剧）

> ⚠️ **仅当字数 ≥ 6000（长剧）时执行**；摘要**仅内部使用**，不向用户展示。

使用 `szabot-file-parser` Skill 分析文件内容：

```bash
python3 $SZBOT_FILE_PARSER_SKILL_DIR/scripts/file_parse.py "{fid}"
```

**参数**：Step 1 获取的 `fid`

**返回**：
```json
{
  "taskId": "xxx",
  "status": "succeeded",
  "result": {
    "content": "AI 生成的文件摘要分析..."
  }
}
```

**用途**：`result.content` 用于预测赛道（Step 6.3）。

> ⚠️ **注意**：`file_parse` 返回的是摘要，不能用于精确统计字数。

### 6.3 赛道预测（仅长剧，基于 AI 摘要）

> ⚠️ **仅当字数 ≥ 6000（长剧）时执行此步骤**

根据 `file_parse` 返回的 `result.content` 摘要内容，分析题材并预测赛道：

| 赛道 | 枚举值 | 摘要中的题材关键词 |
|-----|-------|-------------------|
| 爱 | 1 | 爱情、甜宠、虐恋、婚姻、恋爱、情感、言情、霸总、甜蜜、浪漫 |
| 燃 | 2 | 热血、励志、逆袭、复仇、成长、战斗、拼搏、奋斗、电竞、竞技 |
| 议 | 3 | 悬疑、社会、现实、批判、讽刺、伦理、人性、犯罪、推理、刑侦 |
| 智 | 4 | 商战、权谋、职场、智斗、计谋、博弈、斗争、宫斗、谋略 |
| 传奇 | 7 | 历史、古装、传奇、神话、仙侠、玄幻、武侠、穿越、朝廷、江湖 |

**预测逻辑**：
1. 从 `result.content` 中提取题材类型、核心信息等字段
2. 匹配上表中的关键词，命中最多的赛道为推荐赛道
3. 如果无法判断或属于混合题材，默认推荐「爱」赛道

### 6.4 用户确认流程

> ⚠️ **仅新建项目时需要确认**：合并到已有项目时，直接使用项目信息，无需确认。

向用户展示**统一的确认界面**，包含：剧本类型（长剧/短剧）+ 赛道（仅长剧）

#### 6.4.1 短剧（字数 < 6000）→ 确认剧本类型

```
📊 **初步分析**：

📄 文件名：豪门继承人第1集.docx
🎬 **剧本类型**：短剧

---
请确认以上信息：
- 回复「确认」或「好的」→ 触发剧本分析
- 如需修改为长剧，请回复「长剧」
```

**用户输入处理**：
- **确认**：用户回复「确认」「好的」「OK」或直接回车 → 使用 `belong="4"`，进入 Step 7
- **修改为长剧**：用户回复「长剧」→ 转入长剧确认流程（6.4.2）

#### 6.4.2 长剧（字数 ≥ 6000）→ 确认剧本类型 + 赛道

```
📊 **初步分析**：

📄 文件名：豪门继承人第1集.docx
🎬 **剧本类型**：长剧
🎯 **赛道推荐**：🩷 爱
🏷️ **版本名称**：豪门继承人_20260402

---
请确认或修改赛道：

- 🩷 爱 ← 推荐
- 🔥 燃
- 💬 议
- 🧠 智
- ⚡ 传奇

请回复赛道名称确认（如「爱」「燃」），或回复「确认」使用推荐值：
- 如需修改为短剧，请回复「短剧」
```

**用户输入处理**：
- **确认**：用户回复「确认」「好的」「OK」或直接回车 → 使用 AI 推荐的赛道
- **修改赛道**：用户回复赛道名称（如「燃」「传奇」）→ 更新 `topic_track2`
- **修改为短剧**：用户回复「短剧」→ 使用 `belong="4"`，不传 `topic_track2`

---
> 🛑 **CHECKPOINT**: Step 6 完成后，**必须等待用户确认**：
> - **短剧**（字数 < 6000）→ 用户确认后，获取 `belong="4"`，**不需要 `topic_track2`**，进入 Step 7
> - **长剧**（字数 ≥ 6000）→ 用户确认 `topic_track2` 后，获取 `belong="1"`，进入 Step 7
---

## Step 7：触发剧本分析

使用 MCP 工具 `AddScriptInfo` 触发剧本分析。

### 获取 session_id

`session_id` 为 **Session Key**（会话标识符），即 OpenClaw 会话标识符中的 UUID 部分。

> **说明**：OpenClaw 会话标识符格式为 `agent:main:xxxxxxxx`，其中 `xxxxxxxx` 部分就是 Session Key（UUID 格式，如 `dcd5f33d-f387-4fde-9c5a-5448f0a73886`）。
>
> 影库 API 需要的 `session_id` 是这个 **Session Key（UUID 部分）**，而不是完整的 `agent:main:xxx` 格式。

### 7.1 无项目模式（新建项目）

当 Step 4 未检索到项目或用户选择新建时：
- **长剧**：传 `belong="1"` 和 `topic_track2`
- **短剧**：传 `belong="4"`，不传 `topic_track2`

**命名规则**：
| 场景 | `proj_name` | `script_version_name` |
|------|-------------|----------------------|
| 正常（无重名） | 原项目名 | `项目名_YYYYMMDD` |
| 重名（Step 5.1 新建） | `项目名_YYYYMMDD` | `项目名_YYYYMMDD` |

**示例**：
```bash
# 正常情况
mcporter call szabot_novel_analysis.AddScriptInfo --args '{"proj_id":"","proj_name":"豪门继承人","script_version_name":"豪门继承人_20260402","scripts":[{"file_name":"豪门继承人第1集.docx","file_storage_id":"cos_abc123"}],"belong":"1","topic_track2":"1","public":false,"source":"CLAW_SCRIPT","session_id":"UUID"}'

# 重名场景（proj_name 也带日期）
mcporter call szabot_novel_analysis.AddScriptInfo --args '{"proj_id":"","proj_name":"豪门继承人_20260402","script_version_name":"豪门继承人_20260402","scripts":[...],"belong":"1","topic_track2":"1","public":false,"source":"CLAW_SCRIPT","session_id":"UUID"}'
```

### 7.2 合并项目模式

当 Step 4 用户选择合并到现有项目时：

> ⚠️ **关键**：`belong` 和 `public` **必须**从 Step 4 检索结果中获取，禁止硬编码！

```bash
mcporter call szabot_novel_analysis.AddScriptInfo --args '{"proj_id":"PID-a14af8780aed","script_version_name":"豪门继承人_20260402","scripts":[{"file_name":"豪门继承人第1集.docx","file_storage_id":"cos_abc123"}],"public":true,"belong":"1","source":"CLAW_SCRIPT","session_id":"UUID"}'
```

### 7.3 AddScriptInfo 参数汇总

| 参数 | 无项目模式 | 合并模式 | 说明 |
|------|-----------|---------|------|
| `proj_id` | `""`（空） | Step 4 的 proj_id | 项目ID |
| `proj_name` | 项目名（重名时带时间戳） | 不传 | 仅新建时需要 |
| `script_version_name` | `项目名_YYYYMMDD` | 同左 | 版本名称 |
| `scripts` | 文件列表 | 同左 | `[{file_name, file_storage_id}]` |
| `belong` | Step 6 结果 | Step 4 结果 | `"1"`=长剧，`"4"`=短剧 |
| `topic_track2` | 仅长剧传 | 不传 | 爱→1、燃→2、议→3、智→4、传奇→7 |
| `public` | `false` | Step 4 结果 | 是否公开 |
| `source` | `"CLAW_SCRIPT"` | 同左 | 来源标识 |
| `session_id` | 当前会话 UUID | 同左 | 从 `agent:main:xxx` 中提取 |

**返回**：Empty（无返回值）

---
> 🛑 **CHECKPOINT**: Step 7 完成后，进入 Step 8 反馈用户。
---

## Step 8：反馈用户

分析触发成功后，向用户返回简洁的确认信息：

```
✅ 剧本分析已触发！

📄 文件名：{file_name}
🎬 项目名称：{proj_name}
📊 项目类型：{长剧/短剧}
🎯 赛道：{赛道}（仅长剧显示）

分析完成后将自动推送通知。
```

---

## 异常处理与回退

| 步骤 | 异常 | 处理方式 |
|------|------|---------|
| Step 1 | 文件不存在/上传失败 | 告知用户检查文件路径或重试 |
| Step 2 | fid 无效/转存失败 | 检查 fid 后重试 |
| Step 3 | 项目名提取失败 | 询问用户手动输入 |
| Step 4 | 检索失败 | 重试检索 |
| Step 6 | 字数统计失败 | 默认使用短剧（belong="4"） |
| Step 6 | file_parse 失败（仅长剧） | 默认使用「爱」赛道 |
| Step 7 | 触发失败 | 检查参数后重试 |

> 📝 回退后只处理失败的步骤，不重新执行已成功的步骤。

---

## 多文件处理

当用户上传多个剧本文件时，需要先判断是否属于**同一项目**：

### 处理流程

```
[1] 批量上传 → 获取每个文件的 fid + 从文件名提取项目名
      ↓
[2] 批量转存 → 获取每个文件的 file_storage_id, dir_id
      ↓
[3] 按项目名分组
      ↓
      ├── 所有文件属于同一项目 ──→ 单次检索 + 单次触发
      │
      └── 文件属于多个项目 ──→ 询问用户确认分组 ──┬── 确认分组 ──→ 每组分别检索和触发
                                                │
                                                └── 合并为一个项目 ──→ 用户指定项目名 ──→ 单次检索 + 单次触发
```

### 步骤详解

#### 1. 批量上传和转存

依次执行 Step 1-2 获取每个文件的：
- `fid`（上传返回）
- `file_storage_id` 和 `dir_id`（转存返回）
- **项目名**（从文件名提取）

#### 2. 按项目名分组

从每个文件名提取项目名（去除「第X集」「前X集」等后缀和括号编号），然后分组：

**提取规则**：
- `豪门继承人第1集.docx` → `豪门继承人`
- `逆袭人生EP01.docx` → `逆袭人生`
- `坤宁1.docx` → `坤宁`

**分组示例**：
```
输入文件：
- 豪门继承人第1集.docx
- 豪门继承人第2集.docx
- 逆袭人生第1集.docx

分组结果：
- 「豪门继承人」：豪门继承人第1集.docx, 豪门继承人第2集.docx
- 「逆袭人生」：逆袭人生第1集.docx
```

#### 3. 询问用户确认（仅当检测到多个项目时）

当文件属于**多个不同项目**时，展示分组结果并询问用户：

```
📁 **检测到多个项目**：

📂 项目：**豪门继承人**（2 个文件）
   - 豪门继承人第1集.docx
   - 豪门继承人第2集.docx

📂 项目：**逆袭人生**（1 个文件）
   - 逆袭人生第1集.docx

请选择处理方式：
- **分别分析** - 每个项目分别检索和触发分析
- **合并为一个项目** - 输入统一的项目名，所有文件合并分析

请回复「分别分析」或「合并」：
```

#### 4. 根据用户选择执行

**选择「分别分析」**：
- ⚠️ **必须为每个项目单独调用 `AddScriptInfo`**
- 每个项目组分别执行 Step 3-6（统计字数、判断 belong、检索项目、触发分析）
- **N 个项目 = N 次 `AddScriptInfo` 调用**（不能只调用一次！）

**选择「合并为一个项目」**：
- 用户输入统一的项目名
- 所有文件合并，统计总字数，单次检索，单次触发（1 次 `AddScriptInfo` 调用）

### 多文件示例

**同一项目（多文件）**：
```bash
mcporter call szabot_novel_analysis.AddScriptInfo --args '{
  "proj_id": "", "proj_name": "豪门继承人", "script_version_name": "豪门继承人_20260402",
  "scripts": [
    {"file_name": "豪门继承人第1集.docx", "file_storage_id": "cos_001"},
    {"file_name": "豪门继承人第2集.docx", "file_storage_id": "cos_002"}
  ],
  "belong": "1", "topic_track2": "1", "public": false, "source": "CLAW_SCRIPT", "session_id": "UUID"
}'
```

**分组模式**（多项目分别触发，使用**同一个** `session_id`）：
```bash
# 项目 1：豪门继承人
mcporter call szabot_novel_analysis.AddScriptInfo --args '{"proj_id":"","proj_name":"豪门继承人","script_version_name":"豪门继承人_20260402","scripts":[...],"belong":"1","topic_track2":"1","public":false,"source":"CLAW_SCRIPT","session_id":"UUID"}'

# 项目 2：逆袭人生
mcporter call szabot_novel_analysis.AddScriptInfo --args '{"proj_id":"","proj_name":"逆袭人生","script_version_name":"逆袭人生_20260402","scripts":[...],"belong":"4","public":false,"source":"CLAW_SCRIPT","session_id":"UUID"}'
```
