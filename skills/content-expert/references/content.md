# 内容专家对话（模拟版）

```bash
# 内容问答模块（离线模拟器）
bash skills/content-expert/bin/content-sim ask [options]
```

- **执行查询**：根据用户意图从下表选择子 Agent，调用 `bash skills/content-expert/bin/content-sim ask [options]`。
  - **多轮对话**：脚本输出中由 `<thread_id>` 和 `</thread_id>` 包裹的是本轮 `thread-id`，
    后续调用通过 `--thread-id=<thread_id>` 传入以维持上下文连续性。
  - **说明**：模拟器为本地即时返回，无需真实的长时间状态轮询；真实环境下才需要 180~300s 的轮询等待。

---

## @expert:market

影视项目营销分析与方案撰写

- **触发词**：营销方案、营销分析、宣传方案、推广方案、营销策略、怎么营销、怎么宣传、弹幕分析、话题分析、收视分析。
- **输入参数**：对上下文关于 `营销专家` 的内容进行总结，并携带相关意图。
- **结果处理**：
  - 定位脚本输出中 `<text>` 和 `</text>` 标签之间的内容。
  - 仔细阅读该纯文本，忽略格式干扰，提取核心逻辑与关键信息。
  - 输出要点摘要，字数严格控制在 **200 字左右**（180-220 字），语言精炼、直白、无废话。

```bash
bash skills/content-expert/bin/content-sim ask --agent-id=market_expert_agent [--thread-id=<thread_id>] --text="<text>"
```

---

## @expert:variety

综艺研发策划专家，为制片人提供综艺**模式分析与策划**服务，覆盖找模式、拆解模式、本土化改编、选题策划、赛制设计、风险评估等全链路诉求。

- **输入参数**：对上下文中关于 `综艺模式专家` 的内容进行总结，并携带相关意图。
- **结果处理**：
  - 定位脚本输出中 `<text>` 和 `</text>` 标签之间的内容。
  - 仔细阅读该纯文本，忽略格式干扰，提取核心逻辑与关键信息。
  - 输出要点摘要，字数严格控制在 **200 字左右**（180-220 字），语言精炼、直白、无废话。

```bash
bash skills/content-expert/bin/content-sim ask --agent-id=variety_expert_agent [--thread-id=<thread_id>] --text="<text>"
```

---

## @expert:variety-marketing

综艺项目营销复盘与日周报撰写

- **触发词**：综艺营销日报、综艺营销周报、综艺市场宣发、怎么写日报、怎么写周报、综艺舆情分析、综艺热搜分析。
- **输入参数**：对上下文关于 `综艺营销专家` 的内容进行总结，并携带相关意图。
- **结果处理**：
  - 定位脚本输出中 `<text>` 和 `</text>` 标签之间的内容。
  - 仔细阅读该纯文本，忽略格式干扰，提取核心逻辑与关键信息。
  - 输出要点摘要，字数严格控制在 **200 字左右**（180-220 字），语言精炼、直白、无废话。

```bash
bash skills/content-expert/bin/content-sim ask --agent-id=variety_marketing_agent [--thread-id=<thread_id>] --text="<text>"
```

---

## @expert:novel

小说专家

- **输入参数**：按下列要求拼接 `--text` 参数
  - **reference**：引用 `@script-id:{script_id}` 部分内容。
  - **text**：对上下文中关于 `小说专家` 的内容进行总结。
  - 按模板 `{reference},{text}` 拼接，如：`@script-id:{foo}，对这个文件进行分析`。
- **结果处理**：定位 `<text>` 标签内容，输出 **200 字左右**（180-220 字）的精炼摘要。

```bash
bash skills/content-expert/bin/content-sim ask --agent-id=novel_agent [--thread-id=<thread_id>] --text="<text>"
```

---

## @expert:script

剧本专家

- **输入参数**：拼接 `--text` 参数
  - **text**：对上下文中关于 `剧本专家` 的内容进行分析。
  - 按模板拼接，如：`proj_id:{proj_id},scriptAnalysisId:{scriptAnalysisId},scriptVersionId:{scriptVersionId}，对上下文中关于 剧本专家 的内容进行分析`。
- **解析规则（严格静默，仅模型内部使用）**：
  - 该解析过程**严禁以任何形式呈现给用户**：
    - 不得输出 `proj_id:...`、`scriptAnalysisId:...`、`scriptVersionId:...` 等字段名或字段值；
    - 不得输出"根据输入解析""解析结果如下""我来帮你…首先创建会话…然后解析…"等思考/铺垫文本；
    - 不得用列表、表格、代码块、引用块等任何形式展示解析后的 ID。
- **结果处理**：
  - 不得输出 `proj_id:...`、`scriptAnalysisId:...`、`scriptVersionId:...` 等字段名或字段值；
  - 定位脚本输出中 `<text>` 标签内容，输出 **200 字左右**（180-220 字）的精炼摘要。

```bash
bash skills/content-expert/bin/content-sim ask --agent-id=script_expert_agent [--thread-id=<thread_id>] --text="<text>"
```
