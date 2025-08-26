
package controller

import (
    "fmt"
    "github.com/gin-gonic/gin"
    "hugo-manager-go/config"
    "io/fs"
    "io/ioutil"
    "math"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strconv"
    "strings"
    "time"
)

func EditArticle(c *gin.Context) {
    path := c.Query("path")
    fullPath := filepath.Join(config.GetContentDir(), path)
    data, err := os.ReadFile(fullPath)
    if err != nil {
        c.String(500, "读取失败: %v", err)
        return
    }

    c.HTML(200, "article/editor.html", gin.H{
        "Title":   "编辑文章",
        "Path":    path,
        "Content": string(data),
    })
}

func SaveArticle(c *gin.Context) {
    path := c.PostForm("path")
    content := c.PostForm("content")
    fullPath := filepath.Join(config.GetContentDir(), path)

    os.MkdirAll(filepath.Dir(fullPath), os.ModePerm)
    os.WriteFile(fullPath, []byte(content), 0644)

    c.Redirect(302, "/")
}

type ArticleInfo struct {
    Path          string
    ModTime       time.Time
    Size          int64
    Year          int
    Month         time.Month
    Day           int
    FormattedTime string
    Title         string
    Content       string
    Summary       string
    IsDraft       bool
    HasIssues     bool
    Issues        []string
    Categories    []string
    Tags          []string
    URL           string
    Date          string
}

// ArticleList 显示文章列表页面，支持分页、搜索和按时间筛选
func ArticleList(c *gin.Context) {
    // 获取查询参数
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    if page < 1 {
        page = 1
    }
    
    year, _ := strconv.Atoi(c.Query("year"))
    month, _ := strconv.Atoi(c.Query("month"))
    search := strings.TrimSpace(c.Query("search"))
    status := strings.TrimSpace(c.Query("status")) // "draft" 或空字符串（发布的文章）
    
    pageSize := 20 // 每页显示20条记录
    
    // 获取所有文章信息
    articleInfos, err := getAllArticlesWithContent()
    if err != nil {
        c.HTML(500, "article/list.html", gin.H{
            "Title": "文章列表",
            "Error": err.Error(),
        })
        return
    }
    
    // 原始文章总数
    originalTotal := len(articleInfos)
    
    // 按状态筛选（草稿或发布）
    if status == "draft" {
        articleInfos = filterArticlesByDraft(articleInfos, true)
    } else if status == "published" {
        articleInfos = filterArticlesByDraft(articleInfos, false)
    } else if status == "issues" {
        articleInfos = filterArticlesByIssues(articleInfos)
    }
    
    // 按搜索关键词筛选
    if search != "" {
        articleInfos = filterArticlesBySearch(articleInfos, search)
    }
    
    // 按时间筛选
    if year > 0 {
        articleInfos = filterArticlesByYear(articleInfos, year)
    }
    if month > 0 && month <= 12 {
        articleInfos = filterArticlesByMonth(articleInfos, time.Month(month))
    }
    
    // 计算分页信息
    totalArticles := len(articleInfos)
    totalPages := int(math.Ceil(float64(totalArticles) / float64(pageSize)))
    
    // 获取当前页的文章
    start := (page - 1) * pageSize
    end := start + pageSize
    if end > totalArticles {
        end = totalArticles
    }
    
    var currentPageArticles []ArticleInfo
    if start < totalArticles {
        currentPageArticles = articleInfos[start:end]
    }
    
    // 获取可用的年份列表（基于原始文章列表）
    allArticles, _ := getAllArticlesWithContent()
    availableYears := getAvailableYears(allArticles)
    
    // 统计草稿、发布文章和有问题文章数量
    draftCount := 0
    publishedCount := 0
    issuesCount := 0
    for _, article := range allArticles {
        if article.IsDraft {
            draftCount++
        } else {
            publishedCount++
        }
        if article.HasIssues {
            issuesCount++
        }
    }
    
    c.HTML(200, "article/list.html", gin.H{
        "Title":              "文章列表",
        "Articles":           currentPageArticles,
        "CurrentPage":        page,
        "TotalPages":         totalPages,
        "TotalArticles":      totalArticles,
        "OriginalTotal":      originalTotal,
        "DraftCount":         draftCount,
        "PublishedCount":     publishedCount,
        "IssuesCount":        issuesCount,
        "PageSize":           pageSize,
        "SelectedYear":       year,
        "SelectedMonth":      month,
        "SelectedStatus":     status,
        "SearchKeyword":      search,
        "AvailableYears":     availableYears,
        "HugoProjectPath":    config.GetHugoProjectPath(),
    })
}

