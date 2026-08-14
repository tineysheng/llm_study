package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

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
- 只有用户明确要求读取安全示例目录中的文件时，才调用 file_reader；文件路径必须是相对路径。
- 解释概念、比较方案、总结内容、闲聊、代码设计建议、RAG/Agent/Function Calling 概念问题，都不需要工具。
- 不需要工具时，必须直接用中文简短回答，不要返回 tool call。
- 收到工具 observation 后，优先基于 observation 给出最终回答；只有确实还缺新的外部信息时才再次调用工具。
- 只允许在确实命中工具能力边界时调用工具。`
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

	if relativePath, ok := extractFileReadPath(question); ok {
		payload, err := json.Marshal(FileReaderArgs{RelativePath: relativePath})
		if err != nil {
			return ModelDecision{Content: "生成文件读取参数失败。"}
		}
		return ModelDecision{
			ToolCall: &ToolCall{
				Name:      "file_reader",
				Arguments: string(payload),
			},
		}
	}

	return ModelDecision{
		Content: "这个问题不需要调用工具。Function Calling 的核心是：模型先判断是否需要外部工具，如果需要，就返回结构化 tool call，由应用程序执行工具。",
	}
}

func extractFileReadPath(question string) (string, bool) {
	if !strings.Contains(question, "读取") && !strings.Contains(strings.ToLower(question), "read") {
		return "", false
	}

	quoted := regexp.MustCompile(`["'“”]([^"'“”]+\.(?:md|txt))["'“”]`)
	if matches := quoted.FindStringSubmatch(question); len(matches) == 2 {
		return strings.TrimSpace(matches[1]), true
	}

	filePattern := regexp.MustCompile(`([A-Za-z0-9_./\\-]+\.(?:md|txt))`)
	if matches := filePattern.FindStringSubmatch(question); len(matches) == 2 {
		return strings.TrimSpace(matches[1]), true
	}

	return "", false
}

func mockFollowUpDecision(question string, call ToolCall, result ToolResult) ModelDecision {
	return ModelDecision{
		Content: fmt.Sprintf("我已经收到 `%s` 工具的 observation：%s。基于这个结果，我可以回答原问题：%s", call.Name, summarizeObservation(result.Output, 180), question),
	}
}

func summarizeObservation(output string, maxRunes int) string {
	output = strings.TrimSpace(output)
	if maxRunes <= 0 {
		return output
	}
	runes := []rune(output)
	if len(runes) <= maxRunes {
		return output
	}
	return string(runes[:maxRunes]) + "..."
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
