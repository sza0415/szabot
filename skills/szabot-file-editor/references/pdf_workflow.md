# PDF 模版解析与填写工作流

## 概述

本工作流处理 pdf 类型的模版文件。PDF 与 xlsx/docx 不同，**不能直接修改原文件内容**，需要根据 PDF 类型选择不同的填写策略。

## 内置脚本

| 脚本 | 用途 |
|------|------|
| `scripts/pdf_parser.py` | 解析 pdf 模版结构，检测表单域和文本区域 |
| `scripts/pdf_filler.py` | 将数据写入 pdf 模版（表单域填写或文本叠加） |

---

## 一、模版解析（Phase 3）

### 执行命令

```bash
cd {skill_base_dir}/scripts && python3 pdf_parser.py /path/to/template.pdf
```

### 输出中的关键字段

- `has_fillable_fields`：是否有可填写表单域（决定使用哪种填写模式）
- `fields`：所有待填字段列表
- `tables`：表格区域列表

### 识别模式

| 模式 | 说明 | 依赖库 |
|------|------|--------|
| `fillable_form_field` | PDF 内置的可填写表单域 | pypdf |
| `text_label_value` | 文本中的"标签：____"模式 | pdfplumber |
| `text_placeholder` | 文本中的占位符 | pdfplumber |
| `pdf_table_pair` | 表格中的标签-值配对（类似 docx 的 table_horizontal_pair） | pdfplumber |
| `pdf_table` | 表格数据区域（表头行 + 空数据行） | pdfplumber |

### 字段值状态分类（value_status）

每个解析出的字段都会带有 `value_status` 标记：

| value_status | 含义 | 建议操作 |
|-------------|------|----------|
| `empty` | 字段为空 | 直接填写 |
| `placeholder` | 占位符文本 | 直接填写 |
| `example` | 示例值 | 删除后填写 |
| `has_value` | 已有有效值 | **默认跳过** |

> 对于 `has_value` 的字段，可在 fill_data.json 中设置 `"force": true` 强制覆盖。

---

## 一点五、上传解析结果（Phase 3.5）

模版解析完成后，**必须**将 parser 输出的 JSON 结果保存为本地临时文件并上传，在回复中展示 `fid`。

```bash
# 保存解析结果
cd {skill_base_dir}/scripts && python3 pdf_parser.py /path/to/template.pdf > /tmp/parser_result.json

# 上传
cd {skill_base_dir}/scripts && python3 upload_file.py /tmp/parser_result.json --name "模版名_解析结果_$(date +%s).json"
```

上传成功后，从 `upload_file.py` 的标准输出 JSON 中提取 `fid` 字段的**实际值**，在回复中展示。

> ⛔ **`fid` 必须是 `upload_file.py` 返回的真实值**（如 `fid_5f81b9b77acdcf6bce18`），**绝对禁止**使用占位符（如 `fid_xxx`、`<fid>`、`fid_xxxxxxxx`）。
> ⛔ **不需要调用 `download_url.py` 获取下载链接**，只需展示 `fid` 即可。

---

## 二、填写模式选择

> ⛔ **Phase 4→Phase 5 衔接要点**：Agent 在 Phase 4 完成智能映射后，**必须**按照以下各模式要求的格式构造 `fill_data.json`，而非生成简单的 `{"字段名": "值"}` 扁平字典。具体要求：
> 1. **复用 parser 输出的定位信息**：`form_fields` 模式使用 `field_name`，`text_overlay` 模式使用 `page`/`x`/`y` 坐标（来自 `pdf_coords`），均必须直接来自 Phase 3 parser 的输出，不可自行编造
> 2. **传递 `value_status`**：该字段决定了 filler 的填写策略（直接填写 vs 跳过），必须从 parser 输出中原样传递
> 3. **仅替换 `value`**：Agent 只需将 parser 输出中的 `value`（或空值）替换为映射后的实际数据
> 4. **`has_value` 字段需要 `force`**：如果需要覆盖已有值，必须显式设置 `"force": true`
> 5. **text_overlay 模式必须使用 `pdf_coords`**：坐标、字号、clear_rect 等必须直接使用 parser 输出的 `pdf_coords` 字段，**绝对禁止自行估算坐标**

