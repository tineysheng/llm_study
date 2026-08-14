package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

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
