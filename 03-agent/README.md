# 03 Agent：Function Calling 与工具调用

本目录用于阶段 3：学习 Function Calling、Tool Calling、Agent Loop 和工具安全边界。

当前版本推进到第二课：**LLM Tool Choice 实验**。

本课不再把重点放在“为什么要接口”这类泛工程问题上，而是观察：LLM 如何根据 `tool schema`、`system prompt` 和用户问题决定是否调用工具、调用哪个工具、生成什么 JSON 参数。

## 当前功能

- 定义统一 `Tool` 接口
- 使用 `ToolRegistry` 注册和分发工具
- 注册 `calculator` 和 `current_time` 两个工具
- 用 mock model 判断用户问题是否需要工具
- 支持 `-mode real` 调用真实 LLM Tool Calling API
- 支持 `-show-schema` 打印发给模型的工具说明
- 生成 tool call：工具名 + JSON 参数
- 用 Go 解析和校验工具参数
- 执行工具并返回结果
- 基于工具结果生成最终回答

## 运行方式

离线 mock 模式：

```powershell
go run .\03-agent -mode mock -question "请计算 12 + 30" -show-schema
go run .\03-agent -mode mock -question "现在几点了？" -show-schema
go run .\03-agent -mode mock -question "你好，介绍一下 Function Calling" -show-schema
```

真实 LLM Tool Calling 模式：

```powershell
$env:OPENAI_API_KEY = "sk-你的key"
go run .\03-agent -mode real -question "请计算 12 + 30" -show-schema
go run .\03-agent -mode real -question "现在几点了？" -show-schema
go run .\03-agent -mode real -question "你觉得 RAG 和 Function Calling 有什么区别？" -show-schema
```

错误场景：

```powershell
go run .\03-agent -mode mock -question "请计算 8 / 0"
go run .\03-agent -mode real -question "请计算 8 / 0"
```

运行测试：

```powershell
go test .\03-agent
```

## 学习重点

- `tool schema` 本质上是写给模型看的上下文，会影响模型的工具选择和参数生成
- `description` 太模糊、参数约束太弱，会增加误调用和参数错误概率
- 不需要工具的问题，模型应该能直接回答，而不是为了调用而调用
- 模型不会直接执行工具，只会请求工具调用
- `tool call` 是模型返回的结构化调用请求
- Go 程序必须做参数解析、参数校验和错误处理
- Function Calling 是后续 Agent Loop 的基础

## 当前限制

- 默认使用 mock model，真实 LLM 模式需要 `OPENAI_API_KEY`
- 当前只有两个演示工具：`calculator`、`current_time`
- 当前只演示单次工具调用，还不是完整 Agent Loop
- 后续会继续加入 Agent Loop、最大循环次数、日志和安全边界

