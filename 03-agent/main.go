// Agent Loop 基础演示
//
// 本程序演示最小 Agent Loop：
// 用户问题 -> 模型决定是否调用工具 -> Go 执行工具 -> observation 回传给模型 -> 模型继续生成最终回答。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

type ToolSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ToolCall struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ModelDecision struct {
	Content  string
	ToolCall *ToolCall
}

type CalculatorArgs struct {
	LeftOperand  float64 `json:"left_operand"`
	RightOperand float64 `json:"right_operand"`
	Operation    string  `json:"operation"`
}

type ToolResult struct {
	Name   string
	Output string
}

type AgentRun struct {
	FinalAnswer string
	Steps       []AgentStep
}

type AgentStep struct {
	Number     int
	ToolCall   ToolCall
	ToolResult ToolResult
}

type Tool interface {
	Schema() ToolSchema
	Execute(rawArguments string) (ToolResult, error)
}

type ToolRegistry struct {
	tools map[string]Tool
}

func main() {
	question := flag.String("question", "请计算 12 + 30", "user question")
	mode := flag.String("mode", "mock", "model mode: mock or real")
	model := flag.String("model", getEnvString("OPENAI_MODEL", openai.GPT4oMini), "model name used when -mode real")
	timeoutSeconds := flag.Int("timeout-seconds", getEnvInt("OPENAI_TIMEOUT_SECONDS", 60), "request timeout in seconds when -mode real")
	showSchema := flag.Bool("show-schema", false, "print full tool schemas sent to the model")
	maxSteps := flag.Int("max-steps", 3, "maximum number of tool calls allowed in one agent loop")
	flag.Parse()

	if strings.TrimSpace(*question) == "" {
		fmt.Println("问题不能为空")
		os.Exit(1)
	}

	registry, err := NewToolRegistry(CalculatorTool{}, CurrentTimeTool{})
	if err != nil {
		fmt.Println("注册工具失败:", err)
		os.Exit(1)
	}

	tools := registry.Schemas()
	run, err := runAgentLoop(*mode, *question, registry, *model, time.Duration(*timeoutSeconds)*time.Second, *maxSteps)
	if err != nil {
		fmt.Println("Agent Loop 执行失败:", err)
		os.Exit(1)
	}

	fmt.Println("Agent Loop 基础演示")
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

func runAgentLoop(mode, question string, registry *ToolRegistry, model string, timeout time.Duration, maxSteps int) (AgentRun, error) {
	if maxSteps <= 0 {
		return AgentRun{}, errors.New("max-steps 必须大于 0")
	}

	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "mock":
		return runMockAgentLoop(question, registry, maxSteps)
	case "real":
		apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
		if apiKey == "" {
			return AgentRun{}, errors.New("-mode real 需要先设置 OPENAI_API_KEY")
		}
		if strings.TrimSpace(model) == "" {
			return AgentRun{}, errors.New("model 不能为空")
		}
		if timeout <= 0 {
			return AgentRun{}, errors.New("timeout 必须大于 0")
		}
		client := openai.NewClient(apiKey)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		return runRealAgentLoop(ctx, client, model, question, registry, maxSteps)
	default:
		return AgentRun{}, fmt.Errorf("未知 mode: %s，可选 mock 或 real", mode)
	}
}

func runMockAgentLoop(question string, registry *ToolRegistry, maxSteps int) (AgentRun, error) {
	run := AgentRun{}
	decision := mockModelDecision(question)
	toolCalls := 0

	for {
		var err error
		decision, err = validateModelDecision(decision)
		if err != nil {
			return AgentRun{}, fmt.Errorf("模型输出校验失败: %w", err)
		}
		if decision.ToolCall == nil {
			run.FinalAnswer = decision.Content
			return run, nil
		}
		if toolCalls >= maxSteps {
			return AgentRun{}, fmt.Errorf("超过最大工具调用次数 max-steps=%d，已停止以避免无限循环", maxSteps)
		}

		result, err := executeAndValidateTool(registry, *decision.ToolCall)
		if err != nil {
			return AgentRun{}, err
		}
		toolCalls++
		run.Steps = append(run.Steps, AgentStep{Number: toolCalls, ToolCall: *decision.ToolCall, ToolResult: result})

		decision = mockFollowUpDecision(question, *decision.ToolCall, result)
	}
}

