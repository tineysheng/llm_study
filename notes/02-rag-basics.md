# 02 RAG 基础：为什么需要检索增强生成

## 本课目标

阶段 1 已经解决“如何调用 LLM”。阶段 2 开始解决另一个真实项目问题：模型不知道你的私有文档、最新资料和公司内部规则，直接问模型容易过时、编造或无法引用来源。

RAG 的核心做法是：

```text
用户问题 -> 检索相关文档 -> 把文档拼进 Prompt -> LLM 基于上下文回答 -> 返回来源引用
```

先不用急着理解 Embedding 和向量数据库。本课先把 RAG 里最常见的几个词讲清楚。

## RAG 解决什么问题

直接问模型时，模型主要依赖训练参数里的通用知识。它可能不知道：

- 项目本地文档
- 公司内部知识库
- 最新接口说明
- 经常变化的规则、价格、流程

RAG 不是把知识永久训练进模型，而是在每次回答前把相关知识临时查出来，放进上下文窗口，让模型基于这些资料生成答案。

## 本课必须先懂的术语

### 1. Knowledge Base（知识库）

知识库就是 RAG 可以检索的外部资料集合。

在真实项目里，知识库可能是：

- 公司制度文档
- 产品说明
- API 文档
- Markdown 笔记
- 数据库里的 FAQ

在本课代码里，知识库就是 `02-rag/main.go` 里的 `knowledgeBase`：

```go
var knowledgeBase = []document{
    {
        source:  "notes/01-core-concepts.md",
        title:   "LLM API Key 配置",
        content: "运行本项目的 LLM CLI 前，需要在 PowerShell 中设置 OPENAI_API_KEY...",
    },
}
```

### 2. Document（文档）

Document 是知识库里的一条资料。

本课的 `document` 有三个字段：

```go
type document struct {
    source  string // 来源，例如 notes/01-core-concepts.md
    title   string // 标题，例如 LLM API Key 配置
    content string // 具体内容
}
```

### 3. Chunk（文档片段）

真实 RAG 通常不会把一整篇长文档直接塞给模型，而是先切成很多小段，每一小段叫一个 chunk。

原因是：

- 模型上下文窗口有限，不能无限塞文档
- 长文档里通常只有一小段和用户问题相关
- chunk 太多会贵、慢、噪声大
- chunk 太少可能检索不到关键内容

本课代码为了简单，没有单独做切分，`knowledgeBase` 里的每个 `document` 可以先理解成一个小 chunk。

### 4. Retrieve（检索）

检索就是：根据用户问题，从知识库里找出最相关的几个 chunk。

在本课代码里：

```go
results := retrieve(*question, knowledgeBase, *topK)
```

意思是：拿用户问题去 `knowledgeBase` 里找资料，最多返回 `topK` 条。

### 5. Score（相关性分数）

score 表示某个 chunk 和用户问题有多相关。

在本课 demo 里，score 是用关键词匹配临时算出来的：

- 问题和文档都包含 `OPENAI_API_KEY`，分数就更高
- 问题问 API Key，但文档讲 RAG 和微调，分数就是 0

真实 RAG 里通常不用关键词算 score，而是用 Embedding 向量相似度计算。下一课会学。

### 6. Top-K

Top-K 的意思是：按相关性分数排序后，只取前 K 条结果。

例如：

```text
top-k=1：只取最相关的 1 个 chunk
top-k=3：取最相关的 3 个 chunk
top-k=5：取最相关的 5 个 chunk
```

本课代码默认：

```go
topK := flag.Int("top-k", 2, "number of documents to retrieve")
```

也就是默认取前 2 条检索结果。

Top-K 的工程取舍：

| Top-K 设置 | 好处 | 风险 |
|---|---|---|
| 太小 | Prompt 更短、更便宜、噪声少 | 可能漏掉真正答案 |
| 太大 | 更可能覆盖答案 | 无关 chunk 变多，污染上下文，成本和延迟增加 |

所以 RAG 不是“检索越多越好”，而是要在召回率和噪声之间平衡。

### 7. Context（上下文）

Context 是最终塞进 Prompt 里让模型参考的内容。

在 RAG 里，context 通常来自检索结果：

```text
上下文:
[1] 来源: notes/01-core-concepts.md
标题: LLM API Key 配置
内容: 运行本项目的 LLM CLI 前，需要设置 OPENAI_API_KEY...
```

模型不是直接读你的整个项目，而是只看到你放进 Prompt 的 context。

