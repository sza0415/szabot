# MCP 调用规范

> 🔴 **强制引用**：调用任何 MCP 工具前，**必须**对照本文档校验参数。

---

## 1. 调用方式

评估单 Skill 通过 `mcporter` 命令行工具调用 MCP 工具。

**调用格式：**
```bash
mcporter call <MCP_Server> <Tool> --args '<JSON参数>'
```

**正确示例：**
```bash
mcporter call szbotprojectformal listEvaluation --args '{"projID": 3121}'
mcporter call szbotprojectformal getEstimateFormSchema --args '{"form_id": "qpzpmnwe0z"}'
```

**注意事项：**
- `--args` 后的参数必须是合法的 JSON 字符串，用单引号包裹
- MCP Server 名称和 Tool 名称必须与本文档中定义的一致

---

## 2. 工具清单与调用示例

### 2.1 查看评估单列表

```bash
mcporter call szbotprojectformal listEvaluation --args '{"projID": 3974}'
```

> ⚠️ `projID` 为项目 ID，返回该项目下所有评估单的列表信息（评估单ID、项目名称、上会主题、当前状态、创建时间等）。

### 2.2 创建评估单

```bash
mcporter call szbotprojectformal createEvaluation --args '{"projID": 3974, "meetingState": 100, "meetingSubtype": 3, "subject": "立项评估", "decisionDeadline": "2026-05-01"}'
```

**参数说明：**

| 参数 | 必填 | 类型 | 说明 |
|------|------|------|------|
| `projID` | ✅ | integer | 项目 ID |
| `meetingState` | ✅ | integer | 会议状态，固定为 `100`（决策会） |
| `meetingSubtype` | ✅ | integer | 会议子类型：`3`-定期会议，`5`-临时线上决策 |
| `subject` | ✅ | string | 上会主题：`立项评估` / `开机评估` / `简单报备` / `立项+开机` / `提前锁定` / `成片采买` |
| `decisionDeadline` | ✅ | string | 最晚决策日期，格式 `YYYY-MM-DD` |
| `reportType` | 条件必填 | string | 报备类型（仅 `subject` 为 `简单报备` 时必填）：`主创/成本变更` / `商务条款变更` / `其他` |

> ⚠️ 当 `subject` 为 `简单报备` 时，`reportType` 为必填字段。

### 2.3 获取评估单数据

```bash
mcporter call szbotprojectformal getEstimateFormData --args '{"apply_id": 10007380, "task_key": "base_info_collection"}'
```

**参数说明：**

| 参数 | 必填 | 类型 | 说明 |
|------|------|------|------|
| `apply_id` | ✅ | integer | 评估单 ID |
| `task_key` | ✅ | string | 固定为 `base_info_collection` |

> ⚠️ `task_key` 固定传 `base_info_collection`。

### 2.4 更新评估单数据

```bash
mcporter call szbotprojectformal updateEstimateFormData --args '{"apply_id": 10007380, "task_key": "base_info_collection", "data": {"Foneline_intro": "新的一句话梗概"}}'
```

**参数说明：**

| 参数 | 必填 | 类型 | 说明 |
|------|------|------|------|
| `apply_id` | ✅ | integer | 评估单 ID |
| `task_key` | ✅ | string | 固定为 `base_info_collection` |
| `data` | ✅ | object | 需要更新的字段（仅包含修改的字段，遵循最小化修改原则） |

> ⚠️ `task_key` 固定传 `base_info_collection`。`data` 中只包含需要修改的字段，字段名以 Schema 中的 key 为准。

### 2.5 获取评估单 Schema

```bash
mcporter call szbotprojectformal getEstimateFormSchema --args '{"form_id": "qpzpmnwe0z"}'
```

> ⚠️ `form_id` 为评估单模板的唯一标识，不同类型评估单有不同的 form_id。
> `form_id` 通常无需用户提供，可从 `getEstimateFormData` 的返回结果中获取。

### 2.6 外部数据查询（resolve.py）

评估单字段中需要外部数据的 `resolve` 类字段，统一通过 `scripts/resolve.py` 脚本查询，**不再直接调用 MCP 工具**。

**调用格式：**
```bash
python3 scripts/resolve.py --type <type> --keyword <keyword> [--extra '<json>']
```

**支持的查询类型（`--type` 与 Schema `resolver.ref` 一致）：**

| `--type` | 用途 | 调用示例 |
|----------|------|---------|
| `talent` | 艺人/导演/编剧查询 | `python3 scripts/resolve.py --type talent --keyword "刘亦菲"` |
| `company` | 公司/制作方查询 | `python3 scripts/resolve.py --type company --keyword "正午阳光"` |
| `staff` | 员工信息查询 | `python3 scripts/resolve.py --type staff --keyword "clawjone"` |
| `project` | 项目信息查询 | `python3 scripts/resolve.py --type project --keyword "庆余年"` |
| `ip` | IP/台账查询 | `python3 scripts/resolve.py --type ip --keyword "甄嬛传" --extra '{"play_type":"80"}'` |

> 💡 Schema 中 `resolver.ref` 的值直接作为 `--type` 参数传入，无需映射。

