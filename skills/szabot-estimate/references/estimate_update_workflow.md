# 评估单更新工作流

> 🚫 **严格使用限制**：本工作流**仅适用于"修改/更新评估单"场景**。当且仅当用户明确表达了修改已有评估单的意图时，才可执行本文件。

> 🔒 **上下文隔离**：执行本工作流期间，仅遵循本文件和 `mcp_call_rules.md`、`schema-contract.md` 中的规则。

---

## 全局执行规则

> ⚠️ 以下规则在整个会话期间**始终生效**。

1. **步骤追踪**：每轮回复开头必须以内部注释形式标注当前步骤
2. **单步执行**：每轮回复只推进一个步骤，Step 1 可与 Step 2 合并
3. **禁止跳步**：按 Step 1 → 2 → 3 → 4 → 5 → 6 顺序逐步执行
4. **枚举展示约束**：只展示 `label`，**禁止展示 `value`**
5. **Schema 忠实约束**：以动态获取的 Schema 为唯一事实来源
6. **底层操作静默**：MCP 调用过程、Schema 结构等内部细节不向用户展示
7. **最小化修改原则**：只处理用户明确要求修改的字段

---

## 流程总览

```
Step 1  识别意图 + 确定 apply_id + 判定内容来源  ← 意图/标识/来源一次性确认
Step 2  获取当前数据 + Schema（+ 读取文件）      ← 可并行：MCP 调用 ∥ 文件读取
Step 3  根据 Schema 从内容中提取数据             ← 解析 x-source，LLM 抽取字段值
Step 4  根据 Schema 调用三方接口拼接数据         ← 处理 x-source.kind=resolve 的字段
Step 5  结合已有数据，组合最终提交数据            ← 应用 x-active-when / x-readonly-when / readOnly 规则
Step 6  调用 MCP 更新 + 反馈
```

---

## Step 1：识别修改意图 + 确定评估单标识 + 判定内容来源

**意图识别关键词：** 修改、更新、改一下、改成、换成、调整、填写、提交

从用户输入中提取：
- **目标评估单标识**（评估单 ID 或关联的项目名称/ID）
- **修改内容**（要改的字段和新值）
- **内容来源类型**（文件 / 文本 / 混合）

### 1.1 确定 apply_id

| 信息 | 获取方式（按优先级） |
|------|-------------------|
| `apply_id` | ① 用户直接提供 → ② 上下文中已有（如刚创建/刚查看的评估单）→ ③ 用户提供项目 ID，调用 `listEvaluation` 查找 → ④ 询问用户 |

> `task_key` 固定为 `base_info_collection`，无需用户提供或推断。

### 1.2 判定内容来源

用户修改评估单时，数据来源分为两种：

| 来源类型 | 说明 | 示例 |
|---------|------|------|
| **文件** | 用户上传/提供了文件（策划书、评估报告、合同等），需要从文件内容中提取结构化数据 | "帮我把这份策划书的信息填到评估单里" |
| **文本** | 用户在对话中直接描述修改内容，无需读取文件 | "把导演改成张三，集数改为36集" |

按以下优先级判定：

1. **用户已上传/附带文件** → 来源 = 文件
2. **用户提到了文件但未提供** → 提示用户上传文件后继续（**本步骤暂停，等文件就绪后再进入 Step 2**）
3. **用户直接描述了具体字段和值** → 来源 = 文本
4. **无法判定** → 询问用户

**混合来源：** 用户可能同时提供文件和文本补充说明（如"把这份策划书填进去，导演改成李四"）：
- 文件内容为主要提取源
- 用户文本中的明确指令优先级高于文件内容（即文本指定的值覆盖文件中提取的值）

> 💡 **设计意图**：来源判定不依赖 Schema，在第一步即可完成。提前确认来源可避免获取 Schema 后再被文件上传中断上下文。

---

## Step 2：获取当前数据 + Schema（+ 读取文件）

> 静默执行，不向用户展示获取过程。本步骤的多个操作可**并行执行**。

### 2.1 获取当前评估单数据

调用 `getEstimateFormData(apply_id, "base_info_collection")`：
- 提取 `form_id`（供获取 Schema 使用）
- 记录当前各字段的**原始值**（用于 Step 5 合并和 diff 展示）

