## 从命令行发送一条消息的完整处理链路

本文档用于归档 `szabot` 中，从命令行输入一条消息，到最终在终端打印回复的完整处理流程。

涉及的核心文件：

- [main.go](/Users/ziangsun/Documents/szabot/cmd/szabot/main.go)
- [cli.go](/Users/ziangsun/Documents/szabot/internal/channels/cli.go)
- [queue.go](/Users/ziangsun/Documents/szabot/internal/bus/queue.go)
- [events.go](/Users/ziangsun/Documents/szabot/internal/bus/events.go)
- [loop.go](/Users/ziangsun/Documents/szabot/internal/agent/loop.go)
- [runner.go](/Users/ziangsun/Documents/szabot/internal/agent/runner.go)
- [provider.go](/Users/ziangsun/Documents/szabot/internal/providers/provider.go)
- [openai_compatible.go](/Users/ziangsun/Documents/szabot/internal/providers/openai_compatible.go)

### 1. 启动装配图：`main` 把所有模块串起来

```mermaid
flowchart TD
    A["程序启动<br/>cmd/szabot/main.go"] --> B["创建 ctx<br/>signal.NotifyContext<br/>监听 Ctrl+C / SIGTERM"]

    B --> C["创建 MessageBus<br/>bus.New(64)"]

    C --> C1["inbound chan<br/>chan InboundMessage<br/>buffer=64"]
    C --> C2["outbound chan<br/>chan OutboundMessage<br/>buffer=64"]

    B --> D["buildProvider()"]

    D --> E{"读取环境变量<br/>SZABOT_PROVIDER"}

    E -->|"deepseek"| F["读取 DEEPSEEK_API_KEY<br/>DEEPSEEK_BASE_URL<br/>DEEPSEEK_MODEL"]
    F --> G["创建 OpenAICompatibleProvider<br/>ProviderName=deepseek<br/>BaseURL=https://api.deepseek.com/v1"]
    G --> H["model=deepseek-chat<br/>或 DEEPSEEK_MODEL"]

    E -->|"未设置 / 其他"| I["创建 EchoProvider"]
    I --> J["model=echo"]

    H --> K["创建 Runner<br/>Provider=provider<br/>Model=model"]
    J --> K

    K --> L["创建 Agent Loop<br/>Loop{Bus, Runner}"]
    L --> M["loop.Start(ctx)"]
    M --> M1["启动 goroutine #1<br/>Loop.run()<br/>持续监听 bus.Inbound()"]

    C --> N["创建 CLIChannel<br/>ID=cli<br/>Bus=MessageBus"]
    N --> O["cli.Start(ctx)"]
    O --> O1["启动 goroutine #2<br/>readLoop()<br/>stdin -> inbound"]
    O --> O2["启动 goroutine #3<br/>writeLoop()<br/>outbound -> stdout"]

    M1 --> P["main goroutine 等待 ctx.Done()"]
    O1 --> P
    O2 --> P
```

这张图的重点是：

- `main` 不处理业务，只负责装配对象。
- 真正干活的是三个 goroutine：
  - `CLIChannel.readLoop`
  - `agent.Loop.run`
  - `CLIChannel.writeLoop`
- `MessageBus` 是中间的交通枢纽。

### 2. 运行时链路图：用户敲一行消息之后发生什么

