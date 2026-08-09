// Package utils 提供「炼丹炉」项目的通用工具函数
// 包括文件类型判断、ID 生成、字符串处理等辅助功能
package utils

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"time"
)

// 支持的文件类型常量，用于丹方文件类型判断
const (
	// FileTypeWord Word 文档
	FileTypeWord = "doc"
	// FileTypeExcel Excel 表格
	FileTypeExcel = "xlsx"
	// FileTypeMarkdown Markdown 文档
	FileTypeMarkdown = "md"
	// FileTypeText 纯文本
	FileTypeText = "txt"
	// FileTypePDF PDF 文档
	FileTypePDF = "pdf"
	// FileTypeAudio 音频文件
	FileTypeAudio = "audio"
	// FileTypeVideo 视频文件
	FileTypeVideo = "video"
	// FileTypeUnknown 未知类型
	FileTypeUnknown = "unknown"
)

// allowedExtensions 定义允许上传的文件扩展名到文件类型的映射
var allowedExtensions = map[string]string{
	".doc":  FileTypeWord,
	".docx": FileTypeWord,
	".xls":  FileTypeExcel,
	".xlsx": FileTypeExcel,
	".md":   FileTypeMarkdown,
	".txt":  FileTypeText,
	".pdf":  FileTypePDF,
	".mp3":  FileTypeAudio,
	".wav":  FileTypeAudio,
	".m4a":  FileTypeAudio,
	".mp4":  FileTypeVideo,
	".avi":  FileTypeVideo,
	".mov":  FileTypeVideo,
}

// GetFileType 根据文件名判断文件类型
// 返回对应的文件类型字符串，如 "doc", "xlsx", "audio" 等
func GetFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if fileType, ok := allowedExtensions[ext]; ok {
		return fileType
	}
	return FileTypeUnknown
}

// IsAllowedFileType 判断文件类型是否允许上传
func IsAllowedFileType(filename string) bool {
	return GetFileType(filename) != FileTypeUnknown
}

// IsImageFile 判断文件是否为图片类型（用于头像等）
func IsImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg":
		return true
	}
	return false
}

// GetFileExt 获取文件扩展名（不带点）
func GetFileExt(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	return strings.TrimPrefix(ext, ".")
}

// GenerateUniqueFilename 生成唯一的文件名，避免上传文件重名
// 格式: {原始文件名(不含扩展名)}_{时间戳}_{随机数}.{扩展名}
func GenerateUniqueFilename(originalName string) string {
	ext := filepath.Ext(originalName)
	base := strings.TrimSuffix(originalName, ext)
	// 清理原始文件名中的特殊字符
	base = strings.ReplaceAll(base, " ", "_")
	base = strings.ReplaceAll(base, "/", "_")
	base = strings.ReplaceAll(base, "\\", "_")
	if len(base) > 50 {
		base = base[:50]
	}

	timestamp := time.Now().UnixNano() / 1e6 // 毫秒时间戳
	randomNum := rand.Intn(10000)            // 0-9999 随机数

	return fmt.Sprintf("%s_%d_%04d%s", base, timestamp, randomNum, ext)
}

// FormatFileSize 格式化文件大小，返回人类可读的字符串（如 "1.5 MB"）
func FormatFileSize(sizeBytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case sizeBytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(sizeBytes)/float64(GB))
	case sizeBytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(sizeBytes)/float64(MB))
	case sizeBytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(sizeBytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", sizeBytes)
	}
}

// GenerateSessionID 生成唯一的会话 ID（用于 WebSocket 会话标识）
func GenerateSessionID() string {
	return fmt.Sprintf("session_%d_%d", time.Now().UnixNano(), rand.Intn(100000))
}

// SanitizeFileName 清理文件名，移除潜在的危险字符
func SanitizeFileName(filename string) string {
	// 移除路径分隔符和空字符
	filename = filepath.Base(filename)
	filename = strings.ReplaceAll(filename, "\x00", "")
	filename = strings.ReplaceAll(filename, "..", "_")
	return filename
}

// TruncateString 截断字符串到指定长度，超出部分用省略号表示
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// RemoveEmptyStrings 从字符串切片中移除空字符串
func RemoveEmptyStrings(strs []string) []string {
	result := make([]string, 0, len(strs))
	for _, s := range strs {
		if strings.TrimSpace(s) != "" {
			result = append(result, s)
		}
	}
	return result
}

// init 初始化随机数种子
func init() {
	rand.Seed(time.Now().UnixNano())
}
