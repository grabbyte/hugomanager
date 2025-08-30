package controller

import (
    "encoding/json"
    "fmt"
    "github.com/gin-gonic/gin"
    "hugo-manager-go/config"
    "hugo-manager-go/utils"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
)

// Claude Prompt: 创建收藏管理控制器，支持Tools、Books、AI资源等独立分类管理

// 通用收藏项目结构
type CollectionItem struct {
    ID          string            `json:"id"`
    Title       string            `json:"title"`
    Category    string            `json:"category"`
    Type        string            `json:"type"` // tools, books, ai-resources, wiki
    URL         string            `json:"url"`
    Description string            `json:"description"`
    Tags        string            `json:"tags"`
    Metadata    map[string]string `json:"metadata"` // 存储额外字段
    Favorite    bool              `json:"favorite"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

// 收藏集合数据结构
type Collections struct {
    Tools       map[string][]CollectionItem `json:"tools"`
    Books       map[string][]CollectionItem `json:"books"`
    AIResources map[string][]CollectionItem `json:"ai_resources"`
    Wiki        map[string][]CollectionItem `json:"wiki"`
}

// Tools页面
func ToolsPage(c *gin.Context) {
    c.HTML(http.StatusOK, "tools/index.html", gin.H{
        "Title": "工具箱",
        "Page":  "tools",
    })
}

// Books页面
func BooksPage(c *gin.Context) {
    c.HTML(http.StatusOK, "books/index.html", gin.H{
        "Title": "书籍收藏",
        "Page":  "books",
    })
}

// AI编程页面
func AIPage(c *gin.Context) {
    c.HTML(http.StatusOK, "ai/index.html", gin.H{
        "Title": "AI编程",
        "Page":  "ai",
    })
}

// Wiki页面
func WikiPage(c *gin.Context) {
    c.HTML(http.StatusOK, "wiki/index.html", gin.H{
        "Title": "知识库",
        "Page":  "wiki",
    })
}

// 获取收藏数据文件路径
func getCollectionsFilePath() string {
    return filepath.Join(config.GetHugoProjectPath(), "data", "collections.json")
}

// 加载收藏数据
func loadCollections() (*Collections, error) {
    filePath := getCollectionsFilePath()
    
    // 创建data目录如果不存在
    dataDir := filepath.Dir(filePath)
    if err := os.MkdirAll(dataDir, os.ModePerm); err != nil {
        return nil, fmt.Errorf("创建data目录失败: %v", err)
    }
    
    // 如果文件不存在，返回空结构
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        return &Collections{
            Tools:       make(map[string][]CollectionItem),
            Books:       make(map[string][]CollectionItem),
            AIResources: make(map[string][]CollectionItem),
            Wiki:        make(map[string][]CollectionItem),
        }, nil
    }
    
    // 读取文件
    data, err := os.ReadFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("读取收藏文件失败: %v", err)
    }
    
    var collections Collections
    if err := json.Unmarshal(data, &collections); err != nil {
        return nil, fmt.Errorf("解析收藏数据失败: %v", err)
    }
    
    // 初始化空的map如果为nil
    if collections.Tools == nil {
        collections.Tools = make(map[string][]CollectionItem)
    }
    if collections.Books == nil {
        collections.Books = make(map[string][]CollectionItem)
    }
    if collections.AIResources == nil {
        collections.AIResources = make(map[string][]CollectionItem)
    }
    if collections.Wiki == nil {
        collections.Wiki = make(map[string][]CollectionItem)
    }
    
    return &collections, nil
}

// 保存收藏数据
func saveCollections(collections *Collections) error {
    filePath := getCollectionsFilePath()
    
    data, err := json.MarshalIndent(collections, "", "  ")
    if err != nil {
        return fmt.Errorf("序列化收藏数据失败: %v", err)
    }
    
    if err := os.WriteFile(filePath, data, 0644); err != nil {
        return fmt.Errorf("保存收藏文件失败: %v", err)
    }
    
    return nil
}

// 获取工具列表
func GetTools(c *gin.Context) {
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"tools": collections.Tools})
}

// 添加工具
func AddTool(c *gin.Context) {
    var request struct {
        Name        string `json:"name"`
        Category    string `json:"category"`
        Icon        string `json:"icon"`
        URL         string `json:"url"`
        Description string `json:"description"`
        Tags        string `json:"tags"`
        Favorite    bool   `json:"favorite"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
        return
    }
    
    if request.Name == "" || request.URL == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "工具名称和链接不能为空"})
        return
    }
    
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // 生成ID
    now := time.Now()
    id := fmt.Sprintf("tool_%d", now.UnixNano())
    
    // 创建工具项
    tool := CollectionItem{
        ID:          id,
        Title:       request.Name,
        Category:    request.Category,
        Type:        "tools",
        URL:         request.URL,
        Description: request.Description,
        Tags:        request.Tags,
        Favorite:    request.Favorite,
        CreatedAt:   now,
        UpdatedAt:   now,
        Metadata: map[string]string{
            "icon": request.Icon,
        },
    }
    
    // 添加到对应分类
    if collections.Tools[request.Category] == nil {
        collections.Tools[request.Category] = make([]CollectionItem, 0)
    }
    collections.Tools[request.Category] = append(collections.Tools[request.Category], tool)
    
    // 保存数据
    if err := saveCollections(collections); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // 创建Hugo内容文件
    if err := createHugoToolContent(tool); err != nil {
        fmt.Printf("创建Hugo内容文件失败: %v\n", err)
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "工具添加成功",
        "id":      id,
    })
}

