---
name: szabot-file-editor
namespace: szbot
trust-level: builtin
category: file
description: "影库模版文件编辑：解析用户上传的模版文件（pdf/xlsx/docx/pptx），结合影库项目数据或用户提供的内容精准填写模版。当用户提到填写模版、填表、自动填写、模版填充、项目信息填入文件、填写首集表、填写审批表、用XX信息填写XX文件时使用。"
---

# Szbot File Editor

## 🚨 最高优先级总则（必读，一句话记住）

本技能在执行过程中会产生 **3 份必须上传并展示 fid** 的产物，**全部都要在回复中以具体文字出现**，缺一不可：

| # | 产物 | 出现阶段 | 回复中必须出现的文字（示例） |
|---|------|---------|------------------------------|
| 1 | **模版解析结果 JSON** | Phase 3.5 完成后 | `模版解析结果已上传，fid: fid_xxxxxxxxxxxxxxxxxxxxxx`（真实 fid）|
| 2 | **填写数据 fill_data.json** | Phase 5 Step 5.0 完成后 | `填写数据文件已上传，fid: fid_xxxxxxxxxxxxxxxxxxxxxx`（真实 fid）|
| 3 | **填写结果文件** | Phase 7 完成后 | `[结果文件](bvnext://x-callback-url/doc?fid=fid_xxxxxxxxxxxxxxxxxxxxxx)`（真实 fid）|

> ⛔⛔⛔ **绝对禁止的三种失败模式**：
> 1. 执行了 `upload_file.py` 但**没有在回复中输出 fid 文本**（fid 只出现在工具调用日志里不算数，必须作为正文文字输出给用户）
> 2. 使用 `fid_xxx`、`<fid>`、`fid_xxxxxxxx` 等**占位符**代替 `upload_file.py` 返回的真实值
> 3. 只输出最终结果文件 fid，而**跳过中间产物（解析结果 / fill_data.json）的 fid**
>
> ✅ **检查清单**：结束回复前，问自己——「我这次回复里，产物 1 / 2 / 3 的 fid 都以真实文本的形式出现了吗？」任一漏掉，回复都视为**未完成**，必须补上。

---

## 概述

这个技能用于解析用户上传的模版文件（pdf/xlsx/docx/pptx），结合数据源（影库项目信息、用户文本输入、或用户上传的内容文件）精准填写模版。

支持三种数据来源模式：

| 模式 | 数据来源 | 触发场景 |
|------|---------|---------|
| **模式 A** | Agent 调用 `szabot-copilot` 查询影库项目信息 | "用《庆余年第三季》的信息填写这个首集表" |
| **模式 B** | 用户在对话中直接提供文本内容 | "帮我把以下信息填入模版：项目名称：XXX..." |
| **模式 C** | 用户上传内容文件 + 模版文件 | "用文件A的内容填写模版文件B" |

内置脚本：

- `scripts/upload_file.py` — 文件上传主脚本
- `scripts/download_url.py` — 获取文件下载链接脚本
- `scripts/api_client.py` — 公共 HTTP 调用模块
- `scripts/xlsx_parser.py` / `scripts/xlsx_filler.py` — xlsx 模版解析与填写
- `scripts/docx_parser.py` / `scripts/docx_filler.py` — docx 模版解析与填写
- `scripts/pdf_parser.py` / `scripts/pdf_filler.py` — pdf 模版解析与填写
- `scripts/pptx_parser.py` / `scripts/pptx_filler.py` — pptx 模版解析与填写
- `scripts/content_extractor.py` — 内容文件解析统一入口（模式 C，支持 xlsx/docx/pdf/pptx）

## 触发条件

满足以下条件时使用该技能：

1. 用户提供了一个**模版文件**（pdf/xlsx/docx/pptx），或上下文中已有模版文件
2. 用户的意图是**将某些信息填入模版**
3. 数据来源满足以下任一：
   - 用户指定了影库项目名称/ID → 模式 A
   - 用户在对话中直接提供了待填写的文本内容 → 模式 B
   - 用户同时提供了一个内容文件和一个模版文件 → 模式 C

## ⚠️ 模式识别优先级（必读）

当用户提供的待填写信息是**短文本**（如单个词、一个名称、一句话）时，Agent **必须默认先尝试模式 A**（调用 `szabot-copilot` 查询影库项目信息），而不是直接将这个短文本当作单个字段的值填写（模式 B）。

**判断规则**：

- 用户输入为一个词或短短几个字（如《逐玉》、《庆余年第三季》、"值得爱"）→ **默认按项目名称处理**，优先调用 `szabot-copilot` 的 `kb_search` 查询
- 用户明确提供了多个字段的 key-value（如"项目名称：XXX，导演：YYY"） → 模式 B
- 用户明确说明这个短文本仅对应某个具体字段（如"将'逐玉'填入片名字段"） → 模式 B
- 用户提供了一个内容文件 → 模式 C

> ⛔ **绝对禁止的错误做法**：用户说"把逐玉填入模版文件"，Agent 直接把"逐玉"当作片名填入单个字段，然后反问用户"是否需要查询其他信息"。正确做法是：先调用 `szabot-copilot` 查询《逐玉》项目的全量信息，如果查到则按模式 A 填写；若未查到或用户无权限，再考虑降级处理（联网搜索或询问用户）。

> 💡 **简单记忆**："填模版" + "一个名称" → 先查影库项目；查不到再走降级。

## 不适用场景

