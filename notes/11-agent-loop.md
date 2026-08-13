# 11 Agent Loop：工具调用后的继续推理

## 本课目标

前两课已经完成：

```text
用户问题 -> 模型选择工具 -> Go 执行工具 -> 得到工具结果
```

这一课把它升级成真正的最小 Agent Loop：

```text
用户问题
-> 模型判断是否需要工具
-> 返回 tool call
-> Go 校验并执行工具
-> 生成 observation
-> 把 observation 回传给模型
-> 模型基于 observation 继续生成最终回答
```

核心变化：工具结果不再只是 Go 程序自己拼一句答案，而是作为新的上下文交回模型，让模型继续推理。

## 必须先懂的术语

### 1. Agent Loop

Agent Loop 是一个循环控制流程。每一轮都让模型决定下一步：

- 直接回答用户
- 调用某个工具
- 读完工具结果后继续回答或继续调用工具

最小版本不需要复杂规划，只要具备“模型请求工具 -> 应用执行 -> observation 回传 -> 模型继续”的闭环。

### 2. Observation

Observation 是工具执行结果进入模型上下文后的名字。

例如：

```text
Tool Call: calculator {"left_operand":7,"right_operand":6,"operation":"multiply"}
Observation: 42
```

模型下一步应该基于 `42` 回答，而不是重新猜答案。

### 3. 最大循环次数 max-steps

Agent Loop 必须有最大工具调用次数。

原因：真实 LLM 可能因为 prompt、schema、工具错误或上下文污染，反复调用同一个工具，形成无限循环。

当前 demo 增加：

```powershell
go run .\03-agent -mode mock -question "请计算 7 * 6" -max-steps 3
```

`max-steps` 限制的是“最多允许几次工具调用”，不是最多几条消息。

### 4. Trace / 日志

Agent 系统必须记录每次：

- 模型选择了哪个工具
- 生成了什么 arguments
- Go 执行后得到什么 observation
- 最终回答是什么

没有 trace，就很难排查：模型为什么选错工具、参数为什么错、工具结果是否污染了上下文。

## 本课项目落地

本课修改了 `03-agent/main.go`：

- 新增 `AgentRun` 和 `AgentStep`，记录一次 Agent Loop 的最终回答和每轮工具调用
- 新增 `runAgentLoop()`，统一运行 mock / real 模式
- 新增 `runMockAgentLoop()`，离线模拟“工具结果回传后再回答”
- 新增 `runRealAgentLoop()`，真实 LLM 模式下把 tool result 作为 `tool` message 回传给模型
- 新增 `-max-steps`，避免无限工具调用
- 输出 `Agent Loop Trace`，展示 tool call 和 observation

## 运行方式

### 1. 计算工具

```powershell
go run .\03-agent -mode mock -question "请计算 7 * 6" -show-schema
```

观察重点：

```text
Step 1 - Model Tool Call
tool: calculator
arguments: {"left_operand":7,"right_operand":6,"operation":"multiply"}
Observation: 42
```

这说明：模型先请求 `calculator`，Go 执行后得到 `42`，最终回答基于 observation 生成。

### 2. 时间工具

```powershell
go run .\03-agent -mode mock -question "现在几点了？"
```

观察重点：

- 工具名应为 `current_time`
- arguments 应为 `{}`
- observation 是当前本地时间

### 3. 不需要工具的问题

```powershell
go run .\03-agent -mode mock -question "介绍一下 Function Calling"
```

观察重点：

```text
模型决定：不需要工具，直接回答。
```

好的 Agent Loop 不等于每次都调用工具。能判断“不调用工具”也是能力。

### 4. 测试

```powershell
go test .\03-agent
```

## 输出怎么看

当前输出分四块：

| 输出块 | 含义 |
|---|---|
| `Registered Tools` | 当前提供给模型选择的工具列表 |
| `Tool Schemas Sent To Model` | `-show-schema` 时打印，观察模型看到的工具说明 |
| `Agent Loop Trace` | 每轮 tool call、arguments 和 observation |
| `Final Answer` | 模型或 mock follow-up 基于上下文生成的最终回答 |

面试时你要能解释：trace 不是为了好看，而是 Agent 可观测性的最小形态。

## 工程取舍

### 为什么先做最小 loop，而不是直接做复杂 Agent？

因为 Agent 的核心风险不是“代码能不能循环”，而是：

- 模型是否会误调用工具
- 工具参数是否可信
- observation 是否正确回传
- 工具失败后是否停止或恢复
- 是否能防止无限循环

先把这些边界跑通，再加入规划、记忆、文件工具和 RAG 工具更稳。

### 为什么 `max-steps` 必须存在？

真实 Agent 是非确定性的。即使 prompt 写得很好，模型仍可能：

- 反复调用同一工具
- 工具失败后继续重试
- 收到 observation 后没有生成最终回答
- 在多个工具之间来回切换

所以应用层必须设置硬限制。达到上限后应停止，并返回可观测错误。

### 为什么 observation 要回传给模型？

如果不回传，模型无法基于真实工具结果继续推理，只能由 Go 程序硬拼答案。

真实 Agent 的价值在于：模型可以读取工具结果，再决定最终表达、下一步工具调用或错误解释。

## 常见误区

### 误区 1：Agent Loop 就是 while 循环

不只是 while。关键是每轮都要有：模型决策、工具校验、工具执行、observation、退出条件和日志。

### 误区 2：工具调用成功就等于 Agent 成功

不一定。工具成功只是中间步骤。最终还要看模型是否正确理解 observation，并给出符合用户问题的答案。

### 误区 3：让模型自己决定什么时候停就够了

不够。模型可以建议停，但应用层必须有 `max-steps` 这类硬边界。

### 误区 4：mock loop 没价值

mock loop 的价值是稳定观察工程链路。真实 LLM 模式用于观察模型行为，mock 模式用于保证本地测试和学习可重复。

## 面试表达

可以这样说：

> 我实现的 Agent Loop 不是简单调用一次工具，而是让模型先返回 tool call，应用层校验工具名和 JSON 参数后执行工具，再把 tool result 作为 observation 回传给模型，让模型继续生成最终回答。为了防止无限循环，我在应用层增加了 max-steps；为了排查工具误选和参数错误，我记录每轮 tool call、arguments 和 observation。这个设计体现了 Agent 工程里的三个重点：模型输出不可信、工具执行要有边界、Agent 行为必须可观测。

## 本课考核重点

本课问答应覆盖：

1. Agent Loop 和单次 Function Calling 的区别
2. observation 为什么要回传给模型
3. `max-steps` 解决什么生产风险
4. Trace / 日志在 Agent 调试中的价值
5. 工具结果为什么仍然要校验
6. 不需要工具时为什么应该直接回答
7. 真实 LLM 模式下 tool call id 为什么重要
8. 如果模型反复调用工具，应该如何排查

