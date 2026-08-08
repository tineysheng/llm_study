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
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("请先设置环境变量 OPENAI_API_KEY")
		os.Exit(1)
	}

	client := openai.NewClient(apiKey)

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

		switch input {
		case "quit", "/exit":
			return
		case "/help":
			printHelp()
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

		reply, err := streamChat(client, messages)
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

func printHelp() {
	fmt.Println(`
可用命令：
  /help     查看命令说明
  /history  查看当前对话历史
  /reset    清空对话历史，只保留 system prompt
  /exit     退出程序
  quit      退出程序
`)
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

// streamChat 发起流式请求并边收边打印，返回拼接后的完整回复。
func streamChat(client *openai.Client, messages []openai.ChatCompletionMessage) (string, error) {
	req := openai.ChatCompletionRequest{
		Model:       openai.GPT4oMini, // 学习阶段用 mini，成本几乎可忽略
		Messages:    messages,
		Temperature: 0.7, // 0 最确定，2 最发散，改改它观察区别（任务 1.5）
		Stream:      true,
	}

	stream, err := client.CreateChatCompletionStream(context.Background(), req)
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