| 场景 | 应使用的 Skill |
|------|---------------|
| 仅需要解析文件内容（不涉及填写） | `szabot-file-parser` |
| 仅需要上传文件到影库系统 | `szabot-file-uploader` |
| 仅需要查询项目信息（不涉及文件） | `szabot-copilot` |
| 创建/编辑纯 Excel 电子表格（非模版填写） | `xlsx` |
| 创建/修改影库草稿 | `szabot-project` |

## 环境准备（⚠️ 每次会话首次使用本技能前必须执行）

本技能的脚本依赖 Python 3 和第三方库。**在执行任何脚本（包括 parser、filler、content_extractor 等）之前，必须先检查依赖是否已安装。**

> ⛔ **强制约束**：Agent 在每次会话中首次使用本技能时，**必须先执行依赖检查**，确认所有依赖可用后才能进入工作流。**绝对禁止**跳过依赖检查直接执行 parser 或 filler 脚本。

### Step 1：检查依赖（先查后装）

```bash
python3 -c "import openpyxl, docx, pypdf, pdfplumber, reportlab, pptx; print('All dependencies OK')"
```

- 输出 `All dependencies OK` → 依赖已就绪，**直接进入工作流 Phase 1**
- 报 `ModuleNotFoundError` → 执行 Step 2 安装依赖

### Step 2：离线安装 wheel 包（⭐ 首选，绝大多数场景都能成功）

所有依赖的 wheel 包已预置在 `scripts/wheels/` 目录中，无需网络，一条命令同时安装 pip 和所有依赖：

```bash
bash {skill_base_dir}/scripts/wheels/install_offline.sh
```

安装完成后回到 Step 1 验证。

### Step 3：兜底方案（仅方式 A 失败后使用）

如果 Step 2 离线安装失败（如 wheel 包缺失、Python 版本不匹配等），**再阅读** [`references/environment_setup.md`](references/environment_setup.md) 按其中的方式 B/C/D/E 逐一尝试。

> 💡 `references/environment_setup.md` 还包含中文字体安装说明（PDF 填写必需）、备选镜像源、依赖清单等完整细节。首选使用方式 A 成功后无需阅读。

### 依赖清单（简表）

`openpyxl` / `python-docx` / `pypdf` / `pdfplumber` / `reportlab` / `python-pptx` — 详细版本要求见 [`references/environment_setup.md`](references/environment_setup.md)

---

## 工作流

### Phase 0：环境检查（⚠️ 强制执行，每次会话首次使用时必须执行）

> ⛔ **此步骤为强制步骤，不可跳过。** 在进入 Phase 1 之前，Agent **必须**先验证 Python 依赖是否已安装：
>
> ```bash
> python3 -c "import openpyxl, docx, pypdf, pdfplumber, reportlab, pptx; print('All dependencies OK')"
> ```
>
> - 输出 `All dependencies OK` → 直接进入 Phase 1
> - 报 `ModuleNotFoundError` → 回到上方「环境准备」章节 Step 2（离线安装 wheel 包），安装完成后再进入 Phase 1

---

### Phase 1：文件准备（上传与下载）

> 📖 详细步骤参见 [`references/file_upload_download_workflow.md`](references/file_upload_download_workflow.md)

#### 上传文件获取 fid

当用户提供了本地文件，需要先上传到影库文件系统获取 `fid`：

```bash
cd {skill_base_dir}/scripts && python3 upload_file.py /absolute/path/to/file.xlsx

# 自定义上传文件名（推荐）
cd {skill_base_dir}/scripts && python3 upload_file.py /absolute/path/to/file.xlsx --name "原模版文件名.xlsx"
```

**上传文件名命名规则**：

| 场景 | 命名规则 | 示例 |
|------|---------|------|
| 原模版文件 | 使用原始文件名 | `首集表.xlsx` |
| 填写结果文件 | `{原文件名}_填写结果_{时间戳}.{ext}` | `首集表_填写结果_1713250800.xlsx` |
| 解析结果文件 | `{原文件名}_解析结果_{时间戳}.json` | `首集表_解析结果_1713250800.json` |
| 填写数据文件 | `{原文件名}_填写数据_{时间戳}.json` | `首集表_填写数据_1713250800.json` |

> 💡 时间戳使用 Unix 时间戳（秒级），可通过 `date +%s` 获取。

**输入**：本地文件路径（绝对路径）
**输出**：JSON（包含 `fid` 和文件元信息）

#### 获取文件下载链接

当上下文中已有 `fid`，需要将文件下载到本地进行编辑：

```bash
cd {skill_base_dir}/scripts && python3 download_url.py fid_xxxxxxxxxxxxxxxxxxxxxxxx
```

**输入**：`fid`（由上传接口返回的文件唯一标识）
**输出**：JSON（包含 `download_url` 下载链接）

获取到下载链接后，使用 curl 下载文件到本地：

```bash
curl -o /tmp/template_file.xlsx "<download_url>"
```

### Phase 2：数据获取

> 📖 详细说明参见 [`references/data_source_modes.md`](references/data_source_modes.md)

根据用户意图识别数据来源模式，获取填写数据：

| 模式 | 执行方式 |
|------|----------|
| **模式 A**（默认优先） | 调用 `szabot-copilot` 查询影库项目信息（⚠️ 需完成项目识别和全量字段获取，详见下方说明） |
| **模式 B** | Agent 直接从用户文本中提取 key-value 数据 |
| **模式 C** | 使用 `content_extractor.py` 解析内容文件 |

