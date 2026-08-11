# 03 Agent：Function Calling 与工具调用

本目录用于阶段 3：学习 Function Calling、Tool Calling、Agent Loop 和工具安全边界。

当前版本是第一课：**Function Calling 基础**。

## 当前功能

- 注册一个 `calculator` 工具
- 用 mock model 判断用户问题是否需要工具
- 生成 tool call：工具名 + JSON 参数
- 用 Go 解析和校验工具参数
- 执行工具并返回结果
- 基于工具结果生成最终回答

## 运行方式

默认计算问题：

```powershell
go run .\03-agent
```

指定问题：

```powershell
go run .\03-agent -question "请计算 12 + 30"
go run .\03-agent -question "请计算 9 * 8"
go run .\03-agent -question "请计算 8 / 0"
go run .\03-agent -question "你好，介绍一下 Function Calling"
```

运行测试：

```powershell
go test .\03-agent
```

## 学习重点

- 模型不会直接执行工具，只会请求工具调用
- `tool schema` 是给模型看的工具说明书
- `tool call` 是模型返回的结构化调用请求
- Go 程序必须做参数解析、参数校验和错误处理
- Function Calling 是后续 Agent Loop 的基础

## 当前限制

- 当前使用 mock model，不调用真实 LLM API
- 当前只有一个工具：`calculator`
- 当前只演示单次工具调用，还不是完整 Agent Loop
- 后续会继续加入 Tool 接口、多工具注册、最大循环次数、日志和安全边界

