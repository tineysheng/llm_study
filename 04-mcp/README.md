# 04-mcp: Go 原生 Model Context Protocol (MCP) Server

本项目使用纯 Go 标准库实现了一个符合官方 MCP 规范的 Server。支持通过 `stdio`（标准输入输出）与任何 MCP Client（如 Cursor、Claude Desktop 等）进行 JSON-RPC 2.0 协议通信。

## 功能特性

1. **协议规范对齐**：支持标准 MCP 2024-11-05 协议版本。
2. **核心能力实现**：
   - `initialize`：协商协议版本与能力声明。
   - `notifications/initialized`：客户端就绪通知。
   - `tools/list`：向 Host 暴露标准 JSON Schema 工具列表。
   - `tools/call`：执行工具调用并返回标准化 Content 结果。
3. **零第三方依赖**：仅使用 Go `encoding/json`、`bufio`、`os` 实现，极致轻量，单二进制交付。

## 运行与测试

### 1. 运行协议交互完整演示（时序查看）

```powershell
go run .\04-mcp -demo
```

### 2. 运行单元测试

```powershell
go test .\04-mcp
```

### 3. 配置到 Cursor 或 Claude Desktop

编译为可执行文件后，在 Cursor MCP 配置或 Claude Desktop 的 `claude_desktop_config.json` 中添加：

```json
{
  "mcpServers": {
    "go-tools": {
      "command": "C:\\path\\to\\04-mcp.exe"
    }
  }
}
```