```mermaid
sequenceDiagram
    autonumber

    participant User as 用户<br/>Terminal
    participant ReadLoop as CLIChannel.readLoop<br/>goroutine #2
    participant BusIn as MessageBus.inbound<br/>chan InboundMessage
    participant LoopRun as Loop.run<br/>goroutine #1
    participant Handle as Loop.handle
    participant Runner as Runner.Run
    participant Provider as Provider.Chat
    participant HTTP as DeepSeek/OpenAI API
    participant BusOut as MessageBus.outbound<br/>chan OutboundMessage
    participant WriteLoop as CLIChannel.writeLoop<br/>goroutine #3
    participant Stdout as stdout<br/>Terminal

    Note over User,Stdout: 程序已经启动，readLoop / Loop.run / writeLoop 三个 goroutine 都在等待消息

    User->>ReadLoop: 输入："你好" + 回车

    Note over ReadLoop: bufio.Scanner 从 stdin 读取一整行

    ReadLoop->>ReadLoop: scanner.Text() 得到 line = "你好"

    alt line == ""
        ReadLoop->>Stdout: 打印新的提示符 "> "
        Note over ReadLoop: 空行不会进入 agent
    else line != ""
        ReadLoop->>ReadLoop: 构造 bus.InboundMessage

        Note right of ReadLoop: InboundMessage{<br/>ChannelID: "cli",<br/>SessionID: "cli:local",<br/>UserID: "local",<br/>Text: "你好",<br/>Time: now,<br/>}

        ReadLoop->>BusIn: PublishInbound(ctx, in)

        alt ctx 未取消，inbound 有空间
            BusIn-->>ReadLoop: 写入成功
        else ctx 已取消
            BusIn-->>ReadLoop: 返回 ctx.Err()
            ReadLoop-->>ReadLoop: 退出 readLoop
        end

        BusIn-->>LoopRun: Loop.run 从 Bus.Inbound() 读到 in

        LoopRun->>Handle: l.handle(ctx, in)

        Handle->>Handle: 构造 LLM messages

        Note right of Handle: messages = []providers.Message{<br/>{Role: "user", Content: in.Text}<br/>}<br/><br/>当前阶段没有历史上下文

        Handle->>Runner: Runner.Run(ctx, messages)

        Runner->>Provider: Provider.Chat(ctx, ChatRequest)

        Note right of Runner: ChatRequest{<br/>Model: r.Model,<br/>Messages: messages,<br/>}

        Provider->>HTTP: POST /chat/completions

        alt HTTP 调用成功
            HTTP-->>Provider: 200 OK<br/>choices[0].message.content
            Provider-->>Runner: ChatResponse{Content: "..."}
            Runner-->>Handle: reply = "..."
        else HTTP / API 出错
            HTTP-->>Provider: 4xx/5xx 或网络错误
            Provider-->>Runner: error
            Runner-->>Handle: error
            Handle->>Handle: log.Printf runner error
            Note over Handle,Stdout: 当前代码遇到 runner error 只打日志，不会给 CLI 返回错误消息
        end

        alt 成功拿到 reply
            Handle->>Handle: 构造 bus.OutboundMessage

            Note right of Handle: OutboundMessage{<br/>SessionID: in.SessionID,<br/>ChannelID: in.ChannelID,<br/>Text: reply,<br/>Time: now,<br/>}<br/><br/>关键：ChannelID / SessionID 原样带回

            Handle->>BusOut: PublishOutbound(ctx, out)

            alt ctx 未取消，outbound 有空间
                BusOut-->>Handle: 写入成功
            else ctx 已取消
                BusOut-->>Handle: 返回 ctx.Err()
                Handle->>Handle: log.Printf publish outbound error
            end

            BusOut-->>WriteLoop: writeLoop 从 Bus.Outbound() 读到 out

            WriteLoop->>WriteLoop: 判断 out.ChannelID 是否等于 c.ID

            alt out.ChannelID == "cli"
                WriteLoop->>Stdout: 打印 "\nszabot> {reply}\n> "
                Stdout-->>User: 用户看到回复和新的输入提示符
            else out.ChannelID != "cli"
                WriteLoop->>WriteLoop: continue，忽略这条消息
                Note right of WriteLoop: 这就是多 channel 路由的基础
            end
        end
    end
```

### 3. 数据结构流转图：一条文本如何变形

```mermaid
flowchart LR
    A["终端原始输入<br/>你好"] --> B["CLIChannel.readLoop"]

    B --> C["bus.InboundMessage"]

    C --> C1["ChannelID: cli"]
    C --> C2["SessionID: cli:local"]
    C --> C3["UserID: local"]
    C --> C4["Text: 你好"]
    C --> C5["Time: now"]

    C --> D["Loop.handle"]

    D --> E["[]providers.Message"]

    E --> E1["Role: user"]
    E --> E2["Content: 你好"]

    E --> F["providers.ChatRequest"]

    F --> F1["Model: deepseek-chat / echo"]
    F --> F2["Messages: []Message"]

    F --> G["Provider.Chat"]

    G --> H{"Provider 类型"}

    H -->|"EchoProvider"| I["本地直接生成回复"]
    H -->|"OpenAICompatibleProvider"| J["转成 OpenAI wire format"]

    J --> K["HTTP JSON Request"]

    K --> K1["model"]
    K --> K2["messages: [{role, content}]"]
    K --> K3["stream: false"]

    K --> L["LLM API 返回 JSON"]

    L --> M["providers.ChatResponse"]

    M --> M1["Content: 模型回复文本"]

    M --> N["Loop.handle"]

    N --> O["bus.OutboundMessage"]

    O --> O1["ChannelID: cli<br/>从 inbound 原样复制"]
    O --> O2["SessionID: cli:local<br/>从 inbound 原样复制"]
    O --> O3["Text: 回复文本"]
    O --> O4["Time: now"]

    O --> P["CLIChannel.writeLoop"]

    P --> Q["stdout 输出"]

    Q --> R["szabot> 回复文本"]
```

用户输入不是直接传给 Provider 的，它经过了以下几层包装：