// 获取书籍列表
func GetBooks(c *gin.Context) {
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"books": collections.Books})
}

// 添加书籍
func AddBook(c *gin.Context) {
    var request struct {
        Title       string `json:"title"`
        Category    string `json:"category"`
        Author      string `json:"author"`
        Publisher   string `json:"publisher"`
        URL         string `json:"url"`
        Cover       string `json:"cover"`
        Description string `json:"description"`
        Rating      int    `json:"rating"`
        Status      string `json:"status"`
        Tags        string `json:"tags"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
        return
    }
    
    if request.Title == "" || request.URL == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "书籍标题和链接不能为空"})
        return
    }
    
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // 生成ID
    now := time.Now()
    id := fmt.Sprintf("book_%d", now.UnixNano())
    
    // 创建书籍项
    book := CollectionItem{
        ID:          id,
        Title:       request.Title,
        Category:    request.Category,
        Type:        "books",
        URL:         request.URL,
        Description: request.Description,
        Tags:        request.Tags,
        CreatedAt:   now,
        UpdatedAt:   now,
        Metadata: map[string]string{
            "author":    request.Author,
            "publisher": request.Publisher,
            "cover":     request.Cover,
            "rating":    strconv.Itoa(request.Rating),
            "status":    request.Status,
        },
    }
    
    // 添加到对应分类
    if collections.Books[request.Category] == nil {
        collections.Books[request.Category] = make([]CollectionItem, 0)
    }
    collections.Books[request.Category] = append(collections.Books[request.Category], book)
    
    // 保存数据
    if err := saveCollections(collections); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // 创建Hugo内容文件
    if err := createHugoBookContent(book); err != nil {
        fmt.Printf("创建Hugo内容文件失败: %v\n", err)
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "书籍添加成功",
        "id":      id,
    })
}

// 获取AI资源列表
func GetAIResources(c *gin.Context) {
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"resources": collections.AIResources})
}

// 添加AI资源
func AddAIResource(c *gin.Context) {
    var request struct {
        Title       string `json:"title"`
        Category    string `json:"category"`
        Platform    string `json:"platform"`
        Difficulty  string `json:"difficulty"`
        URL         string `json:"url"`
        Description string `json:"description"`
        Tags        string `json:"tags"`
        Language    string `json:"language"`
        Official    bool   `json:"official"`
        Favorite    bool   `json:"favorite"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
        return
    }
    
    if request.Title == "" || request.URL == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "资源标题和链接不能为空"})
        return
    }
    
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // 生成ID
    now := time.Now()
    id := fmt.Sprintf("ai_%d", now.UnixNano())
    
    // 创建AI资源项
    aiResource := CollectionItem{
        ID:          id,
        Title:       request.Title,
        Category:    request.Category,
        Type:        "ai-resources",
        URL:         request.URL,
        Description: request.Description,
        Tags:        request.Tags,
        Favorite:    request.Favorite,
        CreatedAt:   now,
        UpdatedAt:   now,
        Metadata: map[string]string{
            "platform":   request.Platform,
            "difficulty": request.Difficulty,
            "language":   request.Language,
            "official":   strconv.FormatBool(request.Official),
        },
    }
    
    // 添加到对应分类
    if collections.AIResources[request.Category] == nil {
        collections.AIResources[request.Category] = make([]CollectionItem, 0)
    }
    collections.AIResources[request.Category] = append(collections.AIResources[request.Category], aiResource)
    
    // 保存数据
    if err := saveCollections(collections); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // 创建Hugo内容文件
    if err := createHugoAIContent(aiResource); err != nil {
        fmt.Printf("创建Hugo内容文件失败: %v\n", err)
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "AI资源添加成功",
        "id":      id,
    })
}

