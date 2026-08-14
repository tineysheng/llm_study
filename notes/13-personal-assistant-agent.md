# 13 个人助理 Agent 最小项目

## 本课第一阶段风格版

这一课不引入新复杂理论，只把前面学过的能力收束成一个可以演示的小项目。

一句话目标：

> 把 `calculator`、`current_time`、`file_reader` 三个工具组合成一个最小个人助理 Agent，并能用一条命令演示它如何选择工具、执行工具、读取 observation、生成最终回答。

## 本课只掌握 4 件事

1. **个人助理 Agent 是什么**：不是只聊天，而是能根据问题选择工具完成任务。
2. **工具列表怎么讲**：当前有计算、时间、文件读取 3 个工具。
3. **执行流程怎么看**：用户问题 -> 工具选择 -> Action Input -> Observation -> Final Answer。
4. **作品集怎么讲**：这是一个最小 Agent 项目，展示 tool calling、agent loop、trace 和安全边界。

## 当前工具列表

| 工具 | 解决什么问题 | 示例问题 |
|---|---|---|
| `calculator` | 精确计算 | `请计算 7 * 6` |
| `current_time` | 获取当前本地时间 | `现在几点了？` |
| `file_reader` | 读取安全目录里的学习资料 | `请读取 agent-safety.md` |

## 一条命令演示项目

```powershell
go run .\03-agent -demo
```

这个命令会依次跑 3 个场景：

```text
1. 精确计算：请计算 7 * 6
2. 当前时间：现在几点了？
3. 读取安全学习资料：请读取 agent-safety.md
```

## 输出怎么看

每个场景都看这一段：

```text
Trace:
Step 1 - ReAct Trace
Action: calculator
Action Input: {"left_operand":7,"right_operand":6,"operation":"multiply"}
Observation: 42
```

含义：

| 输出字段 | 含义 |
|---|---|
| `Action` | Agent 选择了哪个工具 |
| `Action Input` | Agent 给工具传了什么参数 |
| `Observation` | Go 程序执行工具后得到什么结果 |
| `Final Answer` | Agent 基于 observation 给用户的回答 |

## 为什么这算 Agent，不是普通 Chatbot

普通 Chatbot：

```text
用户问题 -> 模型直接生成文本
```

当前个人助理 Agent：

```text
用户问题 -> 模型/规则选择工具 -> Go 执行工具 -> observation -> 最终回答
```

区别是：Agent 能使用外部工具完成模型本身不擅长或不能直接完成的任务。

## 本课关键代码位置

| 代码 | 作用 |
|---|---|
| `main.go` | CLI 入口，只负责解析参数和启动 demo / 单次问题 |
| `demo.go` | 定义 3 个内置演示问题，并依次运行 |
| `agent_loop.go` | 执行工具选择、工具调用、observation 回传 |
| `registry.go` | 注册默认 3 个工具，并按工具名分发 |
| `model.go` | mock / real 模型决策 |
| `calculator_tool.go` | 计算器工具 |
| `time_tool.go` | 当前时间工具 |
| `file_reader_tool.go` | 受限文件读取工具 |

推荐阅读顺序：

```text
main.go -> demo.go -> agent_loop.go -> registry.go
```

这 4 个文件读懂后，就能讲清楚项目主流程。具体工具实现可以按需再看。

## 工程取舍

当前项目故意保持最小：

- 默认 `-demo` 使用 mock 模式，保证无需 API Key 也能稳定演示。
- 真实 LLM 模式仍保留在 `-mode real`，用于后续观察真实模型 tool choice。
- `file_reader` 只读取安全示例目录，避免把作品集 demo 变成任意文件读取工具。
- trace 先打印到终端，后续作品集阶段可以改成结构化日志。

## 面试表达

可以这样说：

> 我实现了一个最小个人助理 Agent，内置三个工具：calculator 用于精确计算，current_time 用于获取当前时间，file_reader 用于读取安全目录里的学习资料。用户输入问题后，Agent 会根据问题选择工具，生成 JSON arguments，由 Go 程序校验并执行工具，得到 observation 后再生成最终回答。为了方便调试和面试展示，我加了 trace，能看到每一步的 Action、Action Input 和 Observation。这个项目展示了 Function Calling、Tool Registry、Agent Loop、工具安全边界和可观测性。

## 本课考核边界

只问这些：

1. 为什么这个 demo 算 Agent，而不是普通 Chatbot
2. 三个工具分别解决什么问题
3. `-demo` 输出里的 Action / Action Input / Observation 怎么看
4. 为什么 mock demo 对作品集演示有价值
5. 面试时如何用 1 分钟讲清楚这个 Agent 项目


