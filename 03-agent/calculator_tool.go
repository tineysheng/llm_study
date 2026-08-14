package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type CalculatorTool struct{}

func (CalculatorTool) Schema() ToolSchema {
	return ToolSchema{
		Name:        "calculator",
		Description: "仅当用户给出明确数字并需要加减乘除精确计算时使用。不要用于解释数学概念、闲聊或非计算任务。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"left_operand": map[string]any{
					"type":        "number",
					"description": "数学表达式左侧的数字，例如 7 * 6 中的 7。",
				},
				"right_operand": map[string]any{
					"type":        "number",
					"description": "数学表达式右侧的数字，例如 7 * 6 中的 6。",
				},
				"operation": map[string]any{
					"type":        "string",
					"description": "要执行的运算：add=加法，subtract=减法，multiply=乘法，divide=除法。必须从 enum 中选择。",
					"enum":        []string{"add", "subtract", "multiply", "divide"},
				},
			},
			"required": []string{"left_operand", "right_operand", "operation"},
		},
	}
}

func (CalculatorTool) Execute(rawArguments string) (ToolResult, error) {
	args, err := parseCalculatorArgs(rawArguments)
	if err != nil {
		return ToolResult{}, err
	}

	output, err := executeCalculator(args)
	if err != nil {
		return ToolResult{}, err
	}

	return ToolResult{Name: "calculator", Output: formatNumber(output)}, nil
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

func parseCalculatorArgs(raw string) (CalculatorArgs, error) {
	if strings.TrimSpace(raw) == "" {
		return CalculatorArgs{}, errors.New("calculator arguments 不能为空")
	}

	var payload struct {
		LeftOperand  *float64 `json:"left_operand"`
		RightOperand *float64 `json:"right_operand"`
		Operation    *string  `json:"operation"`
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return CalculatorArgs{}, fmt.Errorf("解析 calculator arguments 失败: %w", err)
	}
	if payload.LeftOperand == nil {
		return CalculatorArgs{}, errors.New("calculator 缺少必填参数: left_operand")
	}
	if payload.RightOperand == nil {
		return CalculatorArgs{}, errors.New("calculator 缺少必填参数: right_operand")
	}
	if payload.Operation == nil || strings.TrimSpace(*payload.Operation) == "" {
		return CalculatorArgs{}, errors.New("calculator 缺少必填参数: operation")
	}

	args := CalculatorArgs{
		LeftOperand:  *payload.LeftOperand,
		RightOperand: *payload.RightOperand,
		Operation:    strings.TrimSpace(*payload.Operation),
	}

	if math.IsNaN(args.LeftOperand) || math.IsInf(args.LeftOperand, 0) || math.IsNaN(args.RightOperand) || math.IsInf(args.RightOperand, 0) {
		return CalculatorArgs{}, errors.New("calculator 参数不能是 NaN 或 Inf")
	}
	if _, ok := allowedOps()[args.Operation]; !ok {
		return CalculatorArgs{}, fmt.Errorf("不支持的运算类型: %s", args.Operation)
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
	switch args.Operation {
	case "add":
		return args.LeftOperand + args.RightOperand, nil
	case "subtract":
		return args.LeftOperand - args.RightOperand, nil
	case "multiply":
		return args.LeftOperand * args.RightOperand, nil
	case "divide":
		if args.RightOperand == 0 {
			return 0, errors.New("除数不能为 0")
		}
		return args.LeftOperand / args.RightOperand, nil
	default:
		return 0, fmt.Errorf("不支持的运算类型: %s", args.Operation)
	}
}

func formatNumber(value float64) string {
	if math.Trunc(value) == value {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}
