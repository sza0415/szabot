# Schema 扩展字段语义

本文档定义 JSON Schema 中扩展字段（`x-*`）及相关标准关键字的结构与语义。

---

## 表单级扩展

### `x-form-context`（表单须知）

- **位置**：schema 根级（`type: "object"` 顶层）
- **类型**：`map[string]any`，key 和结构由配置方自由定义
- **语义**：表单级的自然语言说明信息，描述字段间关系、业务规则、填写指引、术语等
- **消费方式**：Agent 获取 schema 后应**优先阅读** `x-form-context`，据此理解表单整体约束再处理具体字段

**示例**：

```json
{
  "type": "object",
  "title": "版权剧信息评估",
  "x-form-context": {
    "字段关系": [
      "演员的角色(role)取值应来自「主要人物」字段中已填写的角色名称(name)",
      "其他演员的角色(role)同样取自「主要人物」字段中的角色名称(name)"
    ],
    "业务规则": [
      "制片人KPI比例之和必须等于100%",
      "预估成本包含制作费+全部演员费用+承制利润+IP费用，该字段不作为ROC成本数据"
    ],
    "术语": {
      "ROC": "内容投资回报率",
      "独播": "仅在星舟视频播出"
    }
  },
  "properties": { ... }
}
```

> ⚠️ `x-form-context` 的 key 不固定，由配置方按需自定义。Agent 应将其中所有内容作为填写/校验的上下文约束。

---

## 字段级扩展

### `x-source`

声明字段的数据来源方式。顶层必填属性 `kind` 决定内部结构：

```
x-source
├─ kind          ← "extract" | "verbatim" | "resolve"（必填）
│
├─ [kind=extract 时的属性]
│    hint / examples / cardinality / required / onMissing
│
├─ [kind=verbatim 时的属性]
│    span
│
└─ [kind=resolve 时的子对象]
     ├─ extract   （抽取配置）
     └─ resolver  （检索配置）
          ├─ ref
          ├─ inputFrom  → Record<string, InputSource>
          ├─ output     → { mapping, itemsPath }
          └─ overrides  → { errorPolicy, cacheTTL }
```

---

#### kind = `"extract"` — 从文本中抽取数据

属性直接挂在 `x-source` 下：

| 属性 | 类型 | 默认值 | 说明 |
|------|------|:------:|------|
| `hint` | string | — | 引导抽取的提示文本 |
| `examples` | string[] | — | few-shot 示例 |
| `cardinality` | `"single"` \| `"multiple"` | `"single"` | 单值或多值 |
| `required` | boolean | `false` | 是否必须抽取到 |
| `onMissing` | `"ask"` \| `"skip"` \| `"fail"` | `"skip"` | 未抽取到时的行为 |

---

#### kind = `"verbatim"` — 原文直取，不经改写

属性直接挂在 `x-source` 下：

| 属性 | 类型 | 默认值 | 说明 |
|------|------|:------:|------|
| `span` | `"full"` \| `"sentence"` \| `"paragraph"` | `"full"` | 取值粒度：整段 / 单句 / 段落 |

---

#### kind = `"resolve"` — 抽取关键词 + 外部检索

包含两个子对象：`extract`（抽取配置）和 `resolver`（检索配置）。

##### `x-source.extract`

| 属性 | 类型 | 默认值 | 说明 |
|------|------|:------:|------|
| `hint` | string | — | 引导抽取的提示文本 |
| `examples` | string[] | — | few-shot 示例 |
| `cardinality` | `"single"` \| `"multiple"` | `"single"` | 单值或多值 |
| `onMissing` | `"ask"` \| `"skip"` \| `"fail"` | `"skip"` | 未抽取到时的行为 |

##### `x-source.resolver`

| 属性 | 类型 | 必填 | 说明 |
|------|------|:----:|------|
| `ref` | string | ✅ | resolver 类型标识，直接作为 `resolve.py --type` 参数（如 `talent`、`company`、`staff`、`project`、`ip`） |
| `inputFrom` | Record\<string, InputSource\> | — | 入参绑定（key=参数名, value=来源） |
| `output` | object | — | 输出映射（见下） |
| `overrides` | object | — | 字段级覆盖（见下） |

##### `x-source.resolver.inputFrom[*]`（InputSource）

| 属性 | 类型 | 说明 |
|------|------|------|
| `source` | string（必填） | 来源类型，取值见下表 |
| `value` | any | `source="literal"` 时的字面量值 |
| `key` | string | `source="context"` 或 `"env"` 时的键名 |

