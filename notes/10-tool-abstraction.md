# 10 LLM Tool Choice 实验：模型如何决定调用哪个工具

## 本课目标

> 课程调整记录：用户指出“只问为什么要 `Tool` 接口 / `ToolRegistry` 太像八股，对学习 LLM 价值不高”。因此本课重心从 Go 工程抽象改为 **LLM 如何根据工具 schema 做 tool choice**。

这一课不要求你背接口设计。真正要观察的是：

```text
工具 schema + 用户问题 + system prompt
-> LLM 判断是否需要工具
-> LLM 选择工具名
-> LLM 生成 JSON arguments
-> Go 执行工具并校验结果
```

你要学的是 LLM 行为，而不是泛 Go 后端设计。

当前 `03-agent` 支持两种模式：

| 模式 | 命令 | 学习价值 |
|---|---|---|
| mock | `-mode mock` | 离线稳定观察工具调用链路 |
| real | `-mode real` | 让真实 LLM 根据 tool schema 自己决定是否调工具 |

## 为什么这比“接口问答”更有价值

真实 Agent 的难点不是写一个 `map[string]Tool`。真正难的是控制模型行为：

- 工具描述写得模糊，模型会选错工具
- 参数 schema 不清楚，模型会生成错误 JSON
- 工具太多，模型选择空间变大，误调用概率上升
- 用户问题不需要工具时，模型也可能过度调用工具
- 工具返回错误后，Agent 要能解释、重试或停止
- 模型可能返回空内容、非法 JSON、错误工具名、多次 tool calls 等“乱七八糟”的输出

所以本课应该用代码观察：**schema 和 prompt 如何影响 LLM 的 tool choice**。

## 模型输出校验

真实 LLM 返回的内容不能直接执行。当前 `03-agent/main.go` 增加了 `validateModelDecision()`，在工具分发前先校验模型输出。

它会拦截：

- 没有 tool call，也没有文本内容
- 工具名为空
- arguments 为空
- arguments 不是合法 JSON

`realModelDecision()` 还会拒绝当前 demo 暂不支持的多次 tool calls。

这说明 Tool Calling 的可靠工程链路不是：

```text
模型说什么 -> 程序直接执行
```

而是：

```text
模型生成 tool call -> 应用校验结构 -> 注册表检查工具名 -> 工具校验参数 -> 执行工具 -> 校验工具结果
```

核心原则：必须严格控制 AI 使用工具的输入和输出；如果模型越过 schema、参数、工具名或结果边界，应用层应该直接报错返回，不要帮模型猜。

## 参数名也是 Prompt

工具参数名不是只给程序员看的字段名，也是 LLM 看到的上下文。

短参数名虽然省事，但会增加模型理解成本：

| 不推荐 | 问题 | 更推荐 |
|---|---|---|
| `a` | 不清楚是哪个数字 | `left_operand` |
| `b` | 不清楚是哪个数字 | `right_operand` |
| `op` | 不清楚是操作、选项还是别的缩写 | `operation` |

当前 calculator schema 已改为：

```json
{
  "left_operand": 7,
  "right_operand": 6,
  "operation": "multiply"
}
```

这样做的价值是：

- 降低模型生成错误字段名的概率
- 降低模型误解字段含义的概率
- 让 `-show-schema` 输出更接近可读的工具说明
- 让后续真实 LLM tool calling 更稳定

## Required 不是装饰，是缺字段防线

`required` 告诉模型：调用工具时必须生成哪些字段。

当前 calculator schema 要求：

```json
"required": ["left_operand", "right_operand", "operation"]
```

这三个字段缺一不可：

- 缺 `left_operand`：不知道左侧数字，不能计算
- 缺 `right_operand`：不知道右侧数字，不能计算
- 缺 `operation`：不知道执行加减乘除哪一个操作，必须直接报错

Go 里还要特别注意：如果直接把 JSON 解析进普通结构体，缺失的 number 字段会变成 `0`，缺失的 string 字段会变成空字符串。这可能让“模型漏字段”伪装成“合法输入”。

因此当前 `parseCalculatorArgs()` 使用指针临时结构解析参数：

```go
LeftOperand  *float64 `json:"left_operand"`
RightOperand *float64 `json:"right_operand"`
Operation    *string  `json:"operation"`
```

这样可以区分：

- 模型真的传了 `0`
- 模型根本没传这个字段

如果字段缺失，Go 程序会直接返回错误，而不是猜测默认值。

## 工具输出也要校验

工具执行完成后，也不能无脑进入最终回答。当前代码增加了 `validateToolResult()`，用于确认：

- 工具结果名称不能为空
- 工具结果名称必须和本次 tool call 名称一致
- 工具输出不能为空

这样可以避免后续 Agent Loop 中出现“模型请求 calculator，但应用拿 current_time 的结果继续推理”这类上下文污染问题。

## 模型选错工具时怎么调

如果真实 LLM 把“不需要工具的问题”错误路由到某个工具，例如用户问“解释 RAG 和 Function Calling 的区别”，模型却调用 `calculator`，优先按下面顺序排查：

1. **收紧工具 description**：工具描述不要覆盖所有问题，只描述明确适用边界。
2. **收紧 system prompt**：明确告诉模型“不需要工具时直接回答，不要返回 tool call”。
3. **减少候选工具数量**：工具越多，模型选择空间越大，误调用概率越高。
4. **处理问题歧义**：用户问题如果难以命中工具，应让模型直接回答或澄清，而不是强行调用工具。
5. **应用层兜底**：Go 代码必须拒绝未注册工具、非法参数和越界工具结果。

