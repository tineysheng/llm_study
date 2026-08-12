# 阶段 3 学习记录：Function Calling 与 AI Agent

> 从原 `AI_TUTORING.md` 拆分而来。

## Function Calling 第一课记录：工具调用最小闭环（进行中）

### 本课知识点

本课进入阶段 3：Function Calling 与 AI Agent。目标是把模型从“只生成文本”升级为“可以请求应用程序调用外部工具”。

本课重点：

- `Tool / Function`：应用程序提供给模型使用的外部能力
- `Tool Schema / Function Schema`：工具说明书，描述工具名、用途、参数类型和必填字段
- `Tool Call`：模型返回的结构化工具调用请求，包括工具名和 JSON 参数
- `Tool Result / Observation`：Go 程序执行工具后的结果
- 模型不会直接执行工具，真正执行工具的是应用程序
- Go 侧必须解析参数、校验参数、处理工具错误

### 项目落地

已新增：

- `notes/09-function-calling-basics.md`：Function Calling 术语、输出解释和常见误区
- `03-agent/main.go`：mock Function Calling 最小闭环
- `03-agent/main_test.go`：计算表达式解析、参数解析、工具执行和错误处理测试
- `03-agent/README.md`：第三阶段 Agent 模块说明
- `03-agent/DESIGN.md`：当前设计目标、取舍和安全边界

当前 demo 注册了一个 `calculator` 工具：

```text
用户问题 -> mock model 判断是否需要工具 -> tool call -> Go 解析 JSON 参数 -> 执行 calculator -> 输出 final answer
```

### 运行方式

```powershell
go run .\03-agent
go run .\03-agent -question "请计算 12 + 30"
go run .\03-agent -question "你好，介绍一下 Function Calling"
go run .\03-agent -question "请计算 8 / 0"
go test .\03-agent
```

### 当前验证结果

AI 已验证：

- `go test .\03-agent` 通过
- 默认问题“请计算 12 + 30”会触发 `calculator` tool call，并得到 `42`
- 非计算问题不会调用工具，会直接回答 Function Calling 的说明
- “请计算 8 / 0”会触发工具错误：`除数不能为 0`
- 新模块安全扫描通过：Critical/High/Medium/Low 均为 0
- 新模块完整性扫描通过：`README.md`、`DESIGN.md` 和代码结构通过校验

### 下一步

进入本课问答。考核应重点覆盖：Tool Schema 的作用、模型和应用程序的职责边界、为什么 Go 侧必须做参数校验、Function Calling 和完整 Agent Loop 的区别。

## LLM Tool Choice 与工具抽象课程记录：进行中

### 触发原因

用户指出“下一课”不能只停留在文本讲解，必须继续把知识点实现进 Go 项目。因此本课从单个硬编码 `calculator` 工具推进到统一 `Tool` 接口和多工具注册表。

随后用户进一步指出：“为什么要 `Tool` 接口 / `ToolRegistry`”这类问题偏工程八股，对学习 LLM 价值不高。课程已调整为 **LLM Tool Choice 实验课**：用 mock 和真实 LLM 两种模式观察模型如何根据工具 schema、system prompt 和用户问题决定是否调用工具、调用哪个工具、生成什么 JSON arguments。

### 本课知识点

本课的工程支架是把工具调用从“单个工具硬编码分发”升级为“多个工具统一注册和按名称分发”：

- `Tool Interface`：每个工具都必须提供 `Schema()` 和 `Execute()`
- `ToolRegistry`：保存工具名到工具实现的映射
- `Dispatch`：根据模型返回的 `ToolCall.Name` 找到对应工具执行
- 具体工具自己负责 schema、参数解析、参数校验、执行逻辑和错误返回
- 注册表负责拒绝重复工具名和未知工具名

但当前学习重点不是背这些接口职责，而是观察 LLM 行为：

- `tool schema` 如何影响模型选择工具
- `description` 太模糊会不会导致误调用
- `parameters.required` 和 `enum` 如何影响 arguments 生成
- 不需要工具的问题，模型是否能直接回答
- 模型生成的 arguments 为什么仍然必须由 Go 侧校验

### 项目落地

已新增：

- `notes/10-tool-abstraction.md`：LLM Tool Choice 实验、Tool 抽象术语、运行方式、输出解释和常见误区

已修改：

- `03-agent/main.go`：新增 `Tool` 接口、`ToolRegistry`、`CalculatorTool`、`CurrentTimeTool`、`-mode real` 和 `-show-schema`
- `03-agent/main_test.go`：新增注册表分发、未知工具、重复工具、时间工具、参数校验和 OpenAI tool schema 转换测试
- `03-agent/README.md`：同步当前功能和运行方式
- `03-agent/DESIGN.md`：同步 Tool 抽象设计和安全考虑
- `PROGRESS.md`：标记 3.2 Tool 抽象代码任务完成

### 当前验证结果

AI 已验证：

```powershell
go test .\03-agent
go run .\03-agent -mode mock -question "请计算 7 * 6" -show-schema
go run .\03-agent -mode mock -question "现在几点了？" -show-schema
go run .\03-agent -mode mock -question "你好，介绍一下 Function Calling" -show-schema
go run .\03-agent -mode mock -question "请计算 8 / 0"
```

