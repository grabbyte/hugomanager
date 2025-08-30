
package controller

import (
    "fmt"
    "github.com/gin-gonic/gin"
    "hugo-manager-go/config"
    "hugo-manager-go/utils"
    "io/fs"
    "io/ioutil"
    "math"
    "net/http"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strconv"
    "strings"
    "time"
)

// Claude Prompt: 修改EditArticle函数，解析draft状态供编辑器使用
func EditArticle(c *gin.Context) {
    path := c.Query("path")
    fullPath := filepath.Join(config.GetContentDir(), path)
    data, err := os.ReadFile(fullPath)
    if err != nil {
        c.String(500, "读取失败: %v", err)
        return
    }

    content := string(data)
    
    // 分离Front Matter和主体内容
    frontMatter, bodyContent := separateContentAndFrontMatter(content)
    
    // 解析Front Matter获取所有元数据
    title, author, articleType, categories, tags, url, date, isDraft := extractArticleMetadata(content)
    
    // 解析发布日期
    var publishDateStr string
    if date != "" {
        if parsed, err := parseArticleDate(date); err == nil {
            // 转换为HTML datetime-local格式，但保证后续保存时会转为RFC3339
            publishDateStr = parsed.Format("2006-01-02T15:04")
        } else {
            // 如果解析失败，尝试使用原始值
            publishDateStr = date
        }
    }

    c.HTML(200, "article/editor.html", gin.H{
        "Title":           "编辑文章",
        "Path":            path,
        "Content":         content,           // 完整内容，用于保存
        "BodyContent":     bodyContent,       // 主体内容，用于编辑
        "FrontMatter":     frontMatter,       // Front Matter，用于备份
        "IsDraft":         isDraft,
        "ArticleTitle":    title,
        "Author":          author,
        "Type":            articleType,
        "Categories":      categories,
        "Tags":            tags,
        "URL":             url,
        "PublishDate":     publishDateStr,
        "OriginalDate":    date,
    })
}

// Claude Prompt: 修改SaveArticle函数，处理草稿状态更新
func SaveArticle(c *gin.Context) {
    path := c.PostForm("path")
    content := c.PostForm("content")
    isDraftParam := c.PostForm("is_draft")
    fullPath := filepath.Join(config.GetContentDir(), path)

    // 解析草稿状态
    isDraft := isDraftParam == "true"
    
    // 更新Front Matter中的draft字段
    updatedContent := updateDraftStatus(content, isDraft)
    
    os.MkdirAll(filepath.Dir(fullPath), os.ModePerm)
    os.WriteFile(fullPath, []byte(updatedContent), 0644)

    c.Redirect(302, "/articles")
}

// Claude Prompt: 添加updateDraftStatus函数来更新Front Matter中的draft字段
// updateDraftStatus 更新文章内容中的draft状态
func updateDraftStatus(content string, isDraft bool) string {
    lines := strings.Split(content, "\n")
    var result []string
    inFrontMatter := false
    frontMatterProcessed := false
    draftFieldFound := false
    
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        
        // 检查 front matter 开始和结束
        if trimmed == "---" || trimmed == "+++" {
            if !inFrontMatter && !frontMatterProcessed {
                inFrontMatter = true
                result = append(result, line)
                continue
            } else if inFrontMatter {
                // front matter 结束
                // 如果没有找到draft字段，在结束前添加
                if !draftFieldFound {
                    draftValue := "false"
                    if isDraft {
                        draftValue = "true"
                    }
                    result = append(result, "draft: "+draftValue)
                }
                inFrontMatter = false
                frontMatterProcessed = true
                result = append(result, line)
                continue
            }
        }
        
        // 在 front matter 中处理 draft 字段
        if inFrontMatter {
            if strings.HasPrefix(trimmed, "draft:") {
                draftFieldFound = true
                draftValue := "false"
                if isDraft {
                    draftValue = "true"
                }
                // 保持原有的缩进
                indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
                result = append(result, indent+"draft: "+draftValue)
                continue
            }
        }
        
        result = append(result, line)
    }
    
    // 如果没有 front matter，在文件开头添加
    if !frontMatterProcessed {
        var newContent []string
        draftValue := "false"
        if isDraft {
            draftValue = "true"
        }
        
        newContent = append(newContent, "---")
        newContent = append(newContent, "draft: "+draftValue)
        newContent = append(newContent, "---")
        newContent = append(newContent, "")
        newContent = append(newContent, result...)
        
        return strings.Join(newContent, "\n")
    }
    
    return strings.Join(result, "\n")
}