根据 `has_fillable_fields` 选择填写模式：

| 条件 | 模式 | 说明 |
|------|------|------|
| `has_fillable_fields = true` | `form_fields` | 直接填写 PDF 表单域 |
| `has_fillable_fields = false` | `text_overlay` | 在指定坐标叠加文本 |

---

## 三、表单域填写模式（form_fields）

### 准备填写数据

```json
{
  "mode": "form_fields",
  "fills": [
    {"field_name": "last_name", "value": "张三", "value_status": "empty"},
    {"field_name": "phone", "value": "13800138000", "value_status": "example"},
    {"field_name": "company", "value": "光线影业", "value_status": "has_value", "force": true},
    {"field_name": "checkbox_agree", "value": "/On"}
  ]
}
```

> `field_name` 必须与 `pdf_parser.py` 输出的 `field_name` 完全一致。
> `value_status` 为 `has_value` 时默认跳过，设置 `"force": true` 可强制覆盖。

### 执行填写

```bash
cd {skill_base_dir}/scripts && python3 pdf_filler.py /path/to/template.pdf /path/to/fill_data.json /path/to/output.pdf
```

---

## 四、文本叠加模式（text_overlay）

当 PDF 没有可填写表单域时，使用 reportlab 在指定坐标叠加文本。

### 坐标系说明

PDF 坐标系：**左下角为原点**，x 向右增大，y 向上增大。单位为 **点（pt）**，1 英寸 = 72 pt。

常见页面尺寸：
- A4：595 × 842 pt
- US Letter：612 × 792 pt

> ⚠️ **pdfplumber 坐标系**与 PDF 坐标系不同：pdfplumber 以**左上角为原点**，`top` 向下增大。
> 转换公式：`pdf_y = page_height - pdfplumber_top`

### 利用 grid 信息精确定位

`pdf_parser.py` 输出的表格信息中包含 `grid` 字段，提供了表格线条的精确坐标（pdfplumber 坐标系）：

```json
{
  "grid": {
    "h_lines": [121.2, 156.3, 191.4, ...],
    "v_lines": [39.4, 99.4, 370.2, ...],
    "page_height": 841.9,
    "page_width": 595.3
  }
}
```

Agent 应利用 `h_lines`（水平线 y 坐标）和 `v_lines`（竖线 x 坐标）来精确计算每个单元格的边界，确保：
1. **白色遮盖矩形**严格在线条之间，不覆盖边框线（建议向内缩进 1.5pt）
2. **文本位置**在单元格内部适当位置

### 白色遮盖 + 重写（clear_rect）

对于已有值的纯文本 PDF，需要先用白色矩形遮盖旧值，再叠加新文本。每个 fill 项支持 `clear_rect` 字段：

```json
{
  "mode": "text_overlay",
  "fills": [
    {
      "page": 1,
      "x": 105,
      "y": 693,
      "value": "新剧名",
      "font_size": 12,
      "clear_rect": {
        "x0": 100.9,
        "y0": 687.1,
        "x1": 368.7,
        "y1": 719.2
      },
      "value_status": "has_value",
      "force": true
    }
  ]
}
```

`clear_rect` 坐标为 **PDF 坐标系**（左下角原点）。计算方法：
- `x0` = 左侧竖线 x + margin（如 99.4 + 1.5 = 100.9）
- `y0` = page_height - 下方水平线 y - margin（如 841.9 - 156.3 + 1.5 = 687.1）
- `x1` = 右侧竖线 x - margin（如 370.2 - 1.5 = 368.7）
- `y1` = page_height - 上方水平线 y + margin（如 841.9 - 121.2 - 1.5 = 719.2）

> 如果只需要遮盖旧值而不写入新值（清空字段），设置 `"value": ""` 即可。

### 利用 parser 输出的精确坐标（推荐）

`pdf_parser.py` 输出的每个 `pdf_table_pair` 字段中，`location` 包含 `pdf_coords` 字段，提供了**可直接用于 text_overlay 的精确坐标**：

```json
{
  "field_name": "作品名称",
  "location": {
    "type": "pdf_table_cell",
    "page": 2,
    "row": 23,
    "col": 2,
    "value_rect": {"x0": 128.2, "top": 477.6, "x1": 322.4, "bottom": 515.7},
    "pdf_coords": {
      "x": 129.7,
      "y": 696.3,
      "cell_width": 191.2,
      "cell_height": 35.1,
      "original_font_size": 12.0,
      "clear_rect": {"x0": 129.7, "y0": 327.7, "x1": 320.9, "y1": 362.9}
    }
  },
  "pattern": "pdf_table_pair",
  "current_value": "",
  "value_status": "empty"
}
```

Agent 在 Phase 4 映射时，应**直接使用 `pdf_coords.x` 和 `pdf_coords.y`** 作为 text_overlay 的坐标，**不需要手动计算或估算坐标**。

#### 推荐：直接传入 pdf_coords 对象（最简方式）

> ✅ **`pdf_filler.py` 支持在 fill 项中直接传入 `pdf_coords` 对象**，filler 会自动从中提取 `x`、`y`、`original_font_size`（作为 `font_size`）、`cell_width`（作为 `max_width`）、`cell_height`、`clear_rect` 等信息。Agent **不需要手动拆解这些字段**，只需把 parser 输出的 `pdf_coords` 原样传入即可。

```json
{
  "mode": "text_overlay",
  "fills": [
    {
      "page": 2,
      "pdf_coords": {
        "x": 129.7,
        "y": 696.3,
        "cell_width": 191.2,
        "cell_height": 35.1,
        "original_font_size": 12.0,
        "clear_rect": {"x0": 129.7, "y0": 327.7, "x1": 320.9, "y1": 362.9}
      },
      "value": "北京腾讯影业有限公司",
      "value_status": "empty"
    }
  ]
}
```

> 使用 `pdf_coords` 直传时，filler 会自动处理：
> - 坐标定位（`x`、`y`）
> - 字号匹配（`original_font_size`）
> - 长文本自动换行（`cell_width` → `max_width`）
> - 超高自动缩小字号（`cell_height`）
> - 白色遮盖旧值（`clear_rect`）
>
> Agent 仍然可以在 fill 项中显式设置 `x`、`y`、`font_size`、`max_width`、`cell_height`、`clear_rect` 来覆盖 `pdf_coords` 中的值。

#### ⚠️ fill_data.json 格式要点（常见错误提醒）

> **`pdf_filler.py` 已做兼容处理**，但 Agent 仍应尽量生成规范格式以避免歧义。

| 要点 | 正确 ✅ | 错误 ❌ | 说明 |
|------|---------|---------|------|
| 顶层键名 | `"fills": [...]` | `"fields": [...]` | filler 已兼容 `fields`，但规范键名是 `fills` |
| `page` 位置 | 顶层 `"page": 2` | 仅在 `location.page` 中 | filler 已兼容从 `location.page` 自动提取，但建议放顶层 |
| `pdf_coords` 位置 | 顶层 `"pdf_coords": {...}` | 仅在 `location.pdf_coords` 中 | filler 已兼容从 `location.pdf_coords` 自动提取，但建议放顶层 |
| `value` 字段 | 必须有 `"value": "xxx"` | 缺少 value | 没有 value 则不会写入任何文本 |

**最简规范格式**（推荐 Agent 生成此格式）：

```json
{
  "mode": "text_overlay",
  "fills": [
    {
      "page": 1,
      "pdf_coords": { "x": 101.6, "y": 698.6, "cell_width": 266.3, "cell_height": 31.9 },
      "value": "逐玉",
      "value_status": "empty"
    }
  ]
}
```

