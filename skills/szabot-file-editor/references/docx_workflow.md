# DOCX 模版解析与填写工作流

## 概述

本工作流处理 docx 类型的模版文件，包括模版结构解析和精准填写。

使用 `python-docx` 库操作 docx 文件，采用与 xlsx 相同的「先拷贝、再修改」策略保留格式。

## 内置脚本

| 脚本 | 用途 |
|------|------|
| `scripts/docx_parser.py` | 解析 docx 模版结构，提取待填字段和位置 |
| `scripts/docx_filler.py` | 将数据精准写入 docx 模版的指定位置 |

---

## 一、模版解析（Phase 3）

### 执行命令

```bash
cd {skill_base_dir}/scripts && python3 docx_parser.py /path/to/template.docx
```

### 识别模式

| 模式 | 说明 | 示例 |
|------|------|------|
| `table_horizontal_pair` | 表格中水平配对：左列标签，右列空/占位符/示例值/已有值 | 表格第1列="姓名", 第2列="" |
| `table_with_empty_rows` | 表格数据区域：表头行 + 空/示例数据行 | 第0行是表头，1-3行为空 |
| `paragraph_placeholder` | 段落中的占位符文本 | `{{项目名称}}` |
| `paragraph_label_value` | 段落中的"标签：值"模式（冒号后有占位符或已有值） | `项目名称：____`、`*制作国家/地区：中国` |
| `paragraph_label_empty` | 段落中的"标签："模式（冒号结尾，值为空） | `*中文片名：`、`*导演：` |

### 字段值状态分类（value_status）

每个解析出的字段都会带有 `value_status` 标记，用于区分当前值的类型：

| value_status | 含义 | 示例 | 建议操作 |
|-------------|------|------|----------|
| `empty` | 单元格为空 | `""` | 直接填写 |
| `placeholder` | 占位符文本 | `{{项目名称}}`、`____`、`请填写` | 直接填写 |
| `example` | 示例值（无用的占位示例） | `某某公司`、`XXX`、`如：张三` | 删除后填写 |
| `has_value` | 已有有效值 | `北京光线影业有限公司` | **默认跳过，保留原值** |

> ❗ **重要**：Agent 在 Phase 4 智能映射时，应根据 `value_status` 决定是否填写。对于 `has_value` 的字段，应询问用户是否需要覆盖，或在 fill_data.json 中设置 `"force": true` 强制覆盖。

### 位置定位方式

docx 中的位置通过 **索引** 定位（而非 xlsx 的单元格坐标）：

- **表格单元格**：`table_index` + `row` + `col`
- **段落**：`paragraph_index` + `match_text`（用于替换）

---

## 一点五、上传解析结果（Phase 3.5）

模版解析完成后，**必须**将 parser 输出的 JSON 结果保存为本地临时文件并上传，在回复中展示 `fid`。

```bash
# 保存解析结果
cd {skill_base_dir}/scripts && python3 docx_parser.py /path/to/template.docx > /tmp/parser_result.json

# 上传
cd {skill_base_dir}/scripts && python3 upload_file.py /tmp/parser_result.json --name "模版名_解析结果_$(date +%s).json"
```

上传成功后，从 `upload_file.py` 的标准输出 JSON 中提取 `fid` 字段的**实际值**，在回复中展示。

> ⛔ **`fid` 必须是 `upload_file.py` 返回的真实值**（如 `fid_5f81b9b77acdcf6bce18`），**绝对禁止**使用占位符（如 `fid_xxx`、`<fid>`、`fid_xxxxxxxx`）。
> ⛔ **不需要调用 `download_url.py` 获取下载链接**，只需展示 `fid` 即可。

---

## 二、模版填写（Phase 5）

> ⛔ **Phase 4→Phase 5 衔接要点**：Agent 在 Phase 4 完成智能映射后，**必须**按照以下格式构造 `fill_data.json`，而非生成简单的 `{"字段名": "值"}` 扁平字典。具体要求：
> 1. **复用 parser 输出的 `location` 信息**：每个填写项的 `location`（含 `type`、`table_index`/`paragraph_index`、`row`/`col`/`match_text`）必须直接来自 Phase 3 parser 的输出，不可自行编造
> 2. **传递 `value_status` 和 `pattern`**：这两个字段决定了 filler 的填写策略（追加 vs 替换 vs 跳过），必须从 parser 输出中原样传递
> 3. **仅替换 `value`**：Agent 只需将 parser 输出中的 `value`（或空值）替换为映射后的实际数据
> 4. **`has_value` 字段需要 `force`**：如果需要覆盖已有值，必须显式设置 `"force": true`

### 准备填写数据

```json
{
  "fills": [
    {
      "location": {"type": "table_cell", "table_index": 0, "row": 1, "col": 2},
      "value": "某某影视公司",
      "value_status": "empty"
    },
    {
      "location": {"type": "paragraph", "paragraph_index": 5, "match_text": "{{项目名称}}"},
      "value": "庆余年第三季",
      "value_status": "placeholder",
      "pattern": "paragraph_placeholder"
    },
    {
      "location": {"type": "paragraph", "paragraph_index": 13, "match_text": "*中文片名："},
      "value": "逐玉",
      "value_status": "empty",
      "pattern": "paragraph_label_empty"
    },
    {
      "location": {"type": "paragraph", "paragraph_index": 15, "match_text": "*节目集数：（）集"},
      "value": "30集",
      "value_status": "placeholder",
      "pattern": "paragraph_label_value"
    },
    {
      "location": {"type": "table_cell", "table_index": 0, "row": 3, "col": 2},
      "value": "孙皓",
      "value_status": "example"
    },
    {
      "location": {"type": "table_cell", "table_index": 0, "row": 5, "col": 2},
      "value": "张三",
      "value_status": "has_value",
      "force": true
    }
  ],
  "table_fills": [
    {
      "table_index": 1,
      "header_row": 0,
      "data_rows": [
        {"row": 1, "values": {"0": "张若昀", "1": "范闲", "2": "主演"}}
      ]
    }
  ]
}
```

