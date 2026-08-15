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

## 路线图

- [x] M1 项目骨架
- [x] M2 MessageBus
- [x] M3 AgentLoop + AgentRunner（最小循环）
- [x] M4 CLI Channel
- [x] M5 EchoProvider（验证链路）
- [x] M6 OpenAI 兼容 Provider（DeepSeek 已接入）
- [ ] M7 配置加载（~/.szabot/config.json）
- [ ] M8 Session 存储（jsonl）
- [x] M9 Tool 接口 + 文件工具（read/write/edit/list_dir/glob/grep）
- [x] M10 Runner 多轮 + tool calling 循环
- [x] M10.5 执行类工具 bash/python（Docker 沙盒，SZABOT_SANDBOX=1 启用）
- [ ] M11 第二个 Channel（HTTP/WebSocket）
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
