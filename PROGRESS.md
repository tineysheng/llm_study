# 学习任务进度表

用法：完成任务后把 `[ ]` 改成 `[x]`，并在任务后补充完成日期、关键收获或遇到的问题。

## 阶段 1：LLM API 基础

目标：用 Go 掌握 LLM API 的最小工程闭环，包括消息历史、流式输出、参数控制、结构化输出和基本错误处理。

### 1.1 环境与基础运行

- [ ] 配置 `OPENAI_API_KEY`，确认本地可以访问模型 API
- [ ] 跑通 `go run ./01-llm-basics`
- [ ] 阅读 `notes/01-core-concepts.md`，理解 Token / Context Window / Temperature / Streaming
- [ ] 通读 `01-llm-basics/main.go`，能说出每个核心变量的作用

### 1.2 CLI 能力改造

- [x] 增加 `/help` 命令，展示可用指令
- [x] 增加 `/reset` 命令，清空对话历史但保留 system prompt
- [x] 增加 `/history` 命令，查看当前上下文消息数量和角色
- [x] 支持输入 `quit` 或 `/exit` 退出程序

### 1.3 参数与稳定性

- [x] 支持通过命令行参数或环境变量配置 model
- [x] 支持配置 temperature，观察 0 / 0.7 / 1.5 的输出差异
- [x] 支持配置 max tokens，理解输出长度和成本控制
- [x] 给 API 请求增加 timeout，避免网络卡死
- [x] 处理 scanner 错误和流式响应异常

### 1.4 结构化输出

- [x] 让模型按 JSON 格式返回一个固定结构
- [x] 用 Go `encoding/json` 解析模型输出
- [x] 处理 JSON 解析失败的情况，并记录失败原因
- [x] 总结结构化输出在业务系统里的用途和风险

### 1.5 阶段完成标准

- [x] 能解释“LLM API 为什么是无状态的”
- [x] 能解释多轮对话历史为什么会影响成本和延迟
- [x] 能解释 streaming 和普通响应的区别
- [x] 能在 AI 辅助下实现一个最小 LLM CLI，并能解释关键代码
- [x] 在笔记中写下面试版总结：`notes/01-core-concepts.md`

## 阶段 2：RAG 检索增强生成

目标：实现一个本地 Markdown 知识库问答系统，掌握 RAG 从文档到答案的完整链路。

### 2.1 Embedding 基础

- [x] 理解 Embedding 是“文本语义向量”，不是关键词匹配
- [x] 写 Go 代码计算两个向量的余弦相似度
- [ ] 调用 Embedding API，把两段文本转成向量（代码已实现，待 API 额度验证）
- [x] 比较相似问题和不相似问题的向量相似度（使用 mock 向量完成）

### 2.2 文档处理

- [x] 读取本地 Markdown 文档
- [x] 实现按标题或固定长度切分 chunk
- [x] 给每个 chunk 保存 source、title、content、index 等元数据
- [x] 对比不同 chunk 大小对检索效果的影响

### 2.3 向量存储

- [x] 先实现一个内存向量库，降低学习复杂度
- [x] 支持 Top-K 相似度检索
- [ ] 再接入 Qdrant 或 pgvector 作为真实向量数据库
- [x] 写入文档向量并能重新查询（内存版已完成）

### 2.4 RAG 问答链路

- [x] 实现“问题 → Embedding → Top-K 检索”（mock embedding 版）
- [x] 把检索结果拼进 Prompt
- [x] 让模型基于上下文回答问题（mock 生成版）
- [x] 在回答中输出引用来源
- [x] 当上下文没有答案时，要求模型明确说不知道

### 2.5 RAG 调优与面试准备

- 本课材料已新增：`notes/08-rag-tuning.md`、`02-rag/rag-tuning-demo/main.go`。已完成核心问答，后续作品集阶段可继续补流程图和更完整评测集。
- [x] 调整 chunk size，记录效果变化（mock 调优实验已完成）
- [x] 调整 Top-K，记录召回和噪声变化（mock 调优实验已完成）
- [x] 对比“无 RAG 直接问”和“有 RAG 问”的效果
- [x] 总结 RAG 常见问题：召回失败、上下文污染、幻觉、引用错误
- [ ] 能画出 RAG 全流程图并讲清楚每一步

## 阶段 3：Function Calling 与 AI Agent

目标：实现一个可以选择工具、执行工具、读取结果并继续回答的 Agent。

### 3.1 Function Calling 基础

- 本课材料已新增：`notes/09-function-calling-basics.md`、`03-agent/main.go`、`03-agent/main_test.go`、`03-agent/README.md`、`03-agent/DESIGN.md`。当前使用 mock model 学习工具调用闭环。
- [ ] 理解 tool schema / function schema 的作用
- [x] 注册第一个工具：计算器或假天气查询（已注册 `calculator`）
- [x] 让模型根据用户问题决定是否调用工具（mock model 已实现）
- [x] 解析模型返回的工具名和参数（已解析 tool call JSON arguments）

### 3.2 LLM Tool Choice 与工具抽象