```text
stdin line
  -> bus.InboundMessage
  -> []providers.Message
  -> providers.ChatRequest
  -> OpenAI HTTP JSON
  -> providers.ChatResponse
  -> bus.OutboundMessage
  -> stdout
```

### 4. Bus 内部图：为什么它能解耦

`MessageBus` 在 [queue.go](/Users/ziangsun/Documents/szabot/internal/bus/queue.go) 里，本质上就是两条 Go channel：

```mermaid
flowchart TD
    subgraph Channels["外部平台 Channel 层"]
        CLI["CLIChannel"]
        TG["未来：TelegramChannel"]
        Web["未来：WebChannel"]
        Feishu["未来：FeishuChannel"]
    end

    subgraph Bus["MessageBus"]
        In["inbound<br/>chan InboundMessage"]
        Out["outbound<br/>chan OutboundMessage"]
    end

    subgraph Agent["Agent Core"]
        Loop["Loop.run / Loop.handle"]
        Runner["Runner.Run"]
        Provider["Provider.Chat"]
    end

    CLI -->|"PublishInbound"| In
    TG -.->|"PublishInbound"| In
    Web -.->|"PublishInbound"| In
    Feishu -.->|"PublishInbound"| In

    In -->|"Bus.Inbound()"| Loop

    Loop --> Runner
    Runner --> Provider
    Provider --> Runner
    Runner --> Loop

    Loop -->|"PublishOutbound"| Out

    Out -->|"Bus.Outbound()"| CLI
    Out -.->|"Bus.Outbound()"| TG
    Out -.->|"Bus.Outbound()"| Web
    Out -.->|"Bus.Outbound()"| Feishu

    CLI -->|"检查 ChannelID == cli"| CLI
    TG -.->|"检查 ChannelID == telegram"| TG
    Web -.->|"检查 ChannelID == web"| Web
    Feishu -.->|"检查 ChannelID == feishu"| Feishu
```

这里的设计好处是：

- Channel 只管平台格式和统一消息格式之间的翻译。
- Agent 只管处理统一消息。
- Bus 只管搬运消息。
- 新增 Telegram / Web / 飞书，不需要改 `Loop` 和 `Runner`。

### 5. Provider 调用细节图：DeepSeek 这一层怎么走

对应文件：[openai_compatible.go](/Users/ziangsun/Documents/szabot/internal/providers/openai_compatible.go)

```mermaid
flowchart TD
    A["Runner.Run 调用 Provider.Chat"] --> B["OpenAICompatibleProvider.Chat"]

    B --> C{"参数校验"}

    C -->|"BaseURL 为空"| C1["返回错误<br/>provider: BaseURL is empty"]
    C -->|"APIKey 为空"| C2["返回错误<br/>provider: APIKey is empty"]
    C -->|"Model 为空"| C3["返回错误<br/>provider: model is empty"]

    C -->|"参数正常"| D["转换消息格式"]

    D --> D1["内部格式<br/>providers.Message{Role, Content}"]
    D1 --> D2["OpenAI 格式<br/>{role: string, content: string}"]

    D2 --> E["json.Marshal(openAIChatRequest)"]

    E --> F{"JSON 序列化是否成功"}

    F -->|"失败"| F1["返回错误<br/>provider: marshal request"]
    F -->|"成功"| G["拼接 URL<br/>BaseURL + /chat/completions"]

    G --> H["http.NewRequestWithContext"]

    H --> I{"创建请求是否成功"}

    I -->|"失败"| I1["返回错误<br/>provider: new request"]
    I -->|"成功"| J["设置请求头"]

    J --> J1["Content-Type: application/json"]
    J --> J2["Authorization: Bearer APIKey"]

    J --> K["选择 HTTP Client"]

    K --> K1{"p.HTTPClient 是否为空"}

    K1 -->|"不为空"| K2["使用注入的 HTTPClient"]
    K1 -->|"为空"| K3["使用默认 client<br/>Timeout: 30s"]

    K2 --> L["client.Do(httpReq)"]
    K3 --> L

    L --> M{"请求是否成功发出"}

    M -->|"网络错误 / ctx 取消"| M1["返回错误<br/>provider: do request"]
    M -->|"收到响应"| N["io.ReadAll(resp.Body)"]

    N --> O{"读取响应体是否成功"}

    O -->|"失败"| O1["返回错误<br/>provider: read response"]
    O -->|"成功"| P{"HTTP 状态码是否 2xx"}

    P -->|"非 2xx"| P1["返回错误<br/>provider: http 状态码 + 截断响应体"]
    P -->|"2xx"| Q["json.Unmarshal 响应体"]

    Q --> R{"JSON 解析是否成功"}

    R -->|"失败"| R1["返回错误<br/>provider: unmarshal response + body"]
    R -->|"成功"| S{"parsed.Error 是否存在"}

    S -->|"存在"| S1["返回错误<br/>provider: api error"]
    S -->|"不存在"| T{"choices 是否为空"}

    T -->|"为空"| T1["返回错误<br/>provider: no choices in response"]
    T -->|"不为空"| U["取 choices[0].message.content"]

    U --> V["返回 ChatResponse{Content}"]
    V --> W["Runner.Run 返回 reply 文本"]
```

### 6. 当前架构的职责边界图

```mermaid
flowchart TB
    subgraph Main["cmd/szabot/main.go"]
        MainDesc["只做装配<br/>不处理业务逻辑"]
    end

    subgraph Channel["internal/channels"]
        ChannelDesc["平台适配层<br/>负责翻译消息格式"]
        Read["readLoop<br/>平台输入 -> InboundMessage"]
        Write["writeLoop<br/>OutboundMessage -> 平台输出"]
    end

    subgraph Bus["internal/bus"]
        BusDesc["系统中枢<br/>只搬运统一消息"]
        InMsg["InboundMessage"]
        OutMsg["OutboundMessage"]
        Queue["MessageBus<br/>inbound / outbound"]
    end

    subgraph Agent["internal/agent"]
        Loop["Loop<br/>对外协调"]
        Runner["Runner<br/>对内调用模型"]
    end

    subgraph Providers["internal/providers"]
        Interface["Provider interface"]
        Echo["EchoProvider"]
        OpenAI["OpenAICompatibleProvider"]
    end

    MainDesc --> ChannelDesc
    MainDesc --> BusDesc
    MainDesc --> Loop
    MainDesc --> Runner
    MainDesc --> Interface

    Read --> InMsg
    InMsg --> Queue
    Queue --> Loop
    Loop --> Runner
    Runner --> Interface
    Interface --> Echo
    Interface --> OpenAI
    OpenAI --> Interface
    Echo --> Interface
    Interface --> Runner
    Runner --> Loop
    Loop --> OutMsg
    OutMsg --> Queue
    Queue --> Write
```

一句话概括：

**Channel 负责接人话，Bus 负责搬消息，Loop 负责调度，Runner 负责跑模型，Provider 负责对接具体模型服务。**

### 7. 带错误分支的完整流程图

```mermaid
flowchart TD
    A["用户在终端输入一行"] --> B["CLIChannel.readLoop<br/>scanner.Scan()"]

    B --> C{"是否读到内容？"}

    C -->|"空行"| C1["打印提示符 >"]
    C1 --> B

    C -->|"非空行"| D["构造 InboundMessage"]

    D --> E["PublishInbound(ctx, in)"]

    E --> F{"ctx 是否已取消？"}

    F -->|"是"| F1["readLoop 退出"]
    F -->|"否"| G["写入 bus.inbound"]

    G --> H["Loop.run 读取 inbound"]

    H --> I["Loop.handle"]

    I --> J["构造 messages<br/>当前只有一条 user 消息"]

    J --> K["Runner.Run"]

    K --> L["Provider.Chat"]

    L --> M{"Provider 调用是否成功？"}

    M -->|"失败"| M1["Loop 打日志<br/>runner error"]
    M1 --> M2["当前不会给用户返回错误消息"]
    M2 --> B

    M -->|"成功"| N["得到 reply"]

    N --> O["构造 OutboundMessage"]

    O --> P["PublishOutbound(ctx, out)"]

    P --> Q{"ctx 是否已取消？"}

    Q -->|"是"| Q1["Loop 打日志<br/>publish outbound error"]
    Q -->|"否"| R["写入 bus.outbound"]

    R --> S["CLIChannel.writeLoop 读取 outbound"]

    S --> T{"out.ChannelID == c.ID ?"}

    T -->|"否"| T1["忽略该消息<br/>continue"]
    T1 --> S

    T -->|"是"| U["打印到 stdout"]

    U --> V["用户看到<br/>szabot> reply"]
    V --> B
```

### 8. 最终一句话版

这条链路本质是：

```text
命令行输入
  -> CLIChannel.readLoop
  -> bus.InboundMessage
  -> MessageBus.inbound
  -> Agent Loop
  -> Runner
  -> Provider
  -> LLM API / Echo
  -> Runner
  -> Agent Loop
  -> bus.OutboundMessage
  -> MessageBus.outbound
  -> CLIChannel.writeLoop
  -> 命令行输出
```

关键设计点是：

- `ChannelID` 是回信地址。
- `SessionID` 是会话身份。
- `MessageBus` 是解耦核心。
- `Loop` 不关心消息来自 CLI 还是其他平台。
- `Runner` 不关心模型是 DeepSeek 还是 Echo。
- `Provider` 不关心消息最终要回到哪里。
