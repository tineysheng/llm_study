# 03 Agent 设计说明

## 设计目标

阶段 3 的目标是把 LLM 应用从普通聊天扩展为可以调用工具的 Agent。

第一课只实现最小 Function Calling 闭环：

```text
用户问题 -> 模型决定是否调用工具 -> tool call -> Go 执行工具 -> tool result -> 最终回答
```

## 非目标

当前版本暂不实现：

- 真实 LLM Function Calling API
- 多工具 Agent Loop
- 文件读取工具
- MCP Server
- 多轮规划和反思

这些会在后续课程逐步加入。

## 当前组件

### Tool Schema

`ToolSchema` 描述工具名称、用途和参数结构。它的作用是让模型知道有哪些工具可以用，以及参数应该如何组织。

### Tool Call

`ToolCall` 表示模型请求调用工具。它包括：

- `Name`：工具名
- `Arguments`：JSON 字符串参数

### Mock Model

`mockModelDecision` 用确定性规则模拟模型选择工具：

- 如果问题里包含简单数学表达式，就请求 `calculator`
- 否则直接给出自然语言回答

### Calculator Tool

`executeCalculator` 是当前唯一工具，支持：

- `add`
- `subtract`
- `multiply`
- `divide`

它会显式处理除零和未知操作。

## 设计取舍

| 取舍 | 当前选择 | 原因 |
|---|---|---|
| 模型调用 | mock model | 用户当前阶段重点是理解工具调用链路，不依赖 API 额度 |
| 工具数量 | 1 个 calculator | 先把工具 schema、参数解析、执行、错误处理讲清楚 |
| Agent Loop | 暂不实现循环 | 避免第一课知识点过多，后续单独学习最大轮数和错误恢复 |
| 参数格式 | JSON 字符串 | 贴近真实 Function Calling 返回结构 |

## 安全考虑

即使是 mock model，也按真实工程习惯处理：

- 不信任模型返回参数
- 对 JSON 参数做解析和字段校验
- 对除零、未知操作返回显式错误
- 后续涉及文件工具时必须增加目录白名单和路径限制

## 后续演进

1. 抽象 Go `Tool` 接口
2. 支持多个工具注册
3. 实现 Agent Loop：模型请求工具 -> 执行工具 -> 结果回传 -> 模型继续生成
4. 增加最大循环次数，避免无限调用
5. 增加日志，记录 tool call 和 observation
6. 增加安全边界，例如文件读取目录限制

## 变更历史

| 日期 | 变更 | 说明 |
|---|---|---|
| 2026-08-10 | 初始化阶段 3 Agent 模块 | 新增 mock Function Calling demo、calculator 工具、测试、README 和设计说明 |