// getAllArticles 获取所有文章信息并按时间排序
func getAllArticles() ([]ArticleInfo, error) {
    var articleInfos []ArticleInfo
    contentDir := config.GetContentDir()
    
    err := filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d != nil && !d.IsDir() && strings.HasSuffix(path, ".md") {
            rel, _ := filepath.Rel(contentDir, path)
            info, err := os.Stat(path)
            if err == nil {
                modTime := info.ModTime()
                articleInfos = append(articleInfos, ArticleInfo{
                    Path:          rel,
                    ModTime:       modTime,
                    Size:          info.Size(),
                    Year:          modTime.Year(),
                    Month:         modTime.Month(),
                    Day:           modTime.Day(),
                    FormattedTime: modTime.Format("2006-01-02 15:04:05"),
                })
            }
        }
        return nil
    })
    
    if err != nil {
        return nil, err
    }
    
    // 按修改时间倒序排序（最新的在前）
    sort.Slice(articleInfos, func(i, j int) bool {
        return articleInfos[i].ModTime.After(articleInfos[j].ModTime)
    })
    
    return articleInfos, nil
}

// filterArticlesByYear 按年份筛选文章
func filterArticlesByYear(articles []ArticleInfo, year int) []ArticleInfo {
    var filtered []ArticleInfo
    for _, article := range articles {
        if article.Year == year {
            filtered = append(filtered, article)
        }
    }
    return filtered
}

// filterArticlesByMonth 按月份筛选文章
func filterArticlesByMonth(articles []ArticleInfo, month time.Month) []ArticleInfo {
    var filtered []ArticleInfo
    for _, article := range articles {
        if article.Month == month {
            filtered = append(filtered, article)
        }
    }
    return filtered
}

// getAvailableYears 获取可用的年份列表
func getAvailableYears(articles []ArticleInfo) []int {
    yearMap := make(map[int]bool)
    for _, article := range articles {
        yearMap[article.Year] = true
    }
    
    var years []int
    for year := range yearMap {
        years = append(years, year)
    }
    
    // 按年份倒序排序
    sort.Slice(years, func(i, j int) bool {
        return years[i] > years[j]
    })
    
    return years
}

// getAllArticlesWithContent 获取所有文章信息（包含内容）并按时间排序
func getAllArticlesWithContent() ([]ArticleInfo, error) {
    var articleInfos []ArticleInfo
    contentDir := config.GetContentDir()
    
    err := filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d != nil && !d.IsDir() && strings.HasSuffix(path, ".md") {
            rel, _ := filepath.Rel(contentDir, path)
            info, err := os.Stat(path)
            if err == nil {
                modTime := info.ModTime()
                
                // 读取文章内容并检测问题
                articleInfo := readArticleContentWithAnalysis(path, rel, modTime, info.Size())
                articleInfos = append(articleInfos, articleInfo)
            }
        }
        return nil
    })
    
    if err != nil {
        return nil, err
    }
    
    // 按修改时间倒序排序（最新的在前）
    sort.Slice(articleInfos, func(i, j int) bool {
        return articleInfos[i].ModTime.After(articleInfos[j].ModTime)
    })
    
    return articleInfos, nil
}

// readArticleContent 读取文章内容并提取标题、摘要和草稿状态
func readArticleContent(filePath string) (content, title, summary string, isDraft bool) {
    data, err := ioutil.ReadFile(filePath)
    if err != nil {
        return "", "", "", false
    }
    
    content = string(data)
    
    // 提取标题（从 front matter 或第一行 # 标题）
    title = extractTitle(content)
    if title == "" {
        // 如果没有找到标题，使用文件名（去掉扩展名）
        base := filepath.Base(filePath)
        title = strings.TrimSuffix(base, filepath.Ext(base))
    }
    
    // 检查是否为草稿
    isDraft = checkIfDraft(content)
    
    // 生成摘要（去掉 front matter 后的前200个字符）
    summary = generateSummary(content)
    
    return content, title, summary, isDraft
}

