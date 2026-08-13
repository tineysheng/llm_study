# 03 Agent 设计说明

## 设计目标

阶段 3 的目标是把 LLM 应用从普通聊天扩展为可以调用工具的 Agent。

当前版本在 Tool Choice 实验基础上，增加了最小 Agent Loop：

```text
工具 schema + system prompt + 用户问题
-> 模型决定是否调用工具
-> tool call
-> ToolRegistry 分发
-> Go 执行工具
-> observation 回传给模型
-> 模型继续生成最终回答
```

`Tool` 接口和 `ToolRegistry` 仍然是实验支架。课程重点是观察 Agent Loop 如何把模型决策、工具执行、observation 和退出条件串起来。

## 非目标

当前版本暂不实现：

- 文件读取工具
- MCP Server
- 多轮规划和反思
- 工具失败后的自动重试策略

这些会在后续课程逐步加入。

## 当前组件

### Tool Schema

`ToolSchema` 描述工具名称、用途和参数结构。它的作用是让模型知道有哪些工具可以用，以及参数应该如何组织。

### Tool Interface

`Tool` 是 Go 侧的统一工具接口：

```go
type Tool interface {
	Schema() ToolSchema
	Execute(rawArguments string) (ToolResult, error)
}
```

每个工具必须同时提供 schema 和 execute，避免“模型看到的参数说明”和“Go 实际解析的参数”脱节。

### Tool Registry

`ToolRegistry` 保存工具名到工具实现的映射。它负责：

- 注册工具
- 拒绝重复工具名
- 输出所有工具 schema
- 根据 `ToolCall.Name` 分发到具体工具
- 拒绝未知工具

### Tool Call

`ToolCall` 表示模型请求调用工具。它包括：

- `ID`：真实 LLM tool call 的 ID，用于把 tool result 回传到对应 tool call
- `Name`：工具名
- `Arguments`：JSON 字符串参数

### Agent Loop

`runAgentLoop` 是当前核心入口。它负责根据 `-mode` 选择 mock 或 real 执行路径，并强制检查 `max-steps`。

当前循环语义：

1. 模型给出一次决策
2. 如果没有 tool call，直接返回最终回答
3. 如果有 tool call，校验工具名和 arguments
4. 执行工具并校验 tool result
5. 记录 `AgentStep`
6. 把 observation 交回模型继续生成
7. 如果工具调用次数超过 `max-steps`，停止并报错

### Agent Trace

`AgentRun` 保存最终回答和每个 `AgentStep`。每个 step 记录：

- 第几次工具调用
- 模型选择的工具名
- 模型生成的 arguments
- Go 执行工具后的 observation

这个 trace 是当前 demo 的最小可观测性能力，用于排查工具误选、参数错误和 observation 污染。

### Mock Model

`mockModelDecision` 用确定性规则模拟模型选择工具：

- 如果问题里包含简单数学表达式，就请求 `calculator`
- 如果问题询问当前时间，就请求 `current_time`
- 否则直接给出自然语言回答

### Real Model

`realModelDecision` 使用 OpenAI Chat Completions Tool Calling API，让真实 LLM 基于 `tools`、`tool_choice=auto`、`system prompt` 和用户问题决定是否返回 tool call。`runRealAgentLoop` 会在工具执行后追加 `tool` message，把 observation 回传给模型继续回答。

它用于观察真实模型行为：

- 是否调用工具
- 选择哪个工具
- 生成的 JSON arguments 是否符合 schema
- 不需要工具时是否能直接回答
- 工具执行失败时 Go 侧如何拦截错误

### Calculator Tool

`executeCalculator` 是当前唯一工具，支持：

- `add`
- `subtract`
- `multiply`
- `divide`

它会显式处理除零和未知操作。

### Current Time Tool

`CurrentTimeTool` 用于演示第二个工具，支持回答当前本地时间。它不需要参数，如果模型传入多余参数，会显式返回错误。

## 设计取舍

| 取舍 | 当前选择 | 原因 |
|---|---|---|
| 模型调用 | mock + real 两种模式 | mock 保证离线可跑；real 用来观察真实 LLM tool choice |
| 工具抽象 | `Tool` 接口 + `ToolRegistry` | 作为实验支架，统一收集 schema 并分发 LLM 返回的 tool call |
| 工具数量 | 2 个演示工具 | 用 `calculator` 展示带参数工具，用 `current_time` 展示无参数工具 |
| Agent Loop | 实现最小循环 | 先掌握 tool call、observation、最终回答和退出条件，不急着做复杂规划 |
| 最大循环次数 | `-max-steps` 默认 3 | 防止真实模型反复调用工具造成无限循环、成本失控和调试困难 |
| 执行日志 | `Agent Loop Trace` | 记录 tool call、arguments 和 observation，支持排查 Agent 行为 |
| 参数格式 | JSON 字符串 | 贴近真实 Function Calling 返回结构 |

## 安全考虑

即使是 mock model，也按真实工程习惯处理：

- 不信任模型返回参数
- 对 JSON 参数做解析和字段校验
- 对除零、未知操作返回显式错误
- 对未知工具名直接拒绝执行
- 对重复工具名在注册阶段报错
- 对工具结果名称和输出做校验，避免 observation 污染
- 用 `max-steps` 防止无限工具调用
- 后续涉及文件工具时必须增加目录白名单和路径限制

## 后续演进

1. 增加 ReAct 术语和 Thought / Action / Observation 解释
2. 增加安全边界，例如文件读取目录限制
3. 增加更多工具，例如受限文件读取、RAG 检索
4. 增加工具失败后的错误恢复策略
5. 增加更完整的 Agent 评测用例

## 变更历史

| 日期 | 变更 | 说明 |
|---|---|---|
| 2026-08-10 | 初始化阶段 3 Agent 模块 | 新增 mock Function Calling demo、calculator 工具、测试、README 和设计说明 |
| 2026-08-11 | 增加 Tool 抽象与多工具注册 | 新增 `Tool` 接口、`ToolRegistry`、`CurrentTimeTool` 和注册表测试 |
| 2026-08-11 | 调整为 LLM Tool Choice 实验课 | 新增 `-mode real`、`-show-schema`，让真实 LLM 根据 tool schema 决定工具调用 |
| 2026-08-13 | 增加 Agent Loop 基础 | 新增 `runAgentLoop`、`-max-steps`、observation 回传和 `Agent Loop Trace` |