当前 `03-agent/main.go` 已增加 `ToolRegistry.ValidateToolCall()`：模型返回 tool call 后，会先检查工具名是否在注册表中；如果模型选择了不存在的工具，直接报错返回。

## 当前工具列表

真实 Agent 不会只有一个工具。当前项目先注册两个最小工具：

- calculator：精确计算
- current_time：获取当前时间

后续会继续加：

- file_reader：读取受限目录文件
- rag_search：检索知识库
- http_api：调用业务接口

`Tool` 接口和 `ToolRegistry` 只是为了让这些实验能扩展，不是本课的主要考点。

## 本课实验命令

### 1. 离线 mock 模式

```powershell
go run .\03-agent -mode mock -question "请计算 7 * 6" -show-schema
go run .\03-agent -mode mock -question "现在几点了？" -show-schema
go run .\03-agent -mode mock -question "你好，介绍一下 Function Calling" -show-schema
```

观察点：

- 哪些工具 schema 被提供给“模型”
- 哪些问题触发 `calculator`
- 哪些问题触发 `current_time`
- 哪些问题不应该调用工具

### 2. 真实 LLM Tool Calling 模式

先设置 API Key：

```powershell
$env:OPENAI_API_KEY = "sk-你的key"
```

再运行：

```powershell
go run .\03-agent -mode real -question "请计算 7 * 6" -show-schema
go run .\03-agent -mode real -question "现在几点了？" -show-schema
go run .\03-agent -mode real -question "你觉得 RAG 和 Function Calling 有什么区别？" -show-schema
```

观察点：

- 模型是否返回 `tool_calls`
- 模型选择的是哪个工具名
- `arguments` 是否是合法 JSON
- 不需要工具的问题，模型是否能忍住不调用工具
- Go 侧是否能执行或拒绝模型给的参数

### 3. 错误场景

```powershell
go run .\03-agent -mode mock -question "请计算 8 / 0"
go run .\03-agent -mode real -question "请计算 8 / 0"
```

观察点：

- 模型可能正确选择 `calculator`
- 但真正发现除零错误的是 Go 工具层
- 这说明 schema 是软约束，执行校验是硬约束

## 这一课真正要问的问题

不要问“为什么要接口”这种泛工程题。应该问：

1. LLM 是根据哪些输入决定是否调用工具的？
2. `description` 写得太宽泛会导致什么 tool choice 问题？
3. 为什么 `parameters.required` 和 `enum` 会影响模型生成 arguments？
4. 参数名为什么不能只考虑程序员习惯，还要考虑模型是否容易理解？
5. 为什么真实项目不能完全相信模型生成的 arguments？
6. 如果模型选错工具，你会先改 prompt、改 schema，还是改 Go 代码？为什么？
7. 工具数量增加后，LLM 选择错误率为什么可能上升？怎么缓解？
8. 如果模型返回空内容、非法 JSON 或多个 tool calls，应用层应该如何处理？

## Tool 抽象在本课里的位置

`Tool` 接口和 `ToolRegistry` 仍然有用，但它们只是实验支架：

```text
Tool 接口 -> 多个具体工具 -> ToolRegistry 注册表 -> 按 tool name 分发执行
```

你只需要理解到这个程度：

- `Schema()`：把工具说明暴露给 LLM
- `Execute()`：应用程序真正执行工具
- `ToolRegistry.Schemas()`：收集所有工具说明，放进 LLM 请求
- `ToolRegistry.Dispatch()`：根据 LLM 返回的工具名执行对应工具

更重要的是观察：**LLM 看到什么 schema，就会被什么 schema 影响。**

## 本课常见误区

### 误区 1：Tool Calling 的核心是 Go 接口设计

不是。Go 接口只是支架。核心是模型看到哪些工具说明、如何判断是否需要工具、如何生成可执行参数。

### 误区 2：schema 写了 enum，模型就一定不会错

不是。schema 会显著降低错误概率，但模型输出仍然是外部输入。Go 侧必须继续校验 JSON、字段、枚举值和执行边界。

当前代码里，`validateModelDecision()` 负责校验模型返回结构；`parseCalculatorArgs()` 负责校验 calculator 的具体参数。

### 误区 3：工具越多，Agent 越强

不一定。工具越多，模型选择空间越大，误调用和调试成本也越高。真实项目要控制工具数量，并把 description 写清楚。

### 误区 4：不需要工具的问题也应该走工具

不是。好的 tool choice 包括“决定不调用工具”。例如解释 RAG 和 Function Calling 的区别，当前两个工具都帮不上忙，模型应该直接回答。

## 当前重点

- LLM 的 tool choice 受 `system prompt`、`tool schema` 和用户问题共同影响
- `description` 是模型判断工具用途的关键文本
- 参数名也是 prompt，`left_operand/right_operand/operation` 比 `a/b/op` 更适合给 LLM 生成 arguments
- `parameters.required` 和 `enum` 会影响模型生成 arguments
- `required` 只是模型侧提示；Go 侧仍要检测缺字段，避免零值误执行
- Go 侧必须把模型生成的 tool call 当成不可信输入
- 程序要先校验模型输出结构，再执行具体工具，最后校验工具结果
- `Tool` / `ToolRegistry` 只是为了让 schema 收集和工具执行可扩展，不是本课主考点

