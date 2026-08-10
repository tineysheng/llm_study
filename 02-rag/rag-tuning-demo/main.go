// RAG 调优演示
//
// 本程序用 mock 向量和 mock 答案演示 RAG 调优的核心观察点：
//  1. 无 RAG 与有 RAG 的区别
//  2. Top-K 变大后的召回和噪声变化
//  3. threshold 如何决定回答还是说不知道
//  4. chunk size 太小、适中、太大分别带来的影响
package main

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"unicode/utf8"
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

type chunkPolicy struct {
	Name        string
	Description string
	Items       []vectorItem
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println("配置错误:", err)
		os.Exit(1)
	}

	queryVector := mockQueryVector(cfg.question)
	store := newStore(mediumChunkItems())
	results, err := store.Search(queryVector, cfg.topK)
	if err != nil {
		fmt.Println("检索失败:", err)
		os.Exit(1)
	}

	fmt.Println("RAG 调优实验")
	fmt.Println("用户问题:", cfg.question)
	fmt.Println("query vector:", queryVector)
	fmt.Printf("当前参数: top-k=%d threshold=%.2f\n", cfg.topK, cfg.threshold)
	fmt.Println()

	printDirectVsRAG(cfg, results)
	printTopKSweep(cfg.question, queryVector, 4)
	printThresholdSweep(cfg.question, results)
	printChunkPolicySweep(cfg.question, queryVector)
	printInterviewSummary()
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

	return config{question: *question, topK: *topK, threshold: *threshold}, nil
}

func newStore(items []vectorItem) memoryVectorStore {
	return memoryVectorStore{items: items}
}

