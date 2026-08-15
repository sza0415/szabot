# szabot

一个用 Go 实现的 agent 框架，架构借鉴 [nanobot](https://github.com/...)。

## 目录结构

```
szabot/
├── cmd/szabot/             # CLI 入口（main.go：只做装配）
├── internal/
│   ├── bus/                # 消息总线（系统中枢）
│   ├── agent/              # 核心循环
│   │   ├── loop.go         #   外层：消费 bus、协调上下文
│   │   └── runner.go       #   内层：跟 LLM 来回打交道
│   ├── channels/           # 通道（平台翻译官）
│   │   └── cli.go          #   stdin/stdout 实现
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
| **AgentRunner** | 朝内 | 跟 Provider 来回对话，将来负责 tool call 循环 |
| **Provider** | 朝内 | 统一各家 LLM API（OpenAI/DeepSeek/Anthropic/...） |

## 运行

### 默认（Echo，零依赖）

```bash
go run ./cmd/szabot
```

```
> 你好
szabot> echo: 你好
```

### 接 DeepSeek（OpenAI 兼容）

```bash
export SZABOT_PROVIDER=deepseek
export DEEPSEEK_API_KEY=sk-xxxxxxxx
# 可选：
# export DEEPSEEK_MODEL=deepseek-chat        # 默认 deepseek-chat
# export DEEPSEEK_BASE_URL=https://api.deepseek.com/v1

go run ./cmd/szabot
```

按 Ctrl+C 退出。

### Web 界面（HTTP + SSE）

除了 CLI，szabot 还提供一个浏览器聊天界面。设置 `SZABOT_WEB` 即启用（此时不再启动 CLI，
因为两者会争抢同一条出站消息总线）：

```bash
export SZABOT_WEB=1                 # 启用 Web 界面
# 可选：
# export SZABOT_WEB_ADDR=:9000      # 自定义监听地址，默认 :8080

# Provider 照旧（不设则用 echo；接 DeepSeek 见上一节）
export SZABOT_PROVIDER=deepseek
export DEEPSEEK_API_KEY=sk-xxxxxxxx

go run ./cmd/szabot
```

启动后打开 `http://localhost:8080`（或你设的地址）即可对话，回复以打字机效果**流式**呈现。

实现要点：

- **零依赖**：纯标准库 `net/http`，前端单页 HTML 用 `//go:embed` 打进二进制，依旧是"一个可执行文件"。
- **SSE 而非 WebSocket**：agent 输出本就是服务端单向流式推送，SSE 天生贴合；用户发消息走普通 `POST /api/send`，浏览器用 `GET /api/stream?session=...` 收流。
- **按 session 分发**：`bus.Outbound()` 是所有 channel 共享的一条队列，多个连接一起读会互相抢消息。因此 Web channel 用"单个 goroutine 读 bus → 按 `SessionID` 扇出给对应连接"的模型，多个浏览器/标签页互不串台。
- **会话延续**：浏览器把 session ID 存在 `localStorage`，刷新后仍是同一段对话历史。

## 会话历史（Session 存储，M8）

szabot 会按 `SessionID` 把对话历史持久化为 jsonl，从而**同一 session 的后续请求会带上此前的对话上下文**（不再"失忆"）。

### 存储位置

- 默认落在**工作区下的 `sessionlogs/` 目录**（即 `go run ./cmd/szabot` 的启动目录 / 项目根目录下的 `sessionlogs`）。
- 可用环境变量 `SZABOT_SESSION_DIR` 覆盖（支持绝对或相对路径）：

```bash
export SZABOT_SESSION_DIR=/data/szabot-sessions   # 自定义存储目录
```

### 文件与格式

- 每个 session 一个文件，文件名为 `<SessionID>.jsonl`。
  - CLI 渠道的 SessionID 固定为 `cli:local`，因此对应 `sessionlogs/cli:local.jsonl`。
  - Web 渠道的 SessionID 形如 `web:<时间戳>:<随机串>`，每个浏览器会话一个文件。
- 每行是一条消息的 JSON（只含 `role` + `content` 等字段），例如：

```jsonl
{"role":"user","content":"我叫小明"}
{"role":"assistant","content":"你好，小明！"}
```

### 注意事项

- **system prompt 不入库**：它在进程启动时构建、全程不变，由 Loop 在每次请求时恒定拼在最前。这样既避免把它反复写进磁盘，又保证前缀稳定、对 KV Cache 友好。每次请求的实际上下文顺序是：`system prompt（固定） + 历史（从 jsonl 加载） + 本轮 user`。
- **只追加、不改写**：`Append` 往文件尾追加一行并 `fsync`，契合"对话只在末尾增长"的特性，进程/机器异常时历史不丢。
- **重置某个会话**：直接删除对应的 `.jsonl` 文件即可，例如 `rm sessionlogs/cli:local.jsonl`；删除后该 session 下次从空历史重新开始。
- **SessionID 会被清洗**：写盘前经 `filepath.Base` 处理以防路径穿越，因此不会在 `sessionlogs/` 之外生成文件。
- **模型自述的"没有记忆"仅是措辞**：框架层已把历史喂给模型（能答出你之前说过的名字即为证据）；模型有时会声称自己"无跨对话记忆"，那是它的表述习惯，与实际存储行为无关。
- **旧版本迁移**：早期版本默认存储在 `~/.szabot/sessions/`。如需保留旧历史，可手动把其中的 `*.jsonl` 拷到新的 `sessionlogs/` 目录（或用 `SZABOT_SESSION_DIR` 指回旧目录）。

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
- [x] M11 第二个 Channel（HTTP + SSE Web 界面）
- [ ] M12 长期记忆（MEMORY.md）

## 工具箱

| 工具 | 说明 | 依赖 |
|---|---|---|
| `read_file` / `write_file` / `edit_file` | 读 / 覆盖写 / 精确替换，限制在工作区内 | 无 |
| `list_dir` | 列目录（可递归），忽略 .git/node_modules 等噪声 | 无 |
| `glob` | 按文件名模式查找，支持 `**` 递归 | 无 |
| `grep` | 按正则搜索文件内容 | 无 |
| `bash` | 在 Docker 沙盒执行 bash，默认断网 | Docker + `SZABOT_SANDBOX=1` |
| `python` | 代码解释器，在 Docker 沙盒执行 Python | Docker + `SZABOT_SANDBOX=1` |

启用沙盒工具：

```bash
export SZABOT_SANDBOX=1              # 启用 bash + python
# 可选：
# export SZABOT_SANDBOX_NETWORK=1   # 允许容器联网（默认断网）
# export SZABOT_PYTHON_IMAGE=python:3.12-slim
# export SZABOT_BASH_IMAGE=debian:stable-slim
```

未安装 Docker 时会自动跳过 bash/python，文件类工具照常可用。

### 安装 Docker（macOS）

`bash` / `python` 两个执行类工具共用一个沙盒，而该沙盒是 `docker run` 的封装
（见 `internal/tools/sandbox.go`）。因此要用这两个工具，本机必须装有 Docker 且 daemon 在运行。

Apple Silicon / Intel 通用步骤（Homebrew）：

```bash
# 1. 安装 Docker Desktop（含 docker CLI）
brew install --cask docker

# 2. 启动 Docker（装完不会自动起，需手动打开一次）
open -a Docker
# 首次打开需同意条款，等菜单栏鲸鱼图标停止转圈（约 1-2 分钟）

# 3. 确认 docker 可用（能打印 Server 版本即成功）
docker version

# 4.（可选）预拉镜像，避免首次调用时现拉很慢
docker pull debian:stable-slim
docker pull python:3.12-slim
```

> 若第 3 步提示 `command not found`，新开一个终端窗口让 PATH 生效即可。

装好后，开着 Docker 重新启动 szabot 并启用沙盒：

```bash
export SZABOT_SANDBOX=1
go run ./cmd/szabot
```

启动日志出现下面这行，才表示 `bash` / `python` 已就绪：

```
sandbox tools enabled: bash(debian:stable-slim) python(python:3.12-slim) network=false
```

## 技能（Skills）

技能位于 workspace 的 `skills/` 目录，采用三层渐进式披露：L1 元数据常驻 system prompt，
L2 正文（`SKILL.md`）与 L3 子资源由 agent 用 `read_file` 按需读取。详见
`internal/skills/`。

### rycli：影视综专家模式（离线模拟）

`skills/rycli/` 模拟"如影 CLI"的专家问答，覆盖营销 / 综艺模式 / 综艺营销 / 小说 / 剧本
五类专家。底层用 workspace 内自带的纯 bash 脚本 `skills/rycli/bin/sage-sim` 离线模拟
`rycli sage ask` 的输入输出契约（`<thread_id>` 多轮 + `<text>` 正文），无需真实联网服务。

> ⚠️ 该 skill 的执行路径落在 `bash` 工具上，因此**依赖 Docker 沙盒**（`SZABOT_SANDBOX=1`）。
> 未开启沙盒时 agent 会读到 skill 却无法执行脚本。请先按上文装好 Docker。

用法：启用沙盒后，用自然语言直接描述专家意图即可，agent 会自动读取 skill 并调用脚本：

```
> 帮我给一部都市剧做上线首周的营销方案
> 帮我拆解一下这类推理综艺的模式，能不能本土化改编
> 帮我写一份某综艺这周的营销周报
```

## 设计宪法

1. **Core stays small** — 所有新功能挂在 channel/tool/provider 边上，不往 loop 塞业务。
2. **Less structure, more intelligence** — 第二个实现出现时再抽接口。
3. **Prefer duplication over premature abstraction** — 不写 BaseChannel 这种父类。
4. **Explicit over magical** — 所有可配置项必须出现在 config struct 里。

## 已知问题

- `cmd/szabot/main.go` 的 `buildProvider` 目前会用 `DEBUG:` 打印把 `DEEPSEEK_API_KEY`
  明文输出到控制台。仅本地调试无妨，但在共享终端 / CI 日志里会**泄露密钥**，上线前应移除这些打印。
