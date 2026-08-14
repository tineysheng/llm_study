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

## Agent Loop 基础课程记录：已完成

### 触发原因

用户要求“开始下一课”。阶段 3.2 的 Tool Choice 代码任务和核心问答已完成，因此进入阶段 3.3：Agent Loop。

### 本课知识点

本课把前两课的单次工具调用升级为最小 Agent Loop：

```text
用户问题
-> 模型判断是否需要工具
-> 返回 tool call
-> Go 校验并执行工具
-> 生成 observation
-> 把 observation 回传给模型
-> 模型基于 observation 继续生成最终回答
```

重点知识点：

- `Agent Loop`：模型、工具执行、observation 和最终回答之间的循环控制流程
- `Observation`：工具执行结果进入模型上下文后的形式
- `max-steps`：应用层限制最大工具调用次数，防止无限循环
- `Agent Loop Trace`：记录每轮 tool call、arguments 和 observation，用于调试和面试表达
- 真实 LLM 模式下，tool result 需要带着 `tool_call_id` 回传给对应 tool call

### 项目落地

已新增：

- `notes/11-agent-loop.md`：Agent Loop 术语、运行方式、输出解释、常见误区和面试表达

已修改：

- `03-agent/main.go`：新增 `AgentRun`、`AgentStep`、`runAgentLoop`、`runMockAgentLoop`、`runRealAgentLoop`、`-max-steps` 和 `Agent Loop Trace`
- `03-agent/main_test.go`：新增 Agent Loop mock 场景、无需工具场景和 `max-steps` 校验测试
- `03-agent/README.md`：同步当前 Agent Loop 功能和运行方式
- `03-agent/DESIGN.md`：同步 Agent Loop、observation、trace 和安全边界设计
- `PROGRESS.md`：标记 3.3 Agent Loop 代码任务完成

### 当前验证结果

AI 已验证：

```powershell
go test .\03-agent
go run .\03-agent -mode mock -question "请计算 7 * 6" -show-schema
go run .\03-agent -mode mock -question "现在几点了？"
go run .\03-agent -mode mock -question "介绍一下 Function Calling"
```

验证现象：

- 计算问题触发 `calculator`，trace 中出现 arguments 和 `Observation: 42`
- 当前时间问题触发 `current_time`，trace 中出现当前本地时间 observation
- 不需要工具的问题不会调用工具，直接输出最终回答
- `go test .\03-agent` 通过
- 编辑器诊断 `03-agent/main.go` 和 `03-agent/main_test.go` 无错误

### 后续问答安排

进入本课问答。考核应重点覆盖：Agent Loop 和单次 Function Calling 的区别、observation 为什么要回传、`max-steps` 防什么生产风险、trace 如何帮助排查工具误选、真实 LLM 的 `tool_call_id` 为什么重要、模型反复调用工具时如何排查。

### 问答记录

#### Q1：Agent Loop 和单次 Function Calling 的最大区别是什么？为什么需要 `max-steps`？

用户回答：这要看 AI 的回答；如果 AI 想继续调用工具，就让它继续调用。加退出条件是为了防止死循环调用。

结论：回答方向正确。Agent Loop 的关键不是固定调用一次工具，而是让模型在收到 observation 后继续决定下一步：直接生成最终回答，或者在确实需要更多外部信息时继续调用工具。`max-steps` 是应用层硬边界，用来防止模型反复调用工具导致死循环。补充面试表达：除了死循环，`max-steps` 还控制 token 成本、API 调用成本、延迟、外部工具副作用和系统稳定性。

#### Q2：为什么工具执行后的结果要叫 `observation`，并且要回传给模型？

用户回答：不把结果给模型，模型不能继续；结果给模型后，可以判断工具调用次数是否过多并停止继续调用；observation 太长会有问题。

结论：回答方向正确，但需要补全工程细节。observation 回传的核心价值是让模型基于真实工具结果继续决策，而不是凭空猜测。模型拿到 observation 后可以：生成最终回答、解释工具错误、决定是否还需要调用其他工具、或者在达到应用层上限时停止。补充：是否达到 `max-steps` 主要由应用层强制控制，不应完全依赖模型自觉停止。observation 太长或太乱会带来 token 成本上升、上下文窗口占用、模型注意力被噪声分散、误读工具结果、泄露敏感信息和 prompt injection 风险。因此真实项目要对 observation 做结构化、裁剪、脱敏和必要字段保留。

#### Q3：为什么 `Agent Loop Trace` 对真实 Agent 项目很重要？

用户回答：没有日志就像线上程序没有日志一样，查不到 bug 出现的原因。trace 至少要记录模型选择了哪个工具、生成了什么 arguments、Go 执行后得到什么 observation、最终回答是什么。通过日志可以判断 Agent 为什么选择这个工具，以及 observability 是否有问题。

