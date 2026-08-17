# MCP 参数构造规范

> 🔴 **强制引用**：在调用任何 MCP 工具构造参数前，**必须**对照本文档校验字段值。
> 本文档补充 MCP Schema 无法表达的序列化规则和常见陷阱。

---

## 1. 字段值序列化规则

> 以下规则适用于所有草稿和项目操作中的字段值构造。

### 1.1 枚举字段（behavior: enum）— 传 value 不传 label

枚举字段的 `enum_values` 中若存在 `{label, value}` 对象形式，**必须传 `value`**，禁止传 `label`。

| 字段 | ❌ 错误（label） | ✅ 正确（value） |
|------|-----------------|-----------------|
| `playlet` | `"长剧（45分钟/集）"` | `"长剧"` |
| `ifOnlyPlay` | `"独播"` | `"Y"` |
| `isPostBonus` | `"是"` | `"1"` |
| `copyrightArea` | `"全球"` | `"1"` |
| `broadcastOnOTT` | `"是"` | `"Y"` |

**速查规则**：如果 Schema 中 `enum_values` 的某个选项是 `{ "label": "XXX", "value": "YYY" }` 格式，传入 `"YYY"`；如果是纯字符串 `"XXX"`，则直接传 `"XXX"`。

### 1.2 分号序列化（serialization: semicolon）— 数组用分号拼接

> ⚠️ **Schema 标注 `type: "array"` 但实际必须传字符串**，这是 Schema 与实际传参的已知矛盾。

Schema 中标记 `"serialization": "semicolon"` 的字段，传参时必须用**分号 `;`** 将各元素拼接为**一个字符串**。

| 字段 | 原始数据 | ✅ 正确传参 |
|------|---------|-----------|
| `pnamePrevs`（曾用名） | `["旧名1", "旧名2"]` | `"旧名1;旧名2"` |
| `mediaPlatform`（新媒体发行平台） | `["星舟视频", "云溪视频"]` | `"星舟视频;云溪视频"` |
| `theme`（题材类型） | `["悬疑", "犯罪"]` | `"悬疑;犯罪"` |
| `directors`（导演） | `["张三", "李四"]` | `"张三;李四"` |
| `scriptwriters`（编剧） | `["王五"]` | `"王五"` |
| `copyrightArea`（版权区域） | `["1", "2"]` | `"1;2"` |

### 1.3 逗号序列化（serialization: csv）— 级联值用逗号拼接

Schema 中标记 `"serialization": "csv"` 的级联字段（`behavior: cascade`），将各级选中值用**逗号 `,`** 拼接为一个字符串。

| 字段 | 选择 | ✅ 正确传参 |
|------|------|-----------|
| `isCostumeDrama`（是否古装） | 是 → 隋唐 | `"是,隋唐"` |
| `copyrightDramaType`（版权剧分类） | 置换 → 云溪视频独家版权剧 | `"置换,云溪视频独家版权剧"` |

---

## 2. Schema 未覆盖的关键约束

### 2.1 `tv.category = "2"` 是隐藏必填项

MCP Schema 的 `tv` 对象中 **没有 `category` 字段定义**，但实际提交时**必须传** `view.tv.category = "2"`。

### 2.2 草稿更新的禁止规则

- ❌ 禁止在 `tv` 内放 `id` 字段（`id` 只放在 `view` 层级）
- ❌ 禁止传递未修改的字段（最小化修改原则）

### 2.3 `updateProjectPartially` 的 `update_fields` 约束

MCP Schema 的 `update_fields` 仅定义为 `type: "object"`，**无任何内部字段信息**。以下规则必须遵守：
- ✅ 使用英文字段名（Schema 的 `name`），**严禁使用中文 `label`**
- ❌ 禁止将 `project_id` 放入 `update_fields` 内
- ⚠️ 仅允许修改白名单内的字段（见 `project_update_workflow.md`）

### 2.4 `CreateProjectFile` 的 `cate` 字段

`cate` 传数字字符串 ID（如 `"1"`），**不是中文类别名**（如 ~~"剧本"~~）。

---

## 3. 常见错误与正确写法对比

### 错误 1：嵌套结构错误

```json
// ❌ 错误 — tv 不在 view 内
{ "category": "2", "tv": { "pname": "测试" } }

// ✅ 正确
{ "view": { "category": "2", "tv": { "category": "2", "pname": "测试" } } }
```

### 错误 2：缺少 category

```json
// ❌ 错误 — 缺少 view.category 和 tv.category
{ "view": { "tv": { "pname": "测试" } } }

// ✅ 正确
{ "view": { "category": "2", "tv": { "category": "2", "pname": "测试" } } }
```

### 错误 3：semicolon 序列化字段传了数组

```json
// ❌ 错误 — Schema type=array 但实际必须传字符串
{ "pnamePrevs": ["旧名1", "旧名2"] }

// ✅ 正确
{ "pnamePrevs": "旧名1;旧名2" }
```

### 错误 4：update_fields 用了中文 label 作为 key

```json
// ❌ 错误
{ "project_id": 123, "update_fields": { "项目名称": "新名称" } }

// ✅ 正确
{ "project_id": 123, "update_fields": { "pname": "新名称" } }
```

---

## 4. 提交前自检清单

> 在调用任何写入类 MCP 工具前，逐项检查：

- [ ] **嵌套正确**：草稿操作参数是否遵循 `view → category + tv → category + 字段` 三层嵌套？
- [ ] **category 齐全**：`view.category` 和 `view.tv.category` 是否都设为 `"2"`？（Schema 中 tv.category 缺失，但必须传）
- [ ] **id 位置正确**：更新操作中 `id` 是否在正确层级？（草稿更新：`view.id`；正式项目：顶层 `project_id`）
- [ ] **semicolon 序列化**：`serialization: semicolon` 的字段是否已用分号拼接为字符串？（忽略 Schema 的 type=array）
- [ ] **csv 序列化**：`serialization: csv` 的级联字段是否已用逗号拼接各级选中值？
- [ ] **枚举值正确**：所有枚举字段是否传的是 `value` 而非 `label`？
- [ ] **英文字段名**：`update_fields` 内的 key 是否使用英文 `name` 而非中文 `label`？
- [ ] **最小化原则**：是否只包含了需要创建/修改的字段？
