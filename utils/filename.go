package utils

import (
    "regexp"
    "strings"
    "unicode"
    "unicode/utf8"
)

// CleanFilename 清理文件名，移除不安全字符和控制字符
func CleanFilename(filename string) string {
    if !utf8.ValidString(filename) {
        // 如果不是有效的UTF-8字符串，尝试修复
        filename = strings.ToValidUTF8(filename, "")
    }
    
    // 移除控制字符
    result := strings.Map(func(r rune) rune {
        if unicode.IsControl(r) {
            return -1 // 移除控制字符
        }
        return r
    }, filename)
    
    // 移除Windows文件系统不允许的字符
    invalidChars := []string{"<", ">", ":", "\"", "|", "?", "*"}
    for _, char := range invalidChars {
        result = strings.ReplaceAll(result, char, "")
    }
    
    // 移除开头和结尾的空格
    result = strings.TrimSpace(result)
    
    // 如果文件名为空或只包含点，返回默认名称
    if result == "" || result == "." || result == ".." {
        return "unnamed-file"
    }
    
    return result
}

// ValidateFilename 验证文件名是否安全
func ValidateFilename(filename string) bool {
    if filename == "" {
        return false
    }
    
    // 检查是否包含路径分隔符
    if strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
        return false
    }
    
    // 检查是否为保留名称（Windows）
    reservedNames := []string{
        "CON", "PRN", "AUX", "NUL",
        "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
        "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
    }
    
    nameWithoutExt := filename
    if idx := strings.LastIndex(filename, "."); idx != -1 {
        nameWithoutExt = filename[:idx]
    }
    
    for _, reserved := range reservedNames {
        if strings.EqualFold(nameWithoutExt, reserved) {
            return false
        }
    }
    
    return true
}

// SanitizeTitle 清理标题用于生成文件名
func SanitizeTitle(title string) string {
    if !utf8.ValidString(title) {
        title = strings.ToValidUTF8(title, "")
    }
    
    // 移除控制字符
    result := strings.Map(func(r rune) rune {
        if unicode.IsControl(r) {
            return -1
        }
        return r
    }, title)
    
    // 替换空格为连字符
    result = regexp.MustCompile(`\s+`).ReplaceAllString(result, "-")
    
    // 移除或替换特殊字符，保留中文字符
    result = regexp.MustCompile(`[<>:"/\\|?*]`).ReplaceAllString(result, "")
    
    // 移除连续的连字符
    result = regexp.MustCompile(`-+`).ReplaceAllString(result, "-")
    
    // 移除开头和结尾的连字符
    result = strings.Trim(result, "-")
    
    // 如果结果为空，返回默认值
    if result == "" {
        return "untitled"
    }
    
    return result
}

// IsValidUTF8Filename 检查文件名是否为有效的UTF-8编码
func IsValidUTF8Filename(filename string) bool {
    return utf8.ValidString(filename)
}