> ⛔ **模式选择强制约束**：
>
> 当用户输入是**短文本**（单个词/名称/一句话）时，Agent **必须首先尝试模式 A**。具体规则：
>
> 1. **不要把短文本直接当作单个字段值填写**：即使用户只说了"将 XXX 填入模版"，这个 XXX 也很可能是一个项目名称，而不是某个字段的值。
> 2. **必须先调用 `szabot-copilot` 的 `kb_search` 查询**，验证短文本是否对应影库系统中的项目
> 3. **查询到项目后**：按 Step 2.A.2 获取全量字段，执行模式 A 填写流程
> 4. **查询不到或查询失败**：按全局约束第 10 条执行联网搜索降级方案，**并在回复中明确告知用户**查询影库系统失败的事实
> 5. **绝对禁止**在未查询影库系统的情况下，就只填一个字段并反问用户"是否需要查询其他信息"，这种做法综合体验极差
>
> **错误示例**：
> ```
> 用户："把逐玉填入模版文件"
> Agent：直接把"逐玉"填入 B8 片名字段 → 反问用户"是否需要查询其他信息"
> ```
>
> **正确示例**：
> ```
> 用户："把逐玉填入模版文件"
> Agent：先调用 szabot-copilot 查询《逐玉》项目
>        → 拿到全量字段（导演、主演、题材、集数等）
>        → 生成多字段的填写计划，请用户确认
> ```

> ⚠️ **模式 A 关键约束：必须先识别主要项目，再获取全量字段**
>
> Agent 在使用 `szabot-copilot` 查询项目信息时，**必须**按以下步骤执行，**禁止**在第一次搜索后就直接进入模版填写流程：
>
> **Step 2.A.1 — 搜索并识别主要项目**
> 1. 先用项目名称进行模糊搜索，获取搜索结果列表
> 2. 如果搜索结果包含**多个项目**（如同名的电视剧、电影、综艺等），**必须识别出主要项目**：
>    - **优先选择电视剧/剧集类型**的项目（因为模版填写场景中电视剧最常见）
>    - 如果有多个同类型项目，选择管理状态为"进行中"或最近更新的项目
>    - 如果仍无法确定，**询问用户**确认具体是哪个项目
> 3. 确定主要项目后，记录其**项目 ID**
>
> **Step 2.A.2 — 获取项目全量字段**
> 1. 使用项目 ID **再次查询该项目的全部可用字段**（不要仅使用第一次搜索返回的有限字段）
> 2. 查询的字段范围应**尽可能全面**，覆盖模版中可能需要的所有字段类别（基础信息、人员信息、公司信息、进度信息、财务信息等）
> 3. 只有在获取到全量字段数据后，才能进入 Phase 3（模版解析）和 Phase 4（智能映射）
>
> **❌ 错误做法**：第一次搜索返回了项目名称、导演等少量字段后，就直接开始准备 fill_data.json
> **✅ 正确做法**：第一次搜索确认项目 → 用项目 ID 查询全量字段 → 再进入模版解析和填写流程
>
> 详细说明参见 [`references/data_source_modes.md`](references/data_source_modes.md) 中的「模式 A」章节。

模式 C 执行命令：

```bash
cd {skill_base_dir}/scripts && python3 content_extractor.py /path/to/content_file.xlsx
```

当前支持的内容文件类型：`xlsx`、`docx`、`pdf`、`pptx`。

### Phase 3：模版解析

> 📖 各文件类型的详细工作流参见对应的 reference 文档

根据模版文件类型，使用对应的解析脚本提取待填字段和位置信息：

| 文件类型 | 解析脚本 | 详细文档 |
|---------|---------|----------|
| **xlsx** | `scripts/xlsx_parser.py` | [`references/xlsx_workflow.md`](references/xlsx_workflow.md) |
| **docx** | `scripts/docx_parser.py` | [`references/docx_workflow.md`](references/docx_workflow.md) |
| **pdf** | `scripts/pdf_parser.py` | [`references/pdf_workflow.md`](references/pdf_workflow.md) |
| **pptx** | `scripts/pptx_parser.py` | [`references/pptx_workflow.md`](references/pptx_workflow.md) |

解析命令（以 xlsx 为例，其他类型替换脚本名和文件后缀即可）：

```bash
cd {skill_base_dir}/scripts && python3 xlsx_parser.py /path/to/template.xlsx
cd {skill_base_dir}/scripts && python3 docx_parser.py /path/to/template.docx
cd {skill_base_dir}/scripts && python3 pdf_parser.py /path/to/template.pdf
cd {skill_base_dir}/scripts && python3 pptx_parser.py /path/to/template.pptx
```

输出结构化 JSON，包含所有待填字段的名称、位置、样式等信息。

### Phase 3.5：上传解析结果（⚠️ 强制执行）

> **此步骤为强制步骤，不可跳过。**

模版解析完成后，**必须**将 parser 输出的 JSON 结果保存为本地临时文件，上传并在回复中给出下载链接，供用户查看模版的完整字段结构。

#### Step 3.5.1 — 保存解析结果到临时文件

将 parser 的 stdout JSON 输出保存为临时文件：

```bash
cd {skill_base_dir}/scripts && python3 xxx_parser.py /path/to/template.xxx > /tmp/parser_result.json
```

或者在执行 parser 后，将输出内容写入文件：

```bash
cat > /tmp/parser_result.json << 'PARSER_EOF'
<parser 输出的 JSON 内容>
PARSER_EOF
```

#### Step 3.5.2 — 上传解析结果文件