**也兼容的格式**（parser 输出风格，filler 会自动提取）：

```json
{
  "mode": "text_overlay",
  "fills": [
    {
      "field_name": "剧名",
      "value": "逐玉",
      "location": {
        "type": "pdf_table_cell",
        "page": 1,
        "pdf_coords": { "x": 101.6, "y": 698.6, "cell_width": 266.3, "cell_height": 31.9 }
      },
      "value_status": "empty"
    }
  ]
}
```

#### 字号选择

- 如果 `pdf_coords` 中包含 `original_font_size`，**必须使用该字号**，以保持与原始 PDF 风格一致
- 如果没有 `original_font_size`，默认使用 `10.5`

#### 长文本自动换行

当填写内容可能超出单元格宽度时，在 fill 项中传入 `max_width` 和 `cell_height`（直接使用 `pdf_coords.cell_width` 和 `pdf_coords.cell_height`），`pdf_filler.py` 会自动：
1. 按字符拆分为多行
2. 如果行数超出单元格高度，自动缩小字号（最小 6pt）

```json
{
  "page": 1,
  "x": 139.0,
  "y": 441.1,
  "value": "出品：北京腾讯影业有限公司 联合出品：上海拾谷影业有限公司",
  "font_size": 12,
  "max_width": 426.8,
  "cell_height": 54.8
}
```

### 准备填写数据

```json
{
  "mode": "text_overlay",
  "fills": [
    {
      "page": 1,
      "x": 200,
      "y": 700,
      "value": "张三",
      "font_size": 12
    },
    {
      "page": 1,
      "x": 200,
      "y": 650,
      "value": "13800138000",
      "font_size": 12
    }
  ]
}
```

### 执行填写

```bash
cd {skill_base_dir}/scripts && python3 pdf_filler.py /path/to/template.pdf /path/to/fill_data.json /path/to/output.pdf
```

---

---

## 四点五、展示填写计划并等待用户确认（Phase 4.5）

Agent 生成 `fill_data.json` 后，**必须**先向用户展示填写计划并等待确认，**不得直接执行填写**。

```bash
# 保存填写数据（暂不上传，等用户确认后再上传）
cat > /tmp/fill_data.json << 'FILL_EOF'
<Agent 生成的 fill_data.json 内容>
FILL_EOF
```

在回复中输出：
1. 填写计划摘要表格 — 以人类可读的表格展示字段和值
2. 确认提示 — 询问用户是否确认执行填写

> ⛔ **绝对禁止在用户确认之前执行填写。** 必须等待用户明确确认后才能继续。

用户确认后，在调用 `pdf_filler.py` 前，**必须**先上传最终版 `fill_data.json`：

```bash
# 上传最终确认版填写数据
cd {skill_base_dir}/scripts && python3 upload_file.py /tmp/fill_data.json --name "模版名_填写数据_$(date +%s).json"
```

上传成功后，从 `upload_file.py` 的标准输出 JSON 中提取 `fid` 字段的**实际值**，在回复中展示。

> ⛔ **`fid` 必须是 `upload_file.py` 返回的真实值**（如 `fid_5f81b9b77acdcf6bce18`），**绝对禁止**使用占位符（如 `fid_xxx`、`<fid>`、`fid_xxxxxxxx`）。
> ⛔ **不需要调用 `download_url.py` 获取下载链接**，只需展示 `fid` 即可。

---

## 关键约束

