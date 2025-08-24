package utils

import (
    "regexp"
    "strings"
    "gopkg.in/yaml.v2"
)

type FrontMatter struct {
    Author     string   `yaml:"author" json:"author"`
    Categories []string `yaml:"categories" json:"categories"`
    Date       string   `yaml:"date" json:"date"`
    Tags       []string `yaml:"tags" json:"tags"`
    Title      string   `yaml:"title" json:"title"`
    Type       string   `yaml:"type" json:"type"`
    URL        string   `yaml:"url" json:"url"`
}

type ParsedMarkdown struct {
    FrontMatter FrontMatter `json:"front_matter"`
    Content     string      `json:"content"`
    Raw         string      `json:"raw"`
}

// ParseMarkdown 解析Markdown文件，分离front matter和内容
func ParseMarkdown(content string) (*ParsedMarkdown, error) {
    // 使用正则表达式匹配front matter
    re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n(.*)$`)
    matches := re.FindStringSubmatch(content)
    
    result := &ParsedMarkdown{
        Raw: content,
    }
    
    if len(matches) < 3 {
        // 没有front matter，整个内容都是正文
        result.Content = content
        return result, nil
    }
    
    frontMatterYaml := matches[1]
    result.Content = matches[2]
    
    // 解析YAML front matter
    if err := yaml.Unmarshal([]byte(frontMatterYaml), &result.FrontMatter); err != nil {
        return result, err
    }
    
    return result, nil
}

// BuildMarkdown 从结构化数据构建Markdown文件内容
func BuildMarkdown(frontMatter FrontMatter, content string) (string, error) {
    // 构建front matter YAML
    yamlData, err := yaml.Marshal(frontMatter)
    if err != nil {
        return "", err
    }
    
    // 移除YAML末尾的换行符
    yamlStr := strings.TrimSuffix(string(yamlData), "\n")
    
    // 构建完整的Markdown内容
    markdown := "---\n" + yamlStr + "\n---\n" + content
    
    return markdown, nil
}