// Claude Prompt: 修改ArticleInfo结构体，添加发布日期字段用于统计
type ArticleInfo struct {
    Path          string
    ModTime       time.Time
    Size          int64
    Year          int        // 用于筛选的年份（优先使用发布日期）
    Month         time.Month // 用于筛选的月份（优先使用发布日期）
    Day           int        // 用于筛选的日期（优先使用发布日期）
    FormattedTime string     // 格式化的显示时间
    PublishDate   time.Time  // 文章发布日期（从Front Matter解析）
    Title         string
    Content       string
    Summary       string
    IsDraft       bool
    HasIssues     bool
    Issues        []string
    Categories    []string
    Tags          []string
    URL           string
    Date          string     // Front Matter中的原始date字符串
}

// ArticleList 显示文章列表页面（纯模板，数据通过API获取）
func ArticleList(c *gin.Context) {
    c.HTML(200, "article/list.html", gin.H{
        "Title": "文章列表",
        "Page":  "articles",
    })
}

// Claude Prompt: 修改getAllArticles函数，使用发布日期进行排序
// getAllArticles 获取所有文章信息并按发布日期排序
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
                
                // 使用完整的内容分析来获取准确的发布日期
                articleInfo := readArticleContentWithAnalysis(path, rel, modTime, info.Size())
                articleInfos = append(articleInfos, articleInfo)
            }
        }
        return nil
    })
    
    if err != nil {
        return nil, err
    }
    
    // 按发布日期倒序排序（最新的在前）
    sort.Slice(articleInfos, func(i, j int) bool {
        return articleInfos[i].PublishDate.After(articleInfos[j].PublishDate)
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

// Claude Prompt: 修改年份统计函数，包含每个年份的博客数量
// YearStat 年份统计结构
type YearStat struct {
    Year  int `json:"year"`
    Count int `json:"count"`
}

// MonthStat 月份统计结构
type MonthStat struct {
    Month int `json:"month"`
    Count int `json:"count"`
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

// getYearStats 获取年份统计信息（包含博客数量）
func getYearStats(articles []ArticleInfo) []YearStat {
    yearCountMap := make(map[int]int)
    for _, article := range articles {
        yearCountMap[article.Year]++
    }
    
    var yearStats []YearStat
    for year, count := range yearCountMap {
        yearStats = append(yearStats, YearStat{
            Year:  year,
            Count: count,
        })
    }
    
    // 按年份倒序排序
    sort.Slice(yearStats, func(i, j int) bool {
        return yearStats[i].Year > yearStats[j].Year
    })
    
    return yearStats
}

// getMonthStats 获取月份统计信息（包含博客数量）
func getMonthStats(articles []ArticleInfo) []MonthStat {
    monthCountMap := make(map[int]int)
    for _, article := range articles {
        monthCountMap[int(article.Month)]++
    }
    
    var monthStats []MonthStat
    for month, count := range monthCountMap {
        monthStats = append(monthStats, MonthStat{
            Month: month,
            Count: count,
        })
    }
    
    // 按月份正序排序
    sort.Slice(monthStats, func(i, j int) bool {
        return monthStats[i].Month < monthStats[j].Month
    })
    
    return monthStats
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
    
    // 按发布日期倒序排序（最新的在前）
    sort.Slice(articleInfos, func(i, j int) bool {
        return articleInfos[i].PublishDate.After(articleInfos[j].PublishDate)
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

// Claude Prompt: 添加日期解析函数
// parseArticleDate 解析文章日期字符串
func parseArticleDate(dateStr string) (time.Time, error) {
    if dateStr == "" {
        return time.Time{}, fmt.Errorf("empty date string")
    }
    
    // 常见的Hugo日期格式，按优先级排序
    dateFormats := []string{
        // RFC3339格式 (Hugo标准)
        "2006-01-02T15:04:05Z07:00",
        "2006-01-02T15:04:05Z",
        "2006-01-02T15:04:05+07:00",
        "2006-01-02T15:04:05-07:00",
        // ISO8601格式
        time.RFC3339,
        time.RFC3339Nano,
        // 简化格式
        "2006-01-02T15:04:05",
        "2006-01-02T15:04", // HTML datetime-local格式
        "2006-01-02 15:04:05",
        "2006-01-02 15:04",
        "2006-01-02",
        "2006/01/02",
        // 其他常见格式
        "2006-01-02T15:04:05.000Z",
        "2006-01-02T15:04:05.000Z07:00",
    }
    
    for _, format := range dateFormats {
        if parsed, err := time.Parse(format, dateStr); err == nil {
            return parsed, nil
        }
    }
    
    return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// Claude Prompt: 检测时间是否符合Hugo要求的RFC3339格式
// isValidHugoDate 检测日期是否符合Hugo标准的RFC3339格式
func isValidHugoDate(dateStr string) bool {
    if dateStr == "" {
        return false
    }
    
    // Hugo推荐的RFC3339格式
    hugoFormats := []string{
        time.RFC3339,     // 2006-01-02T15:04:05Z07:00
        time.RFC3339Nano, // 2006-01-02T15:04:05.999999999Z07:00
        "2006-01-02T15:04:05Z",        // UTC格式
        "2006-01-02T15:04:05+08:00",   // 带时区格式
        "2006-01-02T15:04:05-07:00",   // 带时区格式
    }
    
    for _, format := range hugoFormats {
        if _, err := time.Parse(format, dateStr); err == nil {
            return true
        }
    }
    
    return false
}

// Claude Prompt: 修复时间格式为Hugo兼容的RFC3339格式
// fixDateFormat 修复日期格式为Hugo兼容的RFC3339格式
func fixDateFormat(dateStr string) (string, error) {
    if dateStr == "" {
        return "", fmt.Errorf("empty date string")
    }
    
    // 如果已经是有效的Hugo格式，直接返回
    if isValidHugoDate(dateStr) {
        return dateStr, nil
    }
    
    // 尝试解析日期
    parsed, err := parseArticleDate(dateStr)
    if err != nil {
        return "", fmt.Errorf("无法解析日期: %v", err)
    }
    
    // 转换为Hugo标准的RFC3339格式
    // 如果没有时区信息，默认使用本地时区(+08:00)
    if parsed.Location() == time.UTC && !strings.Contains(dateStr, "Z") && !strings.Contains(dateStr, "+") && !strings.Contains(dateStr, "-") {
        // 对于没有时区信息的时间，假设为本地时间
        loc, _ := time.LoadLocation("Asia/Shanghai")
        parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 
                          parsed.Hour(), parsed.Minute(), parsed.Second(), 
                          parsed.Nanosecond(), loc)
    }
    
    return parsed.Format(time.RFC3339), nil
}

// Claude Prompt: 修复博客文件中的时间格式
// repairArticleDate 修复单个文章文件的时间格式
func repairArticleDate(filePath string) error {
    // 读取文件内容
    data, err := os.ReadFile(filePath)
    if err != nil {
        return fmt.Errorf("读取文件失败: %v", err)
    }
    
    content := string(data)
    
    // 提取当前的时间字段
    _, _, _, _, _, _, date, _ := extractArticleMetadata(content)
    
    if date == "" {
        return fmt.Errorf("文件中没有找到日期字段")
    }
    
    // 检查是否需要修复
    if isValidHugoDate(date) {
        return nil // 格式已正确，无需修复
    }
    
    // 修复时间格式
    fixedDate, err := fixDateFormat(date)
    if err != nil {
        return fmt.Errorf("时间格式修复失败: %v", err)
    }
    
    // 替换文件中的时间字段
    newContent := replaceDateInFrontMatter(content, fixedDate)
    
    // 写回文件
    err = os.WriteFile(filePath, []byte(newContent), 0644)
    if err != nil {
        return fmt.Errorf("写入文件失败: %v", err)
    }
    
    return nil
}

// Claude Prompt: 提取部分正确的时间信息的容错函数
// extractPartialDateInfo 从错误的日期字符串中提取部分正确的信息
func extractPartialDateInfo(dateStr string, fallbackTime time.Time) (time.Time, int, time.Month, int, string) {
    // 尝试从日期字符串中提取年月日信息
    var year int
    var month int
    var day int
    
    // 使用正则表达式尝试提取年份 (4位数字)
    yearRegex := regexp.MustCompile(`(\d{4})`)
    if matches := yearRegex.FindStringSubmatch(dateStr); len(matches) > 1 {
        if parsedYear, err := strconv.Atoi(matches[1]); err == nil && parsedYear > 1900 && parsedYear < 3000 {
            year = parsedYear
        }
    }
    
    // 尝试提取月份和日期 (格式如 2023-01-15, 2023/01/15 等)
    datePartRegex := regexp.MustCompile(`(\d{4})[-/](\d{1,2})[-/](\d{1,2})`)
    if matches := datePartRegex.FindStringSubmatch(dateStr); len(matches) > 3 {
        if parsedYear, err := strconv.Atoi(matches[1]); err == nil && parsedYear > 1900 && parsedYear < 3000 {
            year = parsedYear
        }
        if parsedMonth, err := strconv.Atoi(matches[2]); err == nil && parsedMonth >= 1 && parsedMonth <= 12 {
            month = parsedMonth
        }
        if parsedDay, err := strconv.Atoi(matches[3]); err == nil && parsedDay >= 1 && parsedDay <= 31 {
            day = parsedDay
        }
    }
    
    // 如果无法提取有效信息，使用fallback时间
    if year == 0 {
        year = fallbackTime.Year()
    }
    if month == 0 {
        month = int(fallbackTime.Month())
    }
    if day == 0 {
        day = fallbackTime.Day()
    }
    
    // 尝试构建时间，如果失败则使用fallback
    var publishDate time.Time
    if month >= 1 && month <= 12 && day >= 1 && day <= 31 {
        // 使用本地时区创建时间
        loc, _ := time.LoadLocation("Asia/Shanghai")
        publishDate = time.Date(year, time.Month(month), day, 12, 0, 0, 0, loc)
        
        // 验证日期是否有效（例如2月30日会自动调整）
        if publishDate.Year() != year || int(publishDate.Month()) != month || publishDate.Day() != day {
            publishDate = fallbackTime
            year = fallbackTime.Year()
            month = int(fallbackTime.Month())
            day = fallbackTime.Day()
        }
    } else {
        publishDate = fallbackTime
        year = fallbackTime.Year()
        month = int(fallbackTime.Month())
        day = fallbackTime.Day()
    }
    
    formattedTime := publishDate.Format("2006-01-02 15:04:05")
    
    return publishDate, year, time.Month(month), day, formattedTime
}

// Claude Prompt: 在Front Matter中替换日期字段
// replaceDateInFrontMatter 在Front Matter中替换日期字段
func replaceDateInFrontMatter(content, newDate string) string {
    lines := strings.Split(content, "\n")
    inFrontMatter := false
    
    for i, line := range lines {
        trimmedLine := strings.TrimSpace(line)
        
        // 检测Front Matter开始/结束
        if trimmedLine == "---" {
            inFrontMatter = !inFrontMatter
            continue
        }
        
        // 在Front Matter中查找日期字段
        if inFrontMatter {
            if strings.HasPrefix(trimmedLine, "date:") {
                // 替换日期行
                lines[i] = "date: " + newDate
                break
            }
        }
    }
    
    return strings.Join(lines, "\n")
}

// Claude Prompt: 批量修复所有博客时间格式的API接口
// RepairAllArticleDates 批量修复所有博客文件的时间格式
func RepairAllArticleDates(c *gin.Context) {
    projectPath := config.GetHugoProjectPath()
    if projectPath == "" {
        i18nManager := utils.GetI18nManager()
        c.JSON(http.StatusBadRequest, gin.H{
            "error": i18nManager.T("settings.hugo.path.not.set"),
        })
        return
    }

    contentPath := filepath.Join(projectPath, "content")
    if _, err := os.Stat(contentPath); os.IsNotExist(err) {
        i18nManager := utils.GetI18nManager()
        c.JSON(http.StatusBadRequest, gin.H{
            "error": i18nManager.T("articles.path.not.exist"),
        })
        return
    }

    var repairedFiles []string
    var failedFiles []map[string]string
    totalFiles := 0

    // 遍历所有markdown文件
    err := filepath.Walk(contentPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        // 只处理markdown文件
        if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
            return nil
        }

        totalFiles++
        relativePath := strings.TrimPrefix(path, contentPath)
        relativePath = strings.TrimPrefix(relativePath, "/")

        // 尝试修复文件的时间格式
        if err := repairArticleDate(path); err != nil {
            failedFiles = append(failedFiles, map[string]string{
                "file":  relativePath,
                "error": err.Error(),
            })
        } else {
            // 检查是否实际进行了修复
            data, _ := os.ReadFile(path)
            content := string(data)
            _, _, _, _, _, _, date, _ := extractArticleMetadata(content)
            
            if date != "" && isValidHugoDate(date) {
                repairedFiles = append(repairedFiles, relativePath)
            }
        }

        return nil
    })

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "扫描文件失败: " + err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message":        "批量修复完成",
        "total_files":    totalFiles,
        "repaired_files": repairedFiles,
        "failed_files":   failedFiles,
        "repaired_count": len(repairedFiles),
        "failed_count":   len(failedFiles),
    })
}

// Claude Prompt: 单个文章时间修复API接口
// RepairSingleArticleDate 修复单个文章文件的时间格式
func RepairSingleArticleDate(c *gin.Context) {
    var request struct {
        Path string `json:"path"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "请求参数错误: " + err.Error(),
        })
        return
    }

    projectPath := config.GetHugoProjectPath()
    if projectPath == "" {
        i18nManager := utils.GetI18nManager()
        c.JSON(http.StatusBadRequest, gin.H{
            "error": i18nManager.T("settings.hugo.path.not.set"),
        })
        return
    }

    // 构建完整路径
    fullPath := filepath.Join(projectPath, "content", request.Path)

    // 检查文件是否存在
    if _, err := os.Stat(fullPath); os.IsNotExist(err) {
        c.JSON(http.StatusNotFound, gin.H{
            "error": "文件不存在: " + request.Path,
        })
        return
    }

    // 修复文件时间格式
    if err := repairArticleDate(fullPath); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "修复失败: " + err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "文章时间格式修复成功",
        "path":    request.Path,
    })
}

// Claude Prompt: 检查时间格式状态API接口
// CheckDateFormats 检查所有文章的时间格式状态
func CheckDateFormats(c *gin.Context) {
    projectPath := config.GetHugoProjectPath()
    if projectPath == "" {
        i18nManager := utils.GetI18nManager()
        c.JSON(http.StatusBadRequest, gin.H{
            "error": i18nManager.T("settings.hugo.path.not.set"),
        })
        return
    }

    contentPath := filepath.Join(projectPath, "content")
    if _, err := os.Stat(contentPath); os.IsNotExist(err) {
        i18nManager := utils.GetI18nManager()
        c.JSON(http.StatusBadRequest, gin.H{
            "error": i18nManager.T("articles.path.not.exist"),
        })
        return
    }

    var validFiles []string
    var invalidFiles []map[string]interface{}
    totalFiles := 0

    // 遍历所有markdown文件
    err := filepath.Walk(contentPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        // 只处理markdown文件
        if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
            return nil
        }

        totalFiles++
        relativePath := strings.TrimPrefix(path, contentPath)
        relativePath = strings.TrimPrefix(relativePath, "/")

        // 读取文件并检查时间格式
        data, err := os.ReadFile(path)
        if err != nil {
            invalidFiles = append(invalidFiles, map[string]interface{}{
                "file":   relativePath,
                "error":  "读取文件失败: " + err.Error(),
                "status": "read_error",
            })
            return nil
        }

        content := string(data)
        _, _, _, _, _, _, date, _ := extractArticleMetadata(content)

        if date == "" {
            invalidFiles = append(invalidFiles, map[string]interface{}{
                "file":     relativePath,
                "error":    "缺少日期字段",
                "status":   "missing_date",
                "current":  "",
            })
        } else if !isValidHugoDate(date) {
            fixedDate, _ := fixDateFormat(date)
            invalidFiles = append(invalidFiles, map[string]interface{}{
                "file":     relativePath,
                "error":    "时间格式不符合Hugo要求",
                "status":   "invalid_format",
                "current":  date,
                "suggested": fixedDate,
            })
        } else {
            validFiles = append(validFiles, relativePath)
        }

        return nil
    })

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "扫描文件失败: " + err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "total_files":    totalFiles,
        "valid_files":    validFiles,
        "invalid_files":  invalidFiles,
        "valid_count":    len(validFiles),
        "invalid_count":  len(invalidFiles),
    })
}

// Claude Prompt: 添加分离Front Matter和主体内容的函数
// separateContentAndFrontMatter 分离Front Matter和文章主体内容
func separateContentAndFrontMatter(content string) (frontMatter, bodyContent string) {
    lines := strings.Split(content, "\n")
    var frontMatterLines []string
    var bodyLines []string
    inFrontMatter := false
    frontMatterEnd := false
    
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        
        // 检查 front matter 开始和结束
        if trimmed == "---" || trimmed == "+++" {
            if !inFrontMatter && !frontMatterEnd {
                inFrontMatter = true
                frontMatterLines = append(frontMatterLines, line)
                continue
            } else if inFrontMatter {
                frontMatterEnd = true
                inFrontMatter = false
                frontMatterLines = append(frontMatterLines, line)
                continue
            }
        }
        
        if inFrontMatter {
            frontMatterLines = append(frontMatterLines, line)
        } else if frontMatterEnd {
            bodyLines = append(bodyLines, line)
        }
    }
    
    frontMatter = strings.Join(frontMatterLines, "\n")
    bodyContent = strings.Join(bodyLines, "\n")
    
    // 去掉开头的空行
    bodyContent = strings.TrimLeft(bodyContent, "\n")
    
    return frontMatter, bodyContent
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
    title, _, _, categories, tags, url, date, isDraft := extractArticleMetadata(content)
    if title == "" {
        // 如果没有找到标题，使用文件名（去掉扩展名）
        base := filepath.Base(filePath)
        title = strings.TrimSuffix(base, filepath.Ext(base))
    }
    
    // 生成摘要
    summary := generateSummary(content)
    
    // 检测文章问题
    issues := detectArticleIssues(content, title, categories, tags, url, date, filePath)
    
    // 解析发布日期，优先使用Front Matter中的date字段
    var publishDate time.Time
    var displayYear int
    var displayMonth time.Month
    var displayDay int
    var formattedTime string
    
    if date != "" {
        if parsed, err := parseArticleDate(date); err == nil {
            publishDate = parsed
            displayYear = parsed.Year()
            displayMonth = parsed.Month()
            displayDay = parsed.Day()
            formattedTime = parsed.Format("2006-01-02 15:04:05")
        } else {
            // 如果日期解析失败，尝试提取部分正确的时间信息
            publishDate, displayYear, displayMonth, displayDay, formattedTime = extractPartialDateInfo(date, modTime)
        }
    } else {
        // 如果没有日期字段，使用文件修改时间
        publishDate = modTime
        displayYear = modTime.Year()
        displayMonth = modTime.Month()
        displayDay = modTime.Day()
        formattedTime = modTime.Format("2006-01-02 15:04:05")
    }
    
    return ArticleInfo{
        Path:          relativePath,
        ModTime:       modTime,
        Size:          size,
        Year:          displayYear,
        Month:         displayMonth,
        Day:           displayDay,
        FormattedTime: formattedTime,
        PublishDate:   publishDate,
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
func extractArticleMetadata(content string) (title, author, articleType string, categories, tags []string, url, date string, isDraft bool) {
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
            } else if strings.HasPrefix(line, "author:") {
                author = strings.TrimSpace(strings.TrimPrefix(line, "author:"))
                author = strings.Trim(author, "\"'")
            } else if strings.HasPrefix(line, "type:") {
                articleType = strings.TrimSpace(strings.TrimPrefix(line, "type:"))
                articleType = strings.Trim(articleType, "\"'")
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
    } else if !isValidHugoDate(date) {
        issues = append(issues, "发布时间格式不符合Hugo要求(需RFC3339格式)")
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

// GetArticlesAPI 通过API返回文章列表数据
func GetArticlesAPI(c *gin.Context) {
    // 获取查询参数
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    if page < 1 {
        page = 1
    }
    
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
    if pageSize < 1 || pageSize > 100 {
        pageSize = 20
    }
    
    year, _ := strconv.Atoi(c.Query("year"))
    month, _ := strconv.Atoi(c.Query("month"))
    search := strings.TrimSpace(c.Query("search"))
    status := strings.TrimSpace(c.Query("status"))
    
    // 获取所有文章信息
    articleInfos, err := getAllArticlesWithContent()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": err.Error(),
        })
        return
    }
    
    
    // 按状态筛选
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
    
    c.JSON(http.StatusOK, gin.H{
        "articles":       currentPageArticles,
        "current_page":   page,
        "total_pages":    totalPages,
        "total_articles": totalArticles,
        "page_size":      pageSize,
        "has_next":       page < totalPages,
        "has_prev":       page > 1,
    })
}

// Claude Prompt: 修改文章统计API，添加年份和月份统计信息
// GetArticleStatsAPI 通过API返回文章统计信息
func GetArticleStatsAPI(c *gin.Context) {
    // 获取所有文章信息
    allArticles, err := getAllArticlesWithContent()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": err.Error(),
        })
        return
    }
    
    // 统计各种状态的文章数量
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
    
    // 获取可用年份（兼容原有前端）
    availableYears := getAvailableYears(allArticles)
    
    // 获取年份统计信息（包含博客数量）
    yearStats := getYearStats(allArticles)
    
    // 获取月份统计信息（包含博客数量）
    monthStats := getMonthStats(allArticles)
    
    c.JSON(http.StatusOK, gin.H{
        "total_articles":   len(allArticles),
        "draft_count":      draftCount,
        "published_count":  publishedCount,
        "issues_count":     issuesCount,
        "available_years":  availableYears,
        "year_stats":       yearStats,
        "month_stats":      monthStats,
        "hugo_project_path": config.GetHugoProjectPath(),
    })
}

// Claude Prompt: 添加Hugo server检测和自动启动API
// CheckHugoServerAPI 检查Hugo server状态，如果未运行则自动启动
func CheckHugoServerAPI(c *gin.Context) {
    hugoManager := utils.GetHugoServeManager()
    
    // 检查当前状态
    if hugoManager.IsRunning() {
        // Hugo server正在运行，返回状态
        status := hugoManager.GetStatus()
        c.JSON(http.StatusOK, gin.H{
            "running": true,
            "status":  "already_running",
            "port":    status["port"],
            "url":     status["url"],
            "message": "Hugo server已经在运行中",
        })
        return
    }
    
    // Hugo server未运行，尝试启动
    err := hugoManager.Start(1313)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "running": false,
            "status":  "start_failed",
            "error":   err.Error(),
            "message": "启动Hugo server失败",
        })
        return
    }
    
    // 等待一小段时间确保服务启动
    time.Sleep(2 * time.Second)
    
    // 返回启动成功状态
    status := hugoManager.GetStatus()
    c.JSON(http.StatusOK, gin.H{
        "running": true,
        "status":  "started",
        "port":    status["port"],
        "url":     status["url"],
        "message": "Hugo server启动成功",
    })
}

// GetHugoServerStatusAPI 获取Hugo server当前状态
func GetHugoServerStatusAPI(c *gin.Context) {
    hugoManager := utils.GetHugoServeManager()
    status := hugoManager.GetStatus()
    
    c.JSON(http.StatusOK, status)
}