```bash
cd {skill_base_dir}/scripts && python3 upload_file.py /tmp/parser_result.json --name "首集表_解析结果_$(date +%s).json"
```

从返回 JSON 中提取 `fid`。

#### Step 3.5.3 — 🚨🚨🚨 必须在回复正文中明文输出 fid（不可省略）

从 `upload_file.py` 的标准输出 JSON 中提取 `fid` 字段的**实际值**，并在当前回复的**正文文字**中以如下格式**明确输出**：

```
模版解析结果已上传，fid: fid_xxxxxxxxxxxxxxxxxxxxxx
```

（上面的 `fid_xxxxxxxxxxxxxxxxxxxxxx` **必须替换为** `upload_file.py` 实际返回的真实 fid）

> ⛔⛔⛔ **三大绝对禁止**：
> 1. **禁止只执行上传而不在回复中输出 fid 文本**——fid 只出现在工具调用日志里不算数，用户看不到。必须作为正文文字呈现给用户。
> 2. **禁止使用占位符**——如 `fid_xxx`、`<fid>`、`fid_xxxxxxxx`、`fid_xxxxxxxxxxxxxxxxxxxxxxxx`。**必须**是 `upload_file.py` 返回的真实值（如 `fid_5f81b9b77acdcf6bce18`）。
> 3. **禁止用「解析完成」「已处理」等模糊措辞代替 fid**——必须明确写出 `fid: <真实值>`。
>
> ⛔ **不需要调用 `download_url.py`**，只需展示 `fid` 即可。
>
> ✅ **自检**：结束本轮回复前，回看自己的回复正文——是否包含形如 `fid: fid_xxxxxxxxxxxxxxxx` 的真实 fid 文本？如果没有，**立即补上**。

### Phase 4：智能映射

由 Agent（LLM）完成，不需要脚本。Agent 阅读模版字段清单和数据源，进行语义匹配：

1. **精确匹配**：模版字段名 == 数据字段名（如 "项目名称" → "项目名称"）
2. **语义匹配**：模版字段名 ≈ 数据字段名（如 "剧名" → "项目名称"）
3. **组合映射**：一个模版字段需要多个数据字段组合
4. **格式转换**：数据需要格式化后填入（如日期格式化）
5. **无法映射**：标记为 unmapped，汇总后询问用户

> ⛔ **关键约束：fill_data.json 中的字段必须严格来自 parser 输出，禁止凭空创造字段**
>
> Agent 在生成 `fill_data.json` 时，**每一个 fill 项的 `location` 和 `sheet` 必须严格来自 Phase 3 parser 输出的 `fields` 列表**。**绝对禁止**以下行为：
> - 凭空创造 parser 输出中不存在的字段（如 parser 没有输出 `单集时长` 字段，Agent 不得自行添加）
> - 自行编造 `location` 坐标（如 parser 输出中没有 `B9` 这个 location，Agent 不得自行使用 `B9`）
> - 将数据源中的字段强行映射到不存在的模版位置
>
> **特别注意合并单元格**：如果 parser 输出中某个字段的 `merged_range` 为 `B8:B9`，说明 B8 和 B9 是同一个合并单元格，只有 `B8` 是有效的写入位置。Agent **不得**将另一个字段映射到 `B9`，否则 filler 会将 `B9` 自动修正为 `B8`，导致覆盖已填写的值。
>
> **正确做法**：如果数据源中有字段无法在 parser 输出的 `fields` 列表中找到对应的模版字段，应将其标记为 unmapped（无法映射），而非自行创造位置。

> ⚠️ **PDF 已有值字段的特殊处理**：PDF parser 可能会将表格中的已有值误识别为标签名（输出中带有 `may_be_value_of` 标记）。Agent 在映射时**必须**识别这类噪音字段，不要将其当作独立的待填字段。如果需要替换已有值，应使用真正标签的 `pdf_coords` 写入新值，同时用 `clear_rect` 清空被误识别字段所在区域。详见 `references/pdf_workflow.md` 中的「已有值字段的识别与处理」章节。

映射完成后，Agent 生成 `fill_data.json` 文件供填写脚本使用。

> ⛔ **关键约束：`fill_data.json` 必须严格遵循各文件类型 filler 脚本要求的结构化格式**，而非简单的 key-value 字典。具体格式要求参见对应的 reference 文档（如 `docx_workflow.md` 的「准备填写数据」章节）。Agent 必须将 Phase 3 parser 输出的 `location`（定位信息）、`value_status`（值状态）、`pattern`（识别模式）等字段**原样传递**到 fill_data.json 中，仅替换 `value` 为映射后的实际数据。**绝对禁止**生成 `{"字段名": "值"}` 这样的扁平字典格式，filler 脚本无法识别此格式。
>
> ⛔ **DOCX 段落填写必须传递 `pattern` 字段**：对于 docx 模版中 `paragraph_label_empty` 和 `paragraph_label_value` 模式的字段（即「标签：」或「标签：值」格式的段落），Agent 在生成 fill_data.json 时**必须**从 parser 输出中原样传递 `pattern` 字段。缺少 `pattern` 会导致 filler 无法正确识别标签模式，可能将标签名称（如 `*中文片名：`）整体替换为填写值，造成标签丢失。虽然 `docx_filler.py` 已增加了冒号检测兜底逻辑，但 Agent 仍应规范传递 `pattern` 字段。
>
> ⚠️ **PDF text_overlay 模式特别注意**：
> - 顶层键名必须是 **`fills`**（不是 `fields`）
> - 每个 fill 项的 **`page`** 和 **`pdf_coords`** 必须在顶层（不要仅嵌套在 `location` 中）
> - 虽然 `pdf_filler.py` 已做兼容处理（支持 `fields` 键名、支持从 `location` 中自动提取 `page`/`pdf_coords`），但 Agent 应尽量生成规范格式

