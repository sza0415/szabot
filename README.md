<p align="center">
  <img src="assets/yomi-logo.svg" alt="yomi" width="620">
</p>

<p align="center">一个用 Go 实现的轻量 agent 框架</p>

<br>

# yomi

一个用 Go 实现的轻量 agent 框架，围绕一条极简的 agent 循环（消息总线 → AgentLoop → Runner → Provider）构建，核心保持精简，能力从边缘扩展。

> 项目名称统一为 **yomi**。当前代码入口 `cmd/szabot`、环境变量 `SZABOT_*` 以及部分存储路径仍保留早期技术标识，以兼容现有用法；后续如需重命名，应连同代码和迁移方案一起调整。

主要特性：

- **多通道**：CLI（stdin/stdout）与 Web（HTTP + SSE 流式聊天）两种前端。
- **多 Provider**：统一 LLM 接口，内置 Echo（零依赖验证）与 OpenAI-compatible 实现；主程序当前通过环境变量装配 Echo 或 DeepSeek。
- **工具箱**：文件类工具（read/write/edit/list_dir/glob/grep）开箱即用，可选的 bash/python 执行类工具跑在 Docker 沙盒里。
- **技能系统（Skills）**：三层渐进式披露，L1 元数据常驻、L2/L3 按需 `read_file` 加载，往 `skills/` 丢个目录即可扩展。
- **会话历史**：按 SessionID 持久化为 jsonl，同一会话自动带上上下文。
- **Skill 评审工具（skill-review）**：独立的本地工作台，对 Skill 做执行路径建模与预期/实际对比，支持 LLM 抽取 Path，不依赖 Docker。

> 设计宪法：Core stays small，所有新功能挂在 channel / tool / provider / skill 边上，不往核心循环塞业务。

## 目录结构

```
szabot/
├── cmd/
│   ├── szabot/             # CLI 入口（main.go：只做装配）
│   └── skill-review/       # Skill 路径评审工具（独立，不依赖 Docker）
├── internal/
│   ├── bus/                # 消息总线（系统中枢）
│   ├── agent/              # 核心循环
│   │   ├── loop.go         #   外层：消费 bus、协调上下文
│   │   └── runner.go       #   内层：跟 LLM 来回打交道
│   ├── channels/           # 通道（平台翻译官）
│   │   └── cli.go          #   stdin/stdout 实现
│   ├── skills/             # 技能系统（三层渐进式披露）
│   ├── skillreview/        # Skill 评审内核（Path/Node/Case/Run/Evaluate/Report）
│   └── providers/          # LLM 提供商
│       ├── provider.go              #   统一接口
│       ├── echo.go                  #   假实现，零依赖验证链路
│       └── openai_compatible.go     #   OpenAI 兼容（DeepSeek/OpenAI/Moonshot/Ollama...）
└── go.mod
```

## 5 个核心角色

| 角色 | 朝向 | 职责 |
|---|---|---|
| **MessageBus** | 中枢 | 用 Go channel 实现的入站/出站两条队列 |
| **Channel** | 朝外 | 把平台原生消息 ⇄ 统一 InboundMessage/OutboundMessage |
| **AgentLoop** | 中间 | 从 bus 接消息、加载 session、调 Runner、推回 bus |
| **AgentRunner** | 朝内 | 跟 Provider 来回对话，驱动多轮 tool calling 循环 |
| **Provider** | 朝内 | 统一 Echo 与 OpenAI-compatible LLM API |

## 运行

### 默认（Echo，零依赖）

```bash
go run ./cmd/szabot
```

```
> 你好
yomi> echo: 你好
```

### 接入 DeepSeek（OpenAI 兼容）

```bash
export SZABOT_PROVIDER=deepseek
export DEEPSEEK_API_KEY=sk-xxxxxxxx

# 思考模式建议显式指定当前支持思考的模型
export DEEPSEEK_MODEL=deepseek-v4-pro

# 可选，默认如下
# export DEEPSEEK_BASE_URL=https://api.deepseek.com/v1

go run ./cmd/szabot
```

当前代码默认模型仍为 `deepseek-chat`，因此建议始终显式设置
`DEEPSEEK_MODEL`。OpenAI-compatible Provider 已支持从非流式响应和 SSE 增量中解析
`reasoning_content`，思考过程会与最终正文分开传递，并可写入 Trace。

> Provider 选择目前由 `cmd/szabot` 的环境变量装配逻辑提供 Echo 和 DeepSeek
> 两个入口；OpenAI-compatible 类型本身可复用，但主程序尚未提供通用 provider
> 配置项。`cmd/skill-review` 另支持 `SZABOT_PROVIDER=openai`。
>
> 安全提示：当前 `buildProvider` 会把 `DEEPSEEK_API_KEY` 明文打印到控制台，
> 请勿在共享终端或 CI 中运行；修复该日志前仅建议用于本地调试。

