package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

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
		fmt.Printf("Step %d - ReAct Trace\n", step.Number)
		fmt.Println("Action:", step.ToolCall.Name)
		fmt.Println("Action Input:", step.ToolCall.Arguments)
		fmt.Println("Observation:", step.ToolResult.Output)
		fmt.Println()
	}
}
