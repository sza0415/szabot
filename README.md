# szabot

一个用 Go 实现的轻量 agent 框架，围绕一条极简的 agent 循环（消息总线 → AgentLoop → Runner → Provider）构建，核心保持精简，能力从边缘扩展。

主要特性：

- **多通道**：CLI（stdin/stdout）与 Web（HTTP + SSE 流式聊天）两种前端。
- **多 Provider**：统一 LLM 接口，内置 Echo（零依赖验证）与 OpenAI 兼容实现（DeepSeek / OpenAI / Moonshot / Ollama 等）。
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
# export SZABOT_SANDBOX_TMP_SIZE=512m # 容器 /tmp 大小，默认 64m
```

未安装 Docker 时会自动跳过 bash/python，文件类工具照常可用。

> 注意：`bash` 工具中的命令实际运行在 `SZABOT_BASH_IMAGE` 容器内，能否执行
> `go`、`git`、`make` 等命令取决于该镜像是否安装了对应工具。默认的
> `debian:stable-slim` 只提供基础 shell 环境，不包含 Go。需要运行 Go 项目时，
> 可以预先拉取带 Go 工具链的镜像，并将它配置为 bash 镜像：
>
> ```bash
> docker pull golang:1.22
> export SZABOT_SANDBOX=1
> export SZABOT_BASH_IMAGE=golang:1.22
> ./szabot
> ```
>
> sandbox 默认关闭网络，因此不能依赖运行时 `apt install`、下载 Go 或拉取依赖。
> 镜像、Go 模块依赖和其他命令行工具都应在启动 szabot 前准备好。若没有 Go 镜像，
> `go version`、`go test ./...`、`go build ./...` 等命令会在容器内失败；这不表示
> szabot 的宿主机必须安装 Go。启动日志中的 `sandbox tools enabled` 表示两个
> sandbox 工具已经注册，不表示 bash 镜像包含所有开发工具。

> 如果命令在容器 `/tmp` 中产生较大的临时文件，默认的 64M 可能不足。可以在启动前
> 设置 `SZABOT_SANDBOX_TMP_SIZE`，例如 `512m` 或 `1g`。该设置同时应用于 bash 和
> python sandbox，且每次执行都会创建新的临时文件系统：
>
> ```bash
> export SZABOT_SANDBOX_TMP_SIZE=512m
> ./szabot
> ```

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
# 如果需要在 bash sandbox 中编译 Go 项目：
# docker pull golang:1.22
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

### content-expert：内容分析专家模式（离线模拟）

`skills/content-expert/` 提供中性的内容分析专家问答模拟，覆盖营销 / 综艺模式 / 综艺营销 /
小说 / 剧本五类专家。底层用 workspace 内自带的纯 bash 脚本
`skills/content-expert/bin/content-sim` 离线生成结构化结果，无需真实联网服务。

> ⚠️ 该 skill 的执行路径落在 `bash` 工具上，因此**依赖 Docker 沙盒**（`SZABOT_SANDBOX=1`）。
> 未开启沙盒时 agent 会读到 skill 却无法执行脚本。请先按上文装好 Docker。

用法：启用沙盒后，用自然语言直接描述专家意图即可，agent 会自动读取 skill 并调用脚本：

```
> 帮我给一部都市剧做上线首周的营销方案
> 帮我拆解一下这类推理综艺的模式，能不能本土化改编
> 帮我写一份某综艺这周的营销周报
```

## Skill 评审（skill-review）

`cmd/skill-review` 是一个**独立的本地工具**，用于对 `skills/` 下的 Skill 做「执行路径（Path）」建模与评审。它不执行 Skill、不依赖 Docker，只读取 `SKILL.md` / `references/` 做路径抽取与预期/实际对比，因此在没有沙盒的机器上也能用。

评审工作台分三大模块，左侧栏目导航切换：

```text
① Skill 管理  ──▶  ② 测试用例构建  ──▶  ③ 评估与对比
  列出/编辑 Skill    收集一次真实运行      预期路径 vs 实际路径
  生成预设 Path      抽取成 Case/Run       指标 + 失败定位
```

### 启动 Web 工作台

```bash
# 最简：在项目根启动，评审 workspace/skills 下的所有 Skill
go run ./cmd/skill-review -serve -workspace .

# 打开浏览器
# http://localhost:8090
```

常用参数：

```bash
go run ./cmd/skill-review \
  -serve \                 # 启动本地 Web 工作台
  -addr :8090 \            # 监听地址，默认 :8090
  -workspace . \           # Skill 读写沙盒根，默认当前目录
  -skills skills \         # 技能目录，默认 <workspace>/skills
  -skill-version local     # 标注本次评审的 Skill 版本（写进报告，便于版本对比）
```

### 生成评审报告（命令行，非 serve）

给定测试用例与实际执行 Trace，直接产出 Markdown / JSON 报告：

```bash
go run ./cmd/skill-review \
  -cases cases.json \      # 测试用例（预期）
  -runs runs.json \        # 实际执行 Trace
  -paths paths.json \      # 可选：Path 定义，用于校验
  -markdown report.md \    # 可选：Markdown 报告输出路径，默认 stdout
  -json report.json \      # 可选：JSON 报告输出路径
  -skill-version $(git rev-parse --short HEAD)
```

不带 `-serve` 时 `-cases` 与 `-runs` 必填。

### 模块一：Skill 管理

- **Skill 列表**：读取 `skills/` 下的每个 Skill（复用 `internal/skills` 的 Loader），显示 name / description / 依赖。
- **编辑 SKILL.md**：在线修改 frontmatter 与正文，保存回磁盘（限制在 `skills/` 内，name 经清洗防路径穿越）。
- **生成预设 Path**：从 SKILL.md 正文推导该 Skill 的执行路径（见下「Path 生成引擎」）。

### 模块二：测试用例构建

构建用例 = **收集一次真实运行**。szabot 运行时产生的 SSE 事件流（`{text, kind, delta, done}`，`kind` 为 `answer/reasoning/tool_call/tool_result`）可以粘贴进工作台，自动解析并抽取成：

- `Run`（实际发生了什么：命中的 Skill、工具调用、有序节点、最终输出）；
- `Case.Expected`（把这次运行固化成一条预期基线，供后续回归比对）。

导出的 JSON 直接可作为命令行模式的 `runs.json` / `cases.json`。

### 模块三：评估与对比

选一条用例，工作台以**预期路径 / 实际路径双轨对齐**展示，不一致的节点标红并给出失败码（`skill_not_selected` / `path_mismatch` / `chain_step_missing` / `tool_missing` / `notice_missed` / `node_missing` / `output_*`），并汇总指标：Skill 命中率、Path/Node 覆盖率、路径匹配率、链路通过率、注意事项通过率、输出通过率。

### Path 生成引擎（LLM 优先，规则版兜底）

`POST /api/skill/paths` 支持两种抽取引擎，用查询参数 `engine` 控制：

| `engine` | 行为 |
|---|---|
| `auto`（默认） | 有 LLM 就用 LLM，抽取失败自动回退规则版，保证总能出草稿 |
| `llm` | 强制 LLM；未配置 Provider 时报错 |
| `rule` | 强制规则版（正则启发式，抓 `bash` 脚本 / 触发词 / references） |

响应头 `X-Path-Source: llm|rule` 标注本次实际用了哪种。

**LLM 引擎复用 szabot 的 Provider 抽象**，环境变量约定与主程序一致（不配则自动走规则版，无需 API key 也能用）：

```bash
# 用 DeepSeek 做 Path 抽取（能识别 MCP 调用、CLI 命令、条件分支、注意事项）
export SZABOT_PROVIDER=deepseek
export DEEPSEEK_API_KEY=sk-xxxxxxxx
# 可选：DEEPSEEK_MODEL（默认 deepseek-chat）、DEEPSEEK_BASE_URL

# 或用 OpenAI 兼容端点
# export SZABOT_PROVIDER=openai
# export OPENAI_API_KEY=sk-xxxxxxxx
# 可选：OPENAI_MODEL（默认 gpt-4o-mini）、OPENAI_BASE_URL

go run ./cmd/skill-review -serve -workspace .
```

> LLM 引擎能读懂 SKILL.md 的语义，把 MCP 工具（`mcp_exec_sql` 等）、CLI 命令、品类分支、铁律注意事项都准确抽成节点；规则版仅识别 `bash` 脚本调用，适合无 API key 时快速出草稿。

### HTTP 接口一览

| 接口 | 方法 | 作用 |
|---|---|---|
| `/` | GET | Web 工作台页面 |
| `/api/data` | GET | 评审模块列表 + 报告 + Path 定义 |
| `/api/skills` | GET | 列出所有 Skill |
| `/api/skill?name=` | GET / PUT | 读取 / 保存某个 SKILL.md |
| `/api/skill/paths?engine=` | POST | 从 SKILL.md 生成预设 Path |

## 设计宪法

1. **Core stays small** — 所有新功能挂在 channel/tool/provider 边上，不往 loop 塞业务。
2. **Less structure, more intelligence** — 第二个实现出现时再抽接口。
3. **Prefer duplication over premature abstraction** — 不写 BaseChannel 这种父类。
4. **Explicit over magical** — 所有可配置项必须出现在 config struct 里。

## 已知问题

- `cmd/szabot/main.go` 的 `buildProvider` 目前会用 `DEBUG:` 打印把 `DEEPSEEK_API_KEY`
  明文输出到控制台。仅本地调试无妨，但在共享终端 / CI 日志里会**泄露密钥**，上线前应移除这些打印。