按 Ctrl+C 退出。

### Web 界面（HTTP + SSE）

除了 CLI，yomi 还提供浏览器聊天界面。设置 `SZABOT_WEB` 即启用；此时不再启动 CLI，
因为两者会争抢同一条出站消息总线：

```bash
export SZABOT_WEB=1

# 建议显式限制为本机访问；代码当前默认监听 :8080（所有可用网卡）
export SZABOT_WEB_ADDR=127.0.0.1:8080

# Provider 照旧（不设则使用 echo；接入 DeepSeek 见上一节）
export SZABOT_PROVIDER=deepseek
export DEEPSEEK_API_KEY=sk-xxxxxxxx
export DEEPSEEK_MODEL=deepseek-v4-pro

go run ./cmd/szabot
```

启动后打开 `http://127.0.0.1:8080` 即可对话，回复以打字机效果**流式**呈现。

> 安全边界：当前 Web 服务没有用户认证和访问控制，只适合可信的本机环境，
> 不应直接暴露到公网或不可信局域网。浏览器提供的 SessionID 只用于消息路由和
> Conversation 关联，不是认证凭据；知道 SessionID 的客户端可能订阅或影响对应会话。
> 文件工具、`web_fetch` 和可选执行工具会进一步扩大服务暴露后的风险。

实现要点：

- **零依赖**：纯标准库 `net/http`，前端单页 HTML 用 `//go:embed` 打进二进制，依旧是“一个可执行文件”。
- **SSE 而非 WebSocket**：Agent 输出是服务端单向流式推送；用户消息通过 `POST /api/send` 提交，浏览器通过 `GET /api/stream?session=...` 接收事件。
- **按 SessionID 分发**：Web Channel 由单个 goroutine 消费共享的 `bus.Outbound()`，再按 SessionID 扇出到对应连接。该机制用于隔离消息路由，不提供身份认证。
- **会话延续**：浏览器把 SessionID 存在 `localStorage`，刷新后继续关联同一段 Conversation。

## Conversation 与 Trace

yomi 将“后续对话需要的主线历史”和“用于观察、排障的完整运行轨迹”分开存储：

```text
sessionlogs/
├── conversations/   # Conversation：按 Session 保存对话主线
└── traces/          # Trace：按 Run 保存完整执行轨迹
```

默认根目录是 yomi 的启动工作目录下的 `sessionlogs/`。可以使用
`SZABOT_SESSION_DIR` 覆盖；相对路径以启动工作目录为基准：

```bash
export SZABOT_SESSION_DIR=/data/yomi-sessions
```

### Conversation：对话上下文

Conversation 会进入同一 Session 的后续模型请求。只有 Run 成功完成后，才会成对追加：

- 本轮原始 user 消息；
- 最终 assistant 正文。

它不保存 system prompt、reasoning、工具调用、工具结果和临时 Agent 状态栏。例如：

```jsonl
{"role":"user","content":"我叫小明"}
{"role":"assistant","content":"你好，小明！"}
```

CLI 的 SessionID 固定为 `cli:local`，默认文件是
`sessionlogs/conversations/cli:local.jsonl`。Web SessionID 通常由浏览器生成并保存在
`localStorage` 中。

模型请求的主要上下文顺序是：

```text
system prompt（启动时构建，不进入 Conversation）
→ Conversation 历史
→ 本轮 user
→ Runner 临时注入的 Agent 状态栏（存在时）
```

### Trace：运行轨迹

Trace 不作为长期对话历史回放给模型。它按 Run 保存 JSONL 事件，包括 Run 生命周期、
system prompt、用户输入、实际模型请求、reasoning、assistant 消息、工具参数与结果、
用户问答、错误、耗时和 usage。每个文件以 RunID 的 SHA-256 值命名。

> Trace 包含大量未脱敏原文，可能涉及提示词、用户输入、文件内容和工具结果。
> 当前没有自动轮转、保留期或脱敏机制，请谨慎保管和清理。

### 重置与并发限制

- 重置 Conversation 时，应先停止 yomi，再删除 `conversations/` 中对应文件并重新启动；运行中直接删除文件不会清除内存缓存。
- Trace 文件不能从文件名直接反推出 SessionID；如需全部清理，可在停止 yomi 后清空 `traces/`。
- 当前锁只覆盖单个进程，不要让多个 yomi 进程共享同一个 `SZABOT_SESSION_DIR`。
- 早期版本位于 `~/.szabot/sessions/` 或 `sessionlogs/` 根目录的历史文件，需要手动迁移到 `conversations/`。

