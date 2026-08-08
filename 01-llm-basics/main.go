// 阶段 1：流式多轮聊天 CLI
//
// 本程序演示 LLM 应用最基础的骨架：
//  1. 调用 OpenAI Chat Completions API
//  2. 流式输出（SSE），打字机效果
//  3. 多轮对话：LLM API 是无状态的，历史消息由我们自己维护并全量回传
//
// 运行前：设置环境变量 OPENAI_API_KEY
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

type chatConfig struct {
	model       string
	temperature float32
	maxTokens   int
	timeout     time.Duration
}

type learningCard struct {
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	Keywords   []string `json:"keywords"`
	Difficulty string   `json:"difficulty"`
	NextAction string   `json:"next_action"`
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println("配置错误:", err)
		os.Exit(1)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("请先设置环境变量 OPENAI_API_KEY")
		os.Exit(1)
	}

	client := openai.NewClient(apiKey)
	fmt.Printf("当前配置：model=%s temperature=%.2f max_tokens=%d timeout=%s\n", cfg.model, cfg.temperature, cfg.maxTokens, cfg.timeout)

	// messages 就是"对话记忆"。system 消息定义模型行为，放在最前面。
	// 注意：随着轮次增加，这个切片会无限增长——这正是阶段 2/3 要解决的上下文管理问题。
	systemMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "你是一个简洁、专业的 Go 编程助手，回答时优先给出代码示例。",
	}
	messages := []openai.ChatCompletionMessage{
		systemMessage,
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("聊天开始，输入 /help 查看命令。")
	for {
		fmt.Print("\n你: ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Println("读取输入出错:", err)
			}
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if strings.HasPrefix(input, "/json ") {
			topic := strings.TrimSpace(strings.TrimPrefix(input, "/json "))
			if topic == "" {
				fmt.Println("用法：/json 你想总结的主题")
				continue
			}
			card, raw, err := generateLearningCard(client, cfg, topic)
			if err != nil {
				fmt.Println("结构化输出解析失败:", err)
				fmt.Println("模型原始输出:", raw)
				continue
			}
			printLearningCard(card)
			continue
		}

		switch input {
		case "quit", "/exit":
			return
		case "/help":
			printHelp()
			continue
		case "/config":
			printConfig(cfg)
			continue
		case "/reset":
			messages = []openai.ChatCompletionMessage{systemMessage}
			fmt.Println("已清空对话历史，只保留 system prompt。")
			continue
		case "/history":
			printHistory(messages)
			continue
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: input,
		})

		reply, err := streamChat(client, messages, cfg)
		if err != nil {
			fmt.Println("调用出错:", err)
			// 请求失败时把刚才那条 user 消息弹掉，避免历史里留下没有回复的消息
			messages = messages[:len(messages)-1]
			continue
		}

		// 把模型的完整回复回填进历史，下一轮请求才能"记得"自己说过什么
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: reply,
		})
	}
}

func loadConfig() (chatConfig, error) {
	defaultModel := getEnvString("OPENAI_MODEL", openai.GPT4oMini)
	defaultTemperature, err := getEnvFloat32("OPENAI_TEMPERATURE", 0.7)
	if err != nil {
		return chatConfig{}, err
	}
	defaultMaxTokens, err := getEnvInt("OPENAI_MAX_TOKENS", 800)
	if err != nil {
		return chatConfig{}, err
	}
	defaultTimeoutSeconds, err := getEnvInt("OPENAI_TIMEOUT_SECONDS", 60)
	if err != nil {
		return chatConfig{}, err
	}

	model := flag.String("model", defaultModel, "model name, for example gpt-4o-mini")
	temperature := flag.Float64("temperature", float64(defaultTemperature), "sampling temperature, range 0-2")
	maxTokens := flag.Int("max-tokens", defaultMaxTokens, "maximum output tokens")
	timeoutSeconds := flag.Int("timeout-seconds", defaultTimeoutSeconds, "request timeout in seconds")
	flag.Parse()

	if *model == "" {
		return chatConfig{}, errors.New("model 不能为空")
	}
	if *temperature < 0 || *temperature > 2 {
		return chatConfig{}, errors.New("temperature 必须在 0 到 2 之间")
	}
	if *maxTokens <= 0 {
		return chatConfig{}, errors.New("max-tokens 必须大于 0")
	}
	if *timeoutSeconds <= 0 {
		return chatConfig{}, errors.New("timeout-seconds 必须大于 0")
	}

	return chatConfig{
		model:       *model,
		temperature: float32(*temperature),
		maxTokens:   *maxTokens,
		timeout:     time.Duration(*timeoutSeconds) * time.Second,
	}, nil
}

