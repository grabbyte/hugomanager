package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"hugo-manager-go/config"
)

type HugoConfig struct {
	Title        string `json:"title"`
	BaseURL      string `json:"baseURL"`
	LanguageCode string `json:"languageCode"`
	Theme        string `json:"theme"`
	Timezone     string `json:"timezone"`
	Paginate     int    `json:"paginate"`
	Description  string `json:"description"`
	Author       string `json:"author"`
	AuthorEmail  string `json:"authorEmail"`
	BuildDrafts  bool   `json:"buildDrafts"`
	BuildFuture  bool   `json:"buildFuture"`
}

// Hugo配置文件的完整内容结构，包含原始内容和解析的UI配置
type HugoConfigFile struct {
	OriginalContent string     // 原始配置文件内容
	UIConfig        HugoConfig // UI中管理的配置项
	managedKeys     []string   // UI管理的配置键
}

// UI管理的配置键列表
func getManagedKeys() []string {
	return []string{
		"title", "baseURL", "languageCode", "theme", "timezone", 
		"paginate", "description", "buildDrafts", "buildFuture",
	}
}

// ParseHugoConfigFile 解析Hugo配置文件，保留原始内容结构
func ParseHugoConfigFile(content string) *HugoConfigFile {
	configFile := &HugoConfigFile{
		OriginalContent: content,
		managedKeys:     getManagedKeys(),
		UIConfig: HugoConfig{
			Title:        "我的Hugo博客",
			BaseURL:      "https://example.com", 
			LanguageCode: "zh-cn",
			Theme:        "",
			Timezone:     "Asia/Shanghai",
			Paginate:     10,
			Description:  "",
			Author:       "",
			AuthorEmail:  "",
			BuildDrafts:  false,
			BuildFuture:  false,
		},
	}
	
	lines := strings.Split(content, "\n")
	inAuthorSection := false
	
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		// 跳过注释和空行
		if strings.HasPrefix(trimmedLine, "#") || trimmedLine == "" {
			continue
		}
		
		// 检查是否进入author section
		if strings.HasPrefix(trimmedLine, "[author]") {
			inAuthorSection = true
			continue
		} else if strings.HasPrefix(trimmedLine, "[") && trimmedLine != "[author]" {
			inAuthorSection = false
			continue
		}
		
		// 解析键值对
		if strings.Contains(trimmedLine, "=") {
			parts := strings.SplitN(trimmedLine, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				value = strings.Trim(value, "\"'")
				
				// 处理author section中的配置
				if inAuthorSection {
					switch key {
					case "name":
						configFile.UIConfig.Author = value
					case "email":
						configFile.UIConfig.AuthorEmail = value
					}
					continue
				}
				
				// 处理主要配置项
				switch key {
				case "title":
					configFile.UIConfig.Title = value
				case "baseURL":
					configFile.UIConfig.BaseURL = value
				case "languageCode":
					configFile.UIConfig.LanguageCode = value
				case "theme":
					configFile.UIConfig.Theme = value
				case "timezone":
					configFile.UIConfig.Timezone = value
				case "paginate":
					if p, err := strconv.Atoi(value); err == nil {
						configFile.UIConfig.Paginate = p
					}
				case "description":
					configFile.UIConfig.Description = value
				case "buildDrafts":
					configFile.UIConfig.BuildDrafts = value == "true"
				case "buildFuture":
					configFile.UIConfig.BuildFuture = value == "true"
				}
			}
		}
	}
	
	return configFile
}

