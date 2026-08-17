# Schema 字段类型处理规范

## 类型系统设计原则

字段定义分两层，另有可选的输出序列化属性：

- **`type`（结构层）**：描述数据的形状，取值为标准 JSON Schema 类型：`string` / `number` / `boolean` / `array` / `object`
- **`behavior`（行为层）**：描述数据的来源与约束方式，取值为：`enum` / `cascade` / `external` / 无（普通输入）
- **`serialization`（序列化层，可选）**：描述数组型字段的输出格式，取值为：`csv` / `pipe` / `semicolon`；未声明时输出为 JSON 数组

两层正交组合，覆盖所有字段场景：

| type | behavior | 含义 |
|------|----------|------|
| `string` / `number` | 无 | 普通输入字段 |
| `string` / `number` | `enum` | 枚举选择，值为标量 |
| `array` | `enum` | 枚举多选，值为数组 |
| `string` | `cascade` | 级联枚举，父子多层选择，序列化为字符串 |
| `string` / `number` | `external` | MCP 查询，结果为标量 |
| `object` | `external` | MCP 查询，结果为单个对象 |
| `array` | `external` | MCP 查询，结果为对象/标量数组 |
| `object` | 无 | 普通嵌套对象，由 `properties` 描述子字段 |
| `array` | 无 | 普通数组，由 `array_item` 描述元素类型 |
| `boolean` | 无 | 布尔值，是/否选择 |

### 各 behavior 的关键属性

| behavior | 需要的属性 | 说明 |
|----------|-----------|------|
| `enum` | `enum_values` | 合法值列表（支持字符串简写和完整对象混用） |
| `cascade` | `cascade_levels` | 层级选项树，包含各层 label/value/children |
| `external` | `mcp_tool`, `mcp_params_template` | MCP 工具名和参数模板（`{input}` 占位符） |
| `external` | `result_path`（可选） | 从 MCP 返回值定位数据的点号路径 |
| `external` | `result_field` 或 `result_fields` | 字段映射规则（单值 vs 多字段对象），值支持 template 语法 |
| `external` | `array_item`（可选） | 描述数组元素的完整结构，复用 `type: object` + `properties` 格式 |
| `external` | `display_fields`（可选） | 多结果时展示给用户的字段列表 |
| 无 | `format`（可选） | 格式约束（`date` / `integer` 等），仅 string 类型 |
| 任意 | `condition`（可选） | 条件表达式，控制字段是否激活 |
| 任意 | `serialization`（可选） | 数组输出序列化格式（`csv` / `pipe` / `semicolon`） |

---

## `$ref` — 共享定义引用

当多个字段共享相同的 `external` 配置时，Schema 通过 `$defs` + `$ref` 消除重复。

- **`$defs`**：定义在 Schema 顶层（与 `fields` 同级），包含若干命名模板
- **`$ref`**：字段中声明 `"$ref": "#/$defs/<模板名>"`，引入该模板的全部属性
- **合并规则**：字段自身属性**优先于** `$ref` 引入的同名属性（即字段可覆盖模板中的 `array_item`、`result_fields` 等）

---

## Schema 结构示例

```json
{
  "category": 2,
  "submit_tool": "create_drama",
  "$defs": {
    "talent_query_config": {
      "mcp_tool": "talent_simple_query",
      "mcp_params_template": { "keyword": "{input}" },
      "result_fields": { "id": "data.id", "name": "data.name" },
      "display_fields": ["id", "name"]
    }
  },
  "fields": [
    {
      "name": "title",
      "label": "剧名",
      "type": "string",
      "required": true
    },
    {
      "name": "episode_count",
      "label": "集数",
      "type": "number",
      "required": false
    },
    {
      "name": "status",
      "label": "播出状态",
      "type": "string",
      "behavior": "enum",
      "required": true,
      "enum_values": [
        { "label": "连载中", "value": 1 },
        { "label": "已完结", "value": 2 },
        { "label": "未开播", "value": 3 }
      ]
    },
    {
      "name": "tags",
      "label": "标签",
      "type": "array",
      "behavior": "enum",
      "required": false,
      "enum_values": ["古装", "都市", "悬疑", "爱情"]
    },
    {
      "name": "actors",
      "label": "主演",
      "type": "array",
      "behavior": "external",
      "required": false,
      "$ref": "#/$defs/talent_query_config",
      "array_item": {
        "type": "object",
        "properties": [
          { "name": "id", "label": "演员ID", "type": "number", "required": true },
          { "name": "name", "label": "演员名", "type": "string", "required": true },
          { "name": "role", "label": "角色", "type": "string" }
        ]
      }
    },
    {
      "name": "director_id",
      "label": "导演",
      "type": "number",
      "behavior": "external",
      "required": true,
      "$ref": "#/$defs/talent_query_config",
      "result_field": "data.id"
    },
    {
      "name": "cover",
      "label": "封面图片",
      "type": "object",
      "required": false,
      "properties": [
        { "name": "url",    "label": "图片链接", "type": "string", "required": true },
        { "name": "width",  "label": "宽度",     "type": "number", "required": false },
        { "name": "height", "label": "高度",     "type": "number", "required": false }
      ]
    },
    {
      "name": "airing_schedule",
      "label": "更新时间",
      "type": "string",
      "required": true,
      "condition": { "field": "status", "operator": "eq", "value": 1 }
    },
    {
      "name": "finale_date",
      "label": "完结日期",
      "type": "string",
      "format": "date",
      "required": true,
      "condition": { "field": "status", "operator": "eq", "value": 2 }
    }
  ]
}
```

---

## 各类型处理规则

### `type: string` / `type: number`，无 behavior — 普通输入

- 直接从用户输入提取对应值，注意去除多余空格、换行
- `number` 类型需将字符串转为数值（如"共36集" → `36`）
- 若携带 `format` 属性，需额外执行格式校验与转换（详见下方 format 说明）

### `type: boolean` — 布尔值

- 从用户输入中识别肯定/否定语义，映射为 `true` / `false`
- 支持的输入写法：

| 用户输入 | 映射结果 |
|---------|---------|
| 是、对、有、需要、true、1 | `true` |
| 否、不是、没有、不需要、false、0 | `false` |

- 无法判断时，以"是/否"形式向用户确认：
  ```
  是否为 IP 作品？请选择：是 / 否
  ```
- 最终写入 JSON 为布尔值（`true` / `false`），不是字符串

---

### `behavior: enum` — 枚举

适用于 `type: string`、`type: number`、`type: array`。

#### 枚举项的两种写法

`enum_values` 中每个元素支持两种形式，可在同一字段内混用：

**字符串简写**（label == value，且均为字符串时使用）：
```json
"enum_values": ["古装", "都市", "悬疑"]
```

**完整对象**（label != value，或 value 为数字时使用）：
```json
"enum_values": [
  { "label": "连载中", "value": 1 },
  { "label": "已完结", "value": 2 }
]
```

**混合写法**（同一字段内共存）：
```json
"enum_values": [
  "古装",
  "都市",
  { "label": "其他", "value": "other" }
]
```

#### 解析规则

```
元素是字符串  →  label = value = 该字符串（value 类型为 string）
元素是对象    →  label 取 label 字段，value 取 value 字段（类型由字段值决定）
```

#### 匹配策略（按优先级）

1. 完全相等匹配（label == 用户输入）
2. 包含匹配（用户输入 contains label 或 label contains 用户输入）
3. 无法匹配时，**仅展示所有 label 中文名供用户选择，不展示 value 数值**

**最终填入 JSON 的值为 `value`，不是 `label`，此映射过程对用户不可见。**

展示示例（正确）：
```
播出状态请选择：连载中 / 已完结 / 未开播
标签请选择（可多选）：古装 / 都市 / 悬疑
```

展示示例（错误，禁止）：
```
播出状态请选择：连载中(1) / 已完结(2) / 未开播(3)
```

#### 枚举选项级 `condition` — 条件可选值

枚举选项支持携带 `condition` 属性，表示该选项仅在条件成立时可选。条件不成立时，该选项**不展示给用户、不允许选择、不允许写入**。

`condition` 语法与字段级 `condition` 完全一致（详见下方 [condition — 条件字段](#condition--条件字段) 章节），支持 `eq` / `neq` / `in` 以及 `and` / `or` 组合。

**使用方式：** 将需要条件限制的选项从字符串简写改为完整对象，并添加 `condition` 属性：

```json
{
  "name": "isIp",
  "label": "是否IP",
  "type": "string",
  "behavior": "enum",
  "required": false,
  "enum_values": [
    {
      "label": "外部IP",
      "value": "外部IP",
      "condition": { "field": "cooperationMode", "operator": "neq", "value": "主控剧" }
    },
    "内部IP",
    {
      "label": "外部原创选题",
      "value": "外部原创选题",
      "condition": { "field": "cooperationMode", "operator": "neq", "value": "主控剧" }
    },
    "内部原创选题"
  ]
}
```

**处理规则：**

1. **过滤阶段**：遍历 `enum_values`，对每个含 `condition` 的选项，按字段级 condition 语法求值；条件不成立则从可选列表中移除
2. **无 `condition` 的选项**：始终可选（等同于条件恒为 true）
3. **匹配与展示**：过滤后的可选列表作为实际枚举，后续匹配策略、展示规则与普通 enum 完全相同
4. **校验**：用户选择的值必须在过滤后的可选列表中，否则拒绝并提示当前可选的选项

**示例：**

```
当 cooperationMode == "主控剧" 时：
  可选: 内部IP / 内部原创选题
  用户输入 "外部IP" → 拒绝，提示："当前合作模式为主控剧，「是否IP」只能选择：内部IP / 内部原创选题"

当 cooperationMode == "定制剧" 时：
  可选: 外部IP / 内部IP / 外部原创选题 / 内部原创选题（全部可选）
```

---

#### `type: array` + `behavior: enum` — 枚举多选

- 将用户输入拆分为多个值（如"古装、悬疑" → ["古装", "悬疑"]）
- 对每个子值按照上述 enum 规则单独映射
- 字符串简写项：最终为字符串数组，如 `["古装", "悬疑"]`
- 完整对象项：最终为 value 数组，如 `["ancient", "mystery"]`

#### `serialization: "csv"` — 枚举多选序列化为逗号字符串

部分历史字段虽然是枚举多选语义，但存储格式为逗号分隔字符串而非数组。此类字段用 `type: array` + `behavior: enum` + `serialization: "csv"` 表达：

```json
{
  "name": "tags",
  "type": "array",
  "behavior": "enum",
  "serialization": "csv",
  "required": false,
  "enum_values": ["古装", "悬疑", "爱情", "都市"]
}
```

**处理规则：**
- 解析阶段：完全按 `array + enum` 规则处理，内部维护为数组
- 输出阶段：检测到 `serialization: "csv"` 时，将数组序列化为逗号分隔字符串再写入最终 JSON

```
内部处理: ["古装", "悬疑"]
最终输出: "古装,悬疑"
```

**支持的 serialization 值：**

| 值 | 分隔符 | 示例输出 |
|----|--------|---------|
| `"csv"` | `,` | `"古装,悬疑,爱情"` |
| `"pipe"` | `\|` | `"古装\|悬疑\|爱情"` |
| `"semicolon"` | `;` | `"古装;悬疑;爱情"` |

> 未声明 `serialization` 时，`array` 类型字段正常输出为 JSON 数组。

---

### `behavior: cascade` — 级联枚举

适用于 `type: string`。父子多层枚举，各层选完后按 `serialization` 拼接为字符串存储。

#### 字段结构

```json
{
  "name": "category_path",
  "label": "内容分类",
  "type": "string",
  "behavior": "cascade",
  "serialization": "csv",
  "required": true,
  "cascade_levels": [
    {
      "label": "一级分类",
      "options": [
        {
          "label": "影视", "value": "影视",
          "children": [
            { "label": "电视剧", "value": "电视剧" },
            { "label": "电影",   "value": "电影" },
            { "label": "纪录片", "value": "纪录片" }
          ]
        },
        {
          "label": "综艺", "value": "综艺",
          "children": [
            { "label": "音乐综艺", "value": "音乐综艺" },
            { "label": "脱口秀",   "value": "脱口秀" }
          ]
        }
      ]
    }
  ]
}
```

#### 处理规则

1. **逐层识别**：从用户输入中先识别一级分类，再根据一级的 `children` 过滤出可选的二级选项，依次类推
2. **向上补全**：用户直接输入子级值（如"电视剧"）但未提及父级时，自动向上查找并补全父级路径
3. **输出序列化**：按层级顺序将各层 `value` 拼接，格式由 `serialization` 决定

```
用户输入: "影视 → 电视剧"   → 输出: "影视,电视剧"
用户输入: "电视剧"（未提父级）→ 自动补全 → 输出: "影视,电视剧"
```

#### 三层及以上（支持任意层级递归）

```json
{ "label": "影视", "value": "影视",
  "children": [
    { "label": "电视剧", "value": "电视剧",
      "children": [
        { "label": "古装剧", "value": "古装剧" },
        { "label": "都市剧", "value": "都市剧" }
      ]
    }
  ]
}
```

输出：`"影视,电视剧,古装剧"`

#### 边界处理

| 情况 | 处理方式 |
|------|---------|
| 只选了父级，未选子级 | 加入「未完成」队列，提示用户补选子级 |
| 子级不在父级的 children 中 | 加入「非法值」队列，提示重新选择 |
| 输入存在歧义（同名子级属于多个父级） | 展示所有路径供用户确认 |

#### 向用户展示时

- 以层级路径形式展示，不暴露内部 value：
  ```
  内容分类请选择：
    一级：影视 / 综艺
    二级（影视下）：电视剧 / 电影 / 纪录片
  ```
- 用户确认后提示当前已选路径：`已选：影视 > 电视剧`

---

### `behavior: external` — 外部 MCP 查询

适用于 `type: string`、`type: number`、`type: object`、`type: array`。

#### MCP 调用方式

schema 中通过以下属性配置外部查询：

- **`mcp_tool`**：要调用的 MCP 工具名
- **`mcp_params_template`**：调用参数模板，其中 `{input}` 为占位符，运行时替换为用户提供的关键词

```json
{
  "mcp_tool": "talent_simple_query",
  "mcp_params_template": { "keyword": "{input}", "role": "director" }
}
```

**`{input}` 替换规则：**
- `{input}` 会被替换为用户输入中提取的对应关键词（如演员名、公司名等）
- 模板中其他字段（如 `"role": "director"`）保持原样，作为固定参数传递
- 示例：用户输入"导演是张三" → `{input}` 替换为 `"张三"` → 实际调用参数为 `{ "keyword": "张三", "role": "director" }`

#### 结果提取方式

提取分为两步：**定位** → **映射**。

**Step 1：定位数据（`result_path`，可选）**

若 schema 定义了 `result_path`，先按点号路径从 MCP 返回值中定位到目标数据：

```json
{ "result_path": "data.data" }
```

- MCP 返回：`{ "data": { "data": [ { "tid": "100008017", "t_name": "刘亦菲" } ], "total": 1 }, "code": "0" }`
- 按 `result_path` 定位：`response.data.data` → `[ { "tid": "100008017", "t_name": "刘亦菲" } ]`

若未定义 `result_path`，直接使用返回值顶层结构。

**Step 2：映射字段（`result_field` / `result_fields`）**

基于 Step 1 定位后的数据，按以下两种模式提取：

**模式 A：`result_field`（单数）— 提取单个标量值**

适用于 `type: string` / `type: number`：

```json
{ "type": "number", "behavior": "external", "result_field": "data.id" }
```

- MCP 返回：`{ "data": { "id": 101, "name": "张三" } }`
- 提取结果：`101`

**模式 B：`result_fields`（复数）— 提取多字段组装对象**

适用于 `type: object` / `type: array`（元素为对象）：

```json
{ "type": "array", "behavior": "external", "result_fields": { "id": "data.id", "name": "data.name" } }
```

- MCP 返回：`{ "data": { "id": 101, "name": "张三" } }`
- 提取结果：`{ "id": 101, "name": "张三" }`
- 若字段为数组，最终为 `[{"id":101,"name":"张三"}, ...]`

#### template 语法 — 多字段拼接

`result_field` 和 `result_fields` 的值除了直接写路径字符串外，还支持 **template 对象语法**，用于将 MCP 返回中的多个字段拼接成一个复合值。

**语法结构：**

```json
{
  "template": "{english_name}({chinese_name})",
  "fields": {
    "english_name": "data.english_name",
    "chinese_name": "data.chinese_name"
  }
}
```

| 属性 | 说明 |
|------|------|
| `template` | 模板字符串，用 `{占位符名}` 引用 `fields` 中定义的变量 |
| `fields` | 占位符到 MCP 返回值路径的映射，key 为占位符名，value 为点号路径 |

**处理规则：**

1. 按 `fields` 中的各路径从 MCP 返回数据中提取值
2. 将提取到的值代入 `template` 中对应的 `{占位符}` 位置
3. 生成最终的字符串值

**在 `result_field`（单数）中使用 — 拼接为单个标量值：**

适用于 `type: array`（元素为字符串）或 `type: string`，将多个返回字段拼接为一个字符串作为最终值。

```json
{
  "name": "marketPMs",
  "label": "市场PM",
  "type": "array",
  "behavior": "external",
      "mcp_tool": "getStaffBaseInfo",
      "mcp_params_template": { "name": "{input}" },
      "result_field": {
        "template": "{english_name}({chinese_name})",
        "fields": {
          "english_name": "data.english_name",
          "chinese_name": "data.chinese_name"
        }
      }
}
```

```
MCP 返回: { "data": { "english_name": "zhangsan", "chinese_name": "张三" } }
提取结果: "zhangsan(张三)"
```

**在 `result_fields`（复数）中的某个字段使用 — 对象内单字段拼接：**

适用于 `type: array`（元素为对象）或 `type: object`，对象中的某个字段需要由多个返回值拼接而成，其他字段仍为普通路径映射。

```json
{
  "name": "producerKpiList",
  "label": "制片人",
  "type": "array",
  "behavior": "external",
  "mcp_tool": "getStaffBaseInfo",
  "mcp_params_template": { "name": "{input}" },
  "result_fields": {
    "name": {
      "template": "{english_name}({chinese_name})",
      "fields": {
        "english_name": "data.english_name",
        "chinese_name": "data.chinese_name"
      }
    },
    "kpi": "data.kpi"
  }
}
```

```
MCP 返回: { "data": { "english_name": "zhangsan", "chinese_name": "张三", "kpi": 50 } }
提取结果: { "name": "zhangsan(张三)", "kpi": 50 }
```

**判断逻辑：**

```
result_field / result_fields 中的值:
  值是字符串  → 普通路径映射（如 "data.id"）
  值是对象且含 template + fields  → template 语法，按模板拼接
```

**边界处理：**

| 情况 | 处理方式 |
|------|---------|
| `fields` 中某路径在 MCP 返回中不存在 | 对应占位符替换为空字符串 |
| `template` 中有未在 `fields` 中定义的占位符 | 保留原始 `{占位符}` 文本不替换 |

---

**模式 C：`array_item`（对象结构）— 用 properties 描述数组元素的完整结构**

适用于 `type: array` + `behavior: external`，当 MCP 返回的 item 是嵌套对象时，可用 `array_item` 复用普通数组的 `type: object` + `properties` 格式来描述元素结构：

```json
{
  "name": "partner",
  "type": "array",
  "behavior": "external",
  "mcp_tool": "company_search",
  "mcp_params_template": { "query": "{input}" },
  "array_item": {
    "type": "object",
    "properties": [
      {
        "name": "company",
        "label": "公司",
        "type": "object",
        "properties": [
          { "name": "value", "label": "公司ID", "type": "string" },
          { "name": "label", "label": "公司名称", "type": "string" },
          { "name": "risk", "label": "风险性", "type": "string" },
          { "name": "risk_statement", "label": "风险说明", "type": "string" }
        ]
      },
      { "name": "product", "label": "产品", "type": "string" },
      { "name": "set", "label": "剧集", "type": "string" }
    ]
  },
  "display_fields": ["company.label", "company.risk", "company.risk_statement"]
}
```

- `array_item` 与普通数组（无 behavior）的 `array_item` 格式完全一致，复用 `type: object` + `properties` 描述嵌套结构
- 当字段同时声明了 `array_item` 时，优先使用 `array_item` 描述元素结构，`result_fields` 可省略
- `display_fields` 中使用点号路径（如 `company.label`）引用嵌套属性

**完整提取示例（含 `result_path`）：**

```
schema 定义:
  "mcp_tool": "talent_simple_query",
  "result_path": "data.data",
  "result_fields": { "tid": "tid", "t_name": "t_name", "gender": "gender" }

MCP 返回值:
  { "data": { "data": [ { "tid": "100008017", "t_name": "刘亦菲", "gender": "女" } ], "total": 1 }, "code": "0" }

处理过程:
  1. 按 result_path "data.data" 定位 → 得到数组 [ { "tid": "100008017", "t_name": "刘亦菲", ... } ]
  2. 按 result_fields 映射 → 提取 { "tid": "100008017", "t_name": "刘亦菲", "gender": "女" }
  3. 数组长度为 1 → 唯一命中，直接使用
```

#### MCP 返回数组时的处理规则

| 返回数量 | 处理方式 |
|---------|---------|
| **恰好 1 条** | 直接使用，无需提示 |
| **0 条** | 加入「未找到」队列 |
| **2 条及以上** | 取前 5 条，加入「需用户选择」队列，按下方规则展示 |

#### 多结果展示规则（基于 `display_fields`）

当 MCP 返回 2 条及以上结果时，需要向用户展示候选列表供选择。

**`display_fields` 属性：** schema 中的 `display_fields` 数组显式声明了多结果时应展示给用户的字段列表，避免由处理方猜测该展示哪些字段。

```json
{
  "name": "leadActor",
  "type": "array",
  "behavior": "external",
  "result_fields": { "tid": "tid", "t_name": "t_name", "gender": "gender", "profession": "profession" },
  "display_fields": ["t_name", "gender", "profession", "birthday"]
}
```

**字段路径解析规则：**

- **`result_fields` 模式**：`display_fields` 中的值为 `result_fields` 的 **key**（如 `"t_name"`、`"gender"`），从提取后的对象中取值展示
- **`result_field` 模式**：`display_fields` 中的值为 MCP **原始返回的点号路径**（如 `"data.name"`、`"data.role"`），因为 `result_field` 只提取单个值，不足以区分，需要从原始返回中取额外字段来辅助展示

**展示格式按 `type` 分两种：**

**单选**（`type: string` / `type: number` / `type: object`）— 只选一条：

```
【需要选择】导演 · 张三（只选一位），请选择序号：
  1. 张三（男，电影导演）
  2. 张三（男，电视剧导演）
```

**多选**（`type: array`）— 可选一条或多条：

```
【需要选择】主要演员 · 李四（可多选，输入序号，多个用空格分隔）：
  1. 李四（女，演员，1990-01-15）
  2. 李四（男，歌手/演员，1985-06-20）
  3. 李思（女，演员，1995-03-10）
```

**各工具类型的展示示例：**

| MCP 工具 | display_fields | 展示效果 |
|----------|---------------|---------|
| `talent_simple_query` | `["t_name", "gender", "profession", "birthday", "pic_url"]` | 李四（女，演员，1990-01-15）[头像](pic_url) |
| `company_search` | `["name", "risk", "risk_statement"]` | 华谊兄弟（有风险，风险说明：xxx） |
| `getStaffBaseInfo`（result_fields） | `["name", "role"]` | 张三（制片人） |
| `getStaffBaseInfo`（result_field） | `["data.name", "data.role"]` | 王一（市场部） |
| `search_ip` | `["ip_name", "ip_use_name", "copyright_start", "copyright_end"]` | 甄嬛传（原名：后宫·甄嬛传，版权期：2020-01-01 ~ 2030-12-31） |

#### `pic_url` 头像展示规则（`talent_simple_query` 专用）

`talent_simple_query` 返回的结果中包含 `pic_url`（头像链接），用于在**多结果或不确定**时帮助用户区分同名/相似人员。

**触发条件：**
- MCP 返回 **2 条及以上** 结果时，展示头像链接
- 仅有 **1 条** 唯一命中时，不需要展示头像（直接使用）

**展示格式：**

```
【需要选择】导演 · 张三，找到以下结果，请选择序号：
  1. 张三（男，导演，1970-05-12）
     头像：https://example.com/pic1.jpg
  2. 张三（男，导演/编剧，1985-08-20）
     头像：https://example.com/pic2.jpg
```

**处理要点：**
- `pic_url` 为空或不存在时，省略头像行，不显示"无头像"等占位文字
- 头像链接仅作辅助区分展示，不写入最终提交的 JSON 数据
- 若即使展示了头像仍无法区分（如头像链接相同），追加其他可区分字段（如代表作等）

**降级处理（无 `display_fields` 时）：**

若 schema 中未声明 `display_fields`，按以下降级策略：
1. 有 `result_fields` → 取 `result_fields` 的所有 key，排除 ID 类字段（key 含 `id`、`tid`、`value`、`pic_url` 等），剩余字段全部展示
2. 有 `result_field` → 展示该单值，若无法区分，追加 MCP 原始返回中的 `name`、`label`、`role` 等字段

#### 多结果展示要点

1. **标题必须包含字段 label 和用户输入的关键词**：如"主要演员 · 李四"，帮助用户定位是哪个关键词的查询结果
2. **单选 vs 多选**：
   - `type: string` / `type: number` / `type: object` → 单选（只选一条）
   - `type: array` → 多选（可选一条或多条），需在提示中注明"可多选"
3. **ID 类字段不展示**：`tid`、`id`、`value` 等内部标识字段不出现在用户展示中
4. **最多展示 5 条**：超过 5 条结果时只取前 5 条，并注明"更多结果可换关键词重新搜索"
5. **无法区分时兜底**：若所有候选项展示字段完全相同，追加 MCP 原始返回中的其他字段（如生日、代表作等）直到可区分

> 执行流程编排（静默处理、统一提示）和提示格式模板详见 `draft_create_workflow.md` Step 3.3 和 Step 4.2。

---

### `type: object`，无 behavior — 嵌套对象

用 `properties` 数组描述子字段，子字段完全复用本规范（支持嵌套）：

```json
{
  "name": "cover",
  "type": "object",
  "properties": [
    { "name": "url",   "type": "string", "required": true,  "label": "图片链接" },
    { "name": "width", "type": "number", "required": false, "label": "宽度" }
  ]
}
```

处理规则：
- 递归展开 `properties`，对每个子字段按其 `type` + `behavior` 独立处理
- 若用户输入中已包含对应信息，直接提取，不重复询问
- 提示格式编排详见 `draft_create_workflow.md` Step 3.4

---

### `type: array`，无 behavior — 普通数组

用 `array_item` 描述元素类型，元素可以是任意 type + behavior 组合（支持嵌套）：

```json
{
  "name": "episodes",
  "type": "array",
  "array_item": {
    "type": "object",
    "properties": [
      { "name": "title",  "type": "string", "label": "集名" },
      { "name": "number", "type": "number", "label": "集数" }
    ]
  }
}
```

对每个元素，按 `array_item` 的类型规则处理，组装后收入数组。

---

## format 属性（string 类型扩展）

`string` 类型字段可携带 `format` 属性，声明值的格式约束：

```json
{ "name": "finale_date", "type": "string", "format": "date" }
```

### 日期格式（`format: "date"`）

支持识别多种输入写法，统一转换为目标格式后写入 JSON。

**支持的输入格式（模糊识别）：**

| 用户输入示例 | 说明 |
|------------|------|
| `2024-02-14` | ISO 标准，直接使用 |
| `2024/02/14` | 斜杠分隔 |
| `2024.02.14` | 点分隔 |
| `20240214` | 纯数字8位 |
| `2024年2月14日` | 中文全写 |
| `2024年2月` | 省略日，补全为 `01` |
| `24-02-14` | 两位年份，补全为 `20xx` |
| `2月14日` / `02-14` | 省略年份，补全为当前年份 |
| `今年2月14日` | 相对表达，结合当前年份推断 |

**目标格式**：默认 `YYYY-MM-DD`，可通过 `format_pattern` 指定：

```json
{ "format": "date", "format_pattern": "YYYY-MM-DD" }
```

**校验规则：**

1. 可识别 → 转换为目标格式，继续流程
2. 无法识别 → 加入「格式错误」队列，统一提示时要求重新输入
3. 日期合法性验证（月份 1–12，日期在当月范围内）

**提示格式：**
```
完结日期（格式：YYYY-MM-DD，如 2024-02-14）：
```

格式错误时：
```
【格式错误】完结日期输入值"xxx"无法识别，请按 YYYY-MM-DD 格式重新输入（如：2024-02-14）
```

### 其他 format 类型（预留）

| format 值 | 含义 | 当前处理 |
|-----------|------|---------|
| `date` | 日期 | ✅ 完整支持 |
| `integer` | 整数（字符串存储） | ✅ 完整支持 |
| `datetime` | 日期时间 | 暂按 string 处理，不做格式校验 |
| `time` | 时间 | 暂按 string 处理 |
| `url` | 链接 | 暂按 string 处理 |
| `email` | 邮箱 | 暂按 string 处理 |

### 整数格式（`format: "integer"`）

用于 `type: string` 但实际存储整数值的字段。输入为自然语言数字描述，输出为**纯数字字符串**。

```json
{ "name": "seriesNumber", "type": "string", "format": "integer" }
```

**支持的输入格式（模糊识别）：**

| 用户输入示例 | 转换结果 |
|------------|---------|
| `36` | `"36"` |
| `共36集` | `"36"` |
| `三十六` | `"36"` |
| `36集` | `"36"` |

**处理规则：**

1. 从用户输入中提取数值部分，去除中文描述、单位等非数字内容
2. 中文数字（如"三十六"）转换为阿拉伯数字
3. 结果必须为非负整数，不含小数点
4. 最终以**字符串形式**写入 JSON（如 `"36"` 而非 `36`）

**校验规则：**

1. 可识别 → 转换为纯数字字符串，继续流程
2. 无法识别或包含小数 → 加入「格式错误」队列，统一提示时要求重新输入

**提示格式：**
```
制作集数（请输入整数，如 36）：
```

格式错误时：
```
【格式错误】制作集数输入值"xxx"无法识别为整数，请重新输入（如：36）
```

---

## condition — 条件字段

字段可携带 `condition` 属性，表示该字段仅在满足条件时才激活。**条件不成立的字段不解析、不验证、不出现在最终 JSON 中。**

### 单条件

```json
"condition": { "field": "status", "operator": "eq", "value": 2 }
```

| operator | 含义 | value 类型 |
|---------|------|-----------|
| `eq` | 等于 | 单值 |
| `neq` | 不等于 | 单值 |
| `in` | 在列表中 | 数组 |

### AND 组合（全部成立）

```json
"condition": {
  "and": [
    { "field": "status", "operator": "eq", "value": 2 },
    { "field": "isIp", "operator": "neq", "value": "内部IP" }
  ]
}
```

### OR 组合（任一成立）

```json
"condition": {
  "or": [
    { "field": "status", "operator": "in", "value": [1, 2] },
    { "field": "type", "operator": "eq", "value": "special" }
  ]
}
```

### 嵌套组合

`and` / `or` 内的子项可继续为单条件或嵌套组合，支持任意层级递归。

```json
"condition": {
  "and": [
    { "field": "category", "operator": "eq", "value": 2 },
    { "or": [
        { "field": "status", "operator": "eq", "value": 1 },
        { "field": "status", "operator": "eq", "value": 2 }
    ]}
  ]
}
```

> 等价于：`category == 2 AND (status == 1 OR status == 2)`

### 求值规则（递归）

1. 含 `field` → 单条件，直接按 operator 比较
2. 含 `and` → 递归求值所有子项，全部 true 则成立
3. 含 `or` → 递归求值所有子项，任一 true 则成立

---

## 数据组装输出格式

最终传给 MCP submit 工具的 JSON：

```json
{
  "category": 2,
  "title": "某某剧",
  "status": 1,
  "actors": [
    { "id": 10086, "name": "刘亦菲" },
    { "id": 20012, "name": "张三" }
  ],
  "episode_count": 36,
  "tags": ["古装", "悬疑"],
  "cover": { "url": "https://...", "width": 800 }
}
```

- 未填写的非必填字段**不包含**在输出中（不填 `null`）
- 条件不成立的字段**不包含**在输出中
- 必填字段缺失时不得提交，需先补全
- `serialization: "csv"` 的字段：输出为逗号分隔字符串，而非数组