完整的数据格式、事件清单、写入时机和安全边界见
[`docs/conversation-and-trace.md`](docs/conversation-and-trace.md)。

## 路线图

- [x] M1 项目骨架
- [x] M2 MessageBus
- [x] M3 AgentLoop + AgentRunner（最小循环）
- [x] M4 CLI Channel
- [x] M5 EchoProvider（验证链路）
- [x] M6 OpenAI 兼容 Provider（DeepSeek 已接入）
- [ ] M7 配置加载（~/.szabot/config.json）
- [x] M8 Session 存储（jsonl）
- [x] M9 Tool 接口 + 文件工具（read/write/edit/list_dir/glob/grep）
- [x] M10 Runner 多轮 + tool calling 循环
- [x] M10.5 执行类工具 bash/python（Docker 沙盒，SZABOT_SANDBOX=1 启用）
- [x] M10.6 通用工具 web_search / web_fetch / todo_write / ask_user_question
- [x] M11 第二个 Channel（HTTP + SSE Web 界面）
- [ ] M12 长期记忆（MEMORY.md）

## 工具箱

| 工具 | 说明 | 依赖 |
|---|---|---|
| `read_file` / `write_file` / `edit_file` | 读取、覆盖写和精确替换工作区文本文件 | 无 |
| `list_dir` / `glob` / `grep` | 浏览目录、按名称查找、按正则搜索内容 | 无 |
| `web_fetch` | 读取 HTTP/HTTPS 页面并提取正文 | 无 |
| `todo_write` | 按 Session 维护任务清单和进度 | 无 |
| `ask_user_question` | 通过当前 Channel 向用户提问并等待回答 | 无 |
| `web_search` | 通过 Tavily 搜索互联网 | `TAVILY_API_KEY` |
| `bash` / `python` | 在临时 Docker 容器中执行命令或代码 | Docker + `SZABOT_SANDBOX=1` |

### 工具注册条件

`read_file`、`write_file`、`edit_file`、`list_dir`、`glob`、`grep`、`web_fetch`、
`todo_write` 和 `ask_user_question` 默认注册。`web_search` 仅在设置
`TAVILY_API_KEY` 后注册；`bash` 和 `python` 仅在显式启用 Docker 沙盒且 Docker
可用时注册：

```bash
export TAVILY_API_KEY=tvly-xxxx
```

> `web_fetch` 当前只校验 URL 为 HTTP/HTTPS，没有阻止访问 localhost、私网地址、
> 云元数据地址或重定向后的内网目标。它不适合直接开放给不可信用户。

### Docker 沙盒

```bash
export SZABOT_SANDBOX=1

# 可选
# export SZABOT_SANDBOX_NETWORK=1       # 允许容器联网，默认关闭
# export SZABOT_PYTHON_IMAGE=python:3.12-slim
# export SZABOT_BASH_IMAGE=debian:stable-slim
# export SZABOT_SANDBOX_TMP_SIZE=512m   # 默认 64m
```

未开启 `SZABOT_SANDBOX` 或找不到 Docker 时，yomi 会跳过 `bash` 和 `python`，
其他工具仍可使用。每次执行都会创建临时容器，默认限制为 30 秒、512 MB 内存、
1 个 CPU、256 个进程和 64 KiB 返回输出；根文件系统只读，`/tmp` 使用临时文件系统，
网络默认关闭。

> Docker 沙盒不是只读工作区：宿主 workspace 会以**读写方式**挂载到容器的
> `/work`。容器内命令可以创建、修改或删除工作区文件，这些变化会直接影响宿主机。
> 沙盒主要隔离工作区之外的文件系统、网络和资源，不能替代权限控制、备份或代码审查。

文件工具拒绝绝对路径和 `..` 路径穿越；`read_file`、`edit_file` 还会拒绝解析后逃出
workspace 的符号链接。当前 `write_file` 只进行词法路径检查，若目标或父目录是指向
workspace 外部的符号链接，仍存在越界写风险。因此不要在工作区放置指向敏感位置的
可写符号链接，也不要把工具能力开放给不可信用户。

Docker 安装、镜像选择、完整限制和安全边界见
[`docs/tools-and-sandbox.md`](docs/tools-and-sandbox.md)。

## 技能（Skills）

yomi 从 workspace 的 `skills/<name>/SKILL.md` 发现技能。当前仓库包含：

