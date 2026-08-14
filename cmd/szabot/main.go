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
	"github.com/ziangsun/szabot/internal/tools"
)

func main() {
	// 监听 Ctrl+C / SIGTERM，触发优雅退出。
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 1. 消息总线。
	b := bus.New(64)

	// 2. 根据环境变量选 Provider。
	provider, model := buildProvider()

	registry := tools.NewRegistry()
	workspace, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: resolve workspace: %v\n", err)
		os.Exit(1)
	}
	registerTools(registry, workspace)

	runner := &agent.Runner{
		Provider: provider,
		Model:    model,
		Tools:    registry,
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

// registerTools 把工作区内的本地工具装进 registry。
//
// 每个工具都被限制在 workspace 内（沙盒边界），任何创建/注册失败都视为致命错误：
// 工具集是 agent 的能力清单，缺失会让行为不可预测，宁可启动即失败。
func registerTools(registry *tools.Registry, workspace string) {
	type factory struct {
		name  string
		build func(string) (tools.Tool, error)
	}
	factories := []factory{
		{"read_file", func(ws string) (tools.Tool, error) { return tools.NewReadFile(ws) }},
		{"write_file", func(ws string) (tools.Tool, error) { return tools.NewWriteFile(ws) }},
		{"edit_file", func(ws string) (tools.Tool, error) { return tools.NewEditFile(ws) }},
		{"list_dir", func(ws string) (tools.Tool, error) { return tools.NewListDir(ws) }},
		{"glob", func(ws string) (tools.Tool, error) { return tools.NewGlob(ws) }},
		{"grep", func(ws string) (tools.Tool, error) { return tools.NewGrep(ws) }},
	}

	for _, f := range factories {
		tool, err := f.build(workspace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: create %s tool: %v\n", f.name, err)
			os.Exit(1)
		}
		if err := registry.Register(tool); err != nil {
			fmt.Fprintf(os.Stderr, "error: register %s tool: %v\n", f.name, err)
			os.Exit(1)
		}
	}

	registerSandboxTools(registry, workspace)
}

// registerSandboxTools 注册需要 Docker 沙盒的执行类工具（bash / python）。
//
// 设计取舍：这两个工具依赖本机 Docker，属于"能力增强"而非"核心必需"。
// 因此当 SZABOT_SANDBOX 未开启，或 Docker 不可用时，只打印提示并跳过，
// 而不是让整个程序启动失败——没装 Docker 的用户仍能用文件类工具。
//
// 开启方式：
//   - export SZABOT_SANDBOX=1            启用 bash + python
//   - export SZABOT_SANDBOX_NETWORK=1    额外允许容器联网（默认断网）
//   - export SZABOT_PYTHON_IMAGE=...     python 镜像，默认 python:3.12-slim
//   - export SZABOT_BASH_IMAGE=...       bash 镜像，默认 debian:stable-slim
func registerSandboxTools(registry *tools.Registry, workspace string) {
	if os.Getenv("SZABOT_SANDBOX") == "" {
		return
	}

	network := os.Getenv("SZABOT_SANDBOX_NETWORK") != ""
	pythonImage := envOr("SZABOT_PYTHON_IMAGE", "python:3.12-slim")
	bashImage := envOr("SZABOT_BASH_IMAGE", "debian:stable-slim")

	bashSandbox, err := tools.NewSandbox(tools.SandboxConfig{
		Image:     bashImage,
		Workspace: workspace,
		Network:   network,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: sandbox unavailable, skipping bash/python: %v\n", err)
		return
	}
	pythonSandbox, err := tools.NewSandbox(tools.SandboxConfig{
		Image:     pythonImage,
		Workspace: workspace,
		Network:   network,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: sandbox unavailable, skipping bash/python: %v\n", err)
		return
	}

	bash, err := tools.NewBash(bashSandbox)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: create bash tool: %v\n", err)
		return
	}
	python, err := tools.NewPython(pythonSandbox)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warn: create python tool: %v\n", err)
		return
	}

	for name, tool := range map[string]tools.Tool{"bash": bash, "python": python} {
		if err := registry.Register(tool); err != nil {
			fmt.Fprintf(os.Stderr, "warn: register %s tool: %v\n", name, err)
		}
	}
	fmt.Printf("sandbox tools enabled: bash(%s) python(%s) network=%v\n", bashImage, pythonImage, network)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
