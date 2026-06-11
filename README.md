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
- [ ] M9 Tool 接口 + 第一个工具
- [ ] M10 Runner 多轮 + tool calling 循环
- [ ] M11 第二个 Channel（HTTP/WebSocket）
- [ ] M12 长期记忆（MEMORY.md）

## 设计宪法

1. **Core stays small** — 所有新功能挂在 channel/tool/provider 边上，不往 loop 塞业务。
2. **Less structure, more intelligence** — 第二个实现出现时再抽接口。
3. **Prefer duplication over premature abstraction** — 不写 BaseChannel 这种父类。
4. **Explicit over magical** — 所有可配置项必须出现在 config struct 里。
