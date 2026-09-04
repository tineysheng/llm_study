# 阶段 4 学习记录：MCP 协议（Model Context Protocol）

## 4.1 MCP 核心概念与最小 JSON-RPC 2.0 Server

### 本课知识点

1. **MCP 是什么**：Anthropic 提出的开放标准协议，解决“每个 AI 框架各自写一套工具绑定代码”的生态孤岛问题。
2. **三层角色**：
   - **Host（宿主）**：Cursor / Claude Desktop / 自研 Agent 应用，负责发起会话和整合上下文。
   - **Client（协议客户端）**：集成在 Host 内部，负责与 Server 维持连接与 JSON-RPC 通信。
   - **Server（工具/资源服务端）**：独立进程或服务（如本课编写的 Go MCP Server），暴露 Tools/Resources/Prompts。
3. **底层通信机制**：
   - 基于标准 JSON-RPC 2.0 协议。
   - 常用传输层：stdio（标准输入输出，适合本地 CLI/IDE 工具）或 SSE / HTTP（适合远程微服务）。
4. **与阶段 3 普通 Function Calling 的对比**：
   - Function Calling：工具写死在当前 Go 进程内，无法跨语言、跨 IDE 复用。
   - MCP：工具独立为进程/微服务，Cursor、Claude Desktop 或任何 MCP Client 都能即插即用。

### 项目落地

- `notes/14-mcp-basics.md`：MCP 架构原理解析与面试对比。
- `04-mcp/`：Go 原生轻量实现的 MCP Server。
  - `types.go`：JSON-RPC 2.0 与 MCP 协议结构体（Initialize、ToolsList、ToolsCall）。
  - `server.go`：标准 stdio 协议处理引擎。
  - `main.go`：CLI 启动入口，支持 stdio 模式与 `-demo` 自动化协议交互演示模式。
  - `main_test.go`：自动化单元测试。
  - `README.md`：模块说明与 Cursor / Claude 接入指南。

### 运行验证

```powershell
go run .\04-mcp -demo
go test .\04-mcp
```

### 问答记录

#### Q1：MCP 和 Function Calling 最大区别是什么？

用户回答：工具单独做成服务，由 HTTP 传给模型；不单独做成 Server 就是本地调用；MCP 更偏分布式，不需要把所有东西放到同一个服务。

结论：回答方向正确，抓住了“工具能力从当前应用进程中拆出来、做成可复用服务”的核心。需要修正一点：MCP Server 通常不是“直接由 HTTP 传给模型”，而是 Host 内部的 MCP Client 通过 stdio / SSE / HTTP 等传输和 MCP Server 通信，再由 Host 把可用工具纳入模型上下文或工具调用流程。MCP 的价值是工具协议标准化、跨客户端复用、能力解耦和更清晰的部署边界。

#### Q2：MCP 里的 Host / Client / Server 分别是什么？

用户回答：Host 是服务地址；Client 是 MCP 服务代理；Server 是 MCP 服务。

结论：`Client` 和 `Server` 的方向基本正确，但 `Host` 需要纠正。Host 不是服务地址，而是承载 AI 会话的宿主应用，例如 Cursor、Claude Desktop 或自研 Agent App。Client 是 Host 内部负责 MCP 协议通信的组件，可以理解为协议代理/连接器。Server 是独立暴露工具、资源或 Prompt 的 MCP 服务端。推荐记忆：Host 是“用工具的 AI 应用”，Client 是“Host 里的协议连接器”，Server 是“提供工具的一侧”。

#### Q3：`04-mcp -demo` 的协议顺序是什么？为什么不能跳过 `tools/list` 直接调用工具？

用户回答：先 MCP 初始化连接；再发现工具列表；再调用工具；不能跳过工具发现，因为要先让 AI 知道有什么工具。

结论：回答通过。当前 demo 的核心顺序是 `initialize -> notifications/initialized -> tools/list -> tools/call`。其中 `tools/list` 的价值是让 Host/MCP Client 发现 Server 暴露了哪些工具、每个工具的 description 和 inputSchema 是什么。更准确地说，不是 MCP Server 直接“让 AI 知道”，而是 Host/Client 先拿到工具目录，再由 Host 决定如何把这些工具能力纳入模型上下文和工具调用流程。不能跳过工具发现，否则客户端不知道工具名、参数 schema 和能力边界，容易调用失败或无法让模型正确选择工具。

### 阶段 4 学习收束记录

用户反馈：MCP 不想继续深挖，知道它是做什么的即可。

处理决定：阶段 4 不再继续协议细节考核，不继续追问 JSON-RPC Notification、stdio、MCP Inspector、Cursor/Claude Desktop 配置等细节。当前只保留以下面试级核心认知：

1. MCP 是把工具能力协议化、服务化的一套标准。
2. Host 是使用工具的 AI 应用，例如 Cursor、Claude Desktop、自研 Agent。
3. MCP Client 是 Host 内部负责连接 MCP Server 的协议组件。
4. MCP Server 是提供工具、资源或 Prompt 的独立服务。
5. MCP 相比普通 Function Calling 的价值是工具复用、生态集成和能力解耦。

下一步转入阶段 5：求职作品集整理，优先把阶段 2 RAG 和阶段 3 Agent 项目打磨成简历与面试材料。