如设置了 `OPENAI_API_KEY`，可运行真实 LLM 模式：

```powershell
go run .\03-agent -mode real -question "请计算 7 * 6" -show-schema
go run .\03-agent -mode real -question "现在几点了？" -show-schema
go run .\03-agent -mode real -question "你觉得 RAG 和 Function Calling 有什么区别？" -show-schema
```

验证现象：

- `Registered Tools` 现在显示 `calculator` 和 `current_time`
- `-show-schema` 会打印发给模型的工具 schema
- 计算问题触发 `calculator`
- 当前时间问题触发 `current_time`
- 普通解释问题不会调用工具
- 除零问题由 Go 工具层返回 `除数不能为 0`

### 后续问答调整

不要再用“为什么真实项目不能一直写 if call.Name == ...”作为主问题。这个问题可以一句话带过。

下一题应该改为：

> 运行 `-show-schema` 后，你看到发给模型的工具信息有哪些？如果 `calculator` 的 description 写得很模糊，模型在什么情况下可能选错工具？

### 问答记录

#### Q1：`tool schema` 里哪些信息会影响 LLM 选择工具？如果 `description` 写得太宽会怎样？

用户回答：主要是 `name` 和 `description`。如果 `description` 范围过大，几乎覆盖所有问题，模型就可能把很多不该调用工具的问题也路由到这个工具。

结论：回答正确，抓住了 Tool Choice 最核心的风险。补充：除 `name` 和 `description` 外，`parameters.properties`、`required`、`enum` 也会影响模型如何生成 arguments。`description` 不应该写成广告词或泛泛能力介绍，而应该写成清晰的路由边界：什么时候该用、什么时候不该用。

#### Q2：为什么 `operation` 要加 `enum`？如果只写 string 会怎样？

用户回答：AI 可能会回复乱七八糟的东西。

结论：回答方向正确。`enum` 的作用是缩小模型输出空间，把 `operation` 限制在 `add/subtract/multiply/divide` 这些 Go 程序能执行的值里。如果不加 `enum`，模型可能生成 `plus`、`+`、`加法`、`times`、`乘`、`calculate` 等不稳定值，导致工具无法执行。补充：即使有 `enum`，Go 侧仍然不能完全信任模型，所以当前代码新增 `validateModelDecision()` 校验模型输出结构，再由 `parseCalculatorArgs()` 校验具体工具参数。

#### Q3：为什么参数名不能太抽象？

用户回答：大模型可能会不清楚这些 `a`、`b`、`op` 是干什么。

结论：回答正确。参数名也是 prompt 的一部分，短字段名对程序员省事，但对 LLM 不够自解释。当前代码已将 calculator 参数从 `a/b/op` 调整为 `left_operand/right_operand/operation`，并补充每个字段的 description，让模型更容易生成稳定、可执行的 arguments。

#### Q4：为什么 `left_operand/right_operand/operation` 要放进 `required`？

用户回答：`operation` 是关键字段，说明进行的是什么命令/操作；Go 程序应该直接报错返回。

结论：回答正确。`operation` 决定执行加减乘除哪一段逻辑，缺失时不能猜。补充：`left_operand` 和 `right_operand` 也同样必须 required，否则 Go 直接解析到普通结构体时，缺失的 number 字段会变成 `0`，可能导致错误计算。当前代码已强化 `parseCalculatorArgs()`：使用指针临时结构检测缺字段，并对旧字段名、空 operation、非法 operation 直接返回错误。

#### Q5：为什么 Tool Schema 不是安全边界？

用户回答：核心就是，必须严格控制 AI 使用工具的输入和输出；如果 AI 越过边界，直接报错返回。

结论：回答正确，可以作为本课核心面试表达。Tool Schema 是给模型看的软约束，不是应用安全边界。应用层必须检查工具名、arguments JSON、required 字段、operation 枚举、业务错误和工具结果。如果模型或工具结果越界，不要帮模型猜测或自动修复，应直接返回显式错误。当前代码已新增 `validateToolResult()`，在工具执行后校验工具结果名称和输出内容，形成“模型输出校验 -> 工具参数校验 -> 工具结果校验”的闭环。

#### Q6：如果真实 LLM 选错工具，应该优先排查哪里？

用户回答：`description` 要写紧一些；prompt 要明说“不需要工具时直接回答”；工具太多会扩大模型可用工具空间；用户问题可能难以命中工具；LLM 代码里面要兜底。

结论：回答完整。工具误选通常不是单点问题，而是 schema、prompt、工具数量、用户问题歧义和应用层兜底共同影响。推荐排查顺序：先收紧工具 description，再强化 system prompt，再控制候选工具数量，最后在 Go 侧用白名单、参数校验、业务边界和结果校验兜底。当前代码已收紧真实 LLM 的工具路由 prompt，并新增 `ToolRegistry.ValidateToolCall()`，提前拒绝模型选择未注册工具。

### 下一步

进入 LLM Tool Choice 实验讲解和问答。考核应重点覆盖：schema 如何影响模型选择工具、description 如何影响误调用、required/enum 如何约束 arguments、为什么 arguments 仍需 Go 校验、工具数量增加后如何降低模型选择错误率。

