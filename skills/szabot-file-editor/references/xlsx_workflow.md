# XLSX 模版解析与填写工作流

## 概述

本工作流处理 xlsx 类型的模版文件，包括模版结构解析、数据提取（作为内容文件时）、以及精准填写。

依赖 `xlsx` skill 提供的 openpyxl 库和 recalc.py 公式重算能力。

## 内置脚本

| 脚本 | 用途 |
|------|------|
| `scripts/xlsx_parser.py` | 解析 xlsx 模版结构，提取待填字段和位置 |
| `scripts/xlsx_filler.py` | 将数据精准写入 xlsx 模版的指定单元格 |
| `scripts/content_extractor.py` | 从 xlsx 内容文件中提取结构化数据（模式 C） |

---

## 一、模版解析（Phase 3）

### 执行命令

```bash
cd {skill_base_dir}/scripts && python3 xlsx_parser.py /path/to/template.xlsx
```

### 输出格式

```json
{
  "template_type": "xlsx",
  "file_name": "电视剧项目信息首集表.xlsx",
  "sheets": [
    {"name": "Sheet1", "max_row": 50, "max_column": 10}
  ],
  "fields": [
    {
      "field_name": "项目名称",
      "location": "B2",
      "sheet": "Sheet1",
      "pattern": "horizontal_pair",
      "label_location": "A2",
      "current_value": "",
      "style": {"font_name": "宋体", "font_size": 12}
    }
  ],
  "tables": [
    {
      "sheet": "Sheet1",
      "header_row": 10,
      "headers": [
        {"column": 1, "coordinate": "A10", "header": "姓名"},
        {"column": 2, "coordinate": "B10", "header": "角色"},
        {"column": 3, "coordinate": "C10", "header": "备注"}
      ],
      "empty_data_rows": [11, 12, 13],
      "pattern": "table_with_empty_rows"
    }
  ],
  "summary": {
    "total_sheets": 1,
    "total_fields": 25,
    "total_tables": 2
  }
}
```

### 识别模式说明

| 模式 | 说明 | 示例 |
|------|------|------|
| `horizontal_pair` | 水平配对：左边是标签，右边是空/占位符 | A2="项目名称", B2="" |
| `placeholder` | 占位符单元格：包含 `{{xxx}}`、`____`、`请填写` 等 | B5="{{导演}}" |
| `table_with_empty_rows` | 表格区域：表头行 + 空数据行 | 第10行是表头，11-13行为空 |

### 多 Sheet 支持

解析脚本会**自动遍历所有工作表**，每个字段和表格都带有 `sheet` 属性标识所属工作表。

**典型多 Sheet 模版示例**（如"北京大视听摄制服务申报表"）：

| Sheet | 内容 | 解析结果 |
|-------|------|---------|
| 申报表1 | 表单字段（片名、题材、主创人员等） | `fields` 中的 `horizontal_pair` 字段 |
| 申报表2 | 取景地信息表格 | `tables` 中的 `table_with_empty_rows` |
| 申报材料 | 说明文档（不需要填写） | 可能产生误识别，Agent 应忽略 |

**Agent 处理多 Sheet 的要点**：

1. 解析结果中的 `fields` 和 `tables` 来自所有 Sheet，通过 `sheet` 字段区分
2. 生成 `fill_data.json` 时，每个 `fill` 项的 `sheet` 必须与解析结果中的 `sheet` 名称**完全一致**
3. 对于纯说明性的 Sheet（如"申报材料"），其中的字段应**跳过不填**
4. 表格类型的 Sheet（如"申报表2"），使用 `table_fills` 方式填写数据行

**多 Sheet 填写数据示例**：

```json
{
  "fills": [
    {"location": "B4", "sheet": "申报表1", "value": "某某影视公司"},
    {"location": "B8", "sheet": "申报表1", "value": "值得爱"}
  ],
  "table_fills": [
    {
      "sheet": "申报表2",
      "header_row": 3,
      "data_rows": [
        {"row": 4, "values": {"A": "1", "B": "故宫", "C": "北京市东城区", "D": "2025-06-01"}},
        {"row": 5, "values": {"A": "2", "B": "颐和园", "C": "北京市海淀区", "D": "2025-06-15"}}
      ]
    }
  ]
}
```

### Agent 后续处理

脚本输出的是**结构化的字段清单**，Agent 需要：

1. 阅读字段清单，理解每个字段的含义
2. 对于 `field_name` 不够明确的字段（如 `占位符_C5`），结合上下文推断含义
3. 将字段清单与数据源进行映射（Phase 4）

