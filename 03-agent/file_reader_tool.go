package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type SafeFileReaderTool struct {
	RootDir  string
	MaxBytes int
}

func NewSafeFileReaderTool(rootDir string, maxBytes int) SafeFileReaderTool {
	if maxBytes <= 0 {
		maxBytes = 4096
	}
	return SafeFileReaderTool{RootDir: rootDir, MaxBytes: maxBytes}
}

func (tool SafeFileReaderTool) Schema() ToolSchema {
	return ToolSchema{
		Name:        "file_reader",
		Description: "仅用于读取安全示例目录内的学习资料文件。只能读取相对路径的 .md 或 .txt 文件，不能读取绝对路径、上级目录或系统文件。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"relative_path": map[string]any{
					"type":        "string",
					"description": "安全示例目录内的相对文件路径，例如 agent-safety.md。不允许 ..、绝对路径、盘符或通配符。",
				},
			},
			"required": []string{"relative_path"},
		},
	}
}

func (tool SafeFileReaderTool) Execute(rawArguments string) (ToolResult, error) {
	args, err := parseFileReaderArgs(rawArguments)
	if err != nil {
		return ToolResult{}, err
	}

	safePath, err := tool.safePath(args.RelativePath)
	if err != nil {
		return ToolResult{}, err
	}

	file, err := os.Open(safePath)
	if err != nil {
		return ToolResult{}, fmt.Errorf("读取文件失败: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, int64(tool.MaxBytes)+1))
	if err != nil {
		return ToolResult{}, fmt.Errorf("读取文件内容失败: %w", err)
	}
	if len(data) > tool.MaxBytes {
		return ToolResult{}, fmt.Errorf("文件超过读取上限: max_bytes=%d", tool.MaxBytes)
	}
	if !utf8.Valid(data) {
		return ToolResult{}, errors.New("file_reader 只允许读取 UTF-8 文本文件")
	}

	return ToolResult{Name: "file_reader", Output: string(data)}, nil
}

func (tool SafeFileReaderTool) safePath(relativePath string) (string, error) {
	cleaned, err := cleanSafeRelativePath(relativePath)
	if err != nil {
		return "", err
	}

	rootAbs, err := filepath.Abs(tool.RootDir)
	if err != nil {
		return "", fmt.Errorf("解析安全目录失败: %w", err)
	}
	rootEval, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return "", fmt.Errorf("安全目录不可用: %w", err)
	}

	targetPath := filepath.Join(rootEval, cleaned)
	targetEval, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		return "", fmt.Errorf("文件不存在或不可访问: %w", err)
	}

	rel, err := filepath.Rel(rootEval, targetEval)
	if err != nil {
		return "", fmt.Errorf("校验文件路径失败: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("拒绝读取安全目录之外的文件")
	}

	info, err := os.Stat(targetEval)
	if err != nil {
		return "", fmt.Errorf("读取文件信息失败: %w", err)
	}
	if info.IsDir() {
		return "", errors.New("file_reader 只能读取文件，不能读取目录")
	}

	return targetEval, nil
}

type FileReaderArgs struct {
	RelativePath string `json:"relative_path"`
}

func parseFileReaderArgs(raw string) (FileReaderArgs, error) {
	if strings.TrimSpace(raw) == "" {
		return FileReaderArgs{}, errors.New("file_reader arguments 不能为空")
	}

	var payload struct {
		RelativePath *string `json:"relative_path"`
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return FileReaderArgs{}, fmt.Errorf("解析 file_reader arguments 失败: %w", err)
	}
	if payload.RelativePath == nil || strings.TrimSpace(*payload.RelativePath) == "" {
		return FileReaderArgs{}, errors.New("file_reader 缺少必填参数: relative_path")
	}

	path, err := cleanSafeRelativePath(*payload.RelativePath)
	if err != nil {
		return FileReaderArgs{}, err
	}
	return FileReaderArgs{RelativePath: path}, nil
}

func cleanSafeRelativePath(rawPath string) (string, error) {
	path := strings.TrimSpace(rawPath)
	if path == "" {
		return "", errors.New("relative_path 不能为空")
	}
	if strings.ContainsRune(path, 0) {
		return "", errors.New("relative_path 不能包含空字符")
	}
	if strings.ContainsAny(path, "*?") {
		return "", errors.New("relative_path 不允许使用通配符")
	}
	if filepath.IsAbs(path) || filepath.VolumeName(path) != "" {
		return "", errors.New("relative_path 必须是相对路径，不能是绝对路径或带盘符路径")
	}

	cleaned := filepath.Clean(path)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", errors.New("relative_path 不能跳出安全目录")
	}

	ext := strings.ToLower(filepath.Ext(cleaned))
	if ext != ".md" && ext != ".txt" {
		return "", errors.New("file_reader 只允许读取 .md 或 .txt 文件")
	}

	return cleaned, nil
}

func defaultSafeFilesDir() string {
	candidates := []string{
		filepath.Join("03-agent", "safe-files"),
		"safe-files",
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate
		}
	}
	return filepath.Join("03-agent", "safe-files")
}
