package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

func newDefaultToolRegistry() (*ToolRegistry, error) {
	return NewToolRegistry(CalculatorTool{}, CurrentTimeTool{}, NewSafeFileReaderTool(defaultSafeFilesDir(), 4096))
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
