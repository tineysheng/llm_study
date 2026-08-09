# 05 文档处理：读取 Markdown 并切成 chunk

## 本课目标

前面已经学了 RAG 的主链路和 Embedding。现在进入 RAG 建库前最重要的一步：文档处理。

真实 RAG 不能直接把一整篇文档都塞给模型，而是要先把文档切成多个小片段，也就是 chunk。

```text
Markdown 文档 -> 解析内容 -> 切 chunk -> 保存 metadata -> 后续做 Embedding
```

## 本课必须先懂的术语

### 1. Markdown Document（Markdown 文档）

Markdown 是常见的文本知识库格式，例如本项目的 `notes/*.md`。

它通常包含：

- 标题：`#`、`##`、`###`
- 段落
- 列表
- 代码块
- 表格

RAG 项目经常从 Markdown、PDF、网页、Word、数据库 FAQ 中读取内容。Markdown 最适合学习，因为它是纯文本，容易读取和切分。

### 2. Chunk（文档片段）

chunk 是从原始文档中切出来的一小段内容。

为什么要切 chunk：

- 模型上下文窗口有限，不能无限塞文档
- 用户问题通常只和文档里的某一小段相关
- chunk 太大，容易带入无关内容
- chunk 太小，可能缺少上下文，导致答案不完整

### 3. Chunk Size（chunk 大小）

chunk size 是每个 chunk 大概包含多少内容。

本课 demo 用“字符数”模拟 chunk size，例如：

```text
chunk-size=300：每个 chunk 最多约 300 个字符
chunk-size=800：每个 chunk 最多约 800 个字符
```

真实项目里更常用 token 数作为 chunk size，因为模型成本和上下文窗口按 token 计算。

### 4. Metadata（元数据）

metadata 是 chunk 的附加信息，用来说明这个 chunk 来自哪里。

本课每个 chunk 会保存：

| 字段 | 含义 |
|---|---|
| `source` | 来源文件路径 |
| `title` | 当前 chunk 对应的标题 |
| `index` | chunk 在文档里的序号 |
| `content` | chunk 正文 |

metadata 很重要，因为 RAG 最终回答需要引用来源。如果只保存 content，不保存 source/title/index，后面很难告诉用户答案来自哪里。

### 5. Heading Split（按标题切分）

按标题切分就是遇到 Markdown 标题时，把一个章节切成一个 chunk。

例如：

```text
## RAG 解决什么问题
...
## 本课必须先懂的术语
...
```

这两个章节可以切成两个 chunk。

优点：

- 保留文档自然结构
- chunk 语义更完整
- 来源标题更清楚

风险：

- 有的章节很长，会超过理想 chunk size
- 有的章节很短，信息量不足

### 6. Fixed Size Split（固定长度切分）

固定长度切分就是按固定字符数切 chunk。

例如：

```text
每 400 个字符切一段
```

优点：

- 实现简单
- chunk 大小比较稳定

风险：

- 可能从一句话中间切开
- 可能破坏 Markdown 结构
- 可能把标题和正文切散

真实项目通常会结合两者：先按标题或段落切，再对过长的 chunk 做二次切分。

## 本课代码

代码位置：

```text
02-rag\chunk-demo\main.go
```

默认运行：

```powershell
go run .\02-rag\chunk-demo
```

默认读取：

```text
notes\02-rag-basics.md
```

按标题切分：

```powershell
go run .\02-rag\chunk-demo -mode heading
```

按固定长度切分：

```powershell
go run .\02-rag\chunk-demo -mode fixed -chunk-size 500
```

指定文件：

```powershell
go run .\02-rag\chunk-demo -file notes\03-embedding-basics.md -mode heading
```

## 如何看懂本课输出

你会看到类似输出：

```text
source: notes\02-rag-basics.md
mode: heading
chunks: 8

Chunk #1
title: 02 RAG 基础：为什么需要检索增强生成
chars: 320
preview: 阶段 1 已经解决“如何调用 LLM”...
```

重点看四件事：

- `source`：chunk 来自哪个文件
- `title`：chunk 属于哪个标题
- `index`：chunk 的序号
- `preview`：chunk 内容预览

这些就是后续向量库要保存的 metadata。

如果按标题切分时出现只有标题、没有正文的 chunk，这类 chunk 通常质量很差，因为它几乎没有可回答问题的信息。本课 demo 会跳过这种 chunk。

## 本课和真实 RAG 的关系

真实 RAG 建库通常是：

```text
读取文档 -> 清洗文本 -> 切 chunk -> 生成 embedding -> 存入向量库
```

本课完成的是前半段：

```text
读取文档 -> 切 chunk -> 保存 metadata
```

下一步会把每个 chunk 转成向量，放进内存向量库里做 Top-K 检索。

## 本课常见误区

### 误区 1：chunk 越大越好

不是。chunk 大更容易保留上下文，但也更容易带入无关内容，增加 token 成本。

### 误区 2：chunk 越小越好

也不是。chunk 小更精确，但可能缺少上下文，导致模型无法回答完整问题。

### 误区 3：只保存 content 就够了

不够。必须保存 metadata，例如 source、title、index。否则后续无法引用来源，也不好排查检索结果。

### 误区 4：固定长度切分一定比标题切分差

不一定。固定长度切分简单稳定，适合结构差的文本；标题切分更适合结构清楚的 Markdown。真实项目要看文档类型。

## 当前重点

- 文档处理是 RAG 建库的第一步
- chunk 是后续 Embedding 和检索的基本单位
- metadata 决定答案能否追溯来源
- chunk size 会影响召回率、噪声、成本和回答质量
- 标题切分和固定长度切分各有取舍