### Phase 4.5：展示填写计划并等待用户确认（⚠️ 强制执行）

> **此步骤为强制步骤，不可跳过。** Agent 在生成 `fill_data.json` 后，**必须**先向用户展示填写计划并等待确认，**不得直接执行填写**。
>
> ⚠️ **此阶段不上传 fill_data.json**。fill_data.json 的上传在用户确认后的 Phase 5 中执行，确保用户下载到的是最终确认版本。

#### Step 4.5.1 — 保存填写数据到临时文件

Agent 将映射后的 `fill_data.json` 写入本地临时文件（暂不上传）：

```bash
cat > /tmp/fill_data.json << 'FILL_EOF'
<Agent 生成的 fill_data.json 内容>
FILL_EOF
```

#### Step 4.5.2 — ⚠️ 必须输出填写计划并等待用户确认

Agent **必须**在回复中输出以下内容：

1. **填写计划摘要表格**：以人类可读的表格形式展示即将填写的字段和值
2. **确认提示**：明确询问用户是否确认执行填写

**输出格式示例**：

```markdown
📋 **填写计划**

| 字段名 | 填写值 | 状态 |
|--------|--------|------|
| 项目名称 | 庆余年第三季 | 新填写 |
| 导演 | 孙皓 | 新填写 |
| 出品公司 | 北京光线影业有限公司 | 覆盖原值 |
| 编剧 | ⚠️ 暂无数据 | 未映射 |

> 共 XX 个字段将被填写，XX 个字段将覆盖原有值，XX 个字段无法映射。

请确认以上填写内容是否正确？如需修改，请告诉我需要调整的字段和值。确认无误后我将执行填写。
```

> ⛔ **绝对禁止在用户确认之前执行 Phase 5 填写操作。** Agent 在输出填写计划和确认提示后，**必须立即结束当前回复**，等待用户在**新的一条消息**中明确回复「确认」「没问题」「可以」等肯定性回复后，才能在**下一轮回复**中继续执行 Phase 5。**绝对禁止**在同一轮回复中既展示填写计划又执行填写。如果用户提出修改意见，Agent 应根据修改意见更新 `/tmp/fill_data.json`，并再次展示修改后的填写计划请求确认。

### Phase 5：执行填写（⚠️ 需用户确认后才可执行）

> ⛔ **前置条件**：Phase 4.5 中用户已确认填写计划。如果用户尚未确认，**禁止执行此步骤**。
>
> 📖 各文件类型的详细填写策略参见对应的 reference 文档

#### Step 5.0 — 上传最终确认版 fill_data.json（⚠️ 强制执行）

用户确认后、调用 filler 脚本前，**必须**先上传最终版的 `fill_data.json`：

```bash
# 上传最终确认版填写数据
cd {skill_base_dir}/scripts && python3 upload_file.py /tmp/fill_data.json --name "首集表_填写数据_$(date +%s).json"
```

从 `upload_file.py` 的标准输出 JSON 中提取 `fid` 字段的**实际值**，并在当前回复的**正文文字**中以如下格式**明确输出**：

```
填写数据文件已上传，fid: fid_xxxxxxxxxxxxxxxxxxxxxx
```

（上面的 `fid_xxxxxxxxxxxxxxxxxxxxxx` **必须替换为** `upload_file.py` 实际返回的真实 fid）

> ⛔⛔⛔ **三大绝对禁止**：
> 1. **禁止只执行上传而不在回复中输出 fid 文本**——fid 只出现在工具调用日志里不算数，用户看不到。必须作为正文文字呈现给用户。
> 2. **禁止使用占位符**——如 `fid_xxx`、`<fid>`、`fid_xxxxxxxx`。**必须**是 `upload_file.py` 返回的真实值。
> 3. **禁止跳过上传步骤直接调用 filler 脚本**——必须先上传并展示 fid，再调用 filler。
>
> ⛔ **不需要调用 `download_url.py`**，只需展示 `fid` 即可。
>
> ✅ **自检**：本轮回复中，填写数据文件的真实 fid 是否以正文文字形式出现？未出现则**立即补上**。

根据模版文件类型，使用对应的填写脚本将数据写入模版：

| 文件类型 | 填写脚本 | 核心库 |
|---------|---------|--------|
| **xlsx** | `scripts/xlsx_filler.py` | openpyxl |
| **docx** | `scripts/docx_filler.py` | python-docx |
| **pdf** | `scripts/pdf_filler.py` | pypdf + reportlab |
| **pptx** | `scripts/pptx_filler.py` | python-pptx |

填写命令（以 xlsx 为例，其他类型替换脚本名和文件后缀即可）：

```bash
cd {skill_base_dir}/scripts && python3 xlsx_filler.py /path/to/template.xlsx /path/to/fill_data.json /path/to/output.xlsx
cd {skill_base_dir}/scripts && python3 docx_filler.py /path/to/template.docx /path/to/fill_data.json /path/to/output.docx
cd {skill_base_dir}/scripts && python3 pdf_filler.py /path/to/template.pdf /path/to/fill_data.json /path/to/output.pdf
cd {skill_base_dir}/scripts && python3 pptx_filler.py /path/to/template.pptx /path/to/fill_data.json /path/to/output.pptx
```