// 删除收藏项
func DeleteCollectionItem(c *gin.Context) {
    itemType, exists := c.Get("type")
    if !exists {
        c.JSON(http.StatusBadRequest, gin.H{"error": "缺少项目类型参数"})
        return
    }
    itemTypeStr := itemType.(string)    // tools, books, ai-resources, wiki
    itemID := c.Param("id")
    
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    var found bool
    
    // 根据类型删除对应项目
    switch itemTypeStr {
    case "tools":
        for category, items := range collections.Tools {
            for i, item := range items {
                if item.ID == itemID {
                    collections.Tools[category] = append(items[:i], items[i+1:]...)
                    found = true
                    break
                }
            }
            if found {
                break
            }
        }
    case "books":
        for category, items := range collections.Books {
            for i, item := range items {
                if item.ID == itemID {
                    collections.Books[category] = append(items[:i], items[i+1:]...)
                    found = true
                    break
                }
            }
            if found {
                break
            }
        }
    case "ai-resources":
        for category, items := range collections.AIResources {
            for i, item := range items {
                if item.ID == itemID {
                    collections.AIResources[category] = append(items[:i], items[i+1:]...)
                    found = true
                    break
                }
            }
            if found {
                break
            }
        }
    case "wiki":
        for category, items := range collections.Wiki {
            for i, item := range items {
                if item.ID == itemID {
                    collections.Wiki[category] = append(items[:i], items[i+1:]...)
                    found = true
                    break
                }
            }
            if found {
                break
            }
        }
    }
    
    if !found {
        c.JSON(http.StatusNotFound, gin.H{"error": "项目不存在"})
        return
    }
    
    // 保存数据
    if err := saveCollections(collections); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// 创建Hugo工具内容文件
func createHugoToolContent(tool CollectionItem) error {
    // 构建Hugo content路径
    toolsDir := filepath.Join(config.GetContentDir(), "tools")
    if err := os.MkdirAll(toolsDir, os.ModePerm); err != nil {
        return err
    }
    
    // 生成文件名
    cleanTitle := utils.SanitizeTitle(tool.Title)
    filename := fmt.Sprintf("%s.md", cleanTitle)
    filePath := filepath.Join(toolsDir, filename)
    
    // 构建Front Matter
    frontMatter := utils.FrontMatter{
        Title:      tool.Title,
        Type:       "tool",
        Date:       tool.CreatedAt.Format("2006-01-02T15:04:05+08:00"),
        Categories: []string{tool.Category},
        Tags:       strings.Split(tool.Tags, ","),
        URL:        "/tools/" + cleanTitle,
    }
    
    // 生成内容
    content := fmt.Sprintf(`# %s

## 工具信息

- **链接**: [%s](%s)
- **分类**: %s
- **标签**: %s

## 描述

%s

## 使用说明

点击上方链接访问该工具。

---
*收录时间: %s*
`, tool.Title, tool.URL, tool.URL, tool.Category, tool.Tags, tool.Description, tool.CreatedAt.Format("2006-01-02 15:04:05"))
    
    // 构建Markdown内容
    markdownContent, err := utils.BuildMarkdown(frontMatter, content)
    if err != nil {
        return err
    }
    
    return os.WriteFile(filePath, []byte(markdownContent), 0644)
}

// 创建Hugo书籍内容文件
func createHugoBookContent(book CollectionItem) error {
    // 构建Hugo content路径
    booksDir := filepath.Join(config.GetContentDir(), "books")
    if err := os.MkdirAll(booksDir, os.ModePerm); err != nil {
        return err
    }
    
    // 生成文件名
    cleanTitle := utils.SanitizeTitle(book.Title)
    filename := fmt.Sprintf("%s.md", cleanTitle)
    filePath := filepath.Join(booksDir, filename)
    
    // 构建Front Matter
    frontMatter := utils.FrontMatter{
        Title:      book.Title,
        Author:     book.Metadata["author"],
        Type:       "book",
        Date:       book.CreatedAt.Format("2006-01-02T15:04:05+08:00"),
        Categories: []string{book.Category},
        Tags:       strings.Split(book.Tags, ","),
        URL:        "/books/" + cleanTitle,
    }
    
    // 生成内容
    content := fmt.Sprintf(`# %s

## 书籍信息

- **作者**: %s
- **出版社**: %s
- **链接**: [%s](%s)
- **分类**: %s
- **评分**: %s/5
- **阅读状态**: %s

## 简介

%s

---
*收录时间: %s*
`, book.Title, book.Metadata["author"], book.Metadata["publisher"], 
   book.URL, book.URL, book.Category, book.Metadata["rating"], 
   book.Metadata["status"], book.Description, book.CreatedAt.Format("2006-01-02 15:04:05"))
    
    // 构建Markdown内容
    markdownContent, err := utils.BuildMarkdown(frontMatter, content)
    if err != nil {
        return err
    }
    
    return os.WriteFile(filePath, []byte(markdownContent), 0644)
}

// 创建Hugo AI资源内容文件
func createHugoAIContent(aiResource CollectionItem) error {
    // 构建Hugo content路径
    aiDir := filepath.Join(config.GetContentDir(), "ai")
    if err := os.MkdirAll(aiDir, os.ModePerm); err != nil {
        return err
    }
    
    // 生成文件名
    cleanTitle := utils.SanitizeTitle(aiResource.Title)
    filename := fmt.Sprintf("%s.md", cleanTitle)
    filePath := filepath.Join(aiDir, filename)
    
    // 构建Front Matter
    frontMatter := utils.FrontMatter{
        Title:      aiResource.Title,
        Type:       "ai-resource",
        Date:       aiResource.CreatedAt.Format("2006-01-02T15:04:05+08:00"),
        Categories: []string{aiResource.Category},
        Tags:       strings.Split(aiResource.Tags, ","),
        URL:        "/ai/" + cleanTitle,
    }
    
    // 生成内容
    content := fmt.Sprintf(`# %s

## 资源信息

- **平台**: %s
- **难度**: %s
- **语言**: %s
- **链接**: [%s](%s)
- **官方资源**: %s

## 描述

%s

---
*收录时间: %s*
`, aiResource.Title, aiResource.Metadata["platform"], aiResource.Metadata["difficulty"],
   aiResource.Metadata["language"], aiResource.URL, aiResource.URL, 
   aiResource.Metadata["official"], aiResource.Description, 
   aiResource.CreatedAt.Format("2006-01-02 15:04:05"))
    
    // 构建Markdown内容
    markdownContent, err := utils.BuildMarkdown(frontMatter, content)
    if err != nil {
        return err
    }
    
    return os.WriteFile(filePath, []byte(markdownContent), 0644)
}

// 获取Wiki条目列表
func GetWikiEntries(c *gin.Context) {
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"entries": collections.Wiki})
}

