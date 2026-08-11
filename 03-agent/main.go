// Function Calling 基础演示
//
// 本程序使用 mock model 演示最小工具调用闭环：
// 用户问题 -> 模型决定是否调用工具 -> Go 解析参数 -> 执行工具 -> 基于工具结果回答。
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type ToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ToolCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ModelDecision struct {
	Content  string
	ToolCall *ToolCall
}

type CalculatorArgs struct {
	A  float64 `json:"a"`
	B  float64 `json:"b"`
	Op string  `json:"op"`
}

type ToolResult struct {
	Name   string
	Output string
}

func main() {
	question := flag.String("question", "请计算 12 + 30", "user question")
	flag.Parse()

	if strings.TrimSpace(*question) == "" {
		fmt.Println("问题不能为空")
		os.Exit(1)
	}

	tools := []ToolSchema{calculatorSchema()}
	decision := mockModelDecision(*question)

	fmt.Println("Function Calling 基础演示")
	fmt.Println("用户问题:", *question)
	fmt.Println()

	fmt.Println("=== Registered Tools ===")
	for _, tool := range tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}
	fmt.Println()

	fmt.Println("=== Model Decision ===")
	if decision.ToolCall == nil {
		fmt.Println("模型决定：不需要工具，直接回答。")
		fmt.Println()
		fmt.Println("=== Final Answer ===")
		fmt.Println(decision.Content)
		return
	}

	fmt.Println("模型决定调用工具:", decision.ToolCall.Name)
	fmt.Println("arguments:", decision.ToolCall.Arguments)
	fmt.Println()

	result, err := dispatchToolCall(*decision.ToolCall)
	if err != nil {
		fmt.Println("工具调用失败:", err)
		os.Exit(1)
	}

	fmt.Println("=== Tool Result / Observation ===")
	fmt.Printf("%s result: %s\n", result.Name, result.Output)
	fmt.Println()

	fmt.Println("=== Final Answer ===")
	fmt.Println(mockFinalAnswer(*question, result))
}

func calculatorSchema() ToolSchema {
	return ToolSchema{
		Name:        "calculator",
		Description: "执行基础数学运算，适合需要精确计算的问题。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"a": map[string]any{
					"type":        "number",
					"description": "第一个数字",
				},
				"b": map[string]any{
					"type":        "number",
					"description": "第二个数字",
				},
				"op": map[string]any{
					"type":        "string",
					"description": "运算类型",
					"enum":        []string{"add", "subtract", "multiply", "divide"},
				},
			},
			"required": []string{"a", "b", "op"},
		},
	}
}

func mockModelDecision(question string) ModelDecision {
	expression, ok := extractMathExpression(question)
	if !ok {
		return ModelDecision{
			Content: "这个问题不需要调用工具。Function Calling 的核心是：模型先判断是否需要外部工具，如果需要，就返回结构化 tool call，由应用程序执行工具。",
		}
	}

	args := CalculatorArgs{A: expression.a, B: expression.b, Op: expression.op}
	payload, err := json.Marshal(args)
	if err != nil {
		return ModelDecision{Content: "生成工具参数失败。"}
	}

	return ModelDecision{
		ToolCall: &ToolCall{
			Name:      "calculator",
			Arguments: string(payload),
		},
	}
}

type mathExpression struct {
	a  float64
	b  float64
	op string
}

func extractMathExpression(question string) (mathExpression, bool) {
	re := regexp.MustCompile(`(-?\d+(?:\.\d+)?)\s*([+\-*/×÷])\s*(-?\d+(?:\.\d+)?)`)
	matches := re.FindStringSubmatch(question)
	if len(matches) != 4 {
		return mathExpression{}, false
	}

	a, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return mathExpression{}, false
	}
	b, err := strconv.ParseFloat(matches[3], 64)
	if err != nil {
		return mathExpression{}, false
	}

	op, ok := normalizeOperator(matches[2])
	if !ok {
		return mathExpression{}, false
	}

	return mathExpression{a: a, b: b, op: op}, true
}

func normalizeOperator(symbol string) (string, bool) {
	switch symbol {
	case "+":
		return "add", true
	case "-":
		return "subtract", true
	case "*", "×":
		return "multiply", true
	case "/", "÷":
		return "divide", true
	default:
		return "", false
	}
}

func dispatchToolCall(call ToolCall) (ToolResult, error) {
	if call.Name != "calculator" {
		return ToolResult{}, fmt.Errorf("未知工具: %s", call.Name)
	}

	args, err := parseCalculatorArgs(call.Arguments)
	if err != nil {
		return ToolResult{}, err
	}

	output, err := executeCalculator(args)
	if err != nil {
		return ToolResult{}, err
	}

	return ToolResult{Name: call.Name, Output: formatNumber(output)}, nil
}

func parseCalculatorArgs(raw string) (CalculatorArgs, error) {
	if strings.TrimSpace(raw) == "" {
		return CalculatorArgs{}, errors.New("calculator arguments 不能为空")
	}

	var args CalculatorArgs
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return CalculatorArgs{}, fmt.Errorf("解析 calculator arguments 失败: %w", err)
	}

	if math.IsNaN(args.A) || math.IsInf(args.A, 0) || math.IsNaN(args.B) || math.IsInf(args.B, 0) {
		return CalculatorArgs{}, errors.New("calculator 参数不能是 NaN 或 Inf")
	}
	if _, ok := allowedOps()[args.Op]; !ok {
		return CalculatorArgs{}, fmt.Errorf("不支持的运算类型: %s", args.Op)
	}

	return args, nil
}

func allowedOps() map[string]struct{} {
	return map[string]struct{}{
		"add":      {},
		"subtract": {},
		"multiply": {},
		"divide":   {},
	}
}

func executeCalculator(args CalculatorArgs) (float64, error) {
	switch args.Op {
	case "add":
		return args.A + args.B, nil
	case "subtract":
		return args.A - args.B, nil
	case "multiply":
		return args.A * args.B, nil
	case "divide":
		if args.B == 0 {
			return 0, errors.New("除数不能为 0")
		}
		return args.A / args.B, nil
	default:
		return 0, fmt.Errorf("不支持的运算类型: %s", args.Op)
	}
}

func mockFinalAnswer(question string, result ToolResult) string {
	return fmt.Sprintf("我通过工具 `%s` 得到结果：%s。\n原问题：%s", result.Name, result.Output, question)
}

func formatNumber(value float64) string {
	if math.Trunc(value) == value {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}
