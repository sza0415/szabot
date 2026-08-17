# 文件同步到项目工作流

将本地文件上传到影库文件服务，获取 `fid` 后，通过 `szbotprojectfile.CreateProjectFile` 关联到项目。

---

## 前置条件

- 用户已提供**本地文件路径**（或可通过对话获取）
- 已知**目标项目 ID**（或可通过 `kb_search` 查询获取项目 ID）

## 工作流步骤

### Step 1：确认信息

向用户确认以下信息：

| 信息 | 必填 | 说明 |
|------|------|------|
| 本地文件路径 | ✅ | 支持绝对路径或相对路径 |
| 目标项目 | ✅ | 项目 ID 或项目名（通过名称查询获取 ID） |
| 文件类别 | ❌ | `cate` 值（category ID），可根据文件名自动匹配，无法匹配时询问用户 |

### Step 2：定位目标项目

> ⚠️ **必须使用 `szabot_tools.kb_search` MCP 工具检索项目**，不得跳过检索步骤。
> ⚠️ **检索关键词必须使用用户的原始输入**，严禁对用户提供的项目名称做任何处理（如添加空格、拆分、补全书名号、去除标点等）。

**场景 A：用户提供了项目 ID**
- 使用 `kb_search` 验证该 ID 对应的项目：

```bash
mcporter call szabot_tools.kb_search --args '{"recall_req_list":[{"domain_knowledge":"影库知识库-ES-0","query":{"condition":[{"项目ID":["<项目ID>"]}]}}]}'
```

- 确认存在后向用户展示项目名称进行确认，然后跳到 Step 3。

**场景 B：用户提供了项目名称**
- 使用 `kb_search` 检索该项目名称：

```bash
mcporter call szabot_tools.kb_search --args '{"recall_req_list":[{"domain_knowledge":"影库知识库-ES-0","query":{"condition":[{"项目名称":["<项目名称>"]}]}}]}'
```

- 检索到唯一结果 → 向用户确认项目信息，提取项目 ID。
- 检索到多条结果 → 列出候选项让用户选择。
- 未检索到结果 → 告知用户未找到，请确认项目名称。

**场景 C：用户未指定目标项目**
- 请用户提供项目名称或 ID：

```
请提供您要同步文件的目标项目名称或项目 ID。
```

### Step 3：上传文件获取 fid

**调用 `szabot-file-uploader` Skill** 将本地文件上传到影库文件服务，获取文件的 `fid`。

> ⚠️ 必须使用 `szabot-file-uploader` Skill 完成上传，`fid` 只能来自上传接口返回结果。

**上传结果需包含以下关键字段：**

| 字段 | 用途 |
|------|------|
| `fid` | 用于下一步 `CreateProjectFile` 的 `doc[].id` 参数，**必须且只能从 `szabot-file-uploader` 上传返回结果中获取** |
| `name`（文件名） | 用于下一步的 `doc[].title` 参数。**必须从用户提供的原始文件信息中获取**（如用户指定的本地文件路径中的文件名），⚠️ 禁止使用 `szabot-file-uploader` 返回的内部标识作为文件名 |

### Step 4：匹配文件类别（cate）

根据文件名或用户指定的类别，匹配对应的 category ID。

> 📄 完整的 category 映射表、匹配规则及常见错误详见 `references/file_categories.md`。

**要点：**
- `cate` 传值为 **category ID 数字字符串**（如 `"123129307"`），不是中文类型名
- 用户明确指定类别时直接使用对应 ID
- 文件名自动匹配时优先匹配子类型（如含"剧本"+"终稿" → `123134052`）
- 无法匹配时按 `file_categories.md` 中的提示格式询问用户选择

### Step 5：将文件添加到项目

调用 MCP 工具 `CreateProjectFile`，将上传后的文件关联到目标项目：

```bash
mcporter call szbotprojectfile.CreateProjectFile --args '{"father":"<项目ID>","doc":[{"id":"<fid>","title":"<文件名>","cate":"<文件类别>"}]}'
```

**参数说明：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `father` | string | ✅ | 目标项目 ID |
| `doc` | array | ✅ | 文件列表，支持一次添加多个文件 |
| `doc[].id` | string | ✅ | 上传返回的 `fid` |
| `doc[].title` | string | ✅ | 文件显示名称（通常使用原始文件名） |
| `doc[].cate` | string | ✅ | 文件类别 category ID，如 `"123129307"`（剧本）、`"123129308"`（故事大纲） |

**完整示例：**

```bash
mcporter call szbotprojectfile.CreateProjectFile --args '{"father":"12345","doc":[{"id":"fid_abc123def456","title":"项目剧本.pdf","cate":"123129307"}]}'
```

### Step 6：确认结果

- 调用成功后，向用户确认文件已同步到项目。
- 可选：调用 `GetDraftFiles` 验证文件是否出现在列表中。

```bash
mcporter call szbotprojectdraft.GetDraftFiles --args '{"id":"<草稿ID>"}'
```

---

## 批量上传场景

若用户提供了多个文件，执行流程：

1. **逐个上传** 每个文件，收集所有 `fid` 和文件名
2. **一次性调用** `CreateProjectFile`，将所有文件放入 `doc` 数组：

```bash
mcporter call szbotprojectfile.CreateProjectFile --args '{"father":"12345","doc":[{"id":"fid_aaa","title":"剧本.pdf","cate":"123129307"},{"id":"fid_bbb","title":"大纲.docx","cate":"123129308"}]}'
```

---

## 错误处理

| 错误场景 | 处理方式 |
|---------|---------|
| 文件不存在 | 提示用户检查文件路径 |
| 上传失败 | 展示错误信息，建议重试 |
| 项目 ID 无效 | 通过 `kb_search` 重新查询 |
| `CreateProjectFile` 失败 | 展示 MCP 返回的错误信息 |
| 网络超时 | 建议重试 |