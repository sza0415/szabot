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

## 设计宪法

1. **Core stays small** — 所有新功能挂在 channel/tool/provider 边上，不往 loop 塞业务。
2. **Less structure, more intelligence** — 第二个实现出现时再抽接口。
3. **Prefer duplication over premature abstraction** — 不写 BaseChannel 这种父类。
4. **Explicit over magical** — 所有可配置项必须出现在 config struct 里。
