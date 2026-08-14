// 个人助理 Agent 最小项目入口
//
// 运行方式：
//
//	go run .\03-agent -demo
//	go run .\03-agent -mode mock -question "请计算 7 * 6"
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

func main() {
	question := flag.String("question", "请计算 12 + 30", "user question")
	mode := flag.String("mode", "mock", "model mode: mock or real")
	model := flag.String("model", getEnvString("OPENAI_MODEL", openai.GPT4oMini), "model name used when -mode real")
	timeoutSeconds := flag.Int("timeout-seconds", getEnvInt("OPENAI_TIMEOUT_SECONDS", 60), "request timeout in seconds when -mode real")
	showSchema := flag.Bool("show-schema", false, "print full tool schemas sent to the model")
	maxSteps := flag.Int("max-steps", 3, "maximum number of tool calls allowed in one agent loop")
	demo := flag.Bool("demo", false, "run built-in personal assistant demo scenarios in mock mode")
	flag.Parse()
	registry, err := newDefaultToolRegistry()
	if err != nil {
		fmt.Println("注册工具失败:", err)
		os.Exit(1)
	}
	if *demo {
		if err := runPersonalAssistantDemo(registry, *maxSteps); err != nil {
			fmt.Println("个人助理 Agent demo 执行失败:", err)
			os.Exit(1)
		}
		return
	}
	if strings.TrimSpace(*question) == "" {
		fmt.Println("问题不能为空")
		os.Exit(1)
	}
	tools := registry.Schemas()
	run, err := runAgentLoop(*mode, *question, registry, *model, time.Duration(*timeoutSeconds)*time.Second, *maxSteps)
	if err != nil {
		fmt.Println("Agent Loop 执行失败:", err)
		os.Exit(1)
	}
	fmt.Println("个人助理 Agent 演示")
	fmt.Println("用户问题:", *question)
	fmt.Println("模型模式:", *mode)
	fmt.Println("最大工具调用次数:", *maxSteps)
	fmt.Println()
	fmt.Println("=== Registered Tools ===")
	for _, tool := range tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}
	if *showSchema {
		fmt.Println()
		fmt.Println("=== Tool Schemas Sent To Model ===")
		printToolSchemas(tools)
	}
	fmt.Println()
	fmt.Println("=== Agent Loop Trace ===")
	printAgentTrace(run)
	fmt.Println("=== Final Answer ===")
	fmt.Println(run.FinalAnswer)
}