结论：回答完整。Agent 行为由模型、prompt、tool schema、用户问题、工具执行结果共同决定，如果没有 trace，只看到最终错误答案，很难判断问题出在工具选择、参数生成、工具执行、observation 回传还是最终回答阶段。最小 trace 应记录 tool name、arguments、observation、final answer；真实项目还应补充 request id、user question、step number、latency、token usage、error、model name、tool_call_id 和是否命中 `max-steps`。面试表达重点：Agent observability 的目标是把黑盒行为拆成可追踪的决策链路，帮助调试、评估、控成本和定位生产问题。

#### Q4：为什么真实 LLM 模式下 `tool_call_id` 很重要？

用户回答：`tool_call_id` 是工具的 id。多个 tool calls 说明调用了多个工具，需要 `tool_call_id` 才能找到对应的工具。observation 回传没有正确关联 `tool_call_id` 可能是因为 AI 使用错工具了。

结论：回答方向部分正确，但需要修正概念。`tool_call_id` 不是工具本身的 id，而是模型生成的“某一次工具调用请求”的 id，用来把 assistant 消息里的 tool call 和后续 role=tool 的 observation 对应起来。多个 tool calls 时尤其重要，因为每个工具调用都有自己的 arguments 和 observation，必须逐一匹配。若 observation 没有正确关联，可能导致模型把 calculator 的结果当成 current_time 的结果，或者把 A 工具的错误解释给 B 工具，造成上下文污染、错误最终回答，甚至被 API 拒绝。当前 demo 暂时只支持单次 tool call，但仍保留并校验 `tool_call_id`，是为了贴近真实 Tool Calling 协议。

#### Q5：如果真实 Agent 反复调用同一个工具，比如一直调用 `calculator`，你会怎么排查？

用户回答：先看 trace 的 Observation，看下返回给 AI 什么信息；可能是 system prompt 没定义清楚；可能是工具 description 写得不好；Go 要控制执行次数，并控制 Observation 输入给 AI。

结论：回答方向正确。排查反复调用工具时，优先看 trace：用户问题、每一步 tool name、arguments、observation、最终回答或是否触发 `max-steps`。如果 observation 已经足够回答但模型仍继续调用，通常要检查 system prompt 是否明确要求“收到足够 observation 后生成最终回答，不要重复调用工具”。如果模型一开始就误选工具，要检查 tool description 是否边界过宽、schema 是否让模型误以为该工具适用。Go 侧必须兜底：设置 `max-steps`、校验工具名和参数、限制 observation 长度和内容、必要时拒绝重复相同 tool call、记录错误并停止，避免死循环、成本失控和外部工具副作用。

#### Q6：为什么只设置 `max-steps` 还不够？真实 Agent 里还需要哪些退出条件或停止策略？

用户回答：除最大工具调用次数外，遇到异常应退出并报错返回；工具连续返回错误也应退出并报错返回；同一个工具用同样 arguments 连续调用说明 prompt 有问题；停止策略可以减少循环调用，提早暴露问题。

结论：回答方向正确。`max-steps` 是最后一道硬上限，但真实 Agent 还需要更细的停止策略：模型返回空内容或非法 tool call 时停止，工具参数校验失败时停止，工具连续错误时停止，重复相同 tool name + arguments 时停止，observation 已经足够回答却仍继续调用时停止，超过总耗时、token 预算或成本预算时停止。工具连续错误不应无限重试，最多做有限次数、可解释的重试；如果是参数错误、权限错误、安全边界错误，通常应直接停止。重复同一工具和同样 arguments 往往说明 prompt、tool description、observation 格式或模型状态出了问题。停止策略的价值是控制成本、降低延迟、避免外部工具副作用，并让问题更早暴露在 trace 和错误信息中。

#### Q7：为什么当前代码要先执行 `executeAndValidateTool()`，成功后才把 `AgentStep` 加进 trace？

用户回答：工具执行失败不要当作成功；必须记录所有记录，特别是失败记录，用于后续优化；失败 trace 字段设计还不清楚。

结论：回答方向正确。当前 demo 的 `AgentStep` 表示“成功产生 observation 的一步”，所以只有 `executeAndValidateTool()` 成功后才 append；失败不能伪装成成功 step，否则 trace 会误导排查。需要补充：当前 demo 的 trace 只记录成功 observation，不记录所有尝试；真实项目应该记录所有 attempt，尤其是失败。更完整的 trace 结构可以包含 `status`、`error`、`started_at`、`ended_at`、`latency_ms`、`tool_name`、`arguments`、`observation`、`retry_count`、`tool_call_id`、`model_name`、`token_usage` 等字段。这样既能保留成功链路，也能分析失败原因、重试策略和成本延迟。

#### Q8：如果面试官问“你这个 Agent Loop 是怎么设计的？如何保证它不会失控？”你会怎么回答？

