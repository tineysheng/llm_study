package main

import (
	"fmt"
	"time"
)

func demoScenarios() []DemoScenario {
	return []DemoScenario{
		{Title: "精确计算", Question: "请计算 7 * 6"},
		{Title: "当前时间", Question: "现在几点了？"},
		{Title: "读取安全学习资料", Question: "请读取 agent-safety.md"},
	}
}

func runPersonalAssistantDemo(registry *ToolRegistry, maxSteps int) error {
	fmt.Println("个人助理 Agent 最小项目演示")
	fmt.Println("模型模式: mock")
	fmt.Println("最大工具调用次数:", maxSteps)
	fmt.Println()

	fmt.Println("=== Demo Scenarios ===")
	for index, scenario := range demoScenarios() {
		fmt.Printf("%d. %s：%s\n", index+1, scenario.Title, scenario.Question)
	}
	fmt.Println()

	for index, scenario := range demoScenarios() {
		run, err := runAgentLoop("mock", scenario.Question, registry, "", time.Second, maxSteps)
		if err != nil {
			return fmt.Errorf("%s: %w", scenario.Title, err)
		}

		fmt.Printf("=== Scenario %d: %s ===\n", index+1, scenario.Title)
		fmt.Println("User Question:", scenario.Question)
		fmt.Println()
		fmt.Println("Trace:")
		printAgentTrace(run)
		fmt.Println("Final Answer:")
		fmt.Println(run.FinalAnswer)
		fmt.Println()
	}

	return nil
}
