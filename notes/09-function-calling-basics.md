# 09 Function Calling 基础：让模型调用工具

## 本课目标

前两阶段里，模型主要做两件事：

```text
聊天生成文本
基于 RAG 上下文生成答案
```

第三阶段开始，我们要把模型从“只会回答”升级成“能请求外部工具帮它完成任务”。

本课目标是理解最小 Function Calling 链路：

```text
用户问题 -> 模型判断是否需要工具 -> 返回 tool call -> Go 解析参数 -> 执行工具 -> 把工具结果交回模型 -> 最终回答
```

本课仍然使用 mock model，不调用真实 LLM API。原因是：先理解 Agent 工程结构，再接真实模型会更清楚。

## 本课必须先懂的术语

### 1. Tool / Function（工具 / 函数）

工具是应用程序提供给模型使用的能力。

例如：

- 计算器：做精确数学计算
- 天气查询：访问天气 API
- 文件读取：读取本地指定目录文件
- RAG 检索：查询知识库

模型本身不会真的执行代码。它只能“请求调用某个工具”。真正执行工具的是你的 Go 程序。

### 2. Tool Schema / Function Schema

Tool Schema 是工具说明书，用来告诉模型：

- 工具叫什么名字
- 工具能做什么
- 参数有哪些
- 参数类型是什么
- 哪些参数必填

示例：

```json
{
  "name": "calculator",
  "description": "执行基础数学运算",
  "parameters": {
    "type": "object",
    "properties": {
      "a": {"type": "number"},
      "b": {"type": "number"},
      "op": {"type": "string", "enum": ["add", "subtract", "multiply", "divide"]}
    },
    "required": ["a", "b", "op"]
  }
}
```

这和前面结构化输出很像：都是让模型输出机器可解析的数据。

### 3. Tool Call（工具调用请求）

Tool Call 是模型返回的“我要调用哪个工具、参数是什么”。

例如：

```json
{
  "name": "calculator",
  "arguments": {"a": 12, "b": 30, "op": "add"}
}
```

注意：模型返回 tool call 不等于工具已经执行。Go 程序还必须：

1. 检查工具名是否合法
2. 解析参数 JSON
3. 校验参数类型和取值
4. 执行对应工具
5. 处理工具错误

### 4. Tool Result / Observation（工具结果 / 观察）

工具执行后的结果叫 Tool Result，也常叫 Observation。

例如：

```text
calculator result: 42
```

真实 Agent Loop 里，这个结果会再次传给模型，让模型基于结果继续回答用户。

### 5. Agent 和普通 Chatbot 的区别

普通 Chatbot：

```text
用户问题 -> 模型直接回答
```

Function Calling / Agent：

```text
用户问题 -> 模型决定调用工具 -> 应用执行工具 -> 模型基于工具结果回答
```

关键区别：Agent 能把模型能力和外部工具连接起来，完成模型自己做不到或不适合做的事情。

## 为什么需要工具调用

LLM 不适合直接承担所有任务：

| 任务 | 直接让模型回答的问题 | 工具调用的价值 |
|---|---|---|
| 精确计算 | 模型可能算错 | 计算器工具保证确定性 |
| 查询实时天气 | 模型没有实时数据 | 天气 API 获取最新数据 |
| 查企业知识库 | 模型没有私有文档 | RAG 工具检索本地知识 |
| 读取文件 | 模型不能访问本地文件系统 | 文件工具在受限目录内读取 |

工具调用的价值不是让模型“更会聊天”，而是让模型能够安全、可控地使用外部能力。

## 本课代码

代码位置：

```text
03-agent\main.go
```

运行默认问题：

```powershell
go run .\03-agent
```

指定问题：

```powershell
go run .\03-agent -question "请计算 12 + 30"
go run .\03-agent -question "你好，介绍一下 Function Calling"
go run .\03-agent -question "请计算 8 / 0"
```

## 如何看懂输出

你会看到几段输出。

### 1. Registered Tools

显示当前注册了哪些工具。

本课只有一个工具：

```text
calculator
```

### 2. Model Decision

显示 mock model 是否决定调用工具。

如果问题是计算题，模型会返回：

```text
模型决定调用工具: calculator
arguments: {"a":12,"b":30,"op":"add"}
```

如果问题不需要工具，会直接回答。

### 3. Parsed Arguments

Go 程序把模型返回的 JSON 参数解析成结构体。

这一步很重要，因为模型输出是外部输入，不能直接信任。

### 4. Tool Result

Go 程序真正执行工具，例如计算：

```text
工具执行结果: 42
```

### 5. Final Answer

模型基于工具结果组织最终回答。

本课是 mock final answer，真实项目里会把 tool result 作为一条 tool message / observation 传回模型。

## 本课常见误区

### 误区 1：模型直接执行工具

不是。模型只返回 tool call。真正执行工具的是应用程序。

### 误区 2：有了 tool schema 就不需要参数校验

不是。Tool Schema 是给模型看的软约束，Go 侧仍然必须做硬校验。

### 误区 3：工具越多越好

不是。工具越多，模型选择错误的概率、参数错误、安全风险和调试成本都会增加。

### 误区 4：Function Calling 就等于完整 Agent

不是。Function Calling 是 Agent 的基础能力。完整 Agent 还需要多轮 loop、多工具注册、错误恢复、最大循环次数、日志、安全边界等。

## 当前重点

- Tool Schema 是工具说明书
- Tool Call 是模型请求调用工具，不是工具已经执行
- Go 程序负责解析参数、校验参数、执行工具和处理错误
- 工具调用让 LLM 能使用外部确定性能力
- Function Calling 是 Agent Loop 的基础，不等于完整 Agent