---

## 二、内容文件解析（Phase 2 模式 C）

当 xlsx 文件作为**数据源**（而非模版）时，使用 `content_extractor.py` 提取数据。

### 执行命令

```bash
cd {skill_base_dir}/scripts && python3 content_extractor.py /path/to/content_file.xlsx
```

### 输出格式

```json
{
  "source_type": "xlsx",
  "file_name": "庆余年三季项目资料.xlsx",
  "key_value_pairs": [
    {
      "key": "项目名称",
      "value": "庆余年第三季",
      "sheet": "Sheet1",
      "location": "A1:B1"
    }
  ],
  "tables": [
    {
      "sheet": "Sheet1",
      "header_row": 5,
      "headers": ["姓名", "角色", "备注"],
      "rows": [
        {"姓名": "张若昀", "角色": "范闲", "备注": "主演"},
        {"姓名": "李沁", "角色": "林婉儿", "备注": "主演"}
      ]
    }
  ]
}
```

---

## 二点五、上传解析结果（Phase 3.5）

模版解析完成后，**必须**将 parser 输出的 JSON 结果保存为本地临时文件并上传，在回复中展示 `fid`。

```bash
# 保存解析结果
cd {skill_base_dir}/scripts && python3 xlsx_parser.py /path/to/template.xlsx > /tmp/parser_result.json

# 上传
cd {skill_base_dir}/scripts && python3 upload_file.py /tmp/parser_result.json --name "首集表_解析结果_$(date +%s).json"
```

上传成功后，从 `upload_file.py` 的标准输出 JSON 中提取 `fid` 字段的**实际值**，在回复中展示。

> ⛔ **`fid` 必须是 `upload_file.py` 返回的真实值**（如 `fid_5f81b9b77acdcf6bce18`），**绝对禁止**使用占位符（如 `fid_xxx`、`<fid>`、`fid_xxxxxxxx`）。
> ⛔ **不需要调用 `download_url.py` 获取下载链接**，只需展示 `fid` 即可。

---

## 三、模版填写（Phase 5）

> ⛔ **Phase 4→Phase 5 衔接要点**：Agent 在 Phase 4 完成智能映射后，**必须**按照以下格式构造 `fill_data.json`，而非生成简单的 `{"字段名": "值"}` 扁平字典。具体要求：
> 1. **复用 parser 输出的 `location` 信息**：每个填写项的 `location`（单元格坐标如 `B2`）和 `sheet`（工作表名）必须直接来自 Phase 3 parser 的输出，不可自行编造
> 2. **传递 `value_status`**：该字段决定了 filler 的填写策略（直接填写 vs 跳过），必须从 parser 输出中原样传递
> 3. **仅替换 `value`**：Agent 只需将 parser 输出中的 `value`（或空值）替换为映射后的实际数据
> 4. **`has_value` 字段需要 `force`**：如果需要覆盖已有值，必须显式设置 `"force": true`
> 5. ⛔ **禁止创造 parser 输出中不存在的字段**：`fill_data.json` 中的每一个 fill 项必须对应 parser 输出 `fields` 列表中的一个字段。如果数据源中有字段在 parser 输出中找不到对应的模版字段，应标记为 unmapped，**绝对禁止**自行编造 `location` 坐标。特别注意：合并单元格（如 `merged_range: "B8:B9"`）只有左上角 `B8` 是有效位置，`B9` 不是独立字段，不得将其他数据映射到 `B9`，否则 filler 会将其修正为 `B8` 并覆盖已有值

### 准备填写数据

Agent 完成字段映射后，需要生成 `fill_data.json` 文件：

```json
{
  "fills": [
    {
      "location": "B2",
      "sheet": "Sheet1",
      "value": "庆余年第三季"
    },
    {
      "location": "B3",
      "sheet": "Sheet1",
      "value": "孙皓"
    }
  ],
  "table_fills": [
    {
      "sheet": "Sheet1",
      "header_row": 10,
      "data_rows": [
        {"row": 11, "values": {"A": "张若昀", "B": "范闲", "C": "主演"}},
        {"row": 12, "values": {"A": "李沁", "B": "林婉儿", "C": "主演"}}
      ]
    }
  ]
}
```

### 执行填写

```bash
cd {skill_base_dir}/scripts && python3 xlsx_filler.py /path/to/template.xlsx /path/to/fill_data.json /path/to/output.xlsx
```

### 输出

