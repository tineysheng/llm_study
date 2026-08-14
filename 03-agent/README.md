# 03 Agent：Function Calling 与工具调用

本目录用于阶段 3：学习 Function Calling、Tool Calling、Agent Loop 和工具安全边界。

当前版本推进到第五课：**个人助理 Agent 最小项目**。

本课重点是把 `calculator`、`current_time`、`file_reader` 三个工具组合成一个可演示、可讲述的最小个人助理 Agent。

## 当前功能

- 定义统一 `Tool` 接口
- 使用 `ToolRegistry` 注册和分发工具
- 注册 `calculator`、`current_time` 和 `file_reader` 三个工具
- 用 mock model 判断用户问题是否需要工具
- 支持 `-mode real` 调用真实 LLM Tool Calling API
- 支持 `-show-schema` 打印发给模型的工具说明
- 生成 tool call：工具名 + JSON 参数
- 用 Go 解析和校验工具参数
- 执行工具并返回结果
- 把工具结果作为 observation 回传给模型或 mock follow-up
- 支持 `-max-steps` 限制最大工具调用次数
- 输出 `Agent Loop Trace`，记录每次 tool call、arguments 和 observation
- 以 ReAct 风格输出 `Action`、`Action Input` 和 `Observation`
- `file_reader` 只能读取 `03-agent/safe-files` 中的 `.md` / `.txt` 文件
- `file_reader` 拒绝 `../`、绝对路径、Windows 盘符路径、通配符、非文本扩展名和 symlink 逃逸
- 基于工具 observation 生成最终回答
- 支持 `-demo` 一次演示计算、时间、文件读取 3 个场景

## 运行方式

个人助理 Agent 一键演示：

```powershell
go run .\03-agent -demo
```

离线 mock 模式：

```powershell
go run .\03-agent -mode mock -question "请计算 12 + 30" -show-schema
go run .\03-agent -mode mock -question "现在几点了？" -show-schema
go run .\03-agent -mode mock -question "你好，介绍一下 Function Calling" -show-schema
go run .\03-agent -mode mock -question "请计算 7 * 6" -max-steps 3
go run .\03-agent -mode mock -question "请读取 agent-safety.md"
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
go run .\03-agent -mode mock -question "请读取 ../agent-safety.md"
go run .\03-agent -mode real -question "请计算 8 / 0"
```

运行测试：

```powershell
go test .\03-agent
```

## 代码阅读顺序

为了避免所有逻辑堆在 `main.go`，当前代码已按职责拆分：

| 文件 | 先看程度 | 作用 |
|---|---|---|
| `main.go` | 必看 | CLI 入口，解析参数，启动单次问题或 `-demo` |
| `demo.go` | 必看 | 个人助理 Agent 的 3 个内置演示场景 |
| `agent_loop.go` | 必看 | Agent Loop：模型决策、工具执行、trace 输出 |
| `registry.go` | 建议看 | 默认工具注册、schema 收集、工具分发 |
| `model.go` | 建议看 | mock / real 模型决策和 tool choice prompt |
| `calculator_tool.go` | 按需看 | 计算器工具 |
| `time_tool.go` | 按需看 | 当前时间工具 |
| `file_reader_tool.go` | 按需看 | 受限文件读取工具和安全校验 |
| `types.go` | 查类型时看 | 核心结构体和接口 |
| `validation.go` | 查校验时看 | 模型输出和工具结果校验 |
| `config.go` | 查配置时看 | 环境变量读取 |

初学时建议只按这个顺序读：`main.go` -> `demo.go` -> `agent_loop.go` -> `registry.go`。

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
- ReAct 的工程可观测性重点是 `Action`、`Action Input`、`Observation`，不是暴露完整隐藏思维链
- 文件读取工具属于危险工具，必须由 Go 侧实现目录白名单和路径限制
- Tool schema 是软约束，安全边界必须由应用层强制执行
- 最小个人助理 Agent 的作品集价值在于：能展示工具选择、工具执行、Agent Loop、trace 和安全边界

## 当前限制

- 默认使用 mock model，真实 LLM 模式需要 `OPENAI_API_KEY`
- 当前有三个演示工具：`calculator`、`current_time`、`file_reader`
- 当前 Agent Loop 仍是最小版本，暂不包含复杂规划、长期记忆和错误自动恢复
- 当前 `file_reader` 只用于读取安全示例目录，不支持任意文件读取
- 当前 `-demo` 使用 mock 模式，保证无需 API Key 也能稳定演示
- 后续会在作品集阶段补更完整的项目 README、架构图和面试材料

