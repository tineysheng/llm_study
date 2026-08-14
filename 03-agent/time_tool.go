package main

import (
	"errors"
	"strings"
	"time"
)

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