### 2.2 获取评估单 Schema

调用 `getEstimateFormSchema(form_id)`：
- Schema 获取失败：按 `mcp_call_rules.md` §4 重试策略处理（网络失败自动重试 3 次；其他错误告知用户并终止）
- Schema 为空 → 告知用户并终止
- **优先读取 schema 根级的 `x-form-context`**（表单须知），将其中的字段关系、业务规则等作为后续 Step 3~5 的约束上下文

### 2.3 读取/解析文件内容（如来源含文件）

若 Step 1 判定来源为文件或混合，根据文件类型选择不同处理路径：

#### 2.3.1 判断文件类型

| 文件类型 | 扩展名 | 处理方式 |
|----------|--------|----------|
| **文本类** | `.md` `.txt` `.markdown` | **直接读取**，无需调用脚本 |
| **富格式** | `.xlsx` `.docx` `.pptx` `.pdf` `.html` | 需调用 `scripts/convert.py` 解析为 Markdown |

#### 2.3.2 文本类文件 → 直接读取

文本类文件（`.md` / `.txt` / `.markdown`）已经是可读文本，**直接用 read_file 读取原文件**即可：
1. 读取文件全文
2. 将全文作为后续 Step 3 的**提取源文本**
3. 告知用户已读取文件，概要说明内容

> 不需要调用 `convert.py`，不产生缓存产物，零延迟。

#### 2.3.3 富格式文件 → 调用脚本解析

对于富格式文件（`.xlsx` / `.docx` / `.pptx` / `.pdf` / `.html`），需要调用脚本转为 Markdown：

**a) 调用解析脚本：**

```bash
python3 scripts/convert.py --input "/abs/path/to/file.pdf" --session "default"
```

脚本会输出 JSON 到 stdout，关键字段：
- `output_path`：解析产物的 `.md` 文件路径
- `parser`：使用的解析器（`markitdown` / `ocr` / `cache`）
- `cache_hit`：是否命中缓存（缓存键为 `sha1(abspath + mtime_ns + size)[:12]`，同一文件未修改时秒级返回）
- `pages` / `sheets`：页数/工作表数（用于概要说明）
- `warnings`：解析警告（如 OCR 识别率低等）

> ⚠️ PDF 文件会自动进行"文本层探测"：文本型走 markitdown，图片型自动 fallback 到 OCR（多进程 tesseract）。大文件可能耗时较长（55 页约 40-60 秒），注意等待。

**b) 读取解析产物：**

从 stdout JSON 的 `output_path` 获取 `.md` 文件路径，读取其内容：
1. 读取 `.md` 产物全文
2. 将全文作为后续 Step 3 的**提取源文本**
3. 告知用户已读取文件，概要说明文件内容（如"已读取策划书，共 X 页，包含项目基本信息、主创团队等内容"）

**c) 异常处理：**

| 情况 | 处理 |
|------|------|
| 脚本不存在或依赖缺失 | 提示用户运行 `bash scripts/check_deps.sh` 检查环境 |
| 解析失败（exitCode ≠ 0）| 告知用户文件解析失败，展示 `warnings` 信息，询问是否换文件或手动输入 |
| `bytes < 200` 且 `fallback_used: false` | 文件可能为纯图片型但 OCR 未触发，提示用户确认文件内容 |

#### 2.3.4 通用规则

> ⚠️ 文件可能包含大量信息，涵盖评估单的多个字段。此时**不局限于用户明确提到的字段**——应结合 Schema 尽可能从文件中提取所有可填字段的值。

---

若来源为文本：
1. 将用户的对话文本作为后续 Step 3 的**提取源文本**
2. 遵循**最小化修改原则**：只处理用户明确要求修改的字段

> 💡 **并行优势**：获取 formData → 获取 Schema → 读取/解析文件，三者中前两步有依赖（需 form_id），但"读取/解析文件"可与"获取 formData"并行执行。

**本步骤输出：** Schema + 原始数据 + 确定的提取源文本，交给 Step 3。

**获取到 Schema 后，直接进入 Step 3。**

---

## Step 3：根据 Schema 从内容中提取数据

