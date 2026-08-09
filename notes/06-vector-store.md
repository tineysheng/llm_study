# 06 内存向量库：保存 chunk 向量并做 Top-K 检索

## 本课目标

前面已经学了：

```text
读取 Markdown -> 切 chunk -> 给 chunk 生成向量
```

本课继续往下走：把 chunk、向量和 metadata 存起来，并支持根据用户问题向量检索最相关的 Top-K chunk。

```text
chunk + embedding + metadata -> vector store -> query vector -> Top-K results
```

为了降低学习复杂度，本课先实现一个内存向量库，不接 Qdrant、pgvector 这类真实向量数据库。

## 本课必须先懂的术语

### 1. Vector Store（向量库）

向量库是用来保存和检索向量的存储系统。

在 RAG 里，它通常保存三类东西：

| 内容 | 作用 |
|---|---|
| `embedding` | chunk 的向量，用于相似度检索 |
| `content` | chunk 原文，后续拼进 Prompt |
| `metadata` | 来源信息，例如 source、title、index |

本课的内存向量库就是一个 Go slice：

```go
type memoryVectorStore struct {
    items []vectorItem
}
```

真实项目可能换成：

- Qdrant
- pgvector
- Milvus
- Elasticsearch dense_vector
- Redis Vector Search

但它们解决的核心问题一样：保存向量，并按相似度检索。

### 2. Upsert / Add（写入向量）

写入向量就是把一个 chunk 的内容、metadata 和 embedding 保存到向量库。

本课用 `Add`：

```go
store.Add(vectorItem{
    Chunk: chunk{...},
    Vector: []float64{...},
})
```

真实向量数据库里常叫 `upsert`，意思是：

- 不存在就插入
- 已存在就更新

### 3. Query Vector（查询向量）

用户问题也要先转成向量，这个向量叫 query vector。

```text
用户问题：怎么设置 API Key？
query vector: [0.95, 0.10, 0.00]
```

向量库拿 query vector 去和所有文档向量计算相似度。

### 4. Top-K Search（Top-K 检索）

Top-K Search 是向量库最核心的能力：

```text
query vector -> 计算相似度 -> 排序 -> 返回前 K 个 chunk
```

例如：

```text
top-k=2：返回最相关的 2 个 chunk
```

本课会看到：

- API Key 相关 chunk 排第一
- Streaming 或 JSON 相关 chunk 排后面

### 5. Brute Force Search（暴力检索）

本课的内存向量库会把 query vector 和每个 item 的向量逐个计算相似度。

这叫暴力检索：

```text
for each vector:
    score = cosine(query, vector)
sort by score
return topK
```

优点：

- 实现简单
- 适合学习和小数据量

缺点：

- 数据量大时很慢
- 每次查询都要遍历所有向量

真实向量数据库会使用索引结构提升性能，例如 HNSW。

### 6. Similarity Score（相似度分数）

score 表示 query vector 和 chunk vector 的相似程度。

本课仍然用余弦相似度：

```text
score 接近 1：更相似
score 接近 0：关系弱
```

注意：score 高只说明“更可能相关”，不等于答案一定正确。

## 本课代码

代码位置：

```text
02-rag\vector-store-demo\main.go
```

运行：

```powershell
go run .\02-rag\vector-store-demo
```

指定 Top-K：

```powershell
go run .\02-rag\vector-store-demo -top-k 2
go run .\02-rag\vector-store-demo -top-k 3
```

本课仍然使用 mock 向量，不调用真实 Embedding API。重点是理解向量库存储和检索机制。

## 如何看懂本课输出

你会看到类似输出：

```text
向量库写入 chunk 数: 4
用户问题: 怎么设置 OpenAI API Key？
top-k: 2

Top-K 检索结果:
1. score=0.994 title=API Key 配置 source=notes\01-core-concepts.md index=1
2. score=0.518 title=RAG 和微调的区别 source=notes\02-rag-basics.md index=4
```

重点看：

- `score`：相似度分数
- `title`：命中的 chunk 标题
- `source`：来源文件
- `index`：chunk 序号
- `content`：后续会拼进 Prompt 的原文

这就是 RAG 的检索阶段输出。

## 本课和真实 RAG 的关系

本课实现的是：

```text
chunk + mock vector -> 内存向量库 -> Top-K 检索
```

真实 RAG 会换成：

```text
chunk + real embedding -> Qdrant/pgvector -> Top-K 检索
```

但应用层逻辑相同：

1. 文档入库时保存 chunk、embedding、metadata
2. 查询时把用户问题转成 query vector
3. 向量库返回 Top-K chunk
4. 把这些 chunk 拼进 Prompt
5. Chat Model 基于上下文回答

## 本课常见误区

### 误区 1：向量库只存向量

不是。只存向量没有用。RAG 需要同时存 content 和 metadata，否则检索出来后没法拼 Prompt，也没法引用来源。

### 误区 2：内存向量库不能学真实 RAG

不是。内存向量库不适合生产，但非常适合理解核心机制。真实向量数据库只是把存储、索引和查询性能做得更强。

### 误区 3：Top-K 返回的都是正确答案

不是。Top-K 只是最相关的候选 chunk。最终还要看文档质量、chunk 切分、Prompt 约束和模型生成。

### 误区 4：数据量大时还能暴力遍历

不适合。小数据量可以暴力遍历；大数据量需要向量索引和专门的向量数据库。

## 当前重点

- 向量库存的是 embedding + content + metadata
- 查询时用 query vector 做 Top-K 相似度检索
- 内存向量库适合学习，不适合大规模生产
- metadata 是 RAG 可追溯引用的基础
- 真实向量数据库主要解决规模、索引、持久化和性能问题