### Phase 6：验证与输出

1. **完整性校验**：检查所有必填字段是否已填写
2. **格式校验**：
   - xlsx：可选调用 `xlsx` skill 的 `recalc.py` 重算公式
   - docx/pdf/pptx：检查文件是否可正常打开
3. **输出**：保存填写后的文件到本地

### Phase 7：上传结果文件并输出链接（⚠️ 强制执行）

> **此步骤为强制步骤，不可跳过。**

文件编辑完成后，**必须**执行以下流程，将结果文件的链接输出给用户：

#### Step 7.1 — 上传编辑后的文件

```bash
cd {skill_base_dir}/scripts && python3 upload_file.py /absolute/path/to/edited_file.xlsx --name "首集表_填写结果_$(date +%s).xlsx"
```

从返回 JSON 中提取 `fid`。

#### Step 7.2 — ⚠️ 必须输出链接给用户

使用上传返回的 `fid` 拼接 `bvnext` 协议链接，以 Markdown 超链接格式输出给用户：

```markdown
[结果文件](bvnext://x-callback-url/doc?fid=<fid>)
```

> ⛔ **填写结果文件的链接格式必须为 `bvnext://x-callback-url/doc?fid=<fid>`**，其中 `<fid>` 替换为上传返回的实际 fid 值。**绝对禁止**使用其他格式的链接。
>
> ⛔ **不需要调用 `download_url.py`**，直接用 fid 拼接 bvnext 协议链接即可。
>
> ⛔ **绝对禁止省略此步骤。** 大模型在完成文件编辑后，**必须**将 `[结果文件](bvnext://x-callback-url/doc?fid=<fid>)` 这个 Markdown 超链接输出在回复中，确保用户可以直接点击查看。如果缺少此链接，用户将无法获取编辑后的文件，整个工作流视为未完成。

**正确流程示例**：

```
# Step 7.1: 上传文件
$ python3 upload_file.py /tmp/filled_file.pdf --name "模版_填写结果_$(date +%s).pdf"
→ stdout: {"fid": "fid_abc123...", ...}

# Step 7.2: 输出给用户（使用 bvnext 协议链接）
# ✅ 正确：使用 bvnext 协议 + fid
[结果文件](bvnext://x-callback-url/doc?fid=fid_abc123...)
# ❌ 错误：使用 download_url 或自行拼接的 http 链接
# [结果文件](https://cdn.example.com/...)
```

## 全局约束

1. **fid 必须是上传接口返回的真实值**：`fid` 由 `upload_file.py` 的标准输出 JSON 中的 `fid` 字段返回，**严禁使用用户口头提供、手动输入或凭空编造的 fid**。在回复中展示 fid 或拼接 bvnext 链接时，**必须使用 `upload_file.py` 实际返回的 fid 值**（如 `fid_5f81b9b77acdcf6bce18`），**绝对禁止**使用占位符（如 `fid_xxx`、`<fid>`、`fid_xxxxxxxx`、`fid_xxxxxxxxxxxxxxxxxxxxxxxx`）。
2. **支持的模版文件类型**：`xlsx`、`docx`、`pdf`、`pptx`。
3. **结果必须保留在上下文中**：上传/下载链接获取成功后的结果必须保存在对话上下文中，供后续工作流直接使用。
4. **保留原始格式**：填写模版时**必须使用内置的填写脚本**（如 `xlsx_filler.py`），脚本采用「先拷贝模版副本、再修改副本」的策略，完整保留原始文件的合并单元格、边框、字体、样式、布局。**绝对禁止**大模型自行编写 Python 脚本创建新文件来替代模版填写。
5. ⛔ **编辑完成后必须输出链接**：每次通过本技能完成文件编辑后，**必须**执行「上传结果文件 → 用上传返回的 fid 拼接 `bvnext://x-callback-url/doc?fid=<fid>` 链接 → 以 `[结果文件](bvnext://x-callback-url/doc?fid=<fid>)` 格式输出给用户」这一完整流程。**绝对不允许**在编辑完成后不输出链接就结束回复。**无论数据来源是什么（影库项目查询、用户提供、联网搜索等），只要完成了文件填写并保存，就必须上传并输出 bvnext 链接**。绝对禁止仅输出本地文件路径而不上传、不给链接。
6. ⛔ **禁止自行编写文件操作脚本**：**绝对禁止**大模型自行编写 Python 脚本来操作 xlsx/docx/pdf/pptx 文件（包括但不限于：使用 openpyxl、xlsxwriter、直接操作 XML 等方式）。所有文件操作**必须且只能**通过本技能内置的脚本（如 `xlsx_parser.py`、`xlsx_filler.py`）完成。自行编写脚本会导致文件格式损坏、XML 结构错误、Excel 无法打开等严重问题。
7. 🚨🚨🚨 **中间产物 fid 必须以正文文本形式出现在回复中（与 Phase 7 bvnext 链接同等重要）**：本技能在工作流中产生 3 份必须展示给用户的产物——(a) **模版解析结果 JSON**（Phase 3.5）、(b) **填写数据 fill_data.json**（Phase 5 Step 5.0）、(c) **填写结果文件**（Phase 7）。这三份产物的 fid / 链接**每一个都必须在回复的正文文字中明确出现**，缺一不可：
   - 产物 (a)：`模版解析结果已上传，fid: <真实fid>`
   - 产物 (b)：`填写数据文件已上传，fid: <真实fid>`
   - 产物 (c)：`[结果文件](bvnext://x-callback-url/doc?fid=<真实fid>)`
   
   **绝对禁止**以下行为：
   - ❌ 只执行了 `upload_file.py` 但没有在回复正文中明文输出 fid（工具日志里的 fid 用户看不到，不算数）
   - ❌ 使用占位符（如 `fid_xxx`、`<fid>`、`fid_xxxxxxxx`）代替真实值
   - ❌ 用模糊措辞（如「解析完成」「已上传」「已处理」）代替具体的 fid 文本
   - ❌ 只输出最终结果文件 fid，漏掉中间产物 (a) 和 (b) 的 fid
   - ❌ 调用 `download_url.py` 获取下载链接来展示（中间产物只需展示 fid 即可，无需下载链接）
   
   **自检规则**：每次结束回复前，必须自问——「产物 (a)、(b)、(c) 的 fid/链接是否都以真实文本的形式在我本轮回复的正文里出现了？」任一缺失则回复视为**未完成**，必须立即补齐后再结束。