// extractTitle 从文章内容中提取标题
func extractTitle(content string) string {
    lines := strings.Split(content, "\n")
    inFrontMatter := false
    frontMatterEnd := false
    
    for _, line := range lines {
        line = strings.TrimSpace(line)
        
        // 检查 front matter
        if line == "---" || line == "+++" {
            if !inFrontMatter {
                inFrontMatter = true
                continue
            } else {
                frontMatterEnd = true
                inFrontMatter = false
                continue
            }
        }
        
        // 在 front matter 中查找标题
        if inFrontMatter {
            if strings.HasPrefix(line, "title:") {
                title := strings.TrimSpace(strings.TrimPrefix(line, "title:"))
                title = strings.Trim(title, "\"'")
                return title
            }
        }
        
        // 在正文中查找第一个 # 标题
        if frontMatterEnd && strings.HasPrefix(line, "#") {
            title := strings.TrimSpace(strings.TrimPrefix(line, "#"))
            return title
        }
    }
    
    return ""
}

// generateSummary 生成文章摘要
func generateSummary(content string) string {
    lines := strings.Split(content, "\n")
    var bodyLines []string
    inFrontMatter := false
    frontMatterEnd := false
    
    // 跳过 front matter
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        
        if trimmed == "---" || trimmed == "+++" {
            if !inFrontMatter {
                inFrontMatter = true
                continue
            } else {
                frontMatterEnd = true
                inFrontMatter = false
                continue
            }
        }
        
        if frontMatterEnd && !inFrontMatter {
            // 跳过空行和标题行
            if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
                bodyLines = append(bodyLines, trimmed)
            }
        }
    }
    
    // 合并行并截取前200个字符
    body := strings.Join(bodyLines, " ")
    if len(body) > 200 {
        body = body[:200] + "..."
    }
    
    return body
}

// filterArticlesBySearch 按搜索关键词筛选文章
func filterArticlesBySearch(articles []ArticleInfo, search string) []ArticleInfo {
    if search == "" {
        return articles
    }
    
    var filtered []ArticleInfo
    searchLower := strings.ToLower(search)
    
    for _, article := range articles {
        // 在标题、文件名、内容中搜索
        if strings.Contains(strings.ToLower(article.Title), searchLower) ||
           strings.Contains(strings.ToLower(article.Path), searchLower) ||
           strings.Contains(strings.ToLower(article.Content), searchLower) {
            filtered = append(filtered, article)
        }
    }
    
    return filtered
}

// checkIfDraft 检查文章是否为草稿
func checkIfDraft(content string) bool {
    lines := strings.Split(content, "\n")
    inFrontMatter := false
    
    for _, line := range lines {
        line = strings.TrimSpace(line)
        
        // 检查 front matter
        if line == "---" || line == "+++" {
            if !inFrontMatter {
                inFrontMatter = true
                continue
            } else {
                // front matter 结束
                break
            }
        }
        
        // 在 front matter 中查找 draft 字段
        if inFrontMatter {
            if strings.HasPrefix(line, "draft:") {
                draftValue := strings.TrimSpace(strings.TrimPrefix(line, "draft:"))
                draftValue = strings.ToLower(draftValue)
                return draftValue == "true" || draftValue == "yes" || draftValue == "1"
            }
        }
    }
    
    return false
}

// filterArticlesByDraft 按草稿状态筛选文章
func filterArticlesByDraft(articles []ArticleInfo, isDraft bool) []ArticleInfo {
    var filtered []ArticleInfo
    for _, article := range articles {
        if article.IsDraft == isDraft {
            filtered = append(filtered, article)
        }
    }
    return filtered
}

