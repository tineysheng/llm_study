package main

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

type DemoScenario struct {
	Title    string
	Question string
}

type Tool interface {
	Schema() ToolSchema
	Execute(rawArguments string) (ToolResult, error)
}

type ToolRegistry struct {
	tools map[string]Tool
}