### 8. Source（来源引用）

Source 用来告诉用户答案来自哪里。

例如：

```text
来源：notes/01-core-concepts.md
```

企业知识库问答很重视来源，因为用户需要知道答案有没有依据，也方便排查模型是否引用了错误文档。

## RAG、直接问模型、微调的区别

| 方式 | 适合什么 | 不适合什么 |
|---|---|---|
| 直接问模型 | 通用知识、开放问答、简单解释 | 私有知识、最新事实、需要引用来源的答案 |
| RAG | 私有文档、企业知识库、经常变化的事实知识 | 检索不到或文档质量很差的问题 |
| 微调 | 固定任务格式、领域表达风格、稳定行为模式 | 频繁变化的事实知识 |

面试表达：

> RAG 更适合企业知识库问答，因为企业文档经常变化，更新知识库比重新训练模型更低成本、更可控；同时 RAG 可以返回引用来源，方便排查答案是否来自正确资料。

## 本课代码

代码位置：

```text
02-rag/main.go
```

运行方式：

```powershell
go run .\02-rag -question "怎么设置 OPENAI_API_KEY？"
go run .\02-rag -question "RAG 和微调有什么区别？"
go run .\02-rag -question "streaming 会让模型更聪明吗？"
```

本课代码暂时没有接真实 Embedding，而是用关键词打分模拟检索。这样做是为了先看清楚 RAG 的主链路：

1. 准备本地知识库
2. 根据问题检索 Top-K 文档
3. 把检索结果拼成 Prompt
4. 要求模型只基于上下文回答
5. 返回引用来源

下一课会把“关键词打分”升级成“向量相似度”，正式进入 Embedding。

## 如何看懂本课输出

运行：

```powershell
go run .\02-rag -question "怎么设置 OPENAI_API_KEY？"
```

你会看到几块输出。

### 1. 用户问题

```text
用户问题:
怎么设置 OPENAI_API_KEY？
```

这是用户输入，RAG 会根据这个问题去检索知识库。

### 2. 不使用 RAG 的回答

```text
不使用 RAG 的回答:
我只能根据模型已有知识回答，无法确认你的本地项目文档中是否有关于...
```

这表示如果没有检索项目文档，模型无法确认本地项目里的具体要求。

### 3. 检索到的上下文

```text
检索到的上下文:
1. [notes/01-core-concepts.md] LLM API Key 配置 score=2
2. [README.md] RAG 和微调的区别 score=0
```

这里要看两件事：

- `score=2`：说明第 1 条和问题更相关
- `score=0`：说明第 2 条基本无关，只是因为默认 `top-k=2` 被带出来

这正好说明 Top-K 的作用：如果取太多，可能把无关 chunk 也塞进 Prompt。

### 4. 拼给模型的 Prompt

```text
请只根据下面的上下文回答问题；如果上下文没有答案，就说不知道。
```

这是 RAG Prompt 的核心边界：要求模型只基于检索上下文回答，减少幻觉。

### 5. 使用 RAG 后的回答

```text
使用 RAG 后的回答:
运行本项目的 LLM CLI 前，需要在 PowerShell 中设置 OPENAI_API_KEY...
来源：notes/01-core-concepts.md
```

这就是 RAG 的效果：答案来自检索到的上下文，并且能给出来源。

## 本课常见误区

### 误区 1：RAG 是把所有企业文档都塞进 Prompt

不是。RAG 是先检索，再只把最相关的 Top-K chunk 塞进 Prompt。

### 误区 2：Top-K 越大越好

不是。Top-K 太大会带来无关内容，增加成本、延迟和幻觉风险。

### 误区 3：RAG 能保证答案一定正确

不能。RAG 只是让模型有依据地回答。最终质量还取决于：

- 文档是否正确
- chunk 切分是否合理
- 检索是否召回正确内容
- Prompt 是否限制模型只基于上下文回答
- 代码是否返回来源并处理失败情况

### 误区 4：检索到了内容，回答就一定好

不一定。如果检索到的是错误 chunk 或无关 chunk，模型可能答偏。排查 RAG 问题时，第一步通常是看检索结果，而不是马上换更大的模型。

## 当前要重点理解

- RAG 的关键不是“让模型记住知识”，而是“回答前查知识”
- RAG 回答质量首先受检索质量影响
- 如果没有检索到正确 chunk，模型再强也可能答错
- RAG 比微调更适合频繁变化的事实知识
- Top-K 是“按相关性取前 K 条”，不是越大越好