`source` 取值：

| 值 | 含义 |
|----|------|
| `"extracted"` | 来自 Stage 1 LLM 抽取结果 |
| `"literal"` | 字面量，由 `value` 指定 |
| `"context"` | 运行时上下文，由 `key` 指定（如 `user.id`） |
| `"env"` | 环境变量（白名单），由 `key` 指定 |

##### `x-source.resolver.output`

| 属性 | 类型 | 说明 |
|------|------|------|
| `mapping` | Record\<string, string\> | 目标字段 → JSONPath 映射 |
| `itemsPath` | string | batch 模式时必填：从返回结果中按此路径展开数组 |

##### `x-source.resolver.overrides`

可覆盖的白名单项：

| 属性 | 类型 | 说明 |
|------|------|------|
| `errorPolicy.<scenario>.prompt` | string | 字段级错误话术 |
| `errorPolicy.<scenario>.maxChoices` | number | 多选时最大候选数 |
| `cacheTTL` | number（秒） | 字段级缓存时长 |

> 禁止在 `overrides` 中放 `kind` / `tool`。

---

### `x-biz-data-type`

- **类型**：string
- **语义**：业务类型标识（如 `"company"` / `"talent"` / `"string"` / `"number"`）
- **作用**：指引 resolver 选择、prompt 分组

---

### `x-active-when`

- **类型**：string（expr-lang 表达式）
- **语义**：条件激活。表达式求值为 FALSE 时，字段整个剔除（不抽取、不检索、不提交）

---

### `x-readonly-when`

- **类型**：string（expr-lang 表达式）
- **语义**：条件锁定。表达式求值为 TRUE 时，跳过写入，保留原值
- **约束**：与 `readOnly: true` 互斥；与 `x-active-when` 可共存

---

### `x-required-when`

- **类型**：string（expr-lang 表达式）
- **语义**：条件必填。表达式求值为 TRUE 时，该字段为必填（等效于加入 `required`）
- **约束**：与 `x-active-when` 可共存（未激活时不判定必填）

---

### `readOnly`（JSON Schema 标准）

- **类型**：boolean
- **语义**：恒定只读。字段不抽取、不检索、不提交
- **约束**：不允许与 `x-source` 同时存在；与 `x-readonly-when` 互斥

---

## 数组级扩展

### `x-biz-unique-by`

- **类型**：string
- **语义**：按指定属性名做业务主键去重（如 `"uscc"` / `"id"`）

---

## 数组字段的两种模式

由 `x-source` 配置组合决定：

| 模式 | 条件 | 语义 |
|------|------|------|
| item 模式 | `extract.cardinality: "multiple"` | 抽取多个关键词，逐个调 resolver |
| batch 模式 | `extract.cardinality: "single"` + `output.itemsPath` 存在 | 抽取单个关键词，一次调 resolver 返回多条 |

---

## 示例

```json
{
  "type": "object",
  "properties": {
    "projectName": {
      "type": "string",
      "x-source": {
        "kind": "extract",
        "hint": "影片名称（书名号/引号内）",
        "examples": ["《星际拓荒》", "「无名之辈」"],
        "required": true,
        "onMissing": "ask"
      }
    },
    "director": {
      "type": "object",
      "properties": { "name": { "type": "string" }, "id": { "type": "string" } },
      "x-biz-data-type": "talent",
      "x-source": {
        "kind": "resolve",
        "extract": { "hint": "导演姓名", "cardinality": "single", "onMissing": "ask" },
        "resolver": {
          "ref": "talent",
          "inputFrom": {
            "keyword": { "source": "extracted" },
            "scope": { "source": "literal", "value": "artist" }
          },
          "output": { "mapping": { "name": "$.name", "id": "$.id" } }
        }
      }
    },
    "investors": {
      "type": "array",
      "x-biz-unique-by": "uscc",
      "items": { "type": "object", "properties": { "name": {}, "uscc": {}, "id": {} }, "x-biz-data-type": "company" },
      "x-source": {
        "kind": "resolve",
        "extract": { "hint": "投资方公司", "cardinality": "multiple" },
        "resolver": {
          "ref": "company",
          "inputFrom": { "keyword": { "source": "extracted" } },
          "output": { "mapping": { "name": "$.name", "uscc": "$.credit_no", "id": "$.credit_no" } }
        }
      }
    },
    "rawText": {
      "type": "string",
      "x-source": { "kind": "verbatim", "span": "full" }
    },
    "createdBy": {
      "type": "string",
      "readOnly": true
    }
  }
}
```