func runRealAgentLoop(ctx context.Context, client *openai.Client, model, question string, registry *ToolRegistry, maxSteps int) (AgentRun, error) {
	run := AgentRun{}
	tools := registry.Schemas()
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: toolChoiceSystemPrompt()},
		{Role: openai.ChatMessageRoleUser, Content: question},
	}
	toolCalls := 0

	for {
		decision, assistantMessage, err := nextRealModelDecision(ctx, client, model, messages, tools)
		if err != nil {
			return AgentRun{}, err
		}
		decision, err = validateModelDecision(decision)
		if err != nil {
			return AgentRun{}, fmt.Errorf("模型输出校验失败: %w", err)
		}
		if decision.ToolCall == nil {
			run.FinalAnswer = decision.Content
			return run, nil
		}
		if decision.ToolCall.ID == "" {
			return AgentRun{}, errors.New("真实 LLM 返回的 tool call id 为空，无法回传 observation")
		}
		if toolCalls >= maxSteps {
			return AgentRun{}, fmt.Errorf("超过最大工具调用次数 max-steps=%d，已停止以避免无限循环", maxSteps)
		}

		result, err := executeAndValidateTool(registry, *decision.ToolCall)
		if err != nil {
			return AgentRun{}, err
		}
		toolCalls++
		run.Steps = append(run.Steps, AgentStep{Number: toolCalls, ToolCall: *decision.ToolCall, ToolResult: result})

		messages = append(messages, assistantMessage)
		messages = append(messages, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			ToolCallID: decision.ToolCall.ID,
			Name:       result.Name,
			Content:    result.Output,
		})
	}
}

func executeAndValidateTool(registry *ToolRegistry, call ToolCall) (ToolResult, error) {
	if err := registry.ValidateToolCall(call); err != nil {
		return ToolResult{}, fmt.Errorf("工具选择校验失败: %w", err)
	}
	result, err := registry.Dispatch(call)
	if err != nil {
		return ToolResult{}, fmt.Errorf("工具调用失败: %w", err)
	}
	result, err = validateToolResult(call, result)
	if err != nil {
		return ToolResult{}, fmt.Errorf("工具结果校验失败: %w", err)
	}
	return result, nil
}

func printAgentTrace(run AgentRun) {
	if len(run.Steps) == 0 {
		fmt.Println("模型决定：不需要工具，直接回答。")
		fmt.Println()
		return
	}

	for _, step := range run.Steps {
		fmt.Printf("Step %d - Model Tool Call\n", step.Number)
		fmt.Println("tool:", step.ToolCall.Name)
		fmt.Println("arguments:", step.ToolCall.Arguments)
		fmt.Println("Observation:", step.ToolResult.Output)
		fmt.Println()
	}
}

func decideWithMode(mode, question string, tools []ToolSchema, model string, timeout time.Duration) (ModelDecision, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "mock":
		return mockModelDecision(question), nil
	case "real":
		apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
		if apiKey == "" {
			return ModelDecision{}, errors.New("-mode real 需要先设置 OPENAI_API_KEY")
		}
		if strings.TrimSpace(model) == "" {
			return ModelDecision{}, errors.New("model 不能为空")
		}
		if timeout <= 0 {
			return ModelDecision{}, errors.New("timeout 必须大于 0")
		}
		client := openai.NewClient(apiKey)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		return realModelDecision(ctx, client, model, question, tools)
	default:
		return ModelDecision{}, fmt.Errorf("未知 mode: %s，可选 mock 或 real", mode)
	}
}