8. ⛔ **填写前必须等待用户确认（不可自行判断用户已确认）**：Agent 在生成填写计划后，**必须**在回复中明确输出填写计划摘要表格和确认提示，然后**停止当前回复，等待用户的下一条消息**。只有当用户在**新的一条消息**中明确回复「确认」「没问题」「可以」等肯定性回复后，Agent 才能在**下一轮回复**中执行 Phase 5 填写操作。**绝对禁止**以下行为：
   - 在同一轮回复中既展示填写计划又执行填写（即使 Agent 认为用户「已确认」）
   - Agent 自行判断「用户已确认」而跳过等待步骤
   - 在用户未明确回复确认之前执行任何填写操作
   
   如果用户提出修改意见，Agent 应更新 `fill_data.json` 并重新请求确认。
9. ⛔ **模式 A 必须先识别主要项目再获取全量字段**：当数据来源为影库项目查询（模式 A）时，Agent **必须**先通过搜索结果识别出主要项目（多个同名项目时优先选电视剧/剧集类型），再使用项目 ID 查询该项目的**全部可用字段**。**绝对禁止**在第一次模糊搜索返回少量字段后就直接开始准备 fill_data.json。详见 Phase 2 中的模式 A 说明。
10. ⛔ **模式 A 查询失败时的降级处理（联网搜索）**：当模式 A 查询影库项目信息失败（如用户无权限、项目不存在、接口异常等），Agent 可以尝试通过联网搜索获取相关信息作为降级方案，但**必须遵守以下规则**：
   - **必须明确提醒用户**：在回复中用醒目的方式（如 ⚠️ 或加粗）提醒用户「以下信息来自联网搜索，可能不够准确或不够完整，请仔细核实」
   - **不得将联网信息当作确定事实**：联网搜索结果可能过时、不完整或有误，Agent 应在填写计划中标注哪些字段的数据来自联网搜索
   - **填写完成后仍然必须输出 bvnext 链接**：无论数据来源是影库项目查询还是联网搜索，只要完成了文件填写并保存，就**必须**执行 Phase 7 上传并输出 `[结果文件](bvnext://x-callback-url/doc?fid=<fid>)` 链接。**绝对禁止**仅输出本地文件路径而不上传、不给链接
   
   **正确示例**：
   ```
   > ⚠️ **注意**：由于无法通过影库项目系统获取《逐玉》的信息（可能无权限或项目未录入），以下填写内容来自联网搜索，**可能不够准确或不够完整，请仔细核实**！
   
   ✅ 模板已填写完成！
   [结果文件](bvnext://x-callback-url/doc?fid=fid_abc123...)
   ```
   
   **错误示例**（缺少提醒和链接）：
   ```
   ✅ 模板已填写完成！
   📁 文件已保存至：/root/.openclaw/workspace/xxx.xlsx
   ```
11. ⛔ **每次会话首次使用必须先检查依赖**：Agent 在每次会话中首次使用本技能时，**必须**先执行 Phase 0 环境检查（`python3 -c "import openpyxl, docx, pypdf, pdfplumber, reportlab, pptx; print('All dependencies OK')"`），确认所有依赖可用后才能进入 Phase 1。**绝对禁止**跳过依赖检查直接执行 parser 或 filler 脚本。如果依赖缺失，必须先完成「环境准备」章节的安装步骤。
12. ⛔ **填写结果文件的链接必须使用 `bvnext` 协议格式**：所有输出给用户的填写结果文件链接，**必须**使用 `bvnext://x-callback-url/doc?fid=<fid>` 格式，其中 `<fid>` 为上传返回的实际 fid 值。**绝对禁止**以下行为：
   - 自行拼接 http/https URL（如 `https://claw.szabot.internal/files/<fid>`）
   - 调用 `download_url.py` 获取下载链接作为填写结果的输出链接
   - 使用其他任何格式的链接
   
   对于原模版文件、解析结果文件、填写数据文件等中间数据，上传后只需展示 `fid`，不需要提供下载链接。
   
   `download_url.py` 保留供内部下载文件到本地编辑的场景使用（如 Phase 1 下载模版文件到本地），不再用于生成输出给用户的链接。
