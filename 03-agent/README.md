# 03 Agent：Function Calling 与工具调用

本目录用于阶段 3：学习 Function Calling、Tool Calling、Agent Loop 和工具安全边界。

当前版本推进到第三课：**Agent Loop 基础**。

本课重点是把单次工具调用升级为循环：模型请求工具，Go 执行工具，把 observation 回传给模型，模型继续生成最终回答。

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
- 把工具结果作为 observation 回传给模型或 mock follow-up
- 支持 `-max-steps` 限制最大工具调用次数
- 输出 `Agent Loop Trace`，记录每次 tool call、arguments 和 observation
- 基于工具 observation 生成最终回答

## 运行方式

离线 mock 模式：

```powershell
go run .\03-agent -mode mock -question "请计算 12 + 30" -show-schema
go run .\03-agent -mode mock -question "现在几点了？" -show-schema
go run .\03-agent -mode mock -question "你好，介绍一下 Function Calling" -show-schema
go run .\03-agent -mode mock -question "请计算 7 * 6" -max-steps 3
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
- Agent Loop 要把工具结果作为 observation 回传给模型继续推理
- `max-steps` 是防止无限工具调用的硬边界
- `Agent Loop Trace` 是排查工具误选、参数错误和上下文污染的最小可观测性

## 当前限制

- 默认使用 mock model，真实 LLM 模式需要 `OPENAI_API_KEY`
- 当前只有两个演示工具：`calculator`、`current_time`
- 当前 Agent Loop 仍是最小版本，暂不包含复杂规划、长期记忆和错误自动恢复
- 后续会继续加入 ReAct、安全边界、受限文件读取和更多工具

