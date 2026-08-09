// 文档切分演示
//
// 本程序读取 Markdown 文件，并演示两种常见 chunk 策略：
//  1. heading：按 Markdown 标题切分，保留章节语义
//  2. fixed：按固定字符数切分，方便观察 chunk size 的影响
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type chunk struct {
	Source  string
	Title   string
	Index   int
	Content string
}

type config struct {
	filePath  string
	mode      string
	chunkSize int
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Println("配置错误:", err)
		os.Exit(1)
	}

	content, err := os.ReadFile(cfg.filePath)
	if err != nil {
		fmt.Println("读取 Markdown 文件失败:", err)
		os.Exit(1)
	}

	chunks, err := splitMarkdown(cfg, string(content))
	if err != nil {
		fmt.Println("切分 Markdown 失败:", err)
		os.Exit(1)
	}

	fmt.Println("source:", cfg.filePath)
	fmt.Println("mode:", cfg.mode)
	if cfg.mode == "fixed" {
		fmt.Println("chunk size:", cfg.chunkSize)
	}
	fmt.Println("chunks:", len(chunks))
	fmt.Println()

	for _, c := range chunks {
		fmt.Printf("Chunk #%d\n", c.Index)
		fmt.Println("source:", c.Source)
		fmt.Println("title:", c.Title)
		fmt.Println("chars:", utf8.RuneCountInString(c.Content))
		fmt.Println("preview:", preview(c.Content, 120))
		fmt.Println()
	}
}

func loadConfig() (config, error) {
	filePath := flag.String("file", filepath.Join("notes", "02-rag-basics.md"), "markdown file path")
	mode := flag.String("mode", "heading", "split mode: heading or fixed")
	chunkSize := flag.Int("chunk-size", 500, "max characters per fixed chunk")
	flag.Parse()

	if strings.TrimSpace(*filePath) == "" {
		return config{}, errors.New("file 不能为空")
	}
	if *mode != "heading" && *mode != "fixed" {
		return config{}, errors.New("mode 只能是 heading 或 fixed")
	}
	if *chunkSize <= 0 {
		return config{}, errors.New("chunk-size 必须大于 0")
	}

	return config{
		filePath:  *filePath,
		mode:      *mode,
		chunkSize: *chunkSize,
	}, nil
}

func splitMarkdown(cfg config, content string) ([]chunk, error) {
	switch cfg.mode {
	case "heading":
		return splitByHeading(cfg.filePath, content), nil
	case "fixed":
		return splitByFixedSize(cfg.filePath, content, cfg.chunkSize), nil
	default:
		return nil, errors.New("unsupported split mode")
	}
}

func splitByHeading(source, content string) []chunk {
	lines := strings.Split(content, "\n")
	var chunks []chunk
	var currentTitle string
	var builder strings.Builder

	flush := func() {
		text := strings.TrimSpace(builder.String())
		if text == "" {
			builder.Reset()
			return
		}
		if !hasNonHeadingContent(text) {
			builder.Reset()
			return
		}
		chunks = append(chunks, chunk{
			Source:  source,
			Title:   fallbackTitle(currentTitle),
			Index:   len(chunks) + 1,
			Content: text,
		})
		builder.Reset()
	}

	for _, line := range lines {
		if isMarkdownHeading(line) {
			flush()
			currentTitle = strings.TrimSpace(strings.TrimLeft(line, "#"))
		}
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	flush()

	return chunks
}

func splitByFixedSize(source, content string, chunkSize int) []chunk {
	runes := []rune(strings.TrimSpace(content))
	if len(runes) == 0 {
		return nil
	}

	var chunks []chunk
	for start := 0; start < len(runes); start += chunkSize {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		text := strings.TrimSpace(string(runes[start:end]))
		if text == "" {
			continue
		}
		chunks = append(chunks, chunk{
			Source:  source,
			Title:   fmt.Sprintf("fixed chunk %d", len(chunks)+1),
			Index:   len(chunks) + 1,
			Content: text,
		})
	}

	return chunks
}

func isMarkdownHeading(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "#") {
		return false
	}
	return strings.TrimSpace(strings.TrimLeft(trimmed, "#")) != ""
}

func hasNonHeadingContent(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isMarkdownHeading(trimmed) {
			continue
		}
		return true
	}
	return false
}

func fallbackTitle(title string) string {
	if strings.TrimSpace(title) == "" {
		return "(no heading)"
	}
	return title
}

func preview(content string, limit int) string {
	normalized := strings.Join(strings.Fields(content), " ")
	runes := []rune(normalized)
	if len(runes) <= limit {
		return normalized
	}
	return string(runes[:limit]) + "..."
}