13. ⛔ **fill_data.json 必须传递 `pattern`、`value_status` 字段**：Agent 在生成 fill_data.json 时，**必须**从 parser 输出中原样复制每个字段的 `pattern` 和 `value_status` 字段。特别是 docx 模版中 `paragraph_label_empty` 和 `paragraph_label_value` 模式的字段，缺少 `pattern` 会导致 filler 将标签名称（如 `*中文片名：`）整体替换为填写值，造成标签丢失。正确做法：从 parser 输出中复制完整的 `location`、`pattern`、`value_status`，仅替换 `value`。
14. ⛔ **fill_data.json 中的字段必须严格来自 parser 输出，禁止凭空创造字段**：Agent 在生成 `fill_data.json` 时，**每一个 fill 项的 `location` 和 `sheet` 必须严格来自 Phase 3 parser 输出的 `fields` 列表**。**绝对禁止**以下行为：
   - 凭空创造 parser 输出中不存在的字段（如 parser 没有输出 `单集时长` 字段，Agent 不得自行添加）
   - 自行编造 `location` 坐标（如 parser 输出中没有 `B9` 这个 location，Agent 不得自行使用 `B9`）
   - 将数据源中的字段强行映射到不存在的模版位置
   
   **特别注意合并单元格**：如果 parser 输出中某个字段的 `merged_range` 为 `B8:B9`，说明 B8 和 B9 是同一个合并单元格，只有 `B8` 是有效的写入位置。Agent **不得**将另一个字段映射到 `B9`，否则 filler 会将 `B9` 自动修正为 `B8`，导致覆盖已填写的值。
   
   **正确做法**：如果数据源中有字段无法在 parser 输出的 `fields` 列表中找到对应的模版字段，应将其标记为 unmapped（无法映射），在填写计划中提示用户，而非自行创造位置。

## 输出约定

### upload_file.py

- 标准输出（stdout）：固定输出 JSON，包含 `fid` 和文件元信息
- 标准错误（stderr）：输出关键过程日志
- 退出码：`0` = 成功，`1` = 文件不存在，`2` = 其他错误

### download_url.py

- 标准输出（stdout）：固定输出 JSON，包含 `download_url` 和文件信息
- 标准错误（stderr）：输出关键过程日志
- 退出码：`0` = 成功，`1` = 参数错误，`2` = 其他错误

## 依赖技能

| 技能 | 用途 | 必要性 |
|------|------|--------|
| `szabot-copilot` | 模式 A：查询影库项目信息作为填写数据源 | 模式 A 必需 |
| `xlsx` | xlsx 文件的读写能力、公式重算（recalc.py） | xlsx 模版可选 |
| `docx` | docx 文件的读写能力（unpack/pack 高级编辑） | docx 模版可选 |
| `pdf` | pdf 文件的读写能力（表单填写脚本） | pdf 模版可选 |
| `pptx` | pptx 文件的读写能力（unpack/pack 高级编辑） | pptx 模版可选 |
| `szabot-file-parser` | 可选：辅助解析复杂 PDF 内容文件 | 可选 |

## 与其他技能的衔接

| 下游技能 | 用途 | 使用的字段 |
|---------|------|-----------|
| `szabot-project`（草稿关联文件） | 将填写后的文件关联到项目草稿 | `fid` 作为 `file_id` |

> ⚠️ 下游技能中使用的 `file_id` / `fid` **必须来自本技能上传流程的返回结果**，不可使用用户自行提供的 ID。

## 资源说明

### 脚本

| 脚本 | 用途 |
|------|------|
| `scripts/upload_file.py` | 文件上传，输出 `fid` 与文件元信息 |
| `scripts/download_url.py` | 获取文件下载链接，输出 `download_url` |
| `scripts/api_client.py` | 公共 HTTP 调用模块（上传、下载链接、签名等） |
| `scripts/xlsx_parser.py` | xlsx 模版解析，提取待填字段和位置信息 |
| `scripts/xlsx_filler.py` | xlsx 模版填写，将数据写入指定单元格 |
| `scripts/docx_parser.py` | docx 模版解析，提取表格和段落中的待填字段 |
| `scripts/docx_filler.py` | docx 模版填写，将数据写入表格单元格和段落 |
| `scripts/pdf_parser.py` | pdf 模版解析，检测表单域和文本区域 |
| `scripts/pdf_filler.py` | pdf 模版填写，支持表单域填写和文本叠加 |
| `scripts/pptx_parser.py` | pptx 模版解析，提取幻灯片中的待填字段 |
| `scripts/pptx_filler.py` | pptx 模版填写，将数据写入文本框和表格 |
| `scripts/content_extractor.py` | 内容文件解析统一入口（模式 C，支持 xlsx/docx/pdf/pptx） |
| `scripts/download_wheels.sh` | 离线安装：在本机下载 wheel 包并打包，供容器离线安装 |

### 参考文档

| 文档 | 内容 | 何时阅读 |
|------|------|----------|
| `references/environment_setup.md` | 环境依赖安装完整指南（方式 B/C/D/E、中文字体、备选镜像源） | 仅当 Step 2 离线安装失败时 |
| `references/file_upload_download_workflow.md` | 文件上传与下载工作流 | Phase 1 / Phase 7 |
| `references/data_source_modes.md` | 三种数据来源模式（A/B/C）详细说明 | Phase 2 |
| `references/xlsx_workflow.md` | xlsx 模版的完整解析+填写工作流 | 处理 xlsx 文件时 |
| `references/docx_workflow.md` | docx 模版的完整解析+填写工作流 | 处理 docx 文件时 |
| `references/pdf_workflow.md` | pdf 模版的完整解析+填写工作流 | 处理 pdf 文件时 |
| `references/pptx_workflow.md` | pptx 模版的完整解析+填写工作流 | 处理 pptx 文件时 |