**`--extra` 可选参数说明：**

| `--type` | `--extra` 支持的字段 | 默认值 |
|----------|---------------------|--------|
| `talent` | `page_index`, `page_size` | `0`, `50` |
| `company` | `condition_key`, `target` | `"公司名称"`, `["公司ID","公司名称","公司类型","代表作项目","风险性"]` |
| `project` | `by`（`"id"` 按ID查，默认按名称）、`condition`（自定义） | 按名称查询 |
| `ip` | `page_idx`, `page_size`, `play_type` | `1`, `5`, `""` |
| `staff` | —（无额外参数） | — |

**返回格式（stdout JSON）：**
```json
{
  "status": "ok",
  "resolver_type": "talent",
  "keyword": "刘亦菲",
  "count": 1,
  "results": [{ "...": "..." }]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `status` | `"ok"` \| `"error"` | 调用结果状态 |
| `resolver_type` | string | 实际使用的 resolver 类型 |
| `keyword` | string | 查询关键词 |
| `count` | integer | 匹配到的结果数量 |
| `results` | array | 结构化结果列表 |
| `error` | string | 仅 `status="error"` 时存在 |

> ⚠️ 脚本内部已封装 MCP 工具调用、参数构造、重试（最多 2 次，间隔 1 秒）和结果提取。Agent 无需自行拼接 MCP 参数或处理重试。

---

## 3. 提交前自检清单

> 在调用任何写入类 MCP 工具前，逐项检查：

- [ ] **枚举值正确**：`enum` 字段的值在 Schema 定义的合法范围内
- [ ] **类型正确**：所有字段值的类型与 Schema 中 `type` 定义一致
- [ ] **必填完整**：所有 `required` 字段已填写
- [ ] **条件字段**：`if/then` 条件不成立时，对应的必填字段不参与验证
- [ ] **最小化原则**：更新操作只包含需要修改的字段
- [ ] **resolve 字段**：外部查询统一通过 `scripts/resolve.py` 调用，不直接调用 MCP 查询工具
- [ ] **参数格式**：`mcp_call_tool` 的 `arguments` 为合法 JSON 字符串

---

## 4. 错误分类与重试策略

> 🔴 **强制引用**：所有 MCP 调用失败时，必须按本章节的错误分类判定错误类型，并执行对应的重试策略。各 workflow 中的异常处理均遵循本规则。

### 4.1 错误分类

| 错误类型 | 识别特征 | 说明 |
|---------|---------|------|
| **网络失败** | `timeout`、`ECONNREFUSED`、`ENOTFOUND`、`socket hang up`、`network error`、HTTP 5xx、`connect ETIMEDOUT` | 网络层不可达或后端暂时不可用 |
| **权限问题** | HTTP 401/403、`permission denied`、`unauthorized`、`forbidden`、`no access`、`token expired` | 用户无权限或认证过期 |
| **业务逻辑问题** | HTTP 400/404/409/422、`not found`、`already exists`、`invalid parameter`、`conflict`、明确的业务错误码 | 请求参数不合法、资源不存在、状态冲突等 |
| **Schema 校验失败** | `validation failed`、`schema validation`、`invalid field`、`enum mismatch`、`type mismatch`、`required field missing` | 后端对提交数据进行 Schema 校验未通过 |

> ⚠️ 错误类型判定优先级：Schema 校验失败 > 业务逻辑问题 > 权限问题 > 网络失败（从具体到宽泛）

### 4.2 重试策略

| 错误类型 | 可重试 | 最大重试次数 | 重试间隔 | 重试前操作 | 用户交互 |
|---------|--------|------------|---------|-----------|---------|
| **网络失败** | ✅ 自动重试 | 3 次 | 递增：2s → 4s → 8s | 无需修改参数，直接重试 | 静默重试；全部失败后告知用户"网络异常，建议稍后重试"，提供「立即重试」选项 |
| **权限问题** | ❌ 不重试 | 0 | — | — | 立即告知用户，说明权限要求，引导检查：①确认登录状态 ②确认项目/评估单权限 ③联系管理员授权 |
| **业务逻辑问题** | ⚠️ 条件重试 | 1 次 | 即时 | 根据错误信息**自动修正参数**后重试 | 若自动修正成功 → 静默重试；若无法自动修正 → 展示错误原因，引导用户提供正确信息 |
| **Schema 校验失败** | ⚠️ 条件重试 | 3 次 | 即时 | **重新获取最新 Schema**，对照校验并修正数据后重试 | 第 1~2 次静默修正重试；第 3 次仍失败 → 展示具体校验失败字段和原因，请用户确认或手动修正 |

### 4.3 各错误类型的详细处理流程

#### 4.3.1 网络失败

```
接口调用 → 网络失败
  ├─ 自动重试（第 1 次，等待 2s）
  │    ├─ 成功 → 继续流程
  │    └─ 失败 → 自动重试（第 2 次，等待 4s）
  │         ├─ 成功 → 继续流程
  │         └─ 失败 → 自动重试（第 3 次，等待 8s）
  │              ├─ 成功 → 继续流程
  │              └─ 失败 → 告知用户 ↓
  └─ 输出：
     ⚠️ 网络连接异常，已自动重试 3 次仍未成功。
     可能原因：后端服务暂时不可用或网络波动。
     建议：请稍等片刻后告诉我"重试"，我将再次尝试。
