# mcporter 调用示例

## MCP 调用规范

> 🔴 **强制要求**：调用任何 MCP 工具时，`mcp_call_tool` 的 `arguments` 参数**必须是合法的 JSON 字符串**。

**正确示例：**
```json
arguments: "{\"view\":{\"category\":\"2\",\"tv\":{\"category\":\"2\",\"pname\":\"测试项目\"}}}"
```

**错误示例（严禁）：**
```
arguments: {view: {category: "2", tv: {pname: "测试项目"}}}
```

---

## 草稿操作（szbotprojectdraft）

> ⚠️ **参数规范**：构造参数前必须阅读 `references/mcp_param_rules.md`，对照自检清单逐项校验。

### 创建草稿

> 结构：`view.category="2"` + `view.tv.category="2"` + 业务字段。必填：`pname`、`playlet`、`cooperationMode`。
> 枚举字段传 value 不传 label（如 `playlet` 传 `"长剧"` 而非 `"长剧（45分钟/集）"`）。

```bash
mcporter call szbotprojectdraft.CreateDraft --args '{"view":{"category":"2","tv":{"category":"2","pname":"项目名称","playlet":"长剧","cooperationMode":"主控剧"}}}'
```

### 查看草稿列表

```bash
mcporter call szbotprojectdraft.GetDraftList --args '{"category":"2"}'
```

### 修改草稿（部分更新）

> 结构：`view.id` 必填（草稿 ID），`view.tv` 内只放需要修改的字段。

```bash
mcporter call szbotprojectdraft.UpdateDraftPartially --args '{"view":{"id":"12345","category":"2","tv":{"category":"2","pname":"新名称"}}}'
```

### 删除草稿

```bash
mcporter call szbotprojectdraft.DeleteDraft --args '{"id":"草稿ID"}'
```

### 添加草稿文件
```bash
mcporter call szbotprojectdraft.AddDraftFile --args '{"id":"草稿ID","files":[{"file_id":"文件ID1","name":"文件名1","category":"文件类别1"}]}'
```

### 删除草稿文件
```bash
mcporter call szbotprojectdraft.DeleteDraftFile --args '{"id":"草稿ID","file_id":"文件ID"}'
```

### 获取草稿文件
```bash
mcporter call szbotprojectdraft.GetDraftFiles --args '{"id":"草稿ID"}'
```

---

## 正式项目操作（szbotprojectformal）

> ⚠️ **参数规范**：构造参数前必须阅读 `references/mcp_param_rules.md`，对照自检清单逐项校验。

### 更新正式项目

> 结构：`project_id`（数字类型，非字符串）+ `update_fields`（英文字段名为 key）。
> 仅允许修改白名单内的字段。字段名必须使用 Schema `name`（如 `pname`），严禁使用中文 `label`（如"项目名称"）。

```bash
mcporter call szbotprojectformal.updateProjectPartially --args '{"project_id":111,"update_fields":{"pname":"新项目名称"}}'
```

---

## 项目文件管理（szbotprojectfile）

### 添加单个文件到项目

```bash
mcporter call szbotprojectfile.CreateProjectFile --args '{"father":"项目ID","doc":[{"id":"fid_abc123","title":"故事大纲v2.docx","cate":"123129308"}]}'
```

### 批量添加文件到项目

```bash
mcporter call szbotprojectfile.CreateProjectFile --args '{"father":"项目ID","doc":[{"id":"fid_aaa","title":"终稿剧本第1集.pdf","cate":"123134052"},{"id":"fid_bbb","title":"故事大纲.docx","cate":"123129308"}]}'
```

> ⚠️ `cate` 传值为 **category ID 数字字符串**，不是中文类型名。完整映射及子类型双 category 规则详见 `references/file_categories.md`。

---

## 通用查询工具（szbotprojectdraft）

### 艺人查询

```bash
mcporter call szabot_tools.talent_simple_query --args '{"keyword":"刘亦菲","page_index":0,"page_size":50}'
```

### 台账/IP 查询

```bash
mcporter call szabot_tools.search_ip --args '{"ip_name":"甄嬛传","page_idx":1,"page_size":5,"play_type":"80"}'
```

### 员工信息查询

```bash
mcporter call szabot_tools.getStaffBaseInfo --args '{"name":"clawjone"}'
```

### 公司查询

```bash
mcporter call szabot_tools.company_search --args '{"condition":[{"公司名称":["正午阳光"]}],"target":["公司ID","公司名称","公司类型","代表作项目","风险性"]}'
```

### 搜索项目

```bash
mcporter call szabot_tools.kb_search --args '{"recall_req_list":[{"domain_knowledge":"影库知识库-ES-0","query":{"condition":[{"项目名称":["庆余年"]},{"项目ID":["1234"]}]}}]}'
```

---

## mcporter call 规则

1. `--args` 的值必须用**单引号 `'...'`** 包裹
2. 单引号内部必须是**合法的 JSON**
3. 若 JSON 内部包含单引号字符，使用 `'\''` 转义
4. 无参数的工具调用传 `--args '{}'`，不可省略 `--args`
