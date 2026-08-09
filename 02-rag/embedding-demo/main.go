// Embedding 基础演示
//
// 本程序不调用真实 Embedding API，而是用手写向量演示余弦相似度。
// 目的：先理解 RAG 中“问题向量 vs chunk 向量 -> 相似度排序 -> Top-K”的核心逻辑。
package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
)

type textVector struct {
	title  string
	text   string
	vector []float64
}

type scoredText struct {
	item       textVector
	similarity float64
}

func main() {
	question := textVector{
		title:  "用户问题",
		text:   "如何设置 OpenAI API Key？",
		vector: []float64{0.95, 0.10, 0.00},
	}

	documents := []textVector{
		{
			title:  "API Key 配置",
			text:   "运行前需要设置 OPENAI_API_KEY 环境变量。",
			vector: []float64{0.90, 0.20, 0.00},
		},
		{
			title:  "Streaming 流式输出",
			text:   "流式输出可以改善用户等待体验。",
			vector: []float64{0.05, 0.90, 0.10},
		},
		{
			title:  "JSON 结构化输出",
			text:   "结构化输出要求模型返回可解析的 JSON。",
			vector: []float64{0.20, 0.10, 0.90},
		},
	}

	results, err := rankBySimilarity(question, documents)
	if err != nil {
		fmt.Println("计算相似度失败:", err)
		os.Exit(1)
	}

	fmt.Println("用户问题:")
	fmt.Println(question.text)

	fmt.Println("\n问题向量:")
	fmt.Println(question.vector)

	fmt.Println("\n相似度排序:")
	for i, result := range results {
		fmt.Printf("%d. %s similarity=%.3f\n", i+1, result.item.title, result.similarity)
		fmt.Println("   文本:", result.item.text)
		fmt.Println("   向量:", result.item.vector)
	}

	fmt.Println("\n结论:")
	fmt.Println("相似度最高的文档最可能被选入 RAG 的 Top-K 上下文。")
}

func rankBySimilarity(question textVector, documents []textVector) ([]scoredText, error) {
	results := make([]scoredText, 0, len(documents))
	for _, document := range documents {
		similarity, err := cosineSimilarity(question.vector, document.vector)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", document.title, err)
		}
		results = append(results, scoredText{
			item:       document,
			similarity: similarity,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].similarity > results[j].similarity
	})
	return results, nil
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