```

#### 4.3.2 权限问题

```
接口调用 → 权限错误
  └─ 不重试，直接输出：
     🔒 操作被拒绝：权限不足。
     错误详情：[具体错误信息]

     请检查以下事项：
     1. 确认您已登录且 session 未过期
     2. 确认您对该项目/评估单有编辑权限
     3. 如需权限，请联系项目管理员授权

     如已解决权限问题，请告诉我"继续"。
```

#### 4.3.3 业务逻辑问题

```
接口调用 → 业务逻辑错误
  ├─ 分析错误信息，判断是否可自动修正：
  │    ├─ 可自动修正的情况：
  │    │    - 资源 ID 格式错误（如 string 应为 integer）→ 自动转换类型
  │    │    - 日期格式不对 → 自动转换为 YYYY-MM-DD
  │    │    - 项目 ID 不存在但用户提供了项目名 → 调用 kb_search 获取正确 ID
  │    │    └─ 修正后静默重试（1 次）
  │    │         ├─ 成功 → 继续流程
  │    │         └─ 失败 → 告知用户
  │    └─ 无法自动修正的情况：
  │         - 评估单已存在（重复创建）
  │         - 评估单状态不允许修改（已锁定/已决策）
  │         - 必填参数缺失且无法推断
  │         └─ 直接告知用户
  └─ 输出（无法修正时）：
     ❌ 操作失败：[业务错误描述]
     错误详情：[具体错误信息]

     原因分析：[根据错误码/信息分析具体原因]
     建议操作：[针对性的修复建议]
```

#### 4.3.4 Schema 校验失败

```
接口调用 → Schema 校验失败
  ├─ 第 1 次修正重试：
  │    1. 重新调用 getEstimateFormSchema 获取最新 Schema
  │    2. 对照最新 Schema 检查提交数据：
  │       - 枚举值不在合法范围 → 映射到最接近的合法值
  │       - 类型不匹配 → 转换类型（如 "36" → 36）
  │       - 必填字段为空 → 从上下文/原始数据补充
  │       - 字段名变更 → 按新 Schema 的 key 重新映射
  │    3. 修正后重试
  │    ├─ 成功 → 继续流程
  │    └─ 失败 → 第 2 次修正重试 ↓
  ├─ 第 2 次修正重试：
  │    1. 逐字段对比 Schema 约束，定位全部不合规字段
  │    2. 能自动修正的字段修正，不能的标记出来
  │    3. 修正后重试
  │    ├─ 成功 → 继续流程
  │    └─ 失败 → 第 3 次修正重试 ↓
  ├─ 第 3 次修正重试：
  │    1. 将完整提交数据与 Schema 逐字段严格比对
  │    2. 尝试移除所有非必填的可疑字段，仅保留必填+已确认正确的字段
  │    3. 修正后重试
  │    ├─ 成功 → 继续流程
  │    └─ 失败 → 告知用户 ↓
  └─ 输出（3 次修正仍失败）：
     ❌ 数据校验失败，无法自动修正。

     以下字段未通过后端校验：
     | 字段 | 提交值 | 校验要求 | 失败原因 |
     |------|--------|---------|---------|
     | XXX  | AAA    | BBB     | CCC     |

     请确认以上字段的正确值，或告诉我如何调整。
```

### 4.4 重试计数器与上下文

> Agent 在执行重试时，需维护以下内部状态（不向用户展示）：

| 状态 | 说明 |
|------|------|
| `retry_count` | 当前操作已重试次数 |
| `error_type` | 最近一次错误分类 |
| `last_error_msg` | 最近一次完整错误信息 |
| `auto_fix_applied` | 是否已应用自动修正 |

### 4.5 跨步骤重试规则

- **读取类操作**（`getEstimateFormData`、`getEstimateFormSchema`、`listEvaluation`）：网络失败时自动重试 3 次，无需回退步骤
- **写入类操作**（`createEvaluation`、`updateEstimateFormData`）：
  - 网络失败重试 → 保持当前步骤不变
  - Schema 校验失败重试 → 回退到"获取 Schema"步骤重新获取后再提交
  - 业务逻辑修正重试 → 保持当前步骤，仅修正参数
- **resolve 脚本**：内部已有 2 次重试，Agent 层面不额外重试；若脚本最终返回 `status: "error"`，按 §4.3.3 业务逻辑问题处理

### 4.6 全部失败后的兜底

若所有重试策略均耗尽仍未成功：

```
❌ 操作未能完成

错误类型：[网络失败/权限问题/业务逻辑/Schema校验]
已尝试：[自动重试 N 次 / 自动修正 N 次]
最后错误：[具体错误信息]

建议：
1. [根据错误类型给出的针对性建议]
2. 如需我再次尝试，请回复"重试"
3. 如需手动处理，评估单链接：https://szgate.szabot.internal/x/r/doto9awl4o?applyId=<apply_id>
```
