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