用户回答：严格控制工具详情描述，明确什么情况下 AI 才能使用该工具；必须校验 AI 使用工具的参数；工具调用成功失败都应该记录日志；工具调用失败的信息可以返回给 AI，但必须控制 AI 使用工具的次数；记录日志，记录 AI 使用工具的详情情况；调用工具要有失败条件。

结论：回答通过，已经覆盖工具 schema 边界、参数校验、成功/失败日志、失败反馈和调用次数控制。补充面试表达：完整回答还应先讲整体流程——模型根据用户问题和 tool schema 决定是否调用工具，Go 校验工具名和 arguments，执行工具得到 observation，再把 observation 回传给模型生成最终回答。为了保证不失控，应用层要校验工具名、参数和工具结果，设置 `max-steps`、错误停止、重复调用停止、超时和成本预算；trace 要记录 tool name、arguments、observation、error、latency、token usage 和 final answer。这样可以控制 token/API 成本、降低延迟、防止死循环和外部工具副作用，并让线上问题可追踪。

### 本课通过结论

Agent Loop 基础课已通过。用户已经能够解释：Agent Loop 与单次 Function Calling 的区别、observation 回传价值、`max-steps` 的作用、trace 的排障价值、`tool_call_id` 的关联作用、反复调用工具的排查方式，以及真实 Agent 需要多种停止策略。后续进入阶段 3.4：ReAct 与安全边界，重点学习 Thought / Action / Observation 思想，以及文件读取类工具的白名单和路径限制。

## ReAct 与安全边界课程记录：进行中

### 触发原因

用户要求进入下一课。阶段 3.3 Agent Loop 已通过，因此进入阶段 3.4：ReAct 与安全边界。

### 本课知识点

本课把 Agent Loop 输出调整为更贴近 ReAct 的观察方式：

```text
Reasoning summary / Thought -> Action -> Action Input -> Observation -> Final Answer
```

学习重点：

- `ReAct`：Reasoning + Acting，强调模型决策和工具行动的结合
- `Action`：模型选择的工具
- `Action Input`：模型生成的 JSON arguments
- `Observation`：Go 工具执行后的结果
- 不暴露完整隐藏思维链，只记录可审计的工具行为
- 文件读取工具属于危险工具，必须限制目录、路径、扩展名和读取大小
- tool schema / prompt 是软约束，Go 侧安全校验才是硬边界

### 项目落地

已新增：

- `notes/12-react-security.md`：ReAct 术语、安全边界、运行观察、常见误区和面试表达
- `03-agent/safe-files/agent-safety.md`：受限文件读取示例文件

已修改：

- `03-agent/main.go`：新增 `SafeFileReaderTool`、`file_reader` schema、路径校验、symlink 逃逸防护、读取大小限制和 ReAct 风格 trace
- `03-agent/main_test.go`：新增安全文件读取、路径穿越拒绝、绝对路径拒绝、通配符拒绝、扩展名拒绝、symlink 逃逸拒绝和 mock tool choice 测试
- `03-agent/README.md`：同步 ReAct 与 file_reader 运行方式
- `03-agent/DESIGN.md`：同步威胁模型、安全决策和设计取舍
- `PROGRESS.md`：标记阶段 3.4 代码任务完成

### 当前验证结果

AI 已验证：

```powershell
go test .\03-agent
go run .\03-agent -mode mock -question "请读取 agent-safety.md"
go run .\03-agent -mode mock -question "请读取 ../agent-safety.md"
```

验证现象：

- `go test .\03-agent` 通过
- 正常读取 `agent-safety.md` 时，trace 输出 `Action: file_reader`、`Action Input` 和 `Observation`
- 路径穿越请求 `../agent-safety.md` 被 Go 侧拒绝：`relative_path 不能跳出安全目录`
- 编辑器诊断 `03-agent/main.go` 和 `03-agent/main_test.go` 无错误

### 后续问答安排

进入本课问答。考核应重点覆盖：ReAct 的 Thought / Action / Observation 含义、为什么不暴露完整隐藏思维链、为什么文件读取工具危险、为什么 tool schema 不是安全边界、如何防路径穿越、为什么限制 observation 长度，以及真实项目如何解释 Agent 安全边界。

### 课程节奏反馈与调整

用户反馈：最近几课不如第一阶段学习体验好，问题有时引入文档没充分解释的概念，考核偏向细碎安全工程追问，偏离 LLM / Agent 工程主线。

处理结论：反馈正确。阶段 3.4 从现在起改回第一阶段风格：

1. 先讲少量术语，再看代码和输出。
2. 每课只围绕 3-5 个核心知识点。
3. 考核只问文档明确解释过的概念。
4. 文件系统安全细节只作为扩展，不作为当前核心考核。
5. 面试表达聚焦 Agent 安全边界，而不是泛文件系统安全专项。