```json
{
  "template": "电视剧项目信息首集表.xlsx",
  "output": "电视剧项目信息首集表_filled.xlsx",
  "filled_cells": 22,
  "filled_rows": 5,
  "status": "success"
}
```

---

## 三点五、展示填写计划并等待用户确认（Phase 4.5）

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

用户确认后，在调用 `xlsx_filler.py` 前，**必须**先上传最终版 `fill_data.json`：

```bash
# 上传最终确认版填写数据
cd {skill_base_dir}/scripts && python3 upload_file.py /tmp/fill_data.json --name "首集表_填写数据_$(date +%s).json"
```

上传成功后，从 `upload_file.py` 的标准输出 JSON 中提取 `fid` 字段的**实际值**，在回复中展示。

> ⛔ **`fid` 必须是 `upload_file.py` 返回的真实值**（如 `fid_5f81b9b77acdcf6bce18`），**绝对禁止**使用占位符（如 `fid_xxx`、`<fid>`、`fid_xxxxxxxx`）。
> ⛔ **不需要调用 `download_url.py` 获取下载链接**，只需展示 `fid` 即可。

---

## 四、公式重算（可选）

如果模版中包含公式，填写数据后需要重算：

```bash
cd {xlsx_skill_base_dir}/scripts && python recalc.py /path/to/output.xlsx
```

> ⚠️ 此步骤使用 `xlsx` skill 的 `recalc.py` 脚本，需要 LibreOffice 环境。

---

## 五、完整流程示例

### 场景：用"庆余年第三季"项目信息填写首集表

```bash
# 1. 下载模版文件到本地（如果需要）
cd {skill_base_dir}/scripts && python3 download_url.py fid_xxx
curl -o /tmp/首集表.xlsx "<download_url>"

# 2. 解析模版结构
cd {skill_base_dir}/scripts && python3 xlsx_parser.py /tmp/首集表.xlsx
# → 输出字段清单 JSON

# 3. Agent 调用 szabot-copilot 查询项目信息（模式 A）
# → 获取项目数据

# 4. Agent 进行字段映射，生成 fill_data.json
# → 写入 /tmp/fill_data.json

# 5. 执行填写
cd {skill_base_dir}/scripts && python3 xlsx_filler.py /tmp/首集表.xlsx /tmp/fill_data.json /tmp/首集表_filled.xlsx

# 6. 可选：公式重算
cd {xlsx_skill_base_dir}/scripts && python recalc.py /tmp/首集表_filled.xlsx

# 7. 上传结果文件并输出链接（强制步骤）
cd {skill_base_dir}/scripts && python3 upload_file.py /tmp/首集表_filled.xlsx --name "首集表_填写结果_$(date +%s).xlsx"
# → 从返回 JSON 中提取 fid，输出 [结果文件](bvnext://x-callback-url/doc?fid=<fid>)
```

---

## 关键约束

### ⚠️ 格式保留策略（最重要）

`xlsx_filler.py` 采用**「先拷贝、再修改」**策略：

1. 先用 `shutil.copy2()` 将模版文件**完整拷贝**到输出路径
2. 再用 `openpyxl` 打开**副本**进行修改
3. 修改完成后直接保存副本

这样可以**完整保留**原模版的所有格式：合并单元格、边框、字体、颜色、行高列宽、条件格式等。

> ⛔ **绝对禁止**使用 `load_workbook()` 加载原模版后 `save()` 到新路径的方式——这会导致部分格式信息丢失。
> ⛔ **绝对禁止**大模型自行编写 Python 脚本创建新 xlsx 文件——必须使用 `xlsx_filler.py` 基于模版副本修改。

### 合并单元格处理

模版文件通常包含大量合并单元格（如 `B4:F4` 表示 B4 到 F4 合并为一个单元格）。

**解析阶段**（`xlsx_parser.py`）：
- 自动识别合并单元格区域
- 输出的 `location` 始终是合并区域的**左上角坐标**（如 `B4`）
- 输出中包含 `merged_range` 字段标识合并范围（如 `"B4:F4"`）

**填写阶段**（`xlsx_filler.py`）：
- 自动检测 `location` 是否在合并区域内
- 如果指向合并区域的非左上角单元格，**自动修正**为左上角
- 只向合并区域的左上角单元格写入数据

### 其他约束

1. **不覆盖原模版**：填写结果保存为新文件，原模版文件不被修改
2. **数据类型保留**：数字类型的值应以数字写入（而非字符串），日期类型同理
3. **不传 `data_only=True`**：加载模版时确保保留公式