> 本步骤处理所有需要从提取源文本中抽取值的字段。

### 3.1 字段定位

将提取源文本中的内容映射到 Schema 字段：
- 来源为**文本**时：通过字段的 `title`（中文名）匹配用户提到的字段，确定待修改字段列表
- 来源为**文件**时：遍历 Schema 所有可写字段，尝试从文件内容中提取每个字段的值

### 3.2 按字段 Schema 提取值

对每个待修改字段，根据 Schema 中的类型和约束提取值：

| Schema 约束 | 处理方式 |
|-------------|---------|
| 有 `enum` | 新值必须匹配 `enum` 中的合法值（优先完全相等 → 包含匹配 → 无法匹配则展示选项） |
| `type: string` + `format: date` | 转换为 `YYYY-MM-DD`（支持 `2024/02/14`、`2024年2月14日`、`2月14日` 等） |
| `type: number` / `integer` | 从自然语言中提取数值（如"共36集" → `36`） |
| `type: boolean` | 映射为 `true`/`false`（是/对/有/需要 → true；否/不是/没有 → false） |
| `type: object` | 递归处理 `properties` 中的子字段 |
| `type: array` | 按 `items` 定义处理每个元素 |
| 其余 | 按 `type` 直接赋值 |

### 3.3 x-source 提取指引

对于配置了 `x-source` 的字段，`x-source` 中的信息辅助提取：

| x-source 属性 | 作用 |
|---------------|------|
| `kind=extract` 的 `hint` | 作为提取提示（如"影片名称（书名号/引号内）"） |
| `kind=extract` 的 `examples` | 作为 few-shot 示例辅助抽取 |
| `kind=extract` 的 `onMissing` | 未提取到时：`ask` → 询问用户；`skip` → 跳过；`fail` → 报错终止 |
| `kind=verbatim` | 原文直取，不做改写 |
| `kind=resolve` 的 `extract` 子对象 | 先按 hint/examples 提取关键词（交给 Step 4 调接口） |

**本步骤输出：** 从提取源文本中提取到的字段-值映射（原始提取结果）。

---

## Step 4：根据 Schema 调用三方接口拼接数据

> 本步骤处理所有 `x-source.kind = "resolve"` 的字段——用 Step 3 提取的关键词调用外部接口检索，将结果拼接为结构化数据。

### 4.1 识别 resolve 字段

遍历待修改字段，筛选出 `x-source.kind = "resolve"` 的字段。如果没有 resolve 字段，直接跳到 Step 5。

### 4.2 执行 resolve 流程

> 🚀 **批量执行**：将所有 resolve 字段的查询打包为一次批量调用，脚本内部并发执行，无需 Agent 逐个调用。

对每个 resolve 字段：

1. **准备入参**：按 `resolver.inputFrom` 构造调用参数
   - `source: "extracted"` → 使用 Step 3 的抽取结果
   - `source: "literal"` → 使用 `value` 字面量
   - `source: "context"` → 从运行时上下文按 `key` 取值
   - `source: "env"` → 从环境变量按 `key` 取值

