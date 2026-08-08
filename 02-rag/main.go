// 阶段 2：RAG 最小演示
//
// 本程序先不接真实 Embedding 和向量库，而是用一个小型本地知识库 + 关键词打分
// 演示 RAG 的核心流程：先检索相关资料，再把资料作为上下文交给模型回答。
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

type document struct {
	source  string
	title   string
	content string
}

type retrievedDocument struct {
	document document
	score    int
}

var knowledgeBase = []document{
	{
		source:  "notes/01-core-concepts.md",
		title:   "LLM API Key 配置",
		content: "运行本项目的 LLM CLI 前，需要在 PowerShell 中设置 OPENAI_API_KEY 环境变量，例如：$env:OPENAI_API_KEY = \"sk-你的key\"。",
	},
	{
		source:  "notes/01-core-concepts.md",
		title:   "Streaming 的作用",
		content: "Streaming 流式输出主要改善用户等待体验，让用户看到模型正在持续生成内容，但它不会让模型最终答案本身更聪明。",
	},
	{
		source:  "notes/01-core-concepts.md",
		title:   "结构化输出",
		content: "业务系统需要稳定、可解析的数据结构。让模型返回 JSON 后，Go 程序可以用 encoding/json 解析成结构体，并处理解析失败。",
	},
	{
		source:  "README.md",
		title:   "RAG 的定位",
		content: "RAG 检索增强生成会先从外部知识库检索相关文档，再把检索结果拼进 Prompt，让模型基于上下文回答问题并输出引用来源。",
	},
	{
		source:  "README.md",
		title:   "RAG 和微调的区别",
		content: "RAG 更适合经常变化的事实知识和私有文档；微调更适合让模型学习稳定的任务模式、回答风格或领域表达习惯。",
	},
}

func main() {
	question := flag.String("question", "怎么设置 OPENAI_API_KEY？", "user question")
	topK := flag.Int("top-k", 2, "number of documents to retrieve")
	flag.Parse()

	if strings.TrimSpace(*question) == "" {
		fmt.Println("question 不能为空")
		os.Exit(1)
	}
	if *topK <= 0 {
		fmt.Println("top-k 必须大于 0")
		os.Exit(1)
	}

	results := retrieve(*question, knowledgeBase, *topK)

	fmt.Println("用户问题:")
	fmt.Println(*question)

	fmt.Println("\n不使用 RAG 的回答:")
	fmt.Println(answerWithoutRAG(*question))

	fmt.Println("\n检索到的上下文:")
	for i, result := range results {
		fmt.Printf("%d. [%s] %s score=%d\n", i+1, result.document.source, result.document.title, result.score)
		fmt.Println("   " + result.document.content)
	}

	fmt.Println("\n拼给模型的 Prompt:")
	fmt.Println(buildPrompt(*question, results))

	fmt.Println("\n使用 RAG 后的回答:")
	fmt.Println(answerWithRAG(results))
}

func retrieve(question string, documents []document, topK int) []retrievedDocument {
	results := make([]retrievedDocument, 0, len(documents))
	for _, doc := range documents {
		results = append(results, retrievedDocument{
			document: doc,
			score:    score(question, doc),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].score == results[j].score {
			return results[i].document.title < results[j].document.title
		}
		return results[i].score > results[j].score
	})

	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK]
}

func score(question string, doc document) int {
	q := strings.ToLower(question)
	text := strings.ToLower(doc.title + " " + doc.content)
	keywords := []string{
		"openai_api_key",
		"api key",
		"rag",
		"检索",
		"知识库",
		"上下文",
		"引用",
		"来源",
		"微调",
		"私有",
		"文档",
		"streaming",
		"流式",
		"json",
		"结构化",
		"prompt",
	}

	total := 0
	for _, keyword := range keywords {
		if strings.Contains(q, keyword) && strings.Contains(text, keyword) {
			total += 2
		}
	}
	return total
}

func answerWithoutRAG(question string) string {
	return fmt.Sprintf("我只能根据模型已有知识回答，无法确认你的本地项目文档中是否有关于 %q 的具体要求。", question)
}

func buildPrompt(question string, results []retrievedDocument) string {
	var builder strings.Builder
	builder.WriteString("请只根据下面的上下文回答问题；如果上下文没有答案，就说不知道。\n\n")
	builder.WriteString("上下文:\n")
	for i, result := range results {
		fmt.Fprintf(&builder, "[%d] 来源: %s\n标题: %s\n内容: %s\n\n", i+1, result.document.source, result.document.title, result.document.content)
	}
	builder.WriteString("问题: ")
	builder.WriteString(question)
	return builder.String()
}

func answerWithRAG(results []retrievedDocument) string {
	if len(results) == 0 || results[0].score == 0 {
		return "根据当前知识库上下文，我不知道答案。"
	}
	return fmt.Sprintf("%s\n来源：%s", results[0].document.content, results[0].document.source)
}