1. **PDF 不可直接修改**：与 xlsx/docx 不同，PDF 的文本叠加模式是在原页面上方添加一个透明层
2. **白色遮盖**：对于已有值的字段，使用 `clear_rect` 先用白色矩形遮盖旧值，再叠加新文本。遮盖矩形必须严格在表格线条之间，避免覆盖边框线
3. **中文字体（⚠️ 关键）**：`text_overlay` 模式**必须有中文字体**才能正确显示中文。脚本会按以下顺序尝试注册字体：
   - 系统常见路径的中文字体文件（文泉驿、Noto、苹方等）
   - 系统字体目录中的 CJK 字体文件（glob 搜索）
   - reportlab 内置 CIDFont（STSong-Light）
   
   **⚠️ 如果以上三种方式都找不到中文字体，且填写内容包含中文字符，脚本会直接报错退出（退出码 2），不会生成乱码文件。** Agent 看到此错误后，**必须先安装中文字体再重试**：
   ```bash
   # Debian/Ubuntu
   apt-get install -y fonts-wqy-zenhei
   # CentOS/RHEL
   yum install -y wqy-zenhei-fonts
   # 或安装 Noto CJK
   apt-get install -y fonts-noto-cjk
   ```
4. **坐标精度**：文本叠加模式需要精确的坐标，Agent **必须直接使用** `pdf_parser.py` 输出的 `pdf_coords` 中的坐标，**绝对禁止自行估算坐标**
5. **字号匹配**：必须使用 `pdf_coords.original_font_size`（如果存在）作为 `font_size`，以保持与原始 PDF 风格一致。如果不存在，默认使用 `10.5`
6. **长文本换行**：当填写内容可能超出单元格宽度时，必须在 fill 项中传入 `max_width`（使用 `pdf_coords.cell_width`）和 `cell_height`（使用 `pdf_coords.cell_height`），`pdf_filler.py` 会自动换行和缩小字号
7. **表单域模式优先**：如果 PDF 有可填写表单域，优先使用 `form_fields` 模式，效果更好
8. **已有值被误识别为标签（⚠️ 关键）**：当 PDF 表格中某个字段已经有值时，pdfplumber 可能会把已有值当作独立的标签文本。Parser 会通过 `may_be_value_of` 字段标记这类疑似情况，但 **Agent 在 Phase 4 智能映射时必须主动识别和处理这类噪音字段**。详见下方「已有值字段的识别与处理」章节
9. **避免多次叠加**：每次 `text_overlay` 操作都会在 PDF 上添加一个新的叠加层。如果需要修改已经叠加过的 PDF，应使用**原始模版文件**重新生成，而不是在已叠加的文件上再次叠加（否则白色遮盖层可能覆盖新写入的文本）

---

## 已有值字段的识别与处理（⚠️ Agent 必读）

### 问题背景

PDF 表格中，当某个字段已经被填写过（有已有值）时，`pdf_parser.py` 可能会产生**误识别**：将已有值当作独立的标签名。这是因为 pdfplumber 对合并单元格的处理会把一个逻辑单元格拆成多个物理列，导致已有值文本被当作新的标签。

### 典型场景

原始 PDF 表格结构（视觉上）：

```
┌──────────┬──────────────────────────────┐
│  剧 名   │  值得爱                       │
└──────────┴──────────────────────────────┘
```

pdfplumber 解析后的表格数据（4 列）：

```
row[0] = ["剧 名", "", "值得爱", ""]
           col 0   col 1   col 2    col 3
```

Parser 的标签-值配对逻辑会产生两个字段：
1. `field_name: "剧 名"`, `col: 1`, `value_status: "empty"` — ✅ 正确的标签
2. `field_name: "值得爱"`, `col: 3`, `value_status: "empty"` — ❌ 这是"剧名"的已有值，被误识别为标签

### Parser 的自动检测（`may_be_value_of`）

`pdf_parser.py` 会在后处理阶段检测这类情况，并在疑似字段上添加 `may_be_value_of` 标记：

```json
{
  "field_name": "值得爱",
  "may_be_value_of": "剧 名",
  "_confidence": "low",
  "location": { "type": "pdf_table_cell", "page": 1, "row": 0, "col": 3 },
  "pattern": "pdf_table_pair",
  "value_status": "empty"
}
```

### Agent 在 Phase 4 的处理规则

Agent 在智能映射阶段，**必须**按以下规则处理 parser 输出的字段：