### value_status 与 force 字段说明

| 字段 | 说明 |
|------|------|
| `value_status` | 来自 parser 的值状态分类，可选值：`empty`/`placeholder`/`example`/`has_value` |
| `pattern` | 来自 parser 的字段模式，可选值：`paragraph_label_empty`/`paragraph_label_value`/`paragraph_placeholder`/`table_horizontal_pair` |
| `force` | 当 `value_status="has_value"` 时，设置 `true` 可强制覆盖已有值 |

处理逻辑：
- `empty`/`placeholder`/`example`：直接填写，无需 force
- `has_value` + `force=false`（默认）：**跳过，保留原值**
- `has_value` + `force=true`：强制覆盖原值

> ❗ **重要：段落填写时必须传递 `pattern` 字段**：对于 `paragraph_label_empty` 和 `paragraph_label_value` 模式的字段，filler 会智能地在冒号后面追加值，而不是替换整个段落文本，从而保留标签名称。例如 `*中文片名：` 填写后变为 `*中文片名：逐玉`，而不是 `逐玉`。
>
> ⛔ **缺少 `pattern` 字段的后果**：如果 Agent 未传递 `pattern` 字段，`docx_filler.py` 会尝试通过检测 `match_text` 中的冒号来兜底识别标签模式，但这不是 100% 可靠的。Agent **必须**从 parser 输出中原样复制 `pattern` 字段到 fill_data.json 中，确保 filler 能正确处理每个字段。
>
> **✅ 正确示例**（包含 pattern）：
> ```json
> {"location": {"type": "paragraph", "paragraph_index": 13, "match_text": "*中文片名："}, "value": "逐玉", "pattern": "paragraph_label_empty", "value_status": "empty"}
> ```
> **❌ 错误示例**（缺少 pattern，可能导致标签丢失）：
> ```json
> {"location": {"type": "paragraph", "paragraph_index": 13, "match_text": "*中文片名："}, "value": "逐玉"}
> ```

### 执行填写

```bash
cd {skill_base_dir}/scripts && python3 docx_filler.py /path/to/template.docx /path/to/fill_data.json /path/to/output.docx
```

---

## 二点五、展示填写计划并等待用户确认（Phase 4.5）

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

> ⛔ **绝对禁止在用户确认之前执行填写。** Agent 在输出填写计划和确认提示后，**必须立即结束当前回复**，等待用户在**新的一条消息**中明确回复确认后，才能在**下一轮回复**中继续执行填写。**绝对禁止**在同一轮回复中既展示填写计划又执行填写。

用户确认后，在调用 `docx_filler.py` 前，**必须**先上传最终版 `fill_data.json`：

```bash
# 上传最终确认版填写数据
cd {skill_base_dir}/scripts && python3 upload_file.py /tmp/fill_data.json --name "模版名_填写数据_$(date +%s).json"
```

上传成功后，从 `upload_file.py` 的标准输出 JSON 中提取 `fid` 字段的**实际值**，在回复中展示。

> ⛔ **`fid` 必须是 `upload_file.py` 返回的真实值**（如 `fid_5f81b9b77acdcf6bce18`），**绝对禁止**使用占位符（如 `fid_xxx`、`<fid>`、`fid_xxxxxxxx`）。
> ⛔ **不需要调用 `download_url.py` 获取下载链接**，只需展示 `fid` 即可。

---

## 关键约束

### 格式保留策略

与 xlsx 相同，`docx_filler.py` 采用**「先拷贝、再修改」**策略：

1. 先用 `shutil.copy2()` 完整拷贝模版
2. 再用 `python-docx` 打开副本修改
3. 修改时保留第一个 run 的格式（字体、大小、粗体等）

### 文本替换规则

- **表格单元格**：保留第一个 run 的格式，清空其余 run
- **段落占位符**（`paragraph_placeholder`）：精确替换 `match_text`，保留前后文本和格式
- **段落标签空值**（`paragraph_label_empty`）：在冒号后面追加值，保留标签名称和冒号
- **段落标签有值**（`paragraph_label_value`）：将冒号后面的内容替换为新值，保留标签名称和冒号
- **跨 run 占位符**：自动处理占位符跨越多个 run 的情况

### 示例值检测规则

以下模式会被识别为示例值（`value_status="example"`）：

| 模式 | 示例 |
|------|------|
| XX/xxx/×× | `XX公司`、`xxx` |
| 某某/某X | `某某影视公司`、`某某某` |
| 如：/例：/示例 | `如：张三`、`示例文本` |
| (示例)/（样本） | `张三(示例)` |
| 此处填写/请输入 | `此处填写姓名`、`请输入内容` |