2. **调用 resolve 脚本**：通过 `scripts/resolve.py` 统一调用外部查询接口

   **🚀 推荐：批量模式（一次调用，内部并发）**

   将所有待 resolve 的条目组成 JSON 数组，通过 `--batch` 参数一次性提交：

   ```bash
   python3 scripts/resolve.py --batch '[
     {"type": "talent", "keyword": "张三"},
     {"type": "talent", "keyword": "李四"},
     {"type": "company", "keyword": "正午阳光"},
     {"type": "ip", "keyword": "甄嬛传", "extra": {"play_type": "80"}}
   ]'
   ```

   **批量返回格式：**
   ```json
   {
     "status": "ok",
     "total": 4,
     "results": [
       {"status": "ok", "resolver_type": "talent", "keyword": "张三", "count": 1, "results": [...]},
       {"status": "ok", "resolver_type": "talent", "keyword": "李四", "count": 1, "results": [...]},
       {"status": "ok", "resolver_type": "company", "keyword": "正午阳光", "count": 2, "results": [...]},
       {"status": "error", "resolver_type": "ip", "keyword": "甄嬛传", "error": "..."}
     ]
   }
   ```

   > 💡 批量模式内部使用线程池（最大 8 并发）同时执行所有查询，典型场景下 6~8 个 resolve 耗时从 ~16s 降至 ~2-3s。

   **备选：单条模式（逐个调用）**

   ```bash
   python3 scripts/resolve.py --type <resolver.ref> --keyword <keyword> [--extra '<json>']
   ```

   > `resolver.ref` 的值直接作为 `--type` 参数传入，无需映射。

   **支持的 `resolver.ref` 值：**

   | `resolver.ref` / `--type` | MCP Tool | 用途 |
   |---------------------------|----------|------|
   | `talent` | `talent_simple_query` | 艺人/导演/编剧查询 |
   | `company` | `company_search` | 公司/制作方查询 |
   | `staff` | `getStaffBaseInfo` | 员工信息查询 |
   | `project` | `kb_search` | 项目信息查询 |
   | `ip` | `search_ip` | IP/台账查询 |

   **单条调用示例：**
   ```bash
   # 查艺人
   python3 scripts/resolve.py --type talent --keyword "刘亦菲"
   # → {"status":"ok","resolver_type":"talent","keyword":"刘亦菲","count":1,"results":[...]}

   # 查公司
   python3 scripts/resolve.py --type company --keyword "正午阳光"
   # → {"status":"ok","resolver_type":"company","keyword":"正午阳光","count":1,"results":[...]}

   # 查员工
   python3 scripts/resolve.py --type staff --keyword "clawjone"
   # → {"status":"ok","resolver_type":"staff","keyword":"clawjone","count":1,"results":[...]}

   # 查项目
   python3 scripts/resolve.py --type project --keyword "庆余年"

   # 查 IP/台账
   python3 scripts/resolve.py --type ip --keyword "甄嬛传" --extra '{"play_type":"80"}'
   ```

   **单条返回契约（stdout JSON）：**

   | 字段 | 类型 | 说明 |
   |------|------|------|
   | `status` | `"ok"` \| `"error"` | 调用结果状态 |
   | `resolver_type` | string | 实际使用的 resolver 类型 |
   | `keyword` | string | 查询关键词 |
   | `count` | integer | 匹配到的结果数量 |
   | `results` | array | 结构化结果列表 |
   | `error` | string | 仅 `status="error"` 时存在，错误信息 |

   > ⚠️ 脚本内部已封装 mcporter 调用、参数构造、重试（最多 2 次）和结果提取，Agent 无需自行拼接 MCP 参数。

3. **映射结果**：按 `resolver.output.mapping` 将脚本返回的 `results` 映射到目标字段
   - 批量模式：按 `results` 数组顺序，逐项映射到对应字段
   - item 模式（`extract.cardinality: "multiple"`）：逐个关键词的结果逐个映射
   - batch 模式（`extract.cardinality: "single"` + `output.itemsPath` 存在）：单次调用，按 `itemsPath` 展开数组

4. **数组去重**：若字段有 `x-biz-unique-by`，按指定属性去重

### 4.3 异常处理

| 情况 | 脚本返回 | 处理 |
|------|---------|------|
| 查询到**多条匹配** | `count > 1` | 记录待确认，Step 5 中汇总展示让用户选择 |
| 查询**无结果** | `count = 0` | 按 `onMissing` 处理（ask/skip/fail） |
| 接口**调用失败** | `status = "error"` | 标记该字段为异常，向用户说明 `error` 中的原因 |

> 💡 重试逻辑已内置在 `resolve.py` 中（最多 2 次，间隔 1 秒），Agent 无需手动重试。

**本步骤输出：** resolve 字段的结构化结果（或待确认列表）。

---

## Step 5：结合已有数据，组合最终提交数据

> 本步骤将 Step 3 的提取结果 + Step 4 的 resolve 结果 + Step 2 的原始数据合并，并应用条件规则过滤。

### 5.1 应用条件规则

对每个字段，按以下规则判定是否纳入提交数据：

