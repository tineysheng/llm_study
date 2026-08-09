// RAG 问答链路演示
//
// 本程序用 mock 向量和 mock 生成器串起最小 RAG 闭环：
// 用户问题 -> query vector -> Top-K 检索 -> Prompt 拼接 -> 基于上下文回答 -> 引用来源。
package main

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
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

type config struct {
	question  string
	topK      int
	threshold float64
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println("配置错误:", err)
		os.Exit(1)
	}

	store := memoryVectorStore{}
	for _, item := range mockItems() {
		if err := store.Add(item); err != nil {
			fmt.Println("写入向量库失败:", err)
			os.Exit(1)
		}
	}

	queryVector := mockQueryVector(cfg.question)
	results, err := store.Search(queryVector, cfg.topK)
	if err != nil {
		fmt.Println("检索失败:", err)
		os.Exit(1)
	}

	prompt := buildPrompt(cfg.question, results)
	answer := generateAnswer(results, cfg.threshold)

	fmt.Println("用户问题:", cfg.question)
	fmt.Println("query vector:", queryVector)
	fmt.Println("top-k:", cfg.topK)
	fmt.Printf("threshold: %.2f\n", cfg.threshold)
	fmt.Println()

	fmt.Println("Top-K 检索结果:")
	for i, result := range results {
		c := result.Item.Chunk
		fmt.Printf("%d. score=%.3f title=%s source=%s index=%d\n", i+1, result.Score, c.Title, c.Source, c.Index)
		fmt.Println("   content:", c.Content)
	}
	fmt.Println()

	fmt.Println("拼给模型的 Prompt:")
	fmt.Println(prompt)
	fmt.Println()

	fmt.Println("最终回答:")
	fmt.Println(answer)
}

func loadConfig() (config, error) {
	question := flag.String("question", "怎么设置 OpenAI API Key？", "user question")
	topK := flag.Int("top-k", 2, "number of chunks to retrieve")
	threshold := flag.Float64("threshold", 0.75, "minimum top score required to answer")
	flag.Parse()

	if strings.TrimSpace(*question) == "" {
		return config{}, errors.New("question 不能为空")
	}
	if *topK <= 0 {
		return config{}, errors.New("top-k 必须大于 0")
	}
	if *threshold < -1 || *threshold > 1 {
		return config{}, errors.New("threshold 必须在 -1 到 1 之间")
	}

	return config{
		question:  *question,
		topK:      *topK,
		threshold: *threshold,
	}, nil
}

func (s *memoryVectorStore) Add(item vectorItem) error {
	if len(item.Vector) == 0 {
		return errors.New("vector 不能为空")
	}
	if strings.TrimSpace(item.Chunk.Content) == "" {
		return errors.New("chunk content 不能为空")
	}
	s.items = append(s.items, item)
	return nil
}

func (s memoryVectorStore) Search(queryVector []float64, topK int) ([]searchResult, error) {
	if len(s.items) == 0 {
		return nil, errors.New("向量库为空")
	}

	results := make([]searchResult, 0, len(s.items))
	for _, item := range s.items {
		score, err := cosineSimilarity(queryVector, item.Vector)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", item.Chunk.Title, err)
		}
		results = append(results, searchResult{Item: item, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK], nil
}

func buildPrompt(question string, results []searchResult) string {
	var builder strings.Builder
	builder.WriteString("请只根据下面的上下文回答问题；如果上下文没有答案，就说不知道。\n")
	builder.WriteString("回答后必须列出引用来源。\n\n")
	builder.WriteString("上下文:\n")
	for i, result := range results {
		c := result.Item.Chunk
		fmt.Fprintf(&builder, "[%d] 来源: %s\n标题: %s\nchunk: #%d\n内容: %s\n\n", i+1, c.Source, c.Title, c.Index, c.Content)
	}
	builder.WriteString("问题: ")
	builder.WriteString(question)
	return builder.String()
}

func generateAnswer(results []searchResult, threshold float64) string {
	if len(results) == 0 || results[0].Score < threshold {
		return "根据当前知识库上下文，我不知道答案。"
	}

	top := results[0].Item.Chunk
	var builder strings.Builder
	builder.WriteString(top.Content)
	builder.WriteString("\n\n来源:\n")
	fmt.Fprintf(&builder, "- %s / %s / chunk #%d", top.Source, top.Title, top.Index)
	return builder.String()
}

func mockQueryVector(question string) []float64 {
	normalized := strings.ToLower(question)
	switch {
	case strings.Contains(normalized, "api key") || strings.Contains(normalized, "openai_api_key") || strings.Contains(question, "密钥"):
		return []float64{0.95, 0.10, 0.00}
	case strings.Contains(normalized, "streaming") || strings.Contains(question, "流式"):
		return []float64{0.05, 0.90, 0.10}
	case strings.Contains(normalized, "json") || strings.Contains(question, "结构化"):
		return []float64{0.20, 0.10, 0.90}
	case strings.Contains(normalized, "rag") || strings.Contains(question, "微调"):
		return []float64{0.30, 0.20, 0.50}
	default:
		return []float64{-1.00, 0.00, 0.00}
	}
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
