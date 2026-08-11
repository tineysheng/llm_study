package main

import "testing"

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
	args, err := parseCalculatorArgs(`{"a":12,"b":30,"op":"add"}`)
	if err != nil {
		t.Fatalf("parseCalculatorArgs returned error: %v", err)
	}
	if args.A != 12 || args.B != 30 || args.Op != "add" {
		t.Fatalf("args = %+v", args)
	}
}

func TestParseCalculatorArgsRejectsUnknownOp(t *testing.T) {
	_, err := parseCalculatorArgs(`{"a":12,"b":30,"op":"power"}`)
	if err == nil {
		t.Fatal("expected error for unknown op")
	}
}

func TestExecuteCalculator(t *testing.T) {
	got, err := executeCalculator(CalculatorArgs{A: 9, B: 8, Op: "multiply"})
	if err != nil {
		t.Fatalf("executeCalculator returned error: %v", err)
	}
	if got != 72 {
		t.Fatalf("got %v, want 72", got)
	}
}

func TestExecuteCalculatorRejectsDivideByZero(t *testing.T) {
	_, err := executeCalculator(CalculatorArgs{A: 8, B: 0, Op: "divide"})
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
