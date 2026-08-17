# PPTX 模版解析与填写工作流

## 概述

本工作流处理 pptx 类型的模版文件，包括模版结构解析和精准填写。

使用 `python-pptx` 库操作 pptx 文件，采用与 xlsx 相同的「先拷贝、再修改」策略保留格式。

## 内置脚本

| 脚本 | 用途 |
|------|------|
| `scripts/pptx_parser.py` | 解析 pptx 模版结构，提取待填字段和位置 |
| `scripts/pptx_filler.py` | 将数据精准写入 pptx 模版的指定位置 |

---

## 一、模版解析（Phase 3）

### 执行命令

```bash
cd {skill_base_dir}/scripts && python3 pptx_parser.py /path/to/template.pptx
```

### 识别模式

| 模式 | 说明 | 示例 |
|------|------|------|
| `slide_placeholder` | 幻灯片中的占位符文本 | `{{项目名称}}`、`请填写` |
| `slide_label_value` | 幻灯片中的"标签：____"模式 | `项目名称：____` |
| `slide_table_pair` | 表格中水平配对（左列标签，右列空/占位符/示例值/已有值） | 左列标签，右列空 |
| `slide_table_with_empty_rows` | 表格数据区域 | 表头行 + 空/示例数据行 |

### 字段值状态分类（value_status）

每个解析出的字段都会带有 `value_status` 标记：

| value_status | 含义 | 示例 | 建议操作 |
|-------------|------|------|----------|
| `empty` | 单元格为空 | `""` | 直接填写 |
| `placeholder` | 占位符文本 | `{{项目名称}}`、`____` | 直接填写 |
| `example` | 示例值 | `某某公司`、`XXX` | 删除后填写 |
| `has_value` | 已有有效值 | `北京光线影业` | **默认跳过** |

> 对于 `has_value` 的字段，可在 fill_data.json 中设置 `"force": true` 强制覆盖。

### 位置定位方式

pptx 中的位置通过 **多级索引** 定位：

- **文本框**：`slide_index` + `shape_index` + `paragraph_index`
- **表格单元格**：`slide_index` + `shape_index` + `row` + `col`

---

## 一点五、上传解析结果（Phase 3.5）

模版解析完成后，**必须**将 parser 输出的 JSON 结果保存为本地临时文件并上传，在回复中展示 `fid`。

```bash
# 保存解析结果
cd {skill_base_dir}/scripts && python3 pptx_parser.py /path/to/template.pptx > /tmp/parser_result.json

# 上传
cd {skill_base_dir}/scripts && python3 upload_file.py /tmp/parser_result.json --name "模版名_解析结果_$(date +%s).json"
```

上传成功后，从 `upload_file.py` 的标准输出 JSON 中提取 `fid` 字段的**实际值**，在回复中展示。

> ⛔ **`fid` 必须是 `upload_file.py` 返回的真实值**（如 `fid_5f81b9b77acdcf6bce18`），**绝对禁止**使用占位符（如 `fid_xxx`、`<fid>`、`fid_xxxxxxxx`）。
> ⛔ **不需要调用 `download_url.py` 获取下载链接**，只需展示 `fid` 即可。

---

## 二、模版填写（Phase 5）

> ⛔ **Phase 4→Phase 5 衔接要点**：Agent 在 Phase 4 完成智能映射后，**必须**按照以下格式构造 `fill_data.json`，而非生成简单的 `{"字段名": "值"}` 扁平字典。具体要求：
> 1. **复用 parser 输出的 `location` 信息**：每个填写项的 `location`（含 `type`、`slide_index`、`shape_index`、`paragraph_index` 等）必须直接来自 Phase 3 parser 的输出，不可自行编造
> 2. **传递 `value_status`**：该字段决定了 filler 的填写策略（直接填写 vs 跳过），必须从 parser 输出中原样传递
> 3. **仅替换 `value`**：Agent 只需将 parser 输出中的 `value`（或空值）替换为映射后的实际数据
> 4. **`has_value` 字段需要 `force`**：如果需要覆盖已有值，必须显式设置 `"force": true`

### 准备填写数据

```json
{
  "fills": [
    {
      "location": {
        "type": "text_frame",
        "slide_index": 0,
        "shape_index": 1,
        "paragraph_index": 0
      },
      "value": "庆余年第三季"
    },
    {
      "location": {
        "type": "text_frame",
        "slide_index": 0,
        "shape_index": 2,
        "paragraph_index": 0,
        "match_text": "{{导演}}"
      },
      "value": "孙皓"
    },
    {
      "location": {
        "type": "table_cell",
        "slide_index": 1,
        "shape_index": 0,
        "row": 1,
        "col": 1
      },
      "value": "张若昀"
    }
  ],
  "table_fills": [
    {
      "slide_index": 1,
      "shape_index": 0,
      "data_rows": [
        {"row": 1, "values": {"0": "张若昀", "1": "范闲", "2": "主演"}},
        {"row": 2, "values": {"0": "李沁", "1": "林婉儿", "2": "主演"}}
      ]
    }
  ]
}
```

### 执行填写

```bash
cd {skill_base_dir}/scripts && python3 pptx_filler.py /path/to/template.pptx /path/to/fill_data.json /path/to/output.pptx
```

---

## 关键约束

### 格式保留策略

与 xlsx 相同，`pptx_filler.py` 采用**「先拷贝、再修改」**策略：

1. 先用 `shutil.copy2()` 完整拷贝模版
2. 再用 `python-pptx` 打开副本修改
3. 修改时保留第一个 run 的格式（字体、大小、颜色等）

### 文本替换规则

- **直接替换**：不指定 `match_text` 时，替换整个段落文本
- **精确替换**：指定 `match_text` 时，只替换匹配的部分，保留前后文本
- **跨 run 处理**：如果占位符跨越多个 run，自动合并处理

### value_status 与 force 字段

每个 fill 项可携带 `value_status` 和 `force` 字段：

| 字段 | 说明 |
|------|------|
| `value_status` | 来自 parser 的值状态分类，可选值：`empty`/`placeholder`/`example`/`has_value` |
| `force` | 当 `value_status="has_value"` 时，设置 `true` 可强制覆盖已有值 |

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

> ⛔ **绝对禁止在用户确认之前执行填写。** 必须等待用户明确确认后才能继续。

用户确认后，在调用 `pptx_filler.py` 前，**必须**先上传最终版 `fill_data.json`：

```bash
# 上传最终确认版填写数据
cd {skill_base_dir}/scripts && python3 upload_file.py /tmp/fill_data.json --name "模版名_填写数据_$(date +%s).json"
```

上传成功后，从 `upload_file.py` 的标准输出 JSON 中提取 `fid` 字段的**实际值**，在回复中展示。

> ⛔ **`fid` 必须是 `upload_file.py` 返回的真实值**（如 `fid_5f81b9b77acdcf6bce18`），**绝对禁止**使用占位符（如 `fid_xxx`、`<fid>`、`fid_xxxxxxxx`）。
> ⛔ **不需要调用 `download_url.py` 获取下载链接**，只需展示 `fid` 即可。

---

### 注意事项

- `shape_name` 可以辅助定位，但主要依赖 `shape_index`
- 幻灯片中的形状顺序可能与视觉顺序不同，以 parser 输出的索引为准
- 表格中的 `values` 键是列索引的字符串（如 `"0"`, `"1"`）
