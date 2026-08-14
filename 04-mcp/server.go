package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// MCPServer 实现最小协议服务
type MCPServer struct {
	in  io.Reader
	out io.Writer
}

func NewMCPServer(in io.Reader, out io.Writer) *MCPServer {
	return &MCPServer{
		in:  in,
		out: out,
	}
}

// Serve 启动标准输入输出协议循环
func (s *MCPServer) Serve() error {
	scanner := bufio.NewScanner(s.in)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		resp := s.handleMessage(line)
		if resp != nil {
			data, err := json.Marshal(resp)
			if err != nil {
				return err
			}
			fmt.Fprintf(s.out, "%s\n", data)
		}
	}
	return scanner.Err()
}

// handleMessage 路由 JSON-RPC 消息
func (s *MCPServer) handleMessage(raw []byte) *JSONRPCResponse {
	var req JSONRPCRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      nil,
			Error:   &JSONRPCError{Code: -32700, Message: "Parse error: " + err.Error()},
		}
	}

	// 处理通知（Notification 不需要返回 response）
	if req.ID == nil {
		return nil
	}

	switch req.Method {
	case "initialize":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: InitializeResult{
				ProtocolVersion: "2024-11-05",
				Capabilities: ServerCapabilities{
					Tools: &ToolsCapability{},
				},
				ServerInfo: ImplementationInfo{
					Name:    "go-mcp-demo-server",
					Version: "1.0.0",
				},
			},
		}

	case "tools/list":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: ToolsListResult{
				Tools: []MCPTool{
					{
						Name:        "calculator",
						Description: "执行基础数学运算 (+, -, *, /)",
						InputSchema: map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"left_operand":  map[string]interface{}{"type": "number", "description": "左操作数"},
								"right_operand": map[string]interface{}{"type": "number", "description": "右操作数"},
								"operation": map[string]interface{}{
									"type":        "string",
									"description": "运算符",
									"enum":        []string{"add", "subtract", "multiply", "divide"},
								},
							},
							"required": []string{"left_operand", "right_operand", "operation"},
						},
					},
					{
						Name:        "get_current_time",
						Description: "获取当前系统时间和日期",
						InputSchema: map[string]interface{}{
							"type":       "object",
							"properties": map[string]interface{}{},
						},
					},
				},
			},
		}

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &JSONRPCError{Code: -32602, Message: "Invalid params: " + err.Error()},
			}
		}

		res := s.executeTool(params.Name, params.Arguments)
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  res,
		}

	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &JSONRPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)},
		}
	}
}

// executeTool 执行具体工具逻辑
func (s *MCPServer) executeTool(name string, args map[string]interface{}) CallToolResult {
	switch name {
	case "calculator":
		left, ok1 := args["left_operand"].(float64)
		right, ok2 := args["right_operand"].(float64)
		op, ok3 := args["operation"].(string)
		if !ok1 || !ok2 || !ok3 {
			return CallToolResult{
				IsError: true,
				Content: []ToolContent{{Type: "text", Text: "参数不完整或类型错误"}},
			}
		}

		var res float64
		switch op {
		case "add":
			res = left + right
		case "subtract":
			res = left - right
		case "multiply":
			res = left * right
		case "divide":
			if right == 0 {
				return CallToolResult{
					IsError: true,
					Content: []ToolContent{{Type: "text", Text: "除数不能为0"}},
				}
			}
			res = left / right
		default:
			return CallToolResult{
				IsError: true,
				Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("不支持的操作符: %s", op)}},
			}
		}
		return CallToolResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("%v", res)}},
		}

	case "get_current_time":
		nowStr := time.Now().Format("2006-01-02 15:04:05 MST")
		return CallToolResult{
			Content: []ToolContent{{Type: "text", Text: "当前时间是: " + nowStr}},
		}

	default:
		return CallToolResult{
			IsError: true,
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("未知工具: %s", name)}},
		}
	}
}