// readArticleContentWithAnalysis 读取文章内容并进行问题检测
func readArticleContentWithAnalysis(filePath, relativePath string, modTime time.Time, size int64) ArticleInfo {
    data, err := ioutil.ReadFile(filePath)
    if err != nil {
        return ArticleInfo{
            Path:          relativePath,
            ModTime:       modTime,
            Size:          size,
            Year:          modTime.Year(),
            Month:         modTime.Month(),
            Day:           modTime.Day(),
            FormattedTime: modTime.Format("2006-01-02 15:04:05"),
            HasIssues:     true,
            Issues:        []string{"文件读取失败"},
        }
    }
    
    content := string(data)
    
    // 提取文章元数据
    title, categories, tags, url, date, isDraft := extractArticleMetadata(content)
    if title == "" {
        // 如果没有找到标题，使用文件名（去掉扩展名）
        base := filepath.Base(filePath)
        title = strings.TrimSuffix(base, filepath.Ext(base))
    }
    
    // 生成摘要
    summary := generateSummary(content)
    
    // 检测文章问题
    issues := detectArticleIssues(content, title, categories, tags, url, date, filePath)
    
    return ArticleInfo{
        Path:          relativePath,
        ModTime:       modTime,
        Size:          size,
        Year:          modTime.Year(),
        Month:         modTime.Month(),
        Day:           modTime.Day(),
        FormattedTime: modTime.Format("2006-01-02 15:04:05"),
        Title:         title,
        Content:       content,
        Summary:       summary,
        IsDraft:       isDraft,
        HasIssues:     len(issues) > 0,
        Issues:        issues,
        Categories:    categories,
        Tags:          tags,
        URL:           url,
        Date:          date,
    }
}

// extractArticleMetadata 提取文章元数据
func extractArticleMetadata(content string) (title string, categories, tags []string, url, date string, isDraft bool) {
    lines := strings.Split(content, "\n")
    inFrontMatter := false
    
    for _, line := range lines {
        line = strings.TrimSpace(line)
        
        // 检查 front matter
        if line == "---" || line == "+++" {
            if !inFrontMatter {
                inFrontMatter = true
                continue
            } else {
                // front matter 结束
                break
            }
        }
        
        // 在 front matter 中提取元数据
        if inFrontMatter {
            if strings.HasPrefix(line, "title:") {
                title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
                title = strings.Trim(title, "\"'")
            } else if strings.HasPrefix(line, "categories:") {
                categoriesStr := strings.TrimSpace(strings.TrimPrefix(line, "categories:"))
                if categoriesStr != "" && categoriesStr != "[]" {
                    // 处理数组格式
                    categoriesStr = strings.Trim(categoriesStr, "[]")
                    for _, cat := range strings.Split(categoriesStr, ",") {
                        cat = strings.TrimSpace(strings.Trim(cat, "\"'"))
                        if cat != "" {
                            categories = append(categories, cat)
                        }
                    }
                }
            } else if strings.HasPrefix(line, "tags:") {
                tagsStr := strings.TrimSpace(strings.TrimPrefix(line, "tags:"))
                if tagsStr != "" && tagsStr != "[]" {
                    // 处理数组格式
                    tagsStr = strings.Trim(tagsStr, "[]")
                    for _, tag := range strings.Split(tagsStr, ",") {
                        tag = strings.TrimSpace(strings.Trim(tag, "\"'"))
                        if tag != "" {
                            tags = append(tags, tag)
                        }
                    }
                }
            } else if strings.HasPrefix(line, "url:") {
                url = strings.TrimSpace(strings.TrimPrefix(line, "url:"))
                url = strings.Trim(url, "\"'")
            } else if strings.HasPrefix(line, "date:") {
                date = strings.TrimSpace(strings.TrimPrefix(line, "date:"))
                date = strings.Trim(date, "\"'")
            } else if strings.HasPrefix(line, "draft:") {
                draftValue := strings.TrimSpace(strings.TrimPrefix(line, "draft:"))
                draftValue = strings.ToLower(draftValue)
                isDraft = draftValue == "true" || draftValue == "yes" || draftValue == "1"
            }
        }
    }
    
    return
}