已更新 `notes/12-react-security.md`，新增“本课第一阶段风格版”“本课考核边界”，明确 `symlink`、`filepath.EvalSymlinks()`、`filepath.Rel()` 暂不作为核心考核。

### 问答记录

#### Q1：ReAct 里的 `Thought / Action / Observation` 分别是什么意思？为什么真实项目里不建议暴露完整隐藏 `Thought`？

用户回答：`Action` 是工具；`Action Input` 是输入给工具的数据；`Observation` 是工具输出；为什么不暴露完整 Thought 还不清楚。

结论：前三点回答正确。当前项目里 `Action` 对应模型选择的工具名，例如 `file_reader`；`Action Input` 对应模型生成的 JSON arguments，例如 `{"relative_path":"agent-safety.md"}`；`Observation` 对应 Go 工具执行后的结果，例如文件内容或错误信息。补充：真实项目不建议暴露完整隐藏 `Thought`，因为完整推理链可能包含不稳定推理、敏感信息、提示词细节、安全策略、误导性中间判断，甚至增加被攻击者利用的风险。工程上更推荐记录可审计的高层决策摘要和工具 trace，即 `Action / Action Input / Observation`，这样既能排查问题，又不会依赖或泄露模型完整隐藏思维链。

#### Q2：为什么 `file_reader` 这种工具比 `calculator` 更危险？

用户回答：`calculator` 最坏是不可用；读文件不限制路径可能会读取电脑重要东西；模型有时候会发疯，不能完全相信模型；否则模型就会随便使用工具。

结论：回答方向正确。`calculator` 的风险通常是算错、参数错误、除零、不可用或成本浪费，影响范围相对小；`file_reader` 一旦没有路径限制，可能读取 `.env`、API Key、SSH 私钥、源码、配置文件、用户隐私数据或系统文件，并把这些内容作为 observation 塞回模型上下文，造成敏感信息泄露。模型生成的路径不能直接相信，因为模型可能误解用户意图、被 prompt injection 诱导、生成路径穿越、绝对路径或通配符。白名单目录的作用是把工具能力收窄到明确允许的范围，让模型即使请求危险路径，Go 侧也会拒绝执行。

#### Q3：为什么说 `tool schema` 不是安全边界？当前代码里哪些函数在做真正的安全边界？

用户回答：因为访问路径还是有可能读取权限外的路径；如果真实 LLM 生成 `{"relative_path":"../go.mod"}`，程序应该报错返回；当前代码中哪些函数做安全边界还不知道。

结论：回答方向正确。`tool schema` 和 description 是写给模型看的软约束，只能降低模型生成危险参数的概率，不能保证模型一定遵守。真实 LLM 仍可能因为误解、幻觉或 prompt injection 生成 `../go.mod`、绝对路径或通配符。应用层必须把模型输出当作不可信输入，遇到越权路径要直接报错拒绝，不要帮模型猜测或自动修复。当前真正执行安全边界的代码主要是：`parseFileReaderArgs()` 校验 JSON 结构和必填字段；`cleanSafeRelativePath()` 拒绝空路径、空字符、通配符、绝对路径、盘符路径、`../` 和非 `.md` / `.txt`；`SafeFileReaderTool.safePath()` 通过安全根目录、`filepath.EvalSymlinks()` 和 `filepath.Rel()` 防止 symlink 逃逸；`SafeFileReaderTool.Execute()` 限制读取大小并拒绝非 UTF-8 文本；外层 `ToolRegistry.ValidateToolCall()` 和 `validateToolResult()` 分别校验工具名和工具结果。

#### Q4：为什么只检查 `../` 还不够防路径穿越？

用户回答：各种限制。

结论：方向正确但过于概括，需要补充具体安全点。只检查 `../` 不够，因为危险路径不只有这一种形式，还包括绝对路径、Windows 盘符路径、通配符、空字符、非允许扩展名、路径清理后的变体，以及 symlink 指向安全目录外文件。当前代码中 `cleanSafeRelativePath()` 负责拒绝明显危险路径，`SafeFileReaderTool.safePath()` 再用 `filepath.EvalSymlinks()` 和 `filepath.Rel()` 校验真实路径仍在安全根目录内。面试中不能只说“各种限制”，要能说明“限制什么、为什么限制、代码在哪里限制”。

用户反馈：后续追问里的 `symlink`、`filepath.EvalSymlinks()`、`filepath.Rel()` 在课程文档中没有充分解释，不应该直接考。

处理：反馈正确，该追问不计入考核。已补充 `notes/12-react-security.md` 的术语说明，新增 `symlink / 真实路径 / filepath.Rel()` 小节。后续问答必须先确保概念已在文档中解释清楚，不再突然引入未讲过的概念。