| 规则 | 条件 | 结果 |
|------|------|------|
| `readOnly: true` | 恒定只读 | **排除**，不修改、不提交 |
| `x-active-when` 表达式 = FALSE | 字段未激活 | **排除**，不提交 |
| `x-readonly-when` 表达式 = TRUE | 条件锁定 | **排除**，保留原值不修改 |
| `x-required-when` 表达式 = TRUE | 条件必填 | 该字段必须有值，否则不得提交 |

> 表达式中的变量取值来自**合并后的最新数据**（原始数据 + 本次修改）。

### 5.2 合并数据

```
最终提交数据 = 用户要修改的字段集合
              ∩ 排除 readOnly 字段
              ∩ 排除 x-active-when=FALSE 的字段
              ∩ 排除 x-readonly-when=TRUE 的字段
```

> 遵循**最小化修改原则**：只提交用户明确要求修改的字段，未涉及的字段不包含在提交数据中。

### 5.3 校验

- 所有值的类型必须与 Schema 中 `type` 定义一致
- `enum` 字段的值在合法范围内
- 必填字段（`required` + `x-required-when=TRUE`）不得为空
- 空值的非必填字段不包含在提交数据中

### 5.4 展示修改预览

```
📝 即将修改评估单的以下字段：

| 字段 | 原值 | 新值 |
|------|------|------|
| XXX  | AAA  | BBB  |

确认修改请回复"确认"，如需调整请直接说明。
```

### 5.5 问题汇总（如有）

若存在待确认问题，**合并展示**后等待用户一次性回复：

- **【需要选择】** — 查询到多条结果，请确认
- **【未找到数据】** — 以下条目在系统中不存在
- **【枚举匹配失败】** — 输入值无法匹配枚举选项
- **【条件字段排除】** — 以下字段因条件不满足已被排除（仅在用户尝试修改了被排除字段时提示）

**【快捷操作】**
> 💡 输入「跳过全部」可一次性忽略所有不确定的非必填字段。

---
> 🛑 **CHECKPOINT**: 展示修改预览，等待用户确认。**禁止在未获得用户明确确认前执行 Step 6。**
---

**用户回复处理：**
- 用户确认（"确认"/"好"/"可以"/"OK"）→ 进入 Step 6
- 用户要求调整 → 修改对应字段，重新走 Step 3~5
- 用户说"取消" → 终止流程，不执行任何更新
- 用户补充了选择/缺失字段 → 填入对应值，重新校验后展示更新后的预览

---

## Step 6：调用 MCP 更新 + 反馈

> 静默执行，不向用户展示工具名、参数或调用过程。

### 6.1 参数自检

- [ ] `apply_id` 已正确填写
- [ ] `task_key` 为 `base_info_collection`
- [ ] 只包含需要修改的字段（最小化修改原则）
- [ ] 所有字段已通过 Step 5 校验

### 6.2 执行调用

调用 `updateEstimateFormData(apply_id, "base_info_collection", data)` 提交更新。

**如调用失败：**

> 按 `mcp_call_rules.md` §4 的错误分类与重试策略执行。

| 情况 | 错误类型 | 处理 |
|------|---------|------|
| 网络超时 / 后端不可用 | 网络失败 | 自动重试 3 次（2s→4s→8s），全部失败后告知用户 |
| 权限不足 | 权限问题 | 不重试，引导用户检查评估单编辑权限 |
| 字段值不合法（如 enum 值不匹配） | 业务逻辑 | 对照 Schema 自动修正后重试 1 次；修正失败则展示问题字段让用户确认 |
| 评估单状态不允许修改（已锁定/已决策） | 业务逻辑 | 不重试，告知用户当前评估单状态不可编辑 |
| 后端 Schema 校验失败 | Schema 校验失败 | 回退到 Step 2 重新获取最新 Schema，修正数据后重试最多 3 次；仍失败则展示具体校验失败字段和原因 |

### 6.3 反馈

**成功：**

```
✅ 评估单更新成功！

已更新字段：
  - XXX：AAA → BBB
  - YYY：CCC → DDD

🔗 评估单链接：https://szgate.szabot.internal/x/r/doto9awl4o?applyId=<apply_id>
```

**失败：**

```
❌ 更新失败
错误信息：[MCP 返回的错误详情]
建议：[根据错误类型给出修复建议]
```