// detectArticleIssues 检测文章问题
func detectArticleIssues(content, title string, categories, tags []string, url, date, filePath string) []string {
    var issues []string
    
    // 检测标题问题
    if title == "" {
        issues = append(issues, "标题为空")
    } else if containsSpecialChars(title) {
        issues = append(issues, "标题包含特殊字符")
    }
    
    // 检测内容问题
    bodyContent := extractBodyContent(content)
    if bodyContent == "" {
        issues = append(issues, "文章内容为空")
    } else if len(bodyContent) < 100 {
        issues = append(issues, "文章内容过短(少于100字符)")
    }
    
    // 检测分类问题
    if len(categories) == 0 {
        issues = append(issues, "分类为空")
    }
    
    // 检测标签问题
    if len(tags) == 0 {
        issues = append(issues, "标签为空")
    }
    
    // 检测URL问题
    if url == "" {
        issues = append(issues, "URL为空")
    }
    
    // 检测日期问题
    if date == "" {
        issues = append(issues, "发布时间为空")
    }
    
    // 检测图片链接问题
    brokenImages := detectBrokenImages(content, filePath)
    if len(brokenImages) > 0 {
        issues = append(issues, fmt.Sprintf("存在%d个无效图片链接", len(brokenImages)))
    }
    
    return issues
}

// containsSpecialChars 检测标题是否包含特殊字符
func containsSpecialChars(title string) bool {
    specialChars := []string{"<", ">", ":", "\"", "|", "?", "*", "/", "\\"}
    for _, char := range specialChars {
        if strings.Contains(title, char) {
            return true
        }
    }
    return false
}

// extractBodyContent 提取文章正文内容
func extractBodyContent(content string) string {
    lines := strings.Split(content, "\n")
    var bodyLines []string
    inFrontMatter := false
    frontMatterEnd := false
    
    // 跳过 front matter
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        
        if trimmed == "---" || trimmed == "+++" {
            if !inFrontMatter {
                inFrontMatter = true
                continue
            } else {
                frontMatterEnd = true
                inFrontMatter = false
                continue
            }
        }
        
        if frontMatterEnd && !inFrontMatter {
            bodyLines = append(bodyLines, line)
        }
    }
    
    body := strings.Join(bodyLines, "\n")
    body = strings.TrimSpace(body)
    
    // 移除 markdown 语法字符来计算实际内容长度
    body = regexp.MustCompile(`[#*_\[\]()]+`).ReplaceAllString(body, "")
    body = regexp.MustCompile(`!\[.*?\]\(.*?\)`).ReplaceAllString(body, "")  // 移除图片链接
    body = regexp.MustCompile(`\[.*?\]\(.*?\)`).ReplaceAllString(body, "")   // 移除普通链接
    
    return strings.TrimSpace(body)
}

// detectBrokenImages 检测无效的图片链接
func detectBrokenImages(content, filePath string) []string {
    var brokenImages []string
    
    // 查找所有图片链接
    imageRegex := regexp.MustCompile(`!\[.*?\]\((.*?)\)`)
    matches := imageRegex.FindAllStringSubmatch(content, -1)
    
    baseDir := filepath.Dir(filePath)
    projectDir := config.GetHugoProjectPath()
    
    for _, match := range matches {
        if len(match) > 1 {
            imagePath := strings.TrimSpace(match[1])
            
            // 跳过网络图片
            if strings.HasPrefix(imagePath, "http://") || strings.HasPrefix(imagePath, "https://") {
                continue
            }
            
            // 检查本地图片文件是否存在
            var fullImagePath string
            if strings.HasPrefix(imagePath, "/") {
                // 绝对路径，相对于Hugo项目根目录
                fullImagePath = filepath.Join(projectDir, "static", strings.TrimPrefix(imagePath, "/"))
            } else {
                // 相对路径，相对于文章目录
                fullImagePath = filepath.Join(baseDir, imagePath)
            }
            
            if _, err := os.Stat(fullImagePath); os.IsNotExist(err) {
                brokenImages = append(brokenImages, imagePath)
            }
        }
    }
    
    return brokenImages
}

// filterArticlesByIssues 筛选有问题的文章
func filterArticlesByIssues(articles []ArticleInfo) []ArticleInfo {
    var filtered []ArticleInfo
    for _, article := range articles {
        if article.HasIssues {
            filtered = append(filtered, article)
        }
    }
    return filtered
}
