# szabot：可扩展的 Go Agent 框架与 Skill 评审系统

## 简历项目经历

**项目名称：** szabot 智能 Agent 框架与 Skill 评审平台  
**项目类型：** 个人项目  
**技术栈：** Go、HTTP、SSE、JSONL、Docker、OpenAI-compatible API、HTML/CSS/JavaScript

### 项目简介

基于 Go 设计并实现可扩展的 Agent 框架，支持多轮对话、LLM Tool Calling、Skill 渐进式加载、会话持久化和 CLI/Web 双通道交互；在此基础上实现 Skill Path 评审系统，将 Skill 的输入条件、判断分支、Tool 调用、输出校验和异常处理拆解为可观测路径，并提供覆盖率分析、失败节点定位、回归测试和可视化评审仪表盘。

### 核心工作

- 设计基于 `MessageBus`、`Channel`、`AgentLoop`、`AgentRunner` 和 `Provider` 的分层架构，解耦消息接入、Agent 调度、模型调用和本地工具执行，支持 CLI 与 Web 通道复用同一套核心逻辑。
- 实现 OpenAI-compatible Provider，兼容 DeepSeek、OpenAI、Moonshot、Ollama 等服务，并处理流式响应和增量 Tool Calling 参数组装。
- 实现多轮 Tool Calling 执行链路：解析模型工具请求、校验工具参数、执行本地工具、回传工具结果，并继续请求模型直到生成最终答案。
- 实现受工作区限制的文件工具和执行工具，包括 `read_file`、`write_file`、`edit_file`、`list_dir`、`glob`、`grep`、`bash` 和 `python`，通过路径校验与 Docker 沙盒降低越权访问和宿主机执行风险。
- 设计 Skill 三层渐进式披露机制：启动阶段只注入元数据，触发时按需读取 `SKILL.md`，子资源和脚本按需加载，降低系统提示词上下文开销。
- 实现基于 JSONL 的 Session Store，按 `SessionID` 持久化用户、Assistant 和 Tool 消息，支持会话恢复、追加写入、异常恢复和路径穿越防护。
- 构建 Skill 评审引擎，将 Path 建模为由输入识别、参数校验、条件判断、Tool 调用、输出校验和异常兜底组成的有序节点链路。
- 实现 Path / Node 覆盖率、路径匹配率、节点通过率和输出通过率等指标，并输出 JSON、Markdown 和 Web 仪表盘三种评审结果。
- 实现失败节点诊断和全量回归能力，支持识别 Path 偏移、缺失节点、错误 Tool 调用、禁止节点触发和输出类型不匹配等问题。

### 项目亮点

- **架构可扩展：** Provider、Channel、Tool 和 Skill 均通过明确边界扩展，新增模型供应商或交互入口不需要修改 Agent 核心循环。
- **执行过程可观测：** 不只评估最终答案，还记录实际节点路径、Tool 调用和失败位置，支持从结果反向定位 Skill 规则问题。
- **评审结果可回归：** 测试用例、Path 定义和实际 Trace 均结构化保存，Skill 修改后可以执行全量回归并比较前后结果。
- **安全边界清晰：** 文件操作限制在工作区内，bash/python 通过 Docker 沙盒执行，并支持默认断网和临时目录大小配置。
- **交互方式完整：** 同时支持零依赖 Echo Provider、OpenAI-compatible Provider、CLI 和浏览器 Web 界面，便于本地开发和演示。

## 一页简历精简版

**szabot 智能 Agent 框架与 Skill 评审平台｜Go、SSE、Docker、OpenAI-compatible API**

- 设计并实现分层 Agent 框架，基于 MessageBus、AgentLoop、AgentRunner 和 Provider 解耦消息通道、模型调用与工具执行，支持 CLI/Web 双通道和多轮对话。
- 实现 OpenAI-compatible Provider、流式响应、增量 Tool Calling 组装及本地工具执行链路，支持文件操作、搜索、bash/python 沙盒等能力。
- 设计 Skill 三层渐进式披露机制和 JSONL 会话存储，降低上下文开销并实现跨请求会话恢复。
- 构建 Skill Path 评审系统，将输入识别、条件判断、Tool 调用、输出校验和异常处理建模为可观测节点，提供覆盖率统计、失败节点定位、Markdown/JSON 报告和 Web 可视化仪表盘。
- 通过工作区路径约束、SessionID 清洗、Docker 隔离和默认断网策略完善工具执行安全边界。

## 面试讲解主线

### 1. 为什么设计 MessageBus

MessageBus 统一承载入站和出站消息，使 CLI、Web 等 Channel 不需要直接依赖 Agent 内部实现。AgentLoop 只关心消息处理，Channel 只负责协议转换，降低了组件之间的耦合。

### 2. 为什么 Skill 采用渐进式披露

如果启动时把所有 Skill 正文和资源都放入 system prompt，会增加上下文长度并影响模型注意力。项目将 Skill 拆成元数据、正文和子资源三层：常驻元数据用于触发判断，正文按需读取，脚本和参考资料只在需要时加载。

### 3. 如何保证 Tool Calling 能够连续执行

Runner 收到模型的 Tool Calling 响应后，先根据 Tool 名称查找注册表，再解析和校验参数，执行工具并将结果以 Tool 消息加入上下文，随后继续请求模型。只有模型没有新的 Tool Calling 时，才结束本轮执行并输出最终答案。

### 4. Skill 评审系统解决什么问题

只看最终答案或总分无法定位 Skill 的具体缺陷。评审系统先为 Skill 生成包含判断节点和 Tool 节点的 Path，再将实际执行 Trace 与预期 Path 对比，从而定位是输入识别、分支选择、工具调用、输出校验还是异常处理出了问题。

### 5. 如何处理执行安全

文件类工具统一限制在 workspace 范围内，并对路径进行规范化和穿越检查；bash/python 不直接在宿主机执行，而是通过 Docker 沙盒运行，默认关闭网络，并限制临时目录大小。

## 简历使用建议

- 如果应聘后端或基础架构岗位，优先保留 MessageBus、Provider、Tool Calling、会话存储和沙盒安全部分。
- 如果应聘 AI Agent 岗位，优先突出多轮 Tool Calling、Skill 渐进式披露、Path 评审和执行 Trace。
- 如果简历篇幅有限，使用“一页简历精简版”，保留 4 至 5 条项目要点即可。
- 不建议在没有实际数据时填写性能提升百分比、准确率或吞吐量；可以在完成基准测试后再补充量化结果。
