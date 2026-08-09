# 04 Embedding API：把真实文本转成模型向量

## 本课目标

上一课用手写向量理解了余弦相似度。本课开始调用真实 Embedding API，让模型把自然语言文本转换成真正的语义向量。

本课完成后，你应该能看懂这条链路：

```text
文本 -> Embedding API -> 向量 -> 余弦相似度 -> 判断语义相关性
```

## 本课必须先懂的术语

### 1. Embedding Model（向量模型）

Embedding Model 是专门把文本转换成向量的模型。

它和 Chat Model 不一样：

| 类型 | 输入 | 输出 | 用途 |
|---|---|---|---|
| Chat Model | messages | 自然语言或 JSON | 聊天、总结、问答、生成 |
| Embedding Model | 文本 | 向量数组 | 语义检索、聚类、相似度计算 |

本课默认使用：

```text
text-embedding-3-small
```

它适合学习和一般语义检索场景，成本相对低。

### 2. Input Text（输入文本）

输入文本就是要转换成向量的原始文本。

在 RAG 里，通常有两类文本要转向量：

- 用户问题：例如“怎么配置 OpenAI API Key？”
- 文档 chunk：例如“运行前需要设置 OPENAI_API_KEY 环境变量。”

只有两边都转成向量，才能计算相似度。

### 3. Batch Embedding（批量向量化）

一次 API 请求可以传多段文本，让模型一次返回多个向量。

本课 demo 会一次传入三段文本：

```text
1. 怎么配置 OpenAI API Key？
2. 如何设置 OPENAI_API_KEY 环境变量？
3. 流式输出可以改善用户等待体验。
```

这样可以比较：

- 第 1 段 vs 第 2 段：语义相近
- 第 1 段 vs 第 3 段：语义不太相关

真实 RAG 建库时，也常常批量处理很多 chunk，减少网络请求次数。

### 4. Vector Dimension（向量维度）

真实 Embedding API 返回的向量通常不是 3 维，而是几百到几千维。

你不需要手动理解每一维代表什么。工程上只需要知道：

- 同一个向量库里的向量维度必须一致
- 查询向量和文档向量必须来自同一个 Embedding 模型
- 换 Embedding 模型后，通常要重新生成文档向量

### 5. Cosine Similarity（余弦相似度）

本课继续用余弦相似度比较向量。

它的作用是：

```text
问题向量 vs 文档向量 -> 得到一个相似度分数
```

相似度更高，表示两段文本语义更可能相关。

## 本课代码

代码位置：

```text
02-rag\embedding-api\main.go
```

如果你没有 API 额度，先运行本地模拟版：

```powershell
go run .\02-rag\embedding-api -mock
```

`-mock` 不会联网，也不需要 API Key。它用固定向量模拟 Embedding API 的返回结果，适合先学习“文本 -> 向量 -> 相似度”的工程流程。

如果要调用真实 API，运行前需要设置 API Key：

```powershell
$env:OPENAI_API_KEY = "sk-你的key"
```

然后运行：

```powershell
go run .\02-rag\embedding-api
```

可选参数：

```powershell
go run .\02-rag\embedding-api -model text-embedding-3-small -timeout-seconds 30
```

如果看到：

```text
请先设置环境变量 OPENAI_API_KEY，或使用 -mock 运行本地模拟版
```

说明当前没有读到 API Key。没有余额或暂时不想调用真实 API 时，直接使用：

```powershell
go run .\02-rag\embedding-api -mock
```

## 如何看懂本课输出

你会看到类似输出：

```text
Embedding model: text-embedding-3-small
运行模式: mock，本地模拟向量，不调用 API
向量维度: 1536

文本 1: 怎么配置 OpenAI API Key？
文本 2: 如何设置 OPENAI_API_KEY 环境变量？
文本 3: 流式输出可以改善用户等待体验。

相似度:
文本 1 vs 文本 2 = 0.8xxx
文本 1 vs 文本 3 = 0.2xxx
```

重点观察：

- 文本 1 和文本 2 都在说 API Key 配置，相似度应该更高
- 文本 1 和文本 3 一个说 API Key，一个说流式输出，相似度应该更低

具体分数可能随模型变化，不要死记数字。

如果使用 `-mock`，向量维度会是 8，因为模拟版为了方便观察只写了 8 维向量。真实 API 通常会返回更高维向量，例如 1536 维。

## 本课和真实 RAG 的关系

真实 RAG 会做两类 Embedding：

### 1. 建库阶段

```text
读取文档 -> 切 chunk -> 对每个 chunk 调 Embedding API -> 存入向量库
```

这一步通常离线完成，或者在文档更新时完成。

### 2. 查询阶段

```text
用户问题 -> 调 Embedding API -> 得到问题向量 -> 向量库 Top-K 检索 -> 拼 Prompt -> Chat Model 回答
```

所以 Embedding API 通常会在两个地方用到：文档入库和用户查询。

## 本课常见误区

### 误区 1：Embedding API 会直接回答问题

不会。Embedding API 只返回向量，不返回自然语言答案。

回答问题仍然需要 Chat Model。

### 误区 2：相似度高就等于答案正确

不等于。相似度高只说明语义相关，不能证明文档内容正确。

### 误区 3：查询向量和文档向量可以用不同模型

不建议。不同模型生成的向量空间不同，直接比较没有可靠意义。

### 误区 4：换 Embedding 模型只改查询代码就行

不行。文档向量也要用同一个新模型重新生成，否则向量空间不一致。

## 当前重点

- Embedding API 输入文本，输出向量
- Chat Model 负责生成答案，Embedding Model 负责语义表示
- 相似文本的向量余弦相似度通常更高
- 文档向量和查询向量必须来自同一个 Embedding 模型
- 真实 RAG 会在建库阶段和查询阶段都用到 Embedding