#### 规则 1：识别 `may_be_value_of` 标记

如果字段包含 `may_be_value_of`，说明 parser 认为该字段可能是另一个标签的已有值。Agent 应：
1. **不要将此字段当作独立的待填字段**
2. **将其视为 `may_be_value_of` 所指标签的当前值**
3. 如果用户要求更新该标签的值，需要在 fill_data.json 中：
   - 对**真正的标签字段**（如"剧 名"）设置新值，并使用该字段的 `pdf_coords`
   - 同时用 `clear_rect` 覆盖**被误识别字段**（如"值得爱"）所在的区域
   - 设置 `value_status: "has_value"` 和 `force: true`

#### 规则 2：主动语义判断（即使没有 `may_be_value_of`）

Parser 的自动检测不能覆盖所有情况。Agent 应**主动**根据以下线索判断一个"标签"是否实际上是已有值：

| 线索 | 说明 |
|------|------|
| **字段名不像标签** | 标签通常是通用描述词（如"剧名"、"导演"、"出品公司"），而已有值通常是具体的名称（如"值得爱"、"孙皓"、"光线影业"） |
| **同行有空值标签** | 如果同一行中有另一个标签的值为空，且当前"标签"紧邻其右侧，则当前文本很可能是那个标签的值 |
| **字段名与用户数据匹配** | 如果"标签名"恰好与用户要替换的旧值一致，说明它是已有值而非标签 |
| **位置关系** | 当前"标签"的 `label_col` 等于或紧邻前一个字段的 `col`（值列） |

#### 规则 3：生成正确的 fill_data.json

当需要替换已有值时，fill_data.json 应这样构造：

**场景**：将"剧名"从"值得爱"改为"白日提灯"

```json
{
  "mode": "text_overlay",
  "fills": [
    {
      "page": 1,
      "pdf_coords": {
        "x": 100.9,
        "y": 698.9,
        "cell_width": 267.8,
        "cell_height": 32.1,
        "clear_rect": {
          "x0": 100.9,
          "y0": 687.1,
          "x1": 368.7,
          "y1": 719.2
        },
        "original_font_size": 12.0
      },
      "value": "白日提灯",
      "value_status": "has_value",
      "force": true
    },
    {
      "page": 1,
      "pdf_coords": {
        "x": 371.7,
        "y": 698.2,
        "cell_width": 94.0,
        "cell_height": 32.1,
        "clear_rect": {
          "x0": 371.7,
          "y0": 687.1,
          "x1": 465.6,
          "y1": 719.2
        }
      },
      "value": "",
      "value_status": "has_value",
      "force": true
    }
  ]
}
```

**关键点**：
1. **第一个 fill**：在真正的标签（"剧 名"）的值区域写入新值"白日提灯"，使用 `clear_rect` 清除旧内容
2. **第二个 fill**：清空被误识别字段（"值得爱"）所在区域，`value` 设为空字符串，仅执行白色遮盖
3. 两个 fill 都必须设置 `value_status: "has_value"` 和 `force: true`
4. 如果新值较长，可能需要合并两个区域的 `clear_rect`（将 `x1` 扩展到第二个区域的右边界）

#### 规则 4：合并相邻区域（可选优化）

如果标签的值区域和被误识别字段的区域在物理上相邻（同一行、x 坐标连续），Agent 可以将两个区域合并为一个更大的填写区域：

```json
{
  "page": 1,
  "pdf_coords": {
    "x": 100.9,
    "y": 698.9,
    "cell_width": 363.2,
    "cell_height": 32.1,
    "clear_rect": {
      "x0": 100.9,
      "y0": 687.1,
      "x1": 465.6,
      "y1": 719.2
    },
    "original_font_size": 12.0
  },
  "value": "白日提灯",
  "value_status": "has_value",
  "force": true
}
```

> 合并后 `cell_width` = 第一个区域 x0 到第二个区域 x1 的总宽度，`clear_rect` 覆盖整个合并区域。
