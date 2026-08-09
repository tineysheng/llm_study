// 内存向量库演示
//
// 本程序用 mock 向量演示 RAG 中的向量存储和 Top-K 检索：
//  1. 把 chunk、metadata、vector 写入内存向量库
//  2. 用 query vector 计算相似度
//  3. 按分数排序并返回 Top-K chunk
package main

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
)

type chunk struct {
	Source  string
	Title   string
	Index   int
	Content string
}

type vectorItem struct {
	Chunk  chunk
	Vector []float64
}

type searchResult struct {
	Item  vectorItem
	Score float64
}

type memoryVectorStore struct {
	items []vectorItem
}

func main() {
	topK := flag.Int("top-k", 2, "number of chunks to retrieve")
	flag.Parse()

	if *topK <= 0 {
		fmt.Println("top-k 必须大于 0")
		os.Exit(1)
	}

	store := memoryVectorStore{}
	for _, item := range mockItems() {
		if err := store.Add(item); err != nil {
			fmt.Println("写入向量库失败:", err)
			os.Exit(1)
		}
	}

	query := "怎么设置 OpenAI API Key？"
	queryVector := []float64{0.95, 0.10, 0.00}
	results, err := store.Search(queryVector, *topK)
	if err != nil {
		fmt.Println("检索失败:", err)
		os.Exit(1)
	}

	fmt.Println("向量库写入 chunk 数:", store.Len())
	fmt.Println("用户问题:", query)
	fmt.Println("query vector:", queryVector)
	fmt.Println("top-k:", *topK)
	fmt.Println()

	fmt.Println("Top-K 检索结果:")
	for i, result := range results {
		c := result.Item.Chunk
		fmt.Printf("%d. score=%.3f title=%s source=%s index=%d\n", i+1, result.Score, c.Title, c.Source, c.Index)
		fmt.Println("   content:", c.Content)
		fmt.Println()
	}
}

func (s *memoryVectorStore) Add(item vectorItem) error {
	if len(item.Vector) == 0 {
		return errors.New("vector 不能为空")
	}
	if item.Chunk.Content == "" {
		return errors.New("chunk content 不能为空")
	}
	s.items = append(s.items, item)
	return nil
}

func (s memoryVectorStore) Search(queryVector []float64, topK int) ([]searchResult, error) {
	if len(s.items) == 0 {
		return nil, errors.New("向量库为空")
	}
	if topK <= 0 {
		return nil, errors.New("topK 必须大于 0")
	}

	results := make([]searchResult, 0, len(s.items))
	for _, item := range s.items {
		score, err := cosineSimilarity(queryVector, item.Vector)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", item.Chunk.Title, err)
		}
		results = append(results, searchResult{
			Item:  item,
			Score: score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK], nil
}

func (s memoryVectorStore) Len() int {
	return len(s.items)
}

func mockItems() []vectorItem {
	return []vectorItem{
		{
			Chunk: chunk{
				Source:  "notes\\01-core-concepts.md",
				Title:   "API Key 配置",
				Index:   1,
				Content: "运行本项目的 LLM CLI 前，需要在 PowerShell 中设置 OPENAI_API_KEY 环境变量。",
			},
			Vector: []float64{0.90, 0.20, 0.00},
		},
		{
			Chunk: chunk{
				Source:  "notes\\01-core-concepts.md",
				Title:   "Streaming 流式输出",
				Index:   2,
				Content: "Streaming 流式输出主要改善用户等待体验，但不会让模型最终答案本身更聪明。",
			},
			Vector: []float64{0.05, 0.90, 0.10},
		},
		{
			Chunk: chunk{
				Source:  "notes\\01-core-concepts.md",
				Title:   "JSON 结构化输出",
				Index:   3,
				Content: "业务系统需要稳定、可解析的数据结构，模型返回 JSON 后，Go 程序可以解析成结构体。",
			},
			Vector: []float64{0.20, 0.10, 0.90},
		},
		{
			Chunk: chunk{
				Source:  "notes\\02-rag-basics.md",
				Title:   "RAG 和微调的区别",
				Index:   4,
				Content: "RAG 更适合经常变化的事实知识和私有文档；微调更适合稳定任务模式和回答风格。",
			},
			Vector: []float64{0.30, 0.20, 0.50},
		},
	}
}

func cosineSimilarity(a, b []float64) (float64, error) {
	if len(a) == 0 || len(b) == 0 {
		return 0, errors.New("向量不能为空")
	}
	if len(a) != len(b) {
		return 0, fmt.Errorf("向量维度不一致: %d != %d", len(a), len(b))
	}

	var dotProduct float64
	var normA float64
	var normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0, errors.New("零向量不能计算余弦相似度")
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)), nil
}
