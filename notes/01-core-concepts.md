# 01 核心概念：调用 LLM API 前必须懂的

## 1. Token（词元）

- Token 是模型处理文本的最小单位。英文约 1 个单词 ≈ 1~2 token；中文 1 个汉字 ≈ 1~2 token
- **API 按 token 计费**，输入和输出都收费（输出通常更贵）
- Go 里可以用 `github.com/pkoukk/tiktoken-go` 在本地估算 token 数
- 面试点：如何控制成本？→ 压缩 prompt、限制 max_tokens、用小模型做简单任务

## 2. Context Window（上下文窗口）

- 模型一次能"看到"的最大 token 数（输入 + 输出之和）
- GPT-4o 是 128K，但窗口越大费用越高、注意力越分散
- 多轮对话的本质：**LLM API 是无状态的**，每次请求都要把完整历史发过去。历史太长会超窗口，需要截断或总结

## 3. Messages 的三种角色

| 角色 | 作用 |
|---|---|
| `system` | 设定模型的行为和人设，优先级最高 |
| `user` | 用户的输入 |
| `assistant` | 模型之前的回复（多轮对话时由我们回填） |

后续还会遇到 `tool` 角色（阶段 3 的工具调用）。

## 4. 关键采样参数

- **temperature**（0~2）：越低越确定、越保守；越高越发散。写代码/事实问答用低值，创意写作用高值
- **top_p**：核采样，与 temperature 二选一调即可
- **max_tokens**：限制输出长度，省钱 + 防止失控
- **presence/frequency_penalty**：减少重复

在 `01-llm-basics/main.go` 中，第二课把这些参数做成了配置：

- `-model` 或 `OPENAI_MODEL`：选择使用哪个模型
- `-temperature` 或 `OPENAI_TEMPERATURE`：控制输出确定性和发散程度
- `-max-tokens` 或 `OPENAI_MAX_TOKENS`：限制最大输出长度
- `-timeout-seconds` 或 `OPENAI_TIMEOUT_SECONDS`：限制 API 请求最长等待时间

示例：

```powershell
go run ./01-llm-basics -temperature 0 -max-tokens 200
go run ./01-llm-basics -temperature 1.5 -max-tokens 800
```

工程理解：

- 写代码、事实问答、数据抽取：通常使用较低 temperature
- 创意写作、头脑风暴：可以使用较高 temperature
- `max_tokens` 控制的是输出上限，不是输入上限
- timeout 是后端服务必须有的保护，避免请求长时间卡住

## 5. 流式输出（Streaming / SSE）

- 默认模式：等模型生成完才返回全部内容，用户等待感强
- 流式模式：基于 SSE（Server-Sent Events），生成一点返回一点，打字机效果
- 工程上注意：流式的每个 chunk 只含增量内容（delta），需要自己拼接

## 6. 常见模型定位（OpenAI）

- `gpt-4o`：旗舰多模态，能力强、贵
- `gpt-4o-mini`：便宜快速，学习/简单任务首选
- 学习阶段建议用 mini，成本几乎可忽略

## 自测问题

1. 为什么多轮对话会越来越贵、越来越慢？
2. temperature=0 时输出一定完全相同吗？（提示：不完全保证，但高度确定）
3. 流式和非流式，最终得到的文本内容有区别吗？（没有，只是传输方式不同）

## 第一关问答总结：LLM API 无状态与消息历史

### 核心结论

LLM API 本身不保存会话。它每次只根据当前请求里的 `messages` 生成回答。所谓“模型记得之前聊过什么”，其实是应用程序把历史消息保存下来，并在下一次请求时重新发给模型。

在 `01-llm-basics/main.go` 中：

- `messages`：保存完整对话历史
- `systemMessage`：定义模型角色和回答风格
- `content`：某一条消息的文本内容
- `user` 消息：用户输入
- `assistant` 消息：模型之前的回答

### 学习过程中的关键问题

**问题 1：为什么 LLM API 是无状态的？项目靠什么让模型像记得之前聊过什么？**

答案：模型服务不会自动保存应用的会话历史。项目靠 `messages` 切片保存 system、user、assistant 消息，并在每次请求时传给模型。

**问题 2：为什么 `/reset` 要保留 `systemMessage`？**

答案：`/reset` 清空的是用户和模型之间的历史对话，不应该清掉模型的角色设定。`systemMessage` 负责约束模型行为、语气和回答风格。

**问题 3：为什么要把 assistant 的完整回复 append 回 `messages`？**

答案：多轮对话不只依赖用户说过什么，也依赖模型之前回答过什么。如果不保存 assistant 回复，用户下一轮说“继续”“修改刚才的代码”时，模型就缺少必要上下文。

## 第二关学习记录：模型参数控制

### 本关知识点

LLM 应用不是只“调用模型”，还要通过参数控制模型的成本、稳定性、速度和输出风格。

本关关注四个配置：

- `model`：选择模型，影响能力、价格、速度和上下文长度
- `temperature`：控制输出随机性，越低越稳定，越高越发散
- `max_tokens`：控制最大输出长度，避免输出过长和成本失控
- `timeout`：控制请求最长等待时间，避免后端服务被外部 API 卡死

### 本关实现内容

已在 `01-llm-basics/main.go` 中加入：

- `chatConfig`：集中保存模型调用配置
- `loadConfig`：从环境变量和命令行参数读取配置
- `/config`：在 CLI 中查看当前配置
- `context.WithTimeout`：给每次模型请求增加超时控制

支持的环境变量：

- `OPENAI_MODEL`
- `OPENAI_TEMPERATURE`
- `OPENAI_MAX_TOKENS`
- `OPENAI_TIMEOUT_SECONDS`

支持的命令行参数：

- `-model`
- `-temperature`
- `-max-tokens`
- `-timeout-seconds`

### 建议观察方式

用同一个问题分别运行：

```powershell
go run ./01-llm-basics -temperature 0
go run ./01-llm-basics -temperature 1.5
```

可以问：

```text
给我 5 个学习 Go 并发的建议
```

观察重点：

- 低 temperature 是否更稳定、更保守
- 高 temperature 是否更发散、更有变化
- 降低 `max_tokens` 后输出是否会变短或被截断

### 学习过程中的关键问题

**问题 1：如果要让模型回答更稳定、更适合写代码，temperature 应该调高还是调低？**

答案：应该调低。写代码、事实问答、结构化抽取更需要稳定和准确；高 temperature 更适合创意、头脑风暴、产品设计等发散任务。

**问题 2：`max_tokens` 控制的是什么？设置太小会怎样？**

答案：`max_tokens` 控制本次请求的最大输出长度。设置太小会导致回答不完整、代码被截断、JSON 不合法或关键信息缺失。

**问题 3：为什么调用 LLM API 要设置 timeout？**

答案：LLM API 是外部网络调用，可能因为网络抖动、服务拥塞或限流长时间无响应。后端服务设置 timeout 可以避免请求无限等待，占用 goroutine、连接和内存资源。

**问题 4：`model` 为什么要可配置？**

答案：不同模型在能力、成本、速度、上下文窗口、工具调用、结构化输出、多模态和稳定性上都有差异。真实项目中要根据任务复杂度、预算和效果要求选择模型。

**问题 5：小模型和大模型如何选择？**

答案：要看任务复杂度、答案质量、成本、延迟、上下文长度、结构化输出能力、工具调用能力、稳定性和失败代价。简单任务适合小模型，复杂推理或高失败代价任务适合强模型。

**学习反馈：问题不能为了数量而重复。**

答案：考核应该覆盖知识点，但如果用户已经证明掌握，就应及时收束。问题数量服务于掌握程度，不是机械指标。

### 本关通过标准

本关已通过。应能解释：

- temperature 如何影响输出稳定性
- `max_tokens` 为什么能控制成本和输出风险
- timeout 为什么是后端服务调用外部 API 的必要保护
- model 选择如何影响 AI 应用的效果、成本和稳定性
