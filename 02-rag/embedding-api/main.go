// Embedding API 演示
//
// 本程序调用真实 Embedding API，把三段文本转成向量，再用余弦相似度比较：
//  1. 语义相近文本的向量是否更接近
//  2. 语义不相关文本的向量是否更远
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

type config struct {
	model   string
	timeout time.Duration
	mock    bool
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println("配置错误:", err)
		os.Exit(1)
	}

	inputs := []string{
		"怎么配置 OpenAI API Key？",
		"如何设置 OPENAI_API_KEY 环境变量？",
		"流式输出可以改善用户等待体验。",
	}

	vectors, err := loadEmbeddings(cfg, inputs)
	if err != nil {
		fmt.Println("获取 Embedding 向量失败:", err)
		os.Exit(1)
	}

	if len(vectors) != len(inputs) {
		fmt.Printf("Embedding 数量不符合预期: got=%d want=%d\n", len(vectors), len(inputs))
		os.Exit(1)
	}

	similarScore, err := cosineSimilarity(vectors[0], vectors[1])
	if err != nil {
		fmt.Println("计算相似文本相似度失败:", err)
		os.Exit(1)
	}
	unrelatedScore, err := cosineSimilarity(vectors[0], vectors[2])
	if err != nil {
		fmt.Println("计算不相关文本相似度失败:", err)
		os.Exit(1)
	}

	fmt.Println("Embedding model:", cfg.model)
	fmt.Println("运行模式:", runMode(cfg))
	fmt.Println("向量维度:", len(vectors[0]))
	fmt.Println()

	for i, input := range inputs {
		fmt.Printf("文本 %d: %s\n", i+1, input)
		fmt.Println("向量前 8 维:", previewVector(vectors[i], 8))
		fmt.Println()
	}

	fmt.Println("相似度:")
	fmt.Printf("文本 1 vs 文本 2 = %.4f  （都在说 API Key 配置，应该更高）\n", similarScore)
	fmt.Printf("文本 1 vs 文本 3 = %.4f  （主题不同，应该更低）\n", unrelatedScore)
	fmt.Println()

	if similarScore > unrelatedScore {
		fmt.Println("结论: Embedding 向量能体现语义相似度，适合用于 RAG 检索。")
		return
	}
	fmt.Println("结论: 本次结果不符合预期，建议检查模型、输入文本或向量生成逻辑。")
}

func loadConfig() (config, error) {
	model := flag.String("model", "text-embedding-3-small", "embedding model")
	timeoutSeconds := flag.Int("timeout-seconds", 30, "request timeout in seconds")
	mock := flag.Bool("mock", false, "use local mock embeddings without calling API")
	flag.Parse()

	if *model == "" {
		return config{}, errors.New("model 不能为空")
	}
	if *timeoutSeconds <= 0 {
		return config{}, errors.New("timeout-seconds 必须大于 0")
	}

	return config{
		model:   *model,
		timeout: time.Duration(*timeoutSeconds) * time.Second,
		mock:    *mock,
	}, nil
}

func loadEmbeddings(cfg config, inputs []string) ([][]float32, error) {
	if cfg.mock {
		return mockEmbeddings(inputs)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("请先设置环境变量 OPENAI_API_KEY，或使用 -mock 运行本地模拟版")
	}

	client := openai.NewClient(apiKey)
	return createEmbeddings(client, cfg, inputs)
}

func createEmbeddings(client *openai.Client, cfg config, inputs []string) ([][]float32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	resp, err := client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(cfg.model),
		Input: inputs,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, errors.New("Embedding API 没有返回向量")
	}

	vectors := make([][]float32, 0, len(resp.Data))
	for _, item := range resp.Data {
		vectors = append(vectors, item.Embedding)
	}
	return vectors, nil
}

func mockEmbeddings(inputs []string) ([][]float32, error) {
	if len(inputs) != 3 {
		return nil, fmt.Errorf("mock 模式期望 3 段输入，实际收到 %d 段", len(inputs))
	}
	return [][]float32{
		{0.95, 0.10, 0.00, 0.05, 0.02, 0.00, 0.03, 0.01},
		{0.90, 0.18, 0.02, 0.06, 0.03, 0.00, 0.02, 0.01},
		{0.04, 0.88, 0.14, 0.02, 0.03, 0.05, 0.01, 0.00},
	}, nil
}

func cosineSimilarity(a, b []float32) (float64, error) {
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
		x := float64(a[i])
		y := float64(b[i])
		dotProduct += x * y
		normA += x * x
		normB += y * y
	}

	if normA == 0 || normB == 0 {
		return 0, errors.New("零向量不能计算余弦相似度")
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)), nil
}

func previewVector(vector []float32, size int) []float32 {
	if size > len(vector) {
		size = len(vector)
	}
	return vector[:size]
}

func runMode(cfg config) string {
	if cfg.mock {
		return "mock，本地模拟向量，不调用 API"
	}
	return "api，调用真实 Embedding API"
}