// UpdateWithUIConfig 更新配置文件内容，只修改UI管理的配置项
func (cf *HugoConfigFile) UpdateWithUIConfig(newConfig HugoConfig) string {
	lines := strings.Split(cf.OriginalContent, "\n")
	result := make([]string, 0, len(lines))
	inAuthorSection := false
	authorSectionProcessed := false
	
	// 记录哪些配置项已经被更新
	updatedKeys := make(map[string]bool)
	
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		// 检查是否进入author section
		if strings.HasPrefix(trimmedLine, "[author]") {
			inAuthorSection = true
			result = append(result, line)
			continue
		} else if strings.HasPrefix(trimmedLine, "[") && trimmedLine != "[author]" {
			// 如果离开author section且还没有处理过author信息，则添加
			if inAuthorSection && !authorSectionProcessed {
				cf.addAuthorConfigIfNeeded(&result, newConfig)
				authorSectionProcessed = true
			}
			inAuthorSection = false
			result = append(result, line)
			continue
		}
		
		// 处理配置行
		if strings.Contains(trimmedLine, "=") && !strings.HasPrefix(trimmedLine, "#") {
			parts := strings.SplitN(trimmedLine, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				
				// 处理author section中的配置
				if inAuthorSection {
					switch key {
					case "name":
						if newConfig.Author != "" {
							result = append(result, fmt.Sprintf("  name = \"%s\"", newConfig.Author))
						} else {
							result = append(result, line)
						}
						updatedKeys["author"] = true
						continue
					case "email":
						if newConfig.AuthorEmail != "" {
							result = append(result, fmt.Sprintf("  email = \"%s\"", newConfig.AuthorEmail))
						} else {
							result = append(result, line)
						}
						updatedKeys["authorEmail"] = true
						continue
					}
				}
				
				// 处理主要配置项
				switch key {
				case "title":
					result = append(result, fmt.Sprintf("title = \"%s\"", newConfig.Title))
					updatedKeys["title"] = true
					continue
				case "baseURL":
					result = append(result, fmt.Sprintf("baseURL = \"%s\"", newConfig.BaseURL))
					updatedKeys["baseURL"] = true
					continue
				case "languageCode":
					result = append(result, fmt.Sprintf("languageCode = \"%s\"", newConfig.LanguageCode))
					updatedKeys["languageCode"] = true
					continue
				case "theme":
					if newConfig.Theme != "" {
						result = append(result, fmt.Sprintf("theme = \"%s\"", newConfig.Theme))
					} else {
						// 如果theme为空，保留原行或注释掉
						result = append(result, line)
					}
					updatedKeys["theme"] = true
					continue
				case "timezone":
					result = append(result, fmt.Sprintf("timezone = \"%s\"", newConfig.Timezone))
					updatedKeys["timezone"] = true
					continue
				case "paginate":
					result = append(result, fmt.Sprintf("paginate = %d", newConfig.Paginate))
					updatedKeys["paginate"] = true
					continue
				case "description":
					if newConfig.Description != "" {
						result = append(result, fmt.Sprintf("description = \"%s\"", newConfig.Description))
					} else {
						result = append(result, line)
					}
					updatedKeys["description"] = true
					continue
				case "buildDrafts":
					result = append(result, fmt.Sprintf("buildDrafts = %t", newConfig.BuildDrafts))
					updatedKeys["buildDrafts"] = true
					continue
				case "buildFuture":
					result = append(result, fmt.Sprintf("buildFuture = %t", newConfig.BuildFuture))
					updatedKeys["buildFuture"] = true
					continue
				}
			}
		}
		
		// 原样保留其他行
		result = append(result, line)
	}
	
	// 如果在author section结束时还没有处理过author信息
	if inAuthorSection && !authorSectionProcessed {
		cf.addAuthorConfigIfNeeded(&result, newConfig)
	}
	
	// 添加缺失的配置项
	cf.addMissingConfigs(&result, newConfig, updatedKeys)
	
	return strings.Join(result, "\n")
}

// 添加作者配置信息（如果需要）
func (cf *HugoConfigFile) addAuthorConfigIfNeeded(lines *[]string, config HugoConfig) {
	if config.Author != "" && !strings.Contains(cf.OriginalContent, "name = ") {
		*lines = append(*lines, fmt.Sprintf("  name = \"%s\"", config.Author))
	}
	if config.AuthorEmail != "" && !strings.Contains(cf.OriginalContent, "email = ") {
		*lines = append(*lines, fmt.Sprintf("  email = \"%s\"", config.AuthorEmail))
	}
}

