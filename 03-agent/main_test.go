package main

import (
	"os"
	"strings"
	"testing"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

func TestExtractMathExpression(t *testing.T) {
	tests := []struct {
		name     string
		question string
		wantA    float64
		wantB    float64
		wantOp   string
		wantOK   bool
	}{
		{name: "add", question: "请计算 12 + 30", wantA: 12, wantB: 30, wantOp: "add", wantOK: true},
		{name: "multiply", question: "9 * 8 等于多少", wantA: 9, wantB: 8, wantOp: "multiply", wantOK: true},
		{name: "divide chinese symbol", question: "8 ÷ 2", wantA: 8, wantB: 2, wantOp: "divide", wantOK: true},
		{name: "not math", question: "介绍一下 Function Calling", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractMathExpression(tt.question)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.a != tt.wantA || got.b != tt.wantB || got.op != tt.wantOp {
				t.Fatalf("expression = %+v, want a=%v b=%v op=%s", got, tt.wantA, tt.wantB, tt.wantOp)
			}
		})
	}
}

func TestParseCalculatorArgs(t *testing.T) {
	args, err := parseCalculatorArgs(`{"left_operand":12,"right_operand":30,"operation":"add"}`)
	if err != nil {
		t.Fatalf("parseCalculatorArgs returned error: %v", err)
	}
	if args.LeftOperand != 12 || args.RightOperand != 30 || args.Operation != "add" {
		t.Fatalf("args = %+v", args)
	}
}

func TestParseCalculatorArgsRejectsUnknownOp(t *testing.T) {
	_, err := parseCalculatorArgs(`{"left_operand":12,"right_operand":30,"operation":"power"}`)
	if err == nil {
		t.Fatal("expected error for unknown op")
	}
}

func TestParseCalculatorArgsRejectsMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "missing left operand", raw: `{"right_operand":6,"operation":"multiply"}`},
		{name: "missing right operand", raw: `{"left_operand":7,"operation":"multiply"}`},
		{name: "missing operation", raw: `{"left_operand":7,"right_operand":6}`},
		{name: "empty operation", raw: `{"left_operand":7,"right_operand":6,"operation":" "}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseCalculatorArgs(tt.raw)
			if err == nil {
				t.Fatal("expected error for missing required field")
			}
		})
	}
}

func TestParseCalculatorArgsRejectsUnknownFields(t *testing.T) {
	_, err := parseCalculatorArgs(`{"a":7,"b":6,"op":"multiply"}`)
	if err == nil {
		t.Fatal("expected error for unknown old-style fields")
	}
}

func TestExecuteCalculator(t *testing.T) {
	got, err := executeCalculator(CalculatorArgs{LeftOperand: 9, RightOperand: 8, Operation: "multiply"})
	if err != nil {
		t.Fatalf("executeCalculator returned error: %v", err)
	}
	if got != 72 {
		t.Fatalf("got %v, want 72", got)
	}
}

func TestExecuteCalculatorRejectsDivideByZero(t *testing.T) {
	_, err := executeCalculator(CalculatorArgs{LeftOperand: 8, RightOperand: 0, Operation: "divide"})
	if err == nil {
		t.Fatal("expected divide by zero error")
	}
}

func TestMockModelDecisionRequestsToolForMath(t *testing.T) {
	decision := mockModelDecision("请计算 12 + 30")
	if decision.ToolCall == nil {
		t.Fatal("expected tool call")
	}
	if decision.ToolCall.Name != "calculator" {
		t.Fatalf("tool name = %s, want calculator", decision.ToolCall.Name)
	}
}

func TestToolRegistryDispatchesCalculator(t *testing.T) {
	registry, err := NewToolRegistry(CalculatorTool{}, CurrentTimeTool{})
	if err != nil {
		t.Fatalf("NewToolRegistry returned error: %v", err)
	}

	result, err := registry.Dispatch(ToolCall{Name: "calculator", Arguments: `{"left_operand":7,"right_operand":6,"operation":"multiply"}`})
	if err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}
	if result.Name != "calculator" || result.Output != "42" {
		t.Fatalf("result = %+v, want calculator 42", result)
	}
}

func TestToolRegistryRejectsUnknownTool(t *testing.T) {
	registry, err := NewToolRegistry(CalculatorTool{})
	if err != nil {
		t.Fatalf("NewToolRegistry returned error: %v", err)
	}

	_, err = registry.Dispatch(ToolCall{Name: "unknown", Arguments: `{}`})
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestToolRegistryValidateToolCall(t *testing.T) {
	registry, err := NewToolRegistry(CalculatorTool{}, CurrentTimeTool{})
	if err != nil {
		t.Fatalf("NewToolRegistry returned error: %v", err)
	}

	if err := registry.ValidateToolCall(ToolCall{Name: "calculator", Arguments: `{}`}); err != nil {
		t.Fatalf("ValidateToolCall returned error for registered tool: %v", err)
	}
	if err := registry.ValidateToolCall(ToolCall{Name: "rag_search", Arguments: `{}`}); err == nil {
		t.Fatal("expected error for unregistered tool")
	}
}

func TestToolRegistryRejectsDuplicateTool(t *testing.T) {
	_, err := NewToolRegistry(CalculatorTool{}, CalculatorTool{})
	if err == nil {
		t.Fatal("expected duplicate tool error")
	}
}

func TestMockModelDecisionRequestsCurrentTimeTool(t *testing.T) {
	decision := mockModelDecision("现在几点了？")
	if decision.ToolCall == nil {
		t.Fatal("expected current_time tool call")
	}
	if decision.ToolCall.Name != "current_time" {
		t.Fatalf("tool name = %s, want current_time", decision.ToolCall.Name)
	}
}

func TestCurrentTimeToolRejectsUnexpectedArguments(t *testing.T) {
	_, err := CurrentTimeTool{}.Execute(`{"timezone":"UTC"}`)
	if err == nil {
		t.Fatal("expected current_time argument error")
	}
}

func TestToOpenAIToolsPreservesSchemaForModel(t *testing.T) {
	registry, err := NewToolRegistry(CalculatorTool{}, CurrentTimeTool{})
	if err != nil {
		t.Fatalf("NewToolRegistry returned error: %v", err)
	}

	tools := toOpenAITools(registry.Schemas())
	if len(tools) != 2 {
		t.Fatalf("len(tools) = %d, want 2", len(tools))
	}
	if tools[0].Type != openai.ToolTypeFunction {
		t.Fatalf("tool type = %s, want function", tools[0].Type)
	}
	if tools[0].Function == nil || tools[0].Function.Name == "" || tools[0].Function.Parameters == nil {
		t.Fatalf("tool function schema is incomplete: %+v", tools[0].Function)
	}
}

func TestCalculatorSchemaUsesLLMFriendlyParameterNames(t *testing.T) {
	schema := CalculatorTool{}.Schema()
	properties, ok := schema.Parameters["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties missing: %+v", schema.Parameters)
	}
	for _, name := range []string{"left_operand", "right_operand", "operation"} {
		if _, ok := properties[name]; !ok {
			t.Fatalf("schema missing parameter %q", name)
		}
	}
}

func TestDecideWithModeMockDoesNotNeedAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")

	decision, err := decideWithMode("mock", "请计算 2 + 3", nil, "", time.Second)
	if err != nil {
		t.Fatalf("decideWithMode mock returned error: %v", err)
	}
	if decision.ToolCall == nil || decision.ToolCall.Name != "calculator" {
		t.Fatalf("decision = %+v, want calculator tool call", decision)
	}
}

func TestDecideWithModeRealRequiresAPIKey(t *testing.T) {
	old := os.Getenv("OPENAI_API_KEY")
	t.Setenv("OPENAI_API_KEY", "")
	defer os.Setenv("OPENAI_API_KEY", old)

	_, err := decideWithMode("real", "现在几点了？", nil, openai.GPT4oMini, time.Second)
	if err == nil {
		t.Fatal("expected error when real mode has no API key")
	}
	if !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatalf("error = %v, want OPENAI_API_KEY hint", err)
	}
}

func TestValidateModelDecisionAllowsTextAnswer(t *testing.T) {
	decision, err := validateModelDecision(ModelDecision{Content: "  不需要工具，直接回答。  "})
	if err != nil {
		t.Fatalf("validateModelDecision returned error: %v", err)
	}
	if decision.Content != "不需要工具，直接回答。" {
		t.Fatalf("content = %q, want trimmed text", decision.Content)
	}
}

func TestValidateModelDecisionRejectsEmptyTextAnswer(t *testing.T) {
	_, err := validateModelDecision(ModelDecision{Content: "   "})
	if err == nil {
		t.Fatal("expected error for empty text answer")
	}
}

func TestValidateModelDecisionRejectsEmptyToolName(t *testing.T) {
	_, err := validateModelDecision(ModelDecision{ToolCall: &ToolCall{Name: " ", Arguments: `{}`}})
	if err == nil {
		t.Fatal("expected error for empty tool name")
	}
}

func TestValidateModelDecisionRejectsEmptyToolArguments(t *testing.T) {
	_, err := validateModelDecision(ModelDecision{ToolCall: &ToolCall{Name: "calculator", Arguments: " "}})
	if err == nil {
		t.Fatal("expected error for empty tool arguments")
	}
}

func TestValidateModelDecisionRejectsInvalidJSONArguments(t *testing.T) {
	_, err := validateModelDecision(ModelDecision{ToolCall: &ToolCall{Name: "calculator", Arguments: `op=add`}})
	if err == nil {
		t.Fatal("expected error for invalid JSON arguments")
	}
}

func TestValidateModelDecisionAllowsValidToolCall(t *testing.T) {
	decision, err := validateModelDecision(ModelDecision{ToolCall: &ToolCall{Name: " calculator ", Arguments: ` {"left_operand":1,"right_operand":2,"operation":"add"} `}})
	if err != nil {
		t.Fatalf("validateModelDecision returned error: %v", err)
	}
	if decision.ToolCall.Name != "calculator" || decision.ToolCall.Arguments != `{"left_operand":1,"right_operand":2,"operation":"add"}` {
		t.Fatalf("decision = %+v", decision.ToolCall)
	}
}

func TestValidateToolResultAllowsMatchingNonEmptyResult(t *testing.T) {
	result, err := validateToolResult(
		ToolCall{Name: " calculator ", Arguments: `{"left_operand":1,"right_operand":2,"operation":"add"}`},
		ToolResult{Name: " calculator ", Output: " 3 "},
	)
	if err != nil {
		t.Fatalf("validateToolResult returned error: %v", err)
	}
	if result.Name != "calculator" || result.Output != "3" {
		t.Fatalf("result = %+v, want trimmed calculator result", result)
	}
}

func TestValidateToolResultRejectsMismatchedName(t *testing.T) {
	_, err := validateToolResult(
		ToolCall{Name: "calculator", Arguments: `{"left_operand":1,"right_operand":2,"operation":"add"}`},
		ToolResult{Name: "current_time", Output: "2026-08-12 10:00:00 CST"},
	)
	if err == nil {
		t.Fatal("expected error for mismatched tool result name")
	}
}

func TestValidateToolResultRejectsEmptyOutput(t *testing.T) {
	_, err := validateToolResult(
		ToolCall{Name: "calculator", Arguments: `{"left_operand":1,"right_operand":2,"operation":"add"}`},
		ToolResult{Name: "calculator", Output: "   "},
	)
	if err == nil {
		t.Fatal("expected error for empty tool result output")
	}
}

func TestRunAgentLoopMockUsesToolThenFinalAnswer(t *testing.T) {
	registry, err := NewToolRegistry(CalculatorTool{}, CurrentTimeTool{})
	if err != nil {
		t.Fatalf("NewToolRegistry returned error: %v", err)
	}

	run, err := runAgentLoop("mock", "请计算 7 * 6", registry, "", time.Second, 3)
	if err != nil {
		t.Fatalf("runAgentLoop returned error: %v", err)
	}
	if len(run.Steps) != 1 {
		t.Fatalf("len(steps) = %d, want 1", len(run.Steps))
	}
	if run.Steps[0].ToolCall.Name != "calculator" || run.Steps[0].ToolResult.Output != "42" {
		t.Fatalf("step = %+v, want calculator observation 42", run.Steps[0])
	}
	if !strings.Contains(run.FinalAnswer, "42") {
		t.Fatalf("final answer = %q, want it to include tool result", run.FinalAnswer)
	}
}

func TestRunAgentLoopMockDirectAnswerWithoutTool(t *testing.T) {
	registry, err := NewToolRegistry(CalculatorTool{}, CurrentTimeTool{})
	if err != nil {
		t.Fatalf("NewToolRegistry returned error: %v", err)
	}

	run, err := runAgentLoop("mock", "介绍一下 Function Calling", registry, "", time.Second, 3)
	if err != nil {
		t.Fatalf("runAgentLoop returned error: %v", err)
	}
	if len(run.Steps) != 0 {
		t.Fatalf("len(steps) = %d, want 0", len(run.Steps))
	}
	if run.FinalAnswer == "" {
		t.Fatal("expected final answer")
	}
}

func TestRunAgentLoopRejectsInvalidMaxSteps(t *testing.T) {
	registry, err := NewToolRegistry(CalculatorTool{})
	if err != nil {
		t.Fatalf("NewToolRegistry returned error: %v", err)
	}

	_, err = runAgentLoop("mock", "请计算 1 + 2", registry, "", time.Second, 0)
	if err == nil {
		t.Fatal("expected error for max-steps <= 0")
	}
}
