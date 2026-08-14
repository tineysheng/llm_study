package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
)

func main() {
	demoMode := flag.Bool("demo", false, "运行 MCP 标准握手与工具调用协议交互演示")
	flag.Parse()

	if *demoMode {
		runMCPProtocolDemo()
		return
	}

	// 生产/实际接入模式：绑定 OS 标准输入输出
	server := NewMCPServer(os.Stdin, os.Stdout)
	if err := server.Serve(); err != nil {
		fmt.Fprintf(os.Stderr, "MCP Server error: %v\n", err)
		os.Exit(1)
	}
}

// runMCPProtocolDemo 模拟一个 MCP Client 与 Go MCP Server 的完整 JSON-RPC 2.0 交互时序
func runMCPProtocolDemo() {
	fmt.Println("================================================================")
	fmt.Println("🚀 阶段 4：Go 原生 MCP Server 协议时序演示 (JSON-RPC 2.0 over stdio)")
	fmt.Println("================================================================")
	fmt.Println()

	// 构造交互测试管道
	var inBuf bytes.Buffer
	var outBuf bytes.Buffer
	server := NewMCPServer(&inBuf, &outBuf)

	// 1. 握手阶段 (initialize)
	step1 := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"cursor-client","version":"1.0"}}}`
	fmt.Println("【第 1 步：客户端发起能力握手 (initialize)】")
	fmt.Println("Client -> Server:")
	fmt.Printf("  %s\n", step1)
	resp1 := server.handleMessage([]byte(step1))
	printResp(resp1)

	// 2. 握手确认 (notifications/initialized)
	step2 := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	fmt.Println("【第 2 步：客户端确认握手就绪 (Notification)】")
	fmt.Println("Client -> Server:")
	fmt.Printf("  %s\n", step2)
	resp2 := server.handleMessage([]byte(step2))
	if resp2 == nil {
		fmt.Println("Server -> Client: (Notification 遵循 JSON-RPC 规范，无需返回响应)")
	}
	fmt.Println()

	// 3. 工具目录发现 (tools/list)
	step3 := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	fmt.Println("【第 3 步：客户端发现可用工具 (tools/list)】")
	fmt.Println("Client -> Server:")
	fmt.Printf("  %s\n", step3)
	resp3 := server.handleMessage([]byte(step3))
	printResp(resp3)

	// 4. 工具调用 (tools/call)
	step4 := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"calculator","arguments":{"left_operand":15,"right_operand":27,"operation":"add"}}}`
	fmt.Println("【第 4 步：客户端调用计算器工具 (tools/call: 15 + 27)】")
	fmt.Println("Client -> Server:")
	fmt.Printf("  %s\n", step4)
	resp4 := server.handleMessage([]byte(step4))
	printResp(resp4)

	// 5. 工具调用 (tools/call 时间查询)
	step5 := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"get_current_time","arguments":{}}}`
	fmt.Println("【第 5 步：客户端调用时间查询工具 (tools/call: get_current_time)】")
	fmt.Println("Client -> Server:")
	fmt.Printf("  %s\n", step5)
	resp5 := server.handleMessage([]byte(step5))
	printResp(resp5)

	fmt.Println("================================================================")
	fmt.Println("✅ 演示完毕：展示了完整的 Initialize -> ToolsList -> ToolsCall 协议闭环！")
	fmt.Println("================================================================")
}

func printResp(resp *JSONRPCResponse) {
	fmt.Println("Server -> Client:")
	if resp != nil {
		fmt.Printf("  ID: %v, Result: %+v\n", resp.ID, resp.Result)
	}
	fmt.Println()
}