func (s memoryVectorStore) Search(queryVector []float64, topK int) ([]searchResult, error) {
	if len(s.items) == 0 {
		return nil, errors.New("向量库为空")
	}
	if topK <= 0 {
		return nil, errors.New("top-k 必须大于 0")
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

func printDirectVsRAG(cfg config, results []searchResult) {
	fmt.Println("=== 1. 无 RAG vs 有 RAG ===")
	fmt.Println("无 RAG 直接回答:")
	fmt.Println(directAnswer(cfg.question))
	fmt.Println()
	fmt.Println("有 RAG 回答:")
	fmt.Println(generateAnswer(results, cfg.threshold))
	fmt.Println()
}

func printTopKSweep(question string, queryVector []float64, maxK int) {
	fmt.Println("=== 2. Top-K 调优观察 ===")
	store := newStore(mediumChunkItems())
	goldTitle := expectedTitle(question)
	fmt.Printf("%-6s %-8s %-8s %-10s %s\n", "top-k", "top1", "chars", "noise", "titles")
	for k := 1; k <= maxK; k++ {
		results, err := store.Search(queryVector, k)
		if err != nil {
			fmt.Println("检索失败:", err)
			continue
		}
		fmt.Printf("%-6d %-8.3f %-8d %-10d %s\n", k, results[0].Score, contextChars(results), noiseCount(results, goldTitle), joinedTitles(results))
	}
	fmt.Println("观察：Top-K 变大通常能多召回候选资料，但也会增加上下文长度、成本和无关 chunk 风险。")
	fmt.Println()
}

func printThresholdSweep(question string, results []searchResult) {
	fmt.Println("=== 3. Threshold 调优观察 ===")
	thresholds := []float64{0.50, 0.75, 0.95}
	if len(results) == 0 {
		fmt.Println("没有检索结果。")
		fmt.Println()
		return
	}
	fmt.Printf("当前问题: %s\n", question)
	fmt.Printf("Top-1 score: %.3f (%s)\n", results[0].Score, results[0].Item.Chunk.Title)
	for _, threshold := range thresholds {
		action := "回答"
		if results[0].Score < threshold {
			action = "说不知道"
		}
		fmt.Printf("threshold=%.2f -> %s\n", threshold, action)
	}
	fmt.Println("观察：阈值低，系统更愿意回答但更容易错；阈值高，系统更谨慎但可能拒答本来能答的问题。")
	fmt.Println()
}

func printChunkPolicySweep(question string, queryVector []float64) {
	fmt.Println("=== 4. Chunk Size 调优观察 ===")
	policies := []chunkPolicy{
		{
			Name:        "small",
			Description: "chunk 太小：信息被拆散，Top-1 可能只拿到半句话",
			Items:       smallChunkItems(),
		},
		{
			Name:        "medium",
			Description: "chunk 适中：答案完整，噪声较少",
			Items:       mediumChunkItems(),
		},
		{
			Name:        "large",
			Description: "chunk 太大：答案可能完整，但混入更多无关内容",
			Items:       largeChunkItems(),
		},
	}

	goldTitle := expectedTitle(question)
	fmt.Printf("%-8s %-8s %-8s %-10s %s\n", "policy", "top1", "chars", "noise", "observation")
	for _, policy := range policies {
		results, err := newStore(policy.Items).Search(queryVector, 2)
		if err != nil {
			fmt.Println("检索失败:", err)
			continue
		}
		fmt.Printf("%-8s %-8.3f %-8d %-10d %s\n", policy.Name, results[0].Score, contextChars(results), noiseCount(results, goldTitle), policy.Description)
	}
	fmt.Println("观察：chunk size 没有唯一最优值，要结合文档结构、问题类型、成本预算和评测结果调整。")
	fmt.Println()
}

func printInterviewSummary() {
	fmt.Println("=== 5. 面试表达提示 ===")
	fmt.Println("RAG 调优不是只调一个参数，而是围绕检索质量、上下文噪声、生成约束和可观测性做平衡。")
	fmt.Println("常见排查顺序：先看 Top-K 是否召回正确 chunk，再看 chunk 是否完整，再看 Prompt 是否约束模型，最后看答案引用是否真实对应上下文。")
}

func directAnswer(question string) string {
	if strings.TrimSpace(expectedTitle(question)) == "" {
		return "模型可能会基于通用知识尝试回答，但无法确认你的本地知识库里是否有依据。"
	}
	return "你可以查看相关文档或环境变量配置说明。这个回答没有检索本地知识库，也没有引用来源。"
}

func generateAnswer(results []searchResult, threshold float64) string {
	if len(results) == 0 || results[0].Score < threshold {
		return "根据当前知识库上下文，我不知道答案。"
	}

	var builder strings.Builder
	builder.WriteString(results[0].Item.Chunk.Content)
	builder.WriteString("\n\n来源:\n")
	for _, result := range results {
		if result.Score < threshold {
			continue
		}
		c := result.Item.Chunk
		fmt.Fprintf(&builder, "- %s / %s / chunk #%d\n", c.Source, c.Title, c.Index)
	}
	return strings.TrimSpace(builder.String())
}

func contextChars(results []searchResult) int {
	total := 0
	for _, result := range results {
		total += utf8.RuneCountInString(result.Item.Chunk.Content)
	}
	return total
}

func noiseCount(results []searchResult, goldTitle string) int {
	if goldTitle == "" {
		return len(results)
	}
	count := 0
	for _, result := range results {
		if result.Item.Chunk.Title != goldTitle {
			count++
		}
	}
	return count
}

func joinedTitles(results []searchResult) string {
	titles := make([]string, 0, len(results))
	for _, result := range results {
		titles = append(titles, result.Item.Chunk.Title)
	}
	return strings.Join(titles, " | ")
}

func expectedTitle(question string) string {
	normalized := strings.ToLower(question)
	switch {
	case strings.Contains(normalized, "api key") || strings.Contains(normalized, "openai_api_key") || strings.Contains(question, "密钥"):
		return "API Key 配置"
	case strings.Contains(normalized, "streaming") || strings.Contains(question, "流式"):
		return "Streaming 流式输出"
	case strings.Contains(normalized, "json") || strings.Contains(question, "结构化"):
		return "JSON 结构化输出"
	case strings.Contains(normalized, "rag") || strings.Contains(question, "微调"):
		return "RAG 和微调的区别"
	default:
		return ""
	}
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

func smallChunkItems() []vectorItem {
	return []vectorItem{
		newItem("notes\\01-core-concepts.md", "API Key 配置", 1, "运行 LLM CLI 前需要配置 OpenAI API Key。", []float64{0.94, 0.12, 0.00}),
		newItem("notes\\01-core-concepts.md", "API Key 配置", 2, "在 PowerShell 中设置环境变量：$env:OPENAI_API_KEY = \"sk-你的key\"。", []float64{0.92, 0.14, 0.00}),
		newItem("notes\\01-core-concepts.md", "Streaming 流式输出", 3, "Streaming 流式输出主要改善用户等待体验。", []float64{0.05, 0.90, 0.10}),
		newItem("notes\\01-core-concepts.md", "JSON 结构化输出", 4, "JSON 输出方便 Go 程序解析成结构体。", []float64{0.20, 0.10, 0.90}),
	}
}

func mediumChunkItems() []vectorItem {
	return []vectorItem{
		newItem("notes\\01-core-concepts.md", "API Key 配置", 1, "运行本项目的 LLM CLI 前，需要在 PowerShell 中设置 OPENAI_API_KEY 环境变量。", []float64{0.90, 0.20, 0.00}),
		newItem("notes\\01-core-concepts.md", "Streaming 流式输出", 2, "Streaming 流式输出主要改善用户等待体验，但不会让模型最终答案本身更聪明。", []float64{0.05, 0.90, 0.10}),
		newItem("notes\\01-core-concepts.md", "JSON 结构化输出", 3, "业务系统需要稳定、可解析的数据结构，模型返回 JSON 后，Go 程序可以解析成结构体。", []float64{0.20, 0.10, 0.90}),
		newItem("notes\\02-rag-basics.md", "RAG 和微调的区别", 4, "RAG 更适合经常变化的事实知识和私有文档；微调更适合稳定任务模式和回答风格。", []float64{0.30, 0.20, 0.50}),
	}
}

func largeChunkItems() []vectorItem {
	return []vectorItem{
		newItem("notes\\01-core-concepts.md", "API Key 配置", 1, "运行本项目的 LLM CLI 前，需要在 PowerShell 中设置 OPENAI_API_KEY 环境变量。本段同时还提到：Streaming 主要改善等待体验；JSON 结构化输出适合业务系统解析；temperature 会影响输出稳定性。", []float64{0.88, 0.25, 0.08}),
		newItem("notes\\02-rag-basics.md", "RAG 和微调的区别", 2, "RAG 更适合经常变化的事实知识和私有文档；微调更适合稳定任务模式和回答风格。", []float64{0.30, 0.20, 0.50}),
		newItem("notes\\01-core-concepts.md", "Streaming 流式输出", 3, "Streaming 流式输出主要改善用户等待体验，但不会让模型最终答案本身更聪明。", []float64{0.05, 0.90, 0.10}),
	}
}

func newItem(source, title string, index int, content string, vector []float64) vectorItem {
	return vectorItem{
		Chunk:  chunk{Source: source, Title: title, Index: index, Content: content},
		Vector: vector,
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