- 本课材料已调整：`notes/10-tool-abstraction.md` 从“接口设计问答”改为“LLM Tool Choice 实验”。`03-agent/main.go` 已新增 `Tool` 接口、`ToolRegistry`、`CalculatorTool`、`CurrentTimeTool`，并支持 `-mode mock` / `-mode real` 对比离线规则和真实 LLM tool calling 行为。
- [x] 设计 Go `Tool` 接口
- [x] 为每个工具定义 name、description、parameters、execute
- [x] 实现参数校验
- [x] 实现工具执行错误的显式返回
- [x] 增加真实 LLM Tool Calling 模式，观察模型如何根据 schema 选择工具
- [x] 增加 `-show-schema`，打印发给模型的工具说明

### 3.3 Agent Loop

- 本课材料已新增：`notes/11-agent-loop.md`。`03-agent/main.go` 已新增 `runAgentLoop`、`runMockAgentLoop`、`runRealAgentLoop`、`-max-steps` 和 `Agent Loop Trace`，支持 observation 回传和最终回答生成。
- [x] 实现“模型请求工具 → 执行工具 → 结果回传 → 模型继续生成”的循环
- [x] 支持多个工具注册（沿用 `ToolRegistry` 注册 `calculator` 和 `current_time`）
- [x] 增加最大循环次数，避免无限调用
- [x] 增加日志，记录每次 tool call 和 observation

### 3.4 ReAct 与安全边界

- 本课材料已新增：`notes/12-react-security.md`。`03-agent/main.go` 已新增 `SafeFileReaderTool`、`file_reader` schema、安全目录 `03-agent/safe-files`、路径校验、扩展名限制、symlink 逃逸检测、读取大小限制和 ReAct 风格 trace。
- 用户反馈本课安全细节优先级不高，已决定跳过剩余细节考核。后续只保留核心结论：工具参数不可信，危险工具必须由 Go 侧做硬边界。
- [x] 理解 Thought / Action / Observation 的思想（本课使用可审计的 Action / Action Input / Observation，不暴露完整隐藏思维链）
- [x] 对危险工具增加白名单或路径限制
- [x] 对文件读取类工具做目录限制
- [x] 对工具参数做严格校验，避免模型传入危险参数

### 3.5 阶段项目

- 本课材料已新增：`notes/13-personal-assistant-agent.md`。`03-agent/main.go` 已新增 `-demo`、`demoScenarios()`、`runPersonalAssistantDemo()` 和 `newDefaultToolRegistry()`，一键演示计算、时间、文件读取 3 个工具。
- [x] 实现一个个人助理 Agent
- [x] 至少包含 3 个工具：计算器、本地文件读取、时间或天气查询
- [x] 写 README 说明工具列表、执行流程和安全限制
- [x] 能解释 Agent 和普通 Chatbot 的本质区别
- [x] 完成阶段 3.5 个人助理 Agent 最小项目验收与模拟面试

## 阶段 4：MCP 协议

目标：把工具能力协议化，理解 MCP 如何让 AI 客户端标准化接入外部工具。

### 4.1 MCP 基础

- [ ] 理解 Host / Client / Server 的角色
- [ ] 理解 tools、resources、prompts 的概念
- [ ] 对比 MCP 和 Function Calling 的区别

### 4.2 最小 MCP Server

- [ ] 用 Go 实现一个最小 MCP Server
- [ ] 暴露 1 个工具，例如 calculator
- [ ] 用 MCP Inspector 或兼容客户端验证工具可用
- [ ] 记录请求和响应结构

### 4.3 Agent 工具 MCP 化

- [ ] 把阶段 3 的一个工具改造成 MCP 工具
- [ ] 让外部客户端调用这个工具
- [ ] 总结 MCP 在工具复用、权限隔离、生态集成上的价值

## 阶段 5：求职作品集整理

目标：把学习成果整理成能写进简历、能面试讲清楚的作品。

### 5.1 项目包装

- [ ] 为 RAG 项目补齐 README：背景、功能、架构、运行方式、示例
- [ ] 为 Agent 项目补齐 README：工具列表、Agent Loop、错误处理、安全设计
- [ ] 添加架构图或流程图
- [ ] 准备演示数据和演示问题

### 5.2 简历表达

- [ ] 写一版 RAG 项目简历描述
- [ ] 写一版 Agent 项目简历描述
- [ ] 准备 5 个 RAG 高频面试问题
- [ ] 准备 5 个 Agent 高频面试问题
- [ ] 准备“我是 Go 后端，如何转 AI 应用开发”的自我介绍

## 贯穿全程的要求

- [ ] 每学完一个概念，都用自己的话写进 `notes/`
- [ ] 每完成一个功能，都能说出它解决了什么问题
- [ ] 每个知识点都要完成一次“讲解 → 项目实现 → 运行观察 → 问答考核”
- [ ] 只有能回答核心问题，才进入下一个知识点
- [ ] 每个阶段都保留可运行 demo
- [ ] 不提交 API Key、`.env`、编译产物和 IDE 私有配置
- [ ] 优先把阶段 2 和阶段 3 打磨成求职主项目