func getEnvString(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvFloat32(key string, fallback float32) (float32, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseFloat(value, 32)
	if err != nil {
		return 0, fmt.Errorf("%s 必须是数字: %w", key, err)
	}
	return float32(parsed), nil
}

func getEnvInt(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s 必须是整数: %w", key, err)
	}
	return parsed, nil
}

func printHelp() {
	fmt.Println(`
可用命令：
	/help     查看命令说明
	/config   查看当前模型参数
	/history  查看当前对话历史
	/json     生成结构化 JSON 学习卡片，例如：/json Go channel
	/reset    清空对话历史，只保留 system prompt
	/exit     退出程序
	quit      退出程序
`)
}

func printConfig(cfg chatConfig) {
	fmt.Printf("model=%s temperature=%.2f max_tokens=%d timeout=%s\n", cfg.model, cfg.temperature, cfg.maxTokens, cfg.timeout)
}

func printLearningCard(card learningCard) {
	fmt.Println("结构化学习卡片：")
	fmt.Println("标题:", card.Title)
	fmt.Println("摘要:", card.Summary)
	fmt.Println("关键词:", strings.Join(card.Keywords, "、"))
	fmt.Println("难度:", card.Difficulty)
	fmt.Println("下一步:", card.NextAction)
}

func printHistory(messages []openai.ChatCompletionMessage) {
	fmt.Printf("当前共有 %d 条消息：\n", len(messages))
	for i, msg := range messages {
		content := strings.ReplaceAll(msg.Content, "\n", " ")
		runes := []rune(content)
		if len(runes) > 40 {
			content = string(runes[:40]) + "..."
		}
		fmt.Printf("  %d. role=%s content=%q\n", i+1, msg.Role, content)
	}
}

func generateLearningCard(client *openai.Client, cfg chatConfig, topic string) (learningCard, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	req := openai.ChatCompletionRequest{
		Model:       cfg.model,
		Temperature: 0.2,
		MaxTokens:   cfg.maxTokens,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: `你是一个严格的 JSON 生成器。只返回 JSON，不要返回 Markdown，不要返回代码块。
JSON 必须包含这些字段：
{
  "title": "string",
  "summary": "string",
  "keywords": ["string"],
  "difficulty": "beginner|intermediate|advanced",
  "next_action": "string"
}`,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "请把这个学习主题整理成结构化学习卡片：" + topic,
			},
		},
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return learningCard{}, "", err
	}
	if len(resp.Choices) == 0 {
		return learningCard{}, "", errors.New("模型没有返回 choices")
	}

	raw := strings.TrimSpace(resp.Choices[0].Message.Content)
	var card learningCard
	if err := json.Unmarshal([]byte(raw), &card); err != nil {
		return learningCard{}, raw, err
	}
	if card.Title == "" || card.Summary == "" || len(card.Keywords) == 0 || card.Difficulty == "" || card.NextAction == "" {
		return learningCard{}, raw, errors.New("JSON 字段不完整")
	}

	return card, raw, nil
}

// streamChat 发起流式请求并边收边打印，返回拼接后的完整回复。
func streamChat(client *openai.Client, messages []openai.ChatCompletionMessage, cfg chatConfig) (string, error) {
	req := openai.ChatCompletionRequest{
		Model:       cfg.model,
		Messages:    messages,
		Temperature: cfg.temperature,
		MaxTokens:   cfg.maxTokens,
		Stream:      true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	fmt.Print("\nAI: ")
	var full strings.Builder
	for {
		resp, err := stream.Recv()
		// 流结束的标志是 io.EOF，不是 error
		if errors.Is(err, io.EOF) {
			fmt.Println()
			return full.String(), nil
		}
		if err != nil {
			return "", err
		}

		// 流式响应的每个 chunk 只携带增量内容 delta，需自行拼接
		delta := resp.Choices[0].Delta.Content
		fmt.Print(delta)
		full.WriteString(delta)
	}
}
