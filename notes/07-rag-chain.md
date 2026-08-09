# 07 RAG 问答链路：从问题到带来源答案

## 本课目标

前面几课分别学了：

```text
文档切 chunk -> 生成向量 -> 存入向量库 -> Top-K 检索
```

本课把它们串成一个最小 RAG 问答闭环：

```text
用户问题 -> 问题向量 -> Top-K 检索 -> 拼 Prompt -> 基于上下文回答 -> 输出来源
```

因为目前没有真实 API 额度，本课继续使用 mock 向量和 mock 生成器，但代码结构会按真实 RAG 链路设计。

## 本课必须先懂的术语

### 1. Retrieval（检索）

Retrieval 是根据用户问题，从向量库中找出最相关的 chunk。

输入：

```text
用户问题：怎么设置 OpenAI API Key？
```

输出：

```text
Top-K chunk:
1. API Key 配置
2. RAG 和微调的区别
```

检索阶段决定模型能看到什么资料，所以它直接影响最终回答质量。

### 2. Context（上下文）

Context 是拼进 Prompt 里给模型参考的检索结果。

例如：

```text
[1] 来源: notes\01-core-concepts.md
标题: API Key 配置
内容: 运行本项目前，需要设置 OPENAI_API_KEY 环境变量。
```

RAG 的关键不是让模型读取整个知识库，而是只把 Top-K context 临时放进本次请求。

### 3. Prompt Assembly（Prompt 拼接）

Prompt Assembly 是把系统规则、检索上下文和用户问题组合成完整 Prompt。

本课 Prompt 的核心规则是：

```text
请只根据上下文回答问题。
如果上下文没有答案，就说不知道。
回答后列出引用来源。
```

这个规则用于减少幻觉，并让答案可追溯。

### 4. Grounded Answer（基于上下文的回答）

Grounded Answer 指答案必须有上下文依据。

普通聊天可能基于模型参数里的通用知识回答；RAG 问答要求模型基于检索上下文回答。

如果上下文没有答案，正确行为是：

```text
根据当前知识库上下文，我不知道答案。
```

而不是硬编。

### 5. Citation（引用来源）

Citation 是答案后面的来源信息。

例如：

```text
来源：
- notes\01-core-concepts.md / API Key 配置 / chunk #1
```

引用来源的作用：

- 用户能判断答案依据
- 开发者能排查检索是否正确
- 企业知识库问答更可信

### 6. Score Threshold（相似度阈值）

Score Threshold 是最低相似度要求。

例如：

```text
score >= 0.75 才认为上下文足够相关
```

如果 Top-1 分数太低，说明知识库里可能没有答案。这时应该返回“不知道”，而不是把低相关 chunk 强行交给模型回答。

本课 demo 会用阈值模拟这个判断。

## 本课代码

代码位置：

```text
02-rag\rag-chain-demo\main.go
```

运行默认问题：

```powershell
go run .\02-rag\rag-chain-demo
```

指定问题：

```powershell
go run .\02-rag\rag-chain-demo -question "怎么设置 OpenAI API Key？"
go run .\02-rag\rag-chain-demo -question "streaming 有什么用？"
go run .\02-rag\rag-chain-demo -question "Go goroutine 怎么调度？"
```

指定 Top-K 和阈值：

```powershell
go run .\02-rag\rag-chain-demo -top-k 2 -threshold 0.75
```

## 如何看懂本课输出

你会看到几块输出。

### 1. 用户问题

```text
用户问题: 怎么设置 OpenAI API Key？
```

### 2. Top-K 检索结果

```text
1. score=0.994 title=API Key 配置 source=notes\01-core-concepts.md
```

这里要观察：最高分 chunk 是否真的和问题相关。

### 3. 拼给模型的 Prompt

```text
请只根据下面的上下文回答问题；如果上下文没有答案，就说不知道。
```

这一步把检索结果变成模型可读的上下文。

### 4. 最终回答

```text
运行本项目前，需要在 PowerShell 中设置 OPENAI_API_KEY 环境变量。

来源：
- notes\01-core-concepts.md / API Key 配置 / chunk #1
```

答案来自 chunk content，来源来自 metadata。

### 5. 不知道场景

如果问题和知识库不相关，例如：

```powershell
go run .\02-rag\rag-chain-demo -question "Go goroutine 怎么调度？"
```

应该看到：

```text
根据当前知识库上下文，我不知道答案。
```

这比胡编更符合企业知识库问答要求。

## 本课和真实 RAG 的关系

本课是 mock 版：

```text
mock question vector -> memory vector store -> mock answer
```

真实 RAG 会替换成：

```text
Embedding API -> Qdrant/pgvector -> Chat API
```

但应用层结构一样：

1. 对用户问题生成 query embedding
2. 向量库 Top-K 检索相关 chunk
3. 把 chunk 拼成 Prompt context
4. Chat Model 只基于 context 回答
5. 返回答案和引用来源
6. 如果没有足够相关上下文，就说不知道

## 本课常见误区

### 误区 1：检索到 chunk 就一定要回答

不是。如果最高分很低，说明上下文可能不相关，应该说不知道。

### 误区 2：Prompt 里写“不要胡编”就一定不会幻觉

不是。Prompt 是软约束，还需要阈值、引用、日志和评测来约束系统行为。

### 误区 3：引用来源只是展示效果

不是。引用来源是 RAG 的核心工程能力，用来建立可信度和排查问题。

### 误区 4：Top-K 越大答案越好

不一定。Top-K 太大会把无关 chunk 塞进 Prompt，增加成本和上下文污染。

## 当前重点

- RAG 问答链路是多个步骤串起来，不只是一次模型调用
- 检索结果要拼成 Prompt context
- 答案必须基于上下文，并输出引用来源
- 上下文不够相关时应该说不知道
- 阈值可以帮助避免低相关 chunk 导致模型硬答