// 添加Wiki条目
func AddWikiEntry(c *gin.Context) {
    var request struct {
        Title       string `json:"title"`
        Category    string `json:"category"`
        Type        string `json:"type"`
        Difficulty  string `json:"difficulty"`
        URL         string `json:"url"`
        Description string `json:"description"`
        Tags        string `json:"tags"`
        Keywords    string `json:"keywords"`
        Official    bool   `json:"official"`
        Favorite    bool   `json:"favorite"`
        Frequent    bool   `json:"frequent"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
        return
    }
    
    if request.Title == "" || request.URL == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "条目标题和链接不能为空"})
        return
    }
    
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // 生成ID
    now := time.Now()
    id := fmt.Sprintf("wiki_%d", now.UnixNano())
    
    // 创建Wiki条目
    wikiEntry := CollectionItem{
        ID:          id,
        Title:       request.Title,
        Category:    request.Category,
        Type:        "wiki",
        URL:         request.URL,
        Description: request.Description,
        Tags:        request.Tags,
        Favorite:    request.Favorite,
        CreatedAt:   now,
        UpdatedAt:   now,
        Metadata: map[string]string{
            "type":       request.Type,
            "difficulty": request.Difficulty,
            "keywords":   request.Keywords,
            "official":   strconv.FormatBool(request.Official),
            "frequent":   strconv.FormatBool(request.Frequent),
        },
    }
    
    // 添加到对应分类
    if collections.Wiki[request.Category] == nil {
        collections.Wiki[request.Category] = make([]CollectionItem, 0)
    }
    collections.Wiki[request.Category] = append(collections.Wiki[request.Category], wikiEntry)
    
    // 保存数据
    if err := saveCollections(collections); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // 创建Hugo内容文件
    if err := createHugoWikiContent(wikiEntry); err != nil {
        fmt.Printf("创建Hugo内容文件失败: %v\n", err)
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "Wiki条目添加成功",
        "id":      id,
    })
}

// 搜索Wiki条目
func SearchWikiEntries(c *gin.Context) {
    query := c.Query("q")
    if query == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "搜索关键词不能为空"})
        return
    }
    
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    var results []CollectionItem
    query = strings.ToLower(query)
    
    // 在所有Wiki条目中搜索
    for _, categoryEntries := range collections.Wiki {
        for _, entry := range categoryEntries {
            // 搜索标题、描述、标签和关键词
            if strings.Contains(strings.ToLower(entry.Title), query) ||
               strings.Contains(strings.ToLower(entry.Description), query) ||
               strings.Contains(strings.ToLower(entry.Tags), query) ||
               strings.Contains(strings.ToLower(entry.Metadata["keywords"]), query) {
                results = append(results, entry)
            }
        }
    }
    
    c.JSON(http.StatusOK, gin.H{"results": results})
}

// 创建Hugo Wiki内容文件
func createHugoWikiContent(entry CollectionItem) error {
    // 构建Hugo content路径
    wikiDir := filepath.Join(config.GetContentDir(), "wiki")
    if err := os.MkdirAll(wikiDir, os.ModePerm); err != nil {
        return err
    }
    
    // 生成文件名
    cleanTitle := utils.SanitizeTitle(entry.Title)
    filename := fmt.Sprintf("%s.md", cleanTitle)
    filePath := filepath.Join(wikiDir, filename)
    
    // 构建Front Matter
    frontMatter := utils.FrontMatter{
        Title:      entry.Title,
        Type:       "wiki",
        Date:       entry.CreatedAt.Format("2006-01-02T15:04:05+08:00"),
        Categories: []string{entry.Category},
        Tags:       strings.Split(entry.Tags, ","),
        URL:        "/wiki/" + cleanTitle,
    }
    
    // 生成内容
    content := fmt.Sprintf(`# %s

## 条目信息

- **类型**: %s
- **分类**: %s
- **难度**: %s
- **链接**: [%s](%s)
- **官方文档**: %s
- **关键词**: %s

## 描述

%s

## 相关链接

- [查看原文](%s)

---
*收录时间: %s*
*类型: %s | 难度: %s*
`, entry.Title, entry.Metadata["type"], entry.Category, entry.Metadata["difficulty"],
   entry.URL, entry.URL, entry.Metadata["official"], entry.Metadata["keywords"],
   entry.Description, entry.URL, entry.CreatedAt.Format("2006-01-02 15:04:05"),
   entry.Metadata["type"], entry.Metadata["difficulty"])
    
    // 构建Markdown内容
    wikiMarkdownContent, err := utils.BuildMarkdown(frontMatter, content)
    if err != nil {
        return err
    }
    
    return os.WriteFile(filePath, []byte(wikiMarkdownContent), 0644)
}