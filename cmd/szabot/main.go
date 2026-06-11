// Command szabot 是 CLI 入口。
//
// 这个 main 文件做的事情非常简单——它只做"装配"：
//  1. 创建一条 MessageBus；
//  2. 根据环境变量选择 Provider（echo / deepseek）；
//  3. 用 bus + runner 创建 AgentLoop 并启动；
//  4. 用 bus 创建 CLIChannel 并启动；
//  5. 等系统信号退出。
//
// 没有任何业务逻辑，所有逻辑都被关在了对应的 package 里——
// 这就是 nanobot 设计宪法的第一条："Core stays small; extend at the edges"。
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ziangsun/szabot/internal/agent"
	"github.com/ziangsun/szabot/internal/bus"
	"github.com/ziangsun/szabot/internal/channels"
	"github.com/ziangsun/szabot/internal/providers"
)

func main() {
	// 监听 Ctrl+C / SIGTERM，触发优雅退出。
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 1. 消息总线。
	b := bus.New(64)

	// 2. 根据环境变量选 Provider。
	provider, model := buildProvider()

	runner := &agent.Runner{
		Provider: provider,
		Model:    model,
	}

	// 3. AgentLoop：从 bus 入站读消息 → 调 Runner → 推回 bus 出站。
	loop := &agent.Loop{Bus: b, Runner: runner}
	loop.Start(ctx)

	// 4. CLIChannel：stdin → bus 入站；bus 出站 → stdout。
	cli := &channels.CLIChannel{
		ID:  "cli",
		Bus: b,
	}
	cli.Start(ctx)

	fmt.Printf("szabot started. provider=%s model=%s. type something and press Enter. Ctrl+C to quit.\n",
		provider.Name(), model)

	// 5. 等退出信号。
	<-ctx.Done()
	fmt.Println("\nszabot stopped.")
}

// buildProvider 根据环境变量决定用哪个 Provider。
//
// 切换方式（只看 SZABOT_PROVIDER）：
//   - 不设置 / 设为 "echo"  → 用 EchoProvider（默认，零依赖）
//   - 设为 "deepseek"        → 用 DeepSeek（OpenAI 兼容）
//
// DeepSeek 需要的环境变量：
//   - DEEPSEEK_API_KEY  必填
//   - DEEPSEEK_MODEL    可选，默认 "deepseek-chat"
//   - DEEPSEEK_BASE_URL 可选，默认 "https://api.deepseek.com/v1"
func buildProvider() (providers.Provider, string) {
	providerEnv := os.Getenv("SZABOT_PROVIDER")
	fmt.Printf("DEBUG: SZABOT_PROVIDER=%q\n", providerEnv)

	switch providerEnv {
	case "deepseek":
		key := os.Getenv("DEEPSEEK_API_KEY")
		fmt.Printf("DEBUG: DEEPSEEK_API_KEY=%q\n", key)
		if key == "" {
			fmt.Fprintln(os.Stderr, "error: DEEPSEEK_API_KEY is not set")
			os.Exit(1)
		}
		baseURL := os.Getenv("DEEPSEEK_BASE_URL")
		if baseURL == "" {
			baseURL = "https://api.deepseek.com/v1"
		}
		model := os.Getenv("DEEPSEEK_MODEL")
		if model == "" {
			model = "deepseek-chat"
		}
		fmt.Printf("DEBUG: Using deepseek provider with model=%s\n", model)
		return &providers.OpenAICompatibleProvider{
			ProviderName: "deepseek",
			BaseURL:      baseURL,
			APIKey:       key,
		}, model

	default:
		fmt.Printf("DEBUG: Using default echo provider\n")
		return providers.EchoProvider{}, "echo"
	}
}