// 添加缺失的配置项
func (cf *HugoConfigFile) addMissingConfigs(lines *[]string, config HugoConfig, updatedKeys map[string]bool) {
	// 如果某些必需的配置项在原文件中不存在，则添加它们
	if !updatedKeys["title"] {
		*lines = append(*lines, fmt.Sprintf("title = \"%s\"", config.Title))
	}
	if !updatedKeys["baseURL"] {
		*lines = append(*lines, fmt.Sprintf("baseURL = \"%s\"", config.BaseURL))
	}
	if !updatedKeys["languageCode"] {
		*lines = append(*lines, fmt.Sprintf("languageCode = \"%s\"", config.LanguageCode))
	}
	if !updatedKeys["timezone"] {
		*lines = append(*lines, fmt.Sprintf("timezone = \"%s\"", config.Timezone))
	}
	if !updatedKeys["paginate"] {
		*lines = append(*lines, fmt.Sprintf("paginate = %d", config.Paginate))
	}
	if !updatedKeys["buildDrafts"] {
		*lines = append(*lines, fmt.Sprintf("buildDrafts = %t", config.BuildDrafts))
	}
	if !updatedKeys["buildFuture"] {
		*lines = append(*lines, fmt.Sprintf("buildFuture = %t", config.BuildFuture))
	}
	
	// 处理可选配置项
	if config.Theme != "" && !updatedKeys["theme"] {
		*lines = append(*lines, fmt.Sprintf("theme = \"%s\"", config.Theme))
	}
	if config.Description != "" && !updatedKeys["description"] {
		*lines = append(*lines, fmt.Sprintf("description = \"%s\"", config.Description))
	}
	
	// 处理作者信息
	needsAuthorSection := (config.Author != "" || config.AuthorEmail != "") && 
		!strings.Contains(cf.OriginalContent, "[author]")
	
	if needsAuthorSection {
		*lines = append(*lines, "", "[author]")
		if config.Author != "" {
			*lines = append(*lines, fmt.Sprintf("  name = \"%s\"", config.Author))
		}
		if config.AuthorEmail != "" {
			*lines = append(*lines, fmt.Sprintf("  email = \"%s\"", config.AuthorEmail))
		}
	}
}

// 获取Hugo配置
func GetHugoConfig(c *gin.Context) {
	hugoProjectPath := config.GetHugoProjectPath()
	configPath := filepath.Join(hugoProjectPath, "config.toml")
	
	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 返回默认配置
		defaultConfig := HugoConfig{
			Title:        "我的Hugo博客",
			BaseURL:      "https://example.com",
			LanguageCode: "zh-cn",
			Theme:        "",
			Timezone:     "Asia/Shanghai",
			Paginate:     10,
			Description:  "这是一个用Hugo构建的精美博客",
			Author:       "",
			AuthorEmail:  "",
			BuildDrafts:  false,
			BuildFuture:  false,
		}
		c.JSON(200, defaultConfig)
		return
	}
	
	// 读取现有配置文件
	content, err := os.ReadFile(configPath)
	if err != nil {
		c.JSON(500, gin.H{"error": "读取配置文件失败: " + err.Error()})
		return
	}
	
	// 解析TOML配置文件
	configFile := ParseHugoConfigFile(string(content))
	c.JSON(200, configFile.UIConfig)
}

// 保存Hugo配置
func SaveHugoConfig(c *gin.Context) {
	var hugoConfig HugoConfig
	if err := c.ShouldBindJSON(&hugoConfig); err != nil {
		c.JSON(400, gin.H{"error": "请求格式错误"})
		return
	}
	
	// 验证必填字段
	if hugoConfig.Title == "" || hugoConfig.BaseURL == "" {
		c.JSON(400, gin.H{"error": "站点标题和基础URL为必填项"})
		return
	}
	
	hugoProjectPath := config.GetHugoProjectPath()
	configPath := filepath.Join(hugoProjectPath, "config.toml")
	
	var updatedContent string
	
	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 如果文件不存在，创建新配置文件
		updatedContent = generateTOMLConfig(hugoConfig)
	} else {
		// 读取现有配置文件
		content, err := os.ReadFile(configPath)
		if err != nil {
			c.JSON(500, gin.H{"error": "读取配置文件失败: " + err.Error()})
			return
		}
		
		// 解析配置文件并进行增量更新
		configFile := ParseHugoConfigFile(string(content))
		updatedContent = configFile.UpdateWithUIConfig(hugoConfig)
	}
	
	// 写入配置文件
	err := os.WriteFile(configPath, []byte(updatedContent), 0644)
	if err != nil {
		c.JSON(500, gin.H{"error": "保存配置文件失败: " + err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"message": "Hugo配置保存成功"})
}