| Skill | 加载方式 | 作用 | 主要依赖 |
|---|---|---|---|
| `szbot` | 常驻（`always: true`） | 影库意图识别、路由、输入消歧和公共护栏 | 无 |
| `kbcli` | 按需读取 | 影视综数据查询、结构化 ID 查询和专家问答 | `kbcli` CLI |

技能采用渐进式披露：

1. 非常驻 Skill 的名称、描述、路径和依赖状态作为 L1 摘要注入 system prompt；
2. Agent 命中意图后使用 `read_file` 读取对应 `SKILL.md`；
3. 复杂流程再按 Skill 指引读取 `references/` 等 L3 资源；
4. `always: true` 且依赖满足的 Skill 正文会直接常驻 system prompt，不重复出现在 L1 摘要中。

当前 `szbot` 是常驻路由入口，负责将请求路由到 `kbcli`；`kbcli` 的查数与专家问答
是互斥链路，执行前必须读取其 `SKILL.md` 和对应 `references/`。实际能力与参数以
[`skills/szbot/SKILL.md`](skills/szbot/SKILL.md) 和
[`skills/kbcli/SKILL.md`](skills/kbcli/SKILL.md) 为准。

Skill 是否可用由 frontmatter 的 `requires.bins` 和 `requires.env` 决定。依赖缺失时，
Loader 仍会在 L1 摘要中列出该 Skill，但标记为 `unavailable`。代码还支持 workspace
Skill 覆盖同名内置 Skill 和显式禁用 Skill；当前主程序只装载 workspace 的 `skills/`。

详细设计见 [`docs/skill-execution-path-review.md`](docs/skill-execution-path-review.md)。

## Skill 评审（skill-review）

`cmd/skill-review` 是独立的本地评审工具，用于管理 `SKILL.md`、生成执行 Path，
并将测试用例的预期路径与实际 Run 对比。它本身不执行 Skill，也不要求 Docker。

启动 Web 工作台：

```bash
go run ./cmd/skill-review \
  -serve \
  -addr 127.0.0.1:8090 \
  -workspace .
```

生成 Markdown/JSON 报告：

```bash
go run ./cmd/skill-review \
  -cases cases.json \
  -runs runs.json \
  -paths paths.json \
  -markdown report.md \
  -json report.json \
  -skill-version local
```

非 Web 模式下，`-cases` 和 `-runs` 必填，`-paths` 可选。评审结果包含 Skill 命中、
执行链、Path/Node、注意事项和输出等指标。

Path 生成支持 `auto`、`llm` 和 `rule` 三种引擎。`auto` 默认优先使用已配置的 LLM，
失败后回退规则抽取；生成结果可缓存为 `skills/<name>/PATH.json`。DeepSeek 配置与主程序
一致，建议显式设置模型：

```bash
export SZABOT_PROVIDER=deepseek
export DEEPSEEK_API_KEY=sk-xxxxxxxx
export DEEPSEEK_MODEL=deepseek-v4-pro
```

> Skill Review Web 服务没有认证，并且能够修改 workspace 中的 `SKILL.md` 和
> `PATH.json`。请仅监听 `127.0.0.1`，不要暴露到公网或不可信局域网。

详细的数据模型、评估指标、Path 生成和界面设计见：

- [`docs/skill-execution-path-review.md`](docs/skill-execution-path-review.md)
- [`docs/skill-review-plan.md`](docs/skill-review-plan.md)
- [`docs/skill-review-ui-redesign.md`](docs/skill-review-ui-redesign.md)

## 项目地图（overview）

Overview 会实时读取 workspace 的 `skills/` 和 `docs/`，展示项目规模、L1 摘要、常驻
Skill 正文体积、架构、消息流、工具与文档索引。它不执行 Skill，也不依赖 Docker。

```bash
go run ./cmd/overview -addr 127.0.0.1:8091 -workspace .
```

启动后访问 `http://127.0.0.1:8091`。Overview 当前同样没有认证；它可通过 HTTP API
读取 `docs/` 内容，因此只应在可信本机环境使用。

## 设计宪法

1. **Core stays small** — 所有新功能挂在 channel/tool/provider 边上，不往 loop 塞业务。
2. **Less structure, more intelligence** — 第二个实现出现时再抽接口。
3. **Prefer duplication over premature abstraction** — 不写 BaseChannel 这种父类。
4. **Explicit over magical** — 所有可配置项必须出现在 config struct 里。

## 已知问题

- `cmd/szabot/main.go` 的 `buildProvider` 目前会用 `DEBUG:` 打印把 `DEEPSEEK_API_KEY`
  明文输出到控制台。仅本地调试无妨，但在共享终端 / CI 日志里会**泄露密钥**，上线前应移除这些打印。