func validateModelDecision(decision ModelDecision) (ModelDecision, error) {
	if decision.ToolCall == nil {
		decision.Content = strings.TrimSpace(decision.Content)
		if decision.Content == "" {
			return ModelDecision{}, errors.New("模型既没有返回 tool call，也没有返回可用文本")
		}
		return decision, nil
	}

	decision.ToolCall.ID = strings.TrimSpace(decision.ToolCall.ID)
	decision.ToolCall.Name = strings.TrimSpace(decision.ToolCall.Name)
	if decision.ToolCall.Name == "" {
		return ModelDecision{}, errors.New("模型返回的工具名为空")
	}

	decision.ToolCall.Arguments = strings.TrimSpace(decision.ToolCall.Arguments)
	if decision.ToolCall.Arguments == "" {
		return ModelDecision{}, fmt.Errorf("模型返回的 %s arguments 为空", decision.ToolCall.Name)
	}
	if !json.Valid([]byte(decision.ToolCall.Arguments)) {
		return ModelDecision{}, fmt.Errorf("模型返回的 %s arguments 不是合法 JSON: %s", decision.ToolCall.Name, decision.ToolCall.Arguments)
	}

	return decision, nil
}

func validateToolResult(call ToolCall, result ToolResult) (ToolResult, error) {
	call.Name = strings.TrimSpace(call.Name)
	result.Name = strings.TrimSpace(result.Name)
	result.Output = strings.TrimSpace(result.Output)

	if call.Name == "" {
		return ToolResult{}, errors.New("tool call 名称不能为空")
	}
	if result.Name == "" {
		return ToolResult{}, errors.New("工具结果名称不能为空")
	}
	if result.Name != call.Name {
		return ToolResult{}, fmt.Errorf("工具结果名称不匹配: call=%s result=%s", call.Name, result.Name)
	}
	if result.Output == "" {
		return ToolResult{}, fmt.Errorf("%s 工具结果不能为空", result.Name)
	}

	return result, nil
}

func realModelDecision(ctx context.Context, client *openai.Client, model, question string, tools []ToolSchema) (ModelDecision, error) {
	messages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: toolChoiceSystemPrompt()},
		{Role: openai.ChatMessageRoleUser, Content: question},
	}
	decision, _, err := nextRealModelDecision(ctx, client, model, messages, tools)
	return decision, err
}

func nextRealModelDecision(ctx context.Context, client *openai.Client, model string, messages []openai.ChatCompletionMessage, tools []ToolSchema) (ModelDecision, openai.ChatCompletionMessage, error) {
	req := openai.ChatCompletionRequest{
		Model:               model,
		Temperature:         0,
		Messages:            messages,
		Tools:               toOpenAITools(tools),
		ToolChoice:          "auto",
		MaxCompletionTokens: 300,
		ParallelToolCalls:   false,
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return ModelDecision{}, openai.ChatCompletionMessage{}, err
	}
	if len(resp.Choices) == 0 {
		return ModelDecision{}, openai.ChatCompletionMessage{}, errors.New("模型没有返回 choices")
	}

	message := resp.Choices[0].Message
	if len(message.ToolCalls) == 0 {
		return ModelDecision{Content: strings.TrimSpace(message.Content)}, message, nil
	}
	if len(message.ToolCalls) > 1 {
		return ModelDecision{}, openai.ChatCompletionMessage{}, fmt.Errorf("当前 demo 只支持单次工具调用，但模型返回了 %d 次 tool calls", len(message.ToolCalls))
	}

	call := message.ToolCalls[0]
	if call.Type != openai.ToolTypeFunction {
		return ModelDecision{}, openai.ChatCompletionMessage{}, fmt.Errorf("不支持的 tool call 类型: %s", call.Type)
	}

	return ModelDecision{
		ToolCall: &ToolCall{
			ID:        call.ID,
			Name:      call.Function.Name,
			Arguments: call.Function.Arguments,
		},
	}, message, nil
}

func toolChoiceSystemPrompt() string {
	return `你是一个严格的工具调用路由器和 Agent Loop 控制器。请先判断用户问题是否真的需要外部工具：
- 只有用户给出明确数字并要求加减乘除精确计算时，才调用 calculator。
- 只有用户询问当前日期、当前时间、现在几点、今天几号时，才调用 current_time。
- 解释概念、比较方案、总结内容、闲聊、代码设计建议、RAG/Agent/Function Calling 概念问题，都不需要工具。
- 不需要工具时，必须直接用中文简短回答，不要返回 tool call。
- 收到工具 observation 后，优先基于 observation 给出最终回答；只有确实还缺新的外部信息时才再次调用工具。
- 只允许在确实命中工具能力边界时调用工具。`
}

