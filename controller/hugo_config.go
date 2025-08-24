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
	hugoConfig := parseHugoConfig(string(content))
	c.JSON(200, hugoConfig)
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
	
	// 生成TOML配置内容
	configContent := generateTOMLConfig(hugoConfig)
	
	// 写入配置文件
	err := os.WriteFile(configPath, []byte(configContent), 0644)
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

// 解析Hugo配置文件
func parseHugoConfig(content string) HugoConfig {
	config := HugoConfig{
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
	}
	
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				value = strings.Trim(value, "\"'")
				
				switch key {
				case "title":
					config.Title = value
				case "baseURL":
					config.BaseURL = value
				case "languageCode":
					config.LanguageCode = value
				case "theme":
					config.Theme = value
				case "timezone":
					config.Timezone = value
				case "paginate":
					if p, err := strconv.Atoi(value); err == nil {
						config.Paginate = p
					}
				case "description":
					config.Description = value
				case "author":
					config.Author = value
				case "authorEmail":
					config.AuthorEmail = value
				case "buildDrafts":
					config.BuildDrafts = value == "true"
				case "buildFuture":
					config.BuildFuture = value == "true"
				}
			}
		}
	}
	
	return config
}

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