// 预览Hugo配置文件
func PreviewHugoConfig(c *gin.Context) {
	hugoProjectPath := config.GetHugoProjectPath()
	configPath := filepath.Join(hugoProjectPath, "config.toml")
	
	// 读取配置文件
	content, err := os.ReadFile(configPath)
	if err != nil {
		// 如果文件不存在，生成默认配置预览
		defaultConfig := HugoConfig{
			Title:        "我的Hugo博客",
			BaseURL:      "https://example.com",
			LanguageCode: "zh-cn",
			Theme:        "",
			Timezone:     "Asia/Shanghai",
			Paginate:     10,
			Description:  "这是一个用Hugo构建的精美博客",
			Author:       "",
			AuthorEmail:  "",
			BuildDrafts:  false,
			BuildFuture:  false,
		}
		content = []byte(generateTOMLConfig(defaultConfig))
	}
	
	c.String(200, string(content))
}

// 旧的parseHugoConfig函数已被parseHugoConfigFile取代，该函数保留原始配置结构

// 生成TOML配置内容
func generateTOMLConfig(config HugoConfig) string {
	var content strings.Builder
	
	content.WriteString("# Hugo站点配置文件\n")
	content.WriteString("# 由Hugo Manager自动生成\n\n")
	
	// 基础配置
	content.WriteString("# 站点基础信息\n")
	content.WriteString(fmt.Sprintf("title = \"%s\"\n", config.Title))
	content.WriteString(fmt.Sprintf("baseURL = \"%s\"\n", config.BaseURL))
	content.WriteString(fmt.Sprintf("languageCode = \"%s\"\n", config.LanguageCode))
	
	if config.Theme != "" {
		content.WriteString(fmt.Sprintf("theme = \"%s\"\n", config.Theme))
	}
	
	content.WriteString(fmt.Sprintf("timezone = \"%s\"\n", config.Timezone))
	content.WriteString(fmt.Sprintf("paginate = %d\n", config.Paginate))
	
	if config.Description != "" {
		content.WriteString(fmt.Sprintf("description = \"%s\"\n", config.Description))
	}
	
	content.WriteString("\n# 构建选项\n")
	content.WriteString(fmt.Sprintf("buildDrafts = %t\n", config.BuildDrafts))
	content.WriteString(fmt.Sprintf("buildFuture = %t\n", config.BuildFuture))
	
	// 作者信息
	if config.Author != "" || config.AuthorEmail != "" {
		content.WriteString("\n# 作者信息\n")
		content.WriteString("[author]\n")
		if config.Author != "" {
			content.WriteString(fmt.Sprintf("  name = \"%s\"\n", config.Author))
		}
		if config.AuthorEmail != "" {
			content.WriteString(fmt.Sprintf("  email = \"%s\"\n", config.AuthorEmail))
		}
	}
	
	// 默认的Hugo配置
	content.WriteString("\n# 默认配置\n")
	content.WriteString("enableRobotsTXT = true\n")
	content.WriteString("enableGitInfo = false\n")
	content.WriteString("enableEmoji = true\n")
	
	// 标记和分类
	content.WriteString("\n# 分类设置\n")
	content.WriteString("[taxonomies]\n")
	content.WriteString("  tag = \"tags\"\n")
	content.WriteString("  category = \"categories\"\n")
	
	// 输出格式
	content.WriteString("\n# 输出格式\n")
	content.WriteString("[outputs]\n")
	content.WriteString("  home = [\"HTML\", \"RSS\", \"JSON\"]\n")
	
	return content.String()
}