func toOpenAITools(schemas []ToolSchema) []openai.Tool {
	tools := make([]openai.Tool, 0, len(schemas))
	for _, schema := range schemas {
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        schema.Name,
				Description: schema.Description,
				Parameters:  schema.Parameters,
			},
		})
	}
	return tools
}

func printToolSchemas(tools []ToolSchema) {
	data, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		fmt.Println("打印 tool schema 失败:", err)
		return
	}
	fmt.Println(string(data))
}

func NewToolRegistry(tools ...Tool) (*ToolRegistry, error) {
	registry := &ToolRegistry{tools: make(map[string]Tool)}
	for _, tool := range tools {
		schema := tool.Schema()
		name := strings.TrimSpace(schema.Name)
		if name == "" {
			return nil, errors.New("工具名称不能为空")
		}
		if _, exists := registry.tools[name]; exists {
			return nil, fmt.Errorf("重复注册工具: %s", name)
		}
		registry.tools[name] = tool
	}
	return registry, nil
}

func (r *ToolRegistry) Schemas() []ToolSchema {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	schemas := make([]ToolSchema, 0, len(names))
	for _, name := range names {
		schemas = append(schemas, r.tools[name].Schema())
	}
	return schemas
}

func (r *ToolRegistry) Dispatch(call ToolCall) (ToolResult, error) {
	tool, ok := r.tools[call.Name]
	if !ok {
		return ToolResult{}, fmt.Errorf("未知工具: %s", call.Name)
	}
	return tool.Execute(call.Arguments)
}

func (r *ToolRegistry) ValidateToolCall(call ToolCall) error {
	name := strings.TrimSpace(call.Name)
	if name == "" {
		return errors.New("工具名不能为空")
	}
	if _, ok := r.tools[name]; !ok {
		return fmt.Errorf("模型选择了未注册工具: %s", name)
	}
	return nil
}

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

type CurrentTimeTool struct{}

func (CurrentTimeTool) Schema() ToolSchema {
	return ToolSchema{
		Name:        "current_time",
		Description: "返回当前本地时间，适合回答当前日期、当前时间、现在几点等问题。",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
			"required":   []string{},
		},
	}
}

func (CurrentTimeTool) Execute(rawArguments string) (ToolResult, error) {
	if strings.TrimSpace(rawArguments) != "" && strings.TrimSpace(rawArguments) != "{}" {
		return ToolResult{}, errors.New("current_time 不需要参数")
	}

	now := time.Now().Format("2006-01-02 15:04:05 MST")
	return ToolResult{Name: "current_time", Output: now}, nil
}

func getEnvString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func mockModelDecision(question string) ModelDecision {
	expression, ok := extractMathExpression(question)
	if ok {
		args := CalculatorArgs{LeftOperand: expression.a, RightOperand: expression.b, Operation: expression.op}
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

	if needsCurrentTime(question) {
		return ModelDecision{
			ToolCall: &ToolCall{
				Name:      "current_time",
				Arguments: `{}`,
			},
		}
	}

	return ModelDecision{
		Content: "这个问题不需要调用工具。Function Calling 的核心是：模型先判断是否需要外部工具，如果需要，就返回结构化 tool call，由应用程序执行工具。",
	}
}

func mockFollowUpDecision(question string, call ToolCall, result ToolResult) ModelDecision {
	return ModelDecision{
		Content: fmt.Sprintf("我已经收到 `%s` 工具的 observation：%s。基于这个结果，原问题 `%s` 的答案是：%s。", call.Name, result.Output, question, result.Output),
	}
}

func needsCurrentTime(question string) bool {
	keywords := []string{"当前时间", "现在几点", "几点了", "今天日期", "当前日期", "今天几号"}
	for _, keyword := range keywords {
		if strings.Contains(question, keyword) {
			return true
		}
	}
	return false
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

func mockFinalAnswer(question string, result ToolResult) string {
	return fmt.Sprintf("我通过工具 `%s` 得到结果：%s。\n原问题：%s", result.Name, result.Output, question)
}

func formatNumber(value float64) string {
	if math.Trunc(value) == value {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}
