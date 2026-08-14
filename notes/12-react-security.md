# 12 ReAct 与 Agent 安全边界

## 本课第一阶段风格版

> 课程调整：本课不再深挖文件系统安全细节，而是回到第一阶段那种“小知识点 + 可运行 demo + 面试表达”的节奏。

这一课只要求掌握 4 件事：

1. **ReAct 是什么**：模型先决定动作，再看工具结果继续回答。
2. **Trace 怎么看**：重点看 `Action`、`Action Input`、`Observation`。
3. **文件工具为什么危险**：它可能读取敏感文件，并把内容塞进模型上下文。
4. **安全边界在哪里**：schema / prompt 只是提示，真正拦截危险参数的是 Go 代码。

本课不要求你背文件系统安全细节。下面这些只作为扩展理解，不作为当前核心考核：

- symlink 逃逸
- `filepath.EvalSymlinks()`
- `filepath.Rel()`
- Windows / Linux 各种路径绕过细节

## 一句话目标

给 Agent 增加一个受限 `file_reader` 工具，观察 ReAct 的 `Action -> Observation`，并理解为什么危险工具必须由 Go 程序做硬校验。

## 本课必会术语

| 术语 | 本课含义 | 当前项目里的例子 |
|---|---|---|
| `Action` | 模型选择的工具 | `file_reader` |
| `Action Input` | 模型生成的工具参数 | `{"relative_path":"agent-safety.md"}` |
| `Observation` | Go 工具执行后的结果 | 文件内容或错误信息 |
| 安全边界 | 应用程序强制规定工具能做什么 | 只允许读取 `03-agent/safe-files` |

## 本课只看两条运行输出

### 1. 合法读取

```powershell
go run .\03-agent -mode mock -question "请读取 agent-safety.md"
```

重点看：

```text
Action: file_reader
Action Input: {"relative_path":"agent-safety.md"}
Observation: # Agent Safety Sample ...
```

这说明模型选择了 `file_reader`，Go 程序读取了安全目录里的文件，并把结果作为 observation 返回。

### 2. 危险路径被拒绝

```powershell
go run .\03-agent -mode mock -question "请读取 ../agent-safety.md"
```

重点看：

```text
Agent Loop 执行失败: 工具调用失败: relative_path 不能跳出安全目录
```

这说明：即使模型提出危险路径，Go 程序也不会执行。

## 本课面试表达

可以这样说：

> 我在 Agent Loop 里加入了 ReAct 风格的 trace，用 `Action` 表示模型选择的工具，用 `Action Input` 表示模型生成的参数，用 `Observation` 表示 Go 工具执行后的结果。为了演示安全边界，我新增了一个受限 `file_reader` 工具。这个工具不会信任模型生成的路径，只允许读取白名单目录 `03-agent/safe-files` 下的 `.md` / `.txt` 文件。如果模型传入 `../go.mod` 这类危险路径，Go 程序会直接拒绝。核心原则是：tool schema 和 prompt 只是软约束，真正的安全边界必须由应用层强制执行。

## 本课考核边界

本课只考这些：

1. `Action` / `Action Input` / `Observation` 分别是什么
2. 为什么 `file_reader` 比 `calculator` 更危险
3. 为什么 tool schema 不是安全边界
4. 如果模型传入 `../secret.md`，Go 程序应该怎么处理
5. 为什么 observation 不能无限长、不能包含敏感信息

本课暂不考：

- `symlink` 的完整攻击方式
- `filepath.EvalSymlinks()` 的实现细节
- `filepath.Rel()` 的边界情况
- 跨平台路径安全专项知识

## 本课目标

这一课进入阶段 3.4：ReAct 与安全边界。

前一课已经实现最小 Agent Loop：

```text
模型 -> tool call -> Go 执行工具 -> observation -> 模型继续回答
```

本课继续往真实 Agent 工程靠近：

```text
Reasoning summary / Thought -> Action -> Observation -> Final Answer
```

同时新增一个受限文件读取工具 `file_reader`，用它学习为什么危险工具必须有白名单、路径限制和参数校验。

## 术语表

### 1. ReAct

ReAct 是 Reasoning + Acting 的缩写，常用来描述 Agent 的工作模式：

- 模型先判断下一步需要做什么
- 如果需要外部能力，就选择一个 Action，也就是工具调用
- 工具执行后得到 Observation
- 模型基于 Observation 继续判断或生成最终回答

在工程实现里，我们不需要也不应该记录模型完整隐藏思维链。更推荐记录可审计的高层决策摘要、Action、Action Input 和 Observation。

### 2. Thought / Reasoning Summary

Thought 表示模型做下一步决策前的思考。

真实产品里要注意：

- 不要依赖或暴露模型完整隐藏推理过程
- 可以记录高层 decision summary，例如“需要读取安全示例文件”
- 面试里重点讲清楚模型为什么需要工具，而不是输出完整 chain-of-thought

### 3. Action

Action 是模型决定调用的工具。

当前 demo 中 Action 可以是：

- `calculator`
- `current_time`
- `file_reader`

### 4. Action Input

Action Input 是工具参数，也就是模型生成的 JSON arguments。

例如：

```json
{"relative_path":"agent-safety.md"}
```

Action Input 必须由 Go 程序校验，不能因为 schema 写了限制就直接信任模型。

### 5. Observation

Observation 是工具执行结果。

例如 `file_reader` 读取安全文件后，Observation 是文件内容。这个内容会进入模型上下文，因此必须控制长度、格式和敏感信息。

### 6. 安全边界

安全边界是应用程序明确规定“模型能做什么、不能做什么”的硬限制。

本课的边界：

- `file_reader` 只能读取 `03-agent/safe-files` 目录
- 只能读取相对路径
- 拒绝 `../`
- 拒绝绝对路径和 Windows 盘符路径
- 拒绝通配符
- 只允许 `.md` 和 `.txt`
- 限制最大读取字节数
- 拒绝非 UTF-8 文本
- 校验 symlink 不能逃逸出安全目录

### 7. symlink / 真实路径 / `filepath.Rel()`

这几个概念是文件读取安全里容易被忽略的点。

#### symlink 是什么

symlink 可以理解成“快捷方式”或“软链接”。

例如安全目录里看起来有一个文件：

```text
03-agent/safe-files/link.md
```

但它可能实际指向安全目录外面的文件：

```text
C:\Users\Administrator\.ssh\id_rsa
```

如果程序只检查字符串 `03-agent/safe-files/link.md`，会误以为它在安全目录内；但真正打开文件时，读到的可能是外部敏感文件。

#### 为什么要用 `filepath.EvalSymlinks()`

`filepath.EvalSymlinks()` 的作用是解析 symlink，拿到文件的真实路径。

也就是说，它可以把：

```text
03-agent/safe-files/link.md
```

解析成真正指向的路径，例如：

```text
C:\Users\Administrator\.ssh\id_rsa
```

这样程序才能判断：这个文件虽然“看起来”在安全目录里，但真实位置已经逃出去了。

#### 为什么要用 `filepath.Rel()`

`filepath.Rel(root, target)` 用来计算目标文件相对于安全目录的相对路径。

如果结果以 `..` 开头，说明目标文件不在安全目录内：

```text
..\..\.ssh\id_rsa
```

当前代码就是用这个判断来拒绝 symlink 逃逸。

一句话总结：

```text
cleanSafeRelativePath() 检查模型给的路径字符串是否危险；
safePath() 通过 EvalSymlinks + Rel 检查真实文件是否仍在安全目录内。
```

## 为什么文件读取工具危险

文件读取工具比 calculator 更危险，因为模型可能生成：

```text
../go.mod
C:\Users\Administrator\.ssh\id_rsa
../../.env
*
```

如果 Go 程序直接按模型参数读取文件，就可能造成：

- 路径穿越
- 敏感信息泄露
- 把密钥塞进模型上下文
- 读取系统文件
- 读取超大文件导致成本和上下文失控

因此工具 schema 只是软约束，真正的安全边界必须在 Go 侧实现。

## 本课项目落地

本课修改了 `03-agent/main.go`：

- 新增 `SafeFileReaderTool`
- 新增 `file_reader` tool schema
- 新增 `parseFileReaderArgs()` 校验 JSON 参数
- 新增 `cleanSafeRelativePath()` 校验路径
- 新增 `safePath()` 校验目标文件必须在安全目录内
- 新增 `defaultSafeFilesDir()` 自动定位安全示例目录
- mock model 支持“请读取 agent-safety.md”触发 `file_reader`
- trace 输出改成更接近 ReAct：`Action`、`Action Input`、`Observation`

本课新增示例文件：

- `03-agent/safe-files/agent-safety.md`

## 运行方式

### 1. 正常读取安全文件

```powershell
go run .\03-agent -mode mock -question "请读取 agent-safety.md"
```

观察点：

```text
Step 1 - ReAct Trace
Action: file_reader
Action Input: {"relative_path":"agent-safety.md"}
Observation: ...
```

### 2. 路径穿越被拒绝

```powershell
go run .\03-agent -mode mock -question "请读取 ../agent-safety.md"
```

预期现象：

```text
Agent Loop 执行失败: 工具调用失败: relative_path 不能跳出安全目录
```

这说明模型可以提出危险参数，但 Go 侧必须拒绝执行。

### 3. 运行测试

```powershell
go test .\03-agent
```

测试覆盖：

- 正常读取安全文件
- 拒绝路径穿越
- 拒绝绝对路径
- 拒绝通配符
- 拒绝非 `.md` / `.txt`
- 拒绝 symlink 逃逸
- mock model 选择 `file_reader`

## 输出怎么看

当前 trace：

```text
Step 1 - ReAct Trace
Action: file_reader
Action Input: {"relative_path":"agent-safety.md"}
Observation: # Agent Safety Sample ...
```

含义：

| 字段 | 含义 |
|---|---|
| `Action` | 模型选择的工具 |
| `Action Input` | 模型生成的工具参数 |
| `Observation` | Go 执行工具后的结果 |

面试时要强调：ReAct Trace 不是为了暴露完整隐藏思维链，而是为了让 Agent 的工具行为可审计、可调试、可复盘。

## 常见误区

### 误区 1：工具 description 写了限制就安全了

不是。description 是给模型看的软提示，不能替代 Go 侧校验。

### 误区 2：只禁止 `../` 就能防路径穿越

不够。还要处理绝对路径、Windows 盘符、路径清理、symlink 逃逸、扩展名、文件大小和文本编码。

### 误区 3：读取结果越完整越好

不一定。Observation 会进入模型上下文，太长会增加成本、挤占 context window，也可能泄露敏感信息。

### 误区 4：ReAct 必须打印完整 Thought

不是。真实项目更推荐记录可审计的高层决策、Action、Action Input、Observation，而不是暴露完整隐藏推理链。



## 表达

可以这样说：

> 我在 Agent Loop 基础上加入了 ReAct 风格的 trace，用 Action、Action Input 和 Observation 表示模型选择工具、工具参数和工具结果。为了演示安全边界，我新增了一个受限 file_reader 工具。这个工具不会信任模型生成的路径，而是在 Go 侧校验 relative_path：拒绝绝对路径、盘符、../、通配符和非文本扩展名，并通过真实路径校验防止 symlink 逃逸。读取结果还限制最大字节数和 UTF-8 文本。这样做的核心是：tool schema 和 prompt 只是软约束，真正的安全边界必须由应用层执行。

## 本课考核重点

1. ReAct 的 Thought / Action / Observation 分别是什么意思
2. 为什么不要依赖或暴露完整隐藏 Thought
3. 为什么文件读取工具是危险工具
4. 为什么 tool schema 不是安全边界
5. 如何防路径穿越
6. 为什么要限制安全目录和文件扩展名
7. 为什么要限制 observation 长度
8. 如果模型传入 `../secret.md`，Go 程序应该怎么处理


