package controller

import (
    "fmt"
    "github.com/gin-gonic/gin"
    "hugo-manager-go/config"
    "hugo-manager-go/utils"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"
)

type TreeNode struct {
    Name     string      `json:"name"`
    Path     string      `json:"path"`
    IsDir    bool        `json:"is_dir"`
    Children []*TreeNode `json:"children,omitempty"`
}

type FileInfo struct {
    Name     string    `json:"name"`
    Path     string    `json:"path"`
    IsDir    bool      `json:"is_dir"`
    Size     int64     `json:"size"`
    ModTime  time.Time `json:"mod_time"`
    IsMarkdown bool    `json:"is_markdown"`
}

// 文件管理主页面
func FileManager(c *gin.Context) {
    c.HTML(200, "files/manager.html", gin.H{
        "Title":           "文件管理",
        "HugoProjectPath": config.GetHugoProjectPath(),
    })
}

// 文件编辑页面
func FileEditor(c *gin.Context) {
    filePath := c.Query("file")
    if filePath == "" {
        c.Redirect(302, "/files")
        return
    }
    
    c.HTML(200, "files/editor.html", gin.H{
        "Title":    "编辑文件",
        "FilePath": filePath,
    })
}

// 获取目录树结构
func GetDirectoryTree(c *gin.Context) {
    rootPath := config.GetContentDir()
    
    if _, err := os.Stat(rootPath); os.IsNotExist(err) {
        c.JSON(404, gin.H{"error": "content目录不存在"})
        return
    }
    
    tree, err := buildDirectoryTree(rootPath, "")
    if err != nil {
        c.JSON(500, gin.H{"error": "构建目录树失败: " + err.Error()})
        return
    }
    
    c.JSON(200, tree)
}

// 获取指定目录下的文件列表
func GetFiles(c *gin.Context) {
    relativePath := c.Query("path")
    rootPath := config.GetContentDir()
    fullPath := filepath.Join(rootPath, relativePath)
    
    if _, err := os.Stat(fullPath); os.IsNotExist(err) {
        c.JSON(404, gin.H{"error": "目录不存在"})
        return
    }
    
    entries, err := os.ReadDir(fullPath)
    if err != nil {
        c.JSON(500, gin.H{"error": "读取目录失败: " + err.Error()})
        return
    }
    
    var files []FileInfo
    
    for _, entry := range entries {
        info, err := entry.Info()
        if err != nil {
            continue
        }
        
        originalName := entry.Name()
        
        // 清理文件名用于显示和访问
        cleanName := utils.CleanFilename(originalName)
        
        // 跳过清理后无效的文件名
        if !utils.ValidateFilename(cleanName) {
            continue
        }
        
        // 构建相对路径，使用清理后的文件名
        var relPath string
        if relativePath == "" || relativePath == "." {
            relPath = cleanName
        } else {
            // 确保路径分隔符正确，使用Clean来规范化路径
            relPath = filepath.Clean(filepath.Join(relativePath, cleanName))
            // 转换为正斜杠格式，确保前端能正确处理
            relPath = filepath.ToSlash(relPath)
        }
        
        fileInfo := FileInfo{
            Name:       cleanName,   // 显示清理后的名称
            Path:       relPath,     // 使用清理后的路径
            IsDir:      entry.IsDir(),
            Size:       info.Size(),
            ModTime:    info.ModTime(),
            IsMarkdown: strings.HasSuffix(strings.ToLower(cleanName), ".md"),
        }
        
        files = append(files, fileInfo)
    }
    
    // 排序：目录在前，文件在后，按名称排序
    sort.Slice(files, func(i, j int) bool {
        if files[i].IsDir != files[j].IsDir {
            return files[i].IsDir
        }
        return files[i].Name < files[j].Name
    })
    
    c.JSON(200, gin.H{
        "current_path": relativePath,
        "files":        files,
    })
}

// 获取文件内容
func GetFileContent(c *gin.Context) {
    relativePath := c.Query("path")
    if relativePath == "" {
        c.JSON(400, gin.H{"error": "文件路径不能为空"})
        return
    }
    
    // 标准化路径分隔符，将URL路径转换为系统路径
    relativePath = filepath.FromSlash(relativePath)
    
    // 清理路径中的文件名，但保持路径结构
    dir := filepath.Dir(relativePath)
    filename := filepath.Base(relativePath)
    cleanFilename := utils.CleanFilename(filename)
    
    // 重新构建清理后的路径，确保路径分隔符正确
    var cleanRelativePath string
    if dir == "." || dir == "" {
        cleanRelativePath = cleanFilename
    } else {
        // 确保目录路径正确规范化
        cleanRelativePath = filepath.Clean(filepath.Join(dir, cleanFilename))
    }
    
    fullPath := filepath.Join(config.GetContentDir(), cleanRelativePath)
    
    // 验证路径安全性（防止路径遍历攻击）
    contentDir := config.GetContentDir()
    absFullPath, err := filepath.Abs(fullPath)
    if err != nil {
        c.JSON(400, gin.H{"error": "无效的文件路径"})
        return
    }
    
    absContentDir, err := filepath.Abs(contentDir)
    if err != nil {
        c.JSON(400, gin.H{"error": "内容目录路径错误"})
        return
    }
    
    // 检查文件路径是否在content目录内
    if !strings.HasPrefix(absFullPath, absContentDir) {
        c.JSON(400, gin.H{"error": "访问被拒绝：路径不在允许的目录内"})
        return
    }
    
    content, err := os.ReadFile(fullPath)
    if err != nil {
        c.JSON(500, gin.H{"error": "读取文件失败: " + err.Error()})
        return
    }
    
    isMarkdown := strings.HasSuffix(strings.ToLower(cleanFilename), ".md")
    response := gin.H{
        "path":        cleanRelativePath,
        "content":     string(content),
        "is_markdown": isMarkdown,
    }
    
    // 如果是Markdown文件，解析front matter
    if isMarkdown {
        parsed, err := utils.ParseMarkdown(string(content))
        if err != nil {
            response["parse_error"] = err.Error()
        } else {
            response["parsed"] = parsed
        }
    }
    
    c.JSON(200, response)
}

// 保存文件内容
func SaveFileContent(c *gin.Context) {
    var request struct {
        Path        string           `json:"path"`
        Content     string           `json:"content"`
        IsMarkdown  bool             `json:"is_markdown"`
        FrontMatter *utils.FrontMatter `json:"front_matter,omitempty"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": "请求格式错误"})
        return
    }
    
    if request.Path == "" {
        c.JSON(400, gin.H{"error": "文件路径不能为空"})
        return
    }
    
    var finalContent string
    
    // 如果是Markdown文件且提供了front matter，重新构建文件
    if request.IsMarkdown && request.FrontMatter != nil {
        built, err := utils.BuildMarkdown(*request.FrontMatter, request.Content)
        if err != nil {
            c.JSON(500, gin.H{"error": "构建Markdown失败: " + err.Error()})
            return
        }
        finalContent = built
    } else {
        finalContent = request.Content
    }
    
    // 清理文件路径
    dir := filepath.Dir(request.Path)
    filename := filepath.Base(request.Path)
    cleanFilename := utils.CleanFilename(filename)
    
    var cleanPath string
    if dir == "." || dir == "" {
        cleanPath = cleanFilename
    } else {
        // 确保目录路径正确规范化
        cleanPath = filepath.Clean(filepath.Join(dir, cleanFilename))
    }
    
    fullPath := filepath.Join(config.GetContentDir(), cleanPath)
    
    // 验证路径安全性
    contentDir := config.GetContentDir()
    absFullPath, err := filepath.Abs(fullPath)
    if err != nil {
        c.JSON(400, gin.H{"error": "无效的文件路径"})
        return
    }
    
    absContentDir, err := filepath.Abs(contentDir)
    if err != nil {
        c.JSON(400, gin.H{"error": "内容目录路径错误"})
        return
    }
    
    if !strings.HasPrefix(absFullPath, absContentDir) {
        c.JSON(400, gin.H{"error": "访问被拒绝：路径不在允许的目录内"})
        return
    }
    
    // 确保目录存在
    saveDir := filepath.Dir(fullPath)
    if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
        c.JSON(500, gin.H{"error": "创建目录失败: " + err.Error()})
        return
    }
    
    if err := os.WriteFile(fullPath, []byte(finalContent), 0644); err != nil {
        c.JSON(500, gin.H{"error": "保存文件失败: " + err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"message": "文件保存成功"})
}

// 创建新文章
func CreateNewArticle(c *gin.Context) {
    var request struct {
        Title      string `json:"title"`
        Directory  string `json:"directory"`
        Author     string `json:"author"`
        Type       string `json:"type"`
        Categories []string `json:"categories"`
        Tags       []string `json:"tags"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": "请求格式错误"})
        return
    }
    
    if request.Title == "" {
        c.JSON(400, gin.H{"error": "标题不能为空"})
        return
    }
    
    // 生成文件名：日期+标题
    now := time.Now()
    dateStr := now.Format("2006-01-02")
    
    // 使用新的标题清理函数
    cleanTitle := utils.SanitizeTitle(request.Title)
    
    filename := fmt.Sprintf("%s-%s.md", dateStr, cleanTitle)
    
    // 确定目录路径，默认为posts
    directory := request.Directory
    if directory == "" {
        directory = "posts"
    }
    
    // 构建完整路径
    relativePath := filepath.Join(directory, filename)
    fullPath := filepath.Join(config.GetContentDir(), relativePath)
    
    // 检查文件是否已存在
    if _, err := os.Stat(fullPath); err == nil {
        c.JSON(409, gin.H{"error": "文件已存在: " + relativePath})
        return
    }
    
    // 生成URL规则：/p/年/月/随机数字.html
    year := now.Format("2006")
    month := now.Format("01")
    // 生成4位随机数字
    randomNum := fmt.Sprintf("%04d", time.Now().UnixNano()%10000)
    blogURL := fmt.Sprintf("/p/%s/%s/%s.html", year, month, randomNum)
    
    // 创建Front Matter
    frontMatter := utils.FrontMatter{
        Title:      request.Title,
        Author:     request.Author,
        Type:       request.Type,
        Date:       now.Format("2006-01-02T15:04:05+08:00"),
        Categories: request.Categories,
        Tags:       request.Tags,
        URL:        blogURL,
    }
    
    // 如果没有指定类型，默认为post
    if frontMatter.Type == "" {
        frontMatter.Type = "post"
    }
    
    // 生成默认内容
    defaultContent := fmt.Sprintf("# %s\n\n在这里写入文章内容...", request.Title)
    
    // 构建Markdown内容
    markdownContent, err := utils.BuildMarkdown(frontMatter, defaultContent)
    if err != nil {
        c.JSON(500, gin.H{"error": "生成文章内容失败: " + err.Error()})
        return
    }
    
    // 确保目录存在
    dir := filepath.Dir(fullPath)
    if err := os.MkdirAll(dir, os.ModePerm); err != nil {
        c.JSON(500, gin.H{"error": "创建目录失败: " + err.Error()})
        return
    }
    
    // 写入文件
    if err := os.WriteFile(fullPath, []byte(markdownContent), 0644); err != nil {
        c.JSON(500, gin.H{"error": "创建文件失败: " + err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "message": "文章创建成功",
        "path":    relativePath,
        "filename": filename,
    })
}

// 构建目录树
func buildDirectoryTree(basePath, relativePath string) (*TreeNode, error) {
    fullPath := filepath.Join(basePath, relativePath)
    info, err := os.Stat(fullPath)
    if err != nil {
        return nil, err
    }
    
    name := info.Name()
    if relativePath == "" {
        name = "content"
    }
    
    node := &TreeNode{
        Name:  name,
        Path:  relativePath,
        IsDir: info.IsDir(),
    }
    
    if info.IsDir() {
        entries, err := os.ReadDir(fullPath)
        if err != nil {
            return node, nil // 返回节点但不包含子节点
        }
        
        for _, entry := range entries {
            if entry.IsDir() {
                childPath := filepath.Join(relativePath, entry.Name())
                if child, err := buildDirectoryTree(basePath, childPath); err == nil {
                    node.Children = append(node.Children, child)
                }
            }
        }
        
        // 按名称排序子节点
        sort.Slice(node.Children, func(i, j int) bool {
            return node.Children[i].Name < node.Children[j].Name
        })
    }
    
    return node, nil
}

// Claude Prompt: 实现文章预览API，解析markdown文件内容和元数据
func PreviewArticle(c *gin.Context) {
    relativePath := c.Query("path")
    if relativePath == "" {
        c.JSON(400, gin.H{"error": "文件路径不能为空"})
        return
    }
    
    // 标准化路径分隔符
    relativePath = filepath.FromSlash(relativePath)
    
    // 清理路径
    dir := filepath.Dir(relativePath)
    filename := filepath.Base(relativePath)
    cleanFilename := utils.CleanFilename(filename)
    
    var cleanRelativePath string
    if dir == "." || dir == "" {
        cleanRelativePath = cleanFilename
    } else {
        cleanRelativePath = filepath.Clean(filepath.Join(dir, cleanFilename))
    }
    
    fullPath := filepath.Join(config.GetContentDir(), cleanRelativePath)
    
    // 验证路径安全性
    contentDir := config.GetContentDir()
    absFullPath, err := filepath.Abs(fullPath)
    if err != nil {
        c.JSON(400, gin.H{"error": "无效的文件路径"})
        return
    }
    
    absContentDir, err := filepath.Abs(contentDir)
    if err != nil {
        c.JSON(500, gin.H{"error": "内部错误：无法解析内容目录"})
        return
    }
    
    if !strings.HasPrefix(absFullPath, absContentDir) {
        c.JSON(403, gin.H{"error": "禁止访问此路径"})
        return
    }
    
    // 读取文件内容
    content, err := os.ReadFile(fullPath)
    if err != nil {
        c.JSON(404, gin.H{"error": "文件不存在或无法读取"})
        return
    }
    
    // 解析markdown文件的front matter和内容
    fileContent := string(content)
    frontMatter, markdownContent := parseFrontMatter(fileContent)
    
    // 转换markdown为HTML（简单处理，可后续增强）
    htmlContent := convertMarkdownToHTML(markdownContent)
    
    c.JSON(200, gin.H{
        "title":    frontMatter["title"],
        "metadata": frontMatter,
        "content":  htmlContent,
        "raw_content": markdownContent,
    })
}

// 解析Front Matter
func parseFrontMatter(content string) (map[string]interface{}, string) {
    metadata := make(map[string]interface{})
    
    if !strings.HasPrefix(content, "---") {
        return metadata, content
    }
    
    // 查找第二个---的位置
    parts := strings.SplitN(content, "---", 3)
    if len(parts) < 3 {
        return metadata, content
    }
    
    // 解析YAML front matter
    frontMatterContent := strings.TrimSpace(parts[1])
    lines := strings.Split(frontMatterContent, "\n")
    
    var currentKey string
    var currentArray []string
    
    for _, line := range lines {
        line = strings.TrimSpace(line)
        
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        
        // 检查是否是数组项（以-开头）
        if strings.HasPrefix(line, "-") {
            if currentKey != "" {
                arrayValue := strings.TrimSpace(line[1:])
                // 移除引号
                if strings.HasPrefix(arrayValue, "\"") && strings.HasSuffix(arrayValue, "\"") {
                    arrayValue = arrayValue[1 : len(arrayValue)-1]
                }
                currentArray = append(currentArray, arrayValue)
            }
            continue
        }
        
        // 如果遇到新的键值对，先保存之前的数组
        if currentKey != "" && len(currentArray) > 0 {
            metadata[currentKey] = currentArray
            currentArray = nil
        }
        
        if colonIndex := strings.Index(line, ":"); colonIndex > 0 {
            key := strings.TrimSpace(line[:colonIndex])
            value := strings.TrimSpace(line[colonIndex+1:])
            
            // 如果值为空，可能是数组的开始
            if value == "" {
                currentKey = key
                currentArray = []string{}
                continue
            }
            
            currentKey = ""
            
            // 移除引号
            if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
                value = value[1 : len(value)-1]
            }
            
            // 处理方括号格式的数组
            if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
                value = strings.Trim(value, "[]")
                if value == "" {
                    metadata[key] = []string{}
                } else {
                    items := strings.Split(value, ",")
                    var cleanItems []string
                    for _, item := range items {
                        item = strings.TrimSpace(item)
                        item = strings.Trim(item, "\"'")
                        if item != "" {
                            cleanItems = append(cleanItems, item)
                        }
                    }
                    metadata[key] = cleanItems
                }
            } else {
                metadata[key] = value
            }
        }
    }
    
    // 保存最后的数组
    if currentKey != "" && len(currentArray) > 0 {
        metadata[currentKey] = currentArray
    }
    
    // 返回markdown内容（去除front matter）
    markdownContent := strings.TrimSpace(parts[2])
    return metadata, markdownContent
}

// Claude Prompt: 实现新建文件夹功能，支持在posts目录下创建文件夹
func CreateFolder(c *gin.Context) {
    var request struct {
        ParentPath  string `json:"parent_path"`
        FolderName  string `json:"folder_name"`
        Description string `json:"description"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": "请求格式错误"})
        return
    }
    
    if request.FolderName == "" {
        c.JSON(400, gin.H{"error": "文件夹名称不能为空"})
        return
    }
    
    // 验证文件夹名称格式，只允许字母、数字、中文、中划线和下划线
    if !utils.ValidateFilename(request.FolderName) {
        c.JSON(400, gin.H{"error": "文件夹名称只能包含字母、数字、中文、中划线和下划线"})
        return
    }
    
    // 清理文件夹名称
    cleanFolderName := utils.CleanFilename(request.FolderName)
    
    // 构建完整路径
    var relativePath string
    if request.ParentPath == "" {
        relativePath = cleanFolderName
    } else {
        relativePath = filepath.Join(request.ParentPath, cleanFolderName)
    }
    
    fullPath := filepath.Join(config.GetContentDir(), relativePath)
    
    // 验证路径安全性（防止路径遍历攻击）
    contentDir := config.GetContentDir()
    absFullPath, err := filepath.Abs(fullPath)
    if err != nil {
        c.JSON(400, gin.H{"error": "无效的文件夹路径"})
        return
    }
    
    absContentDir, err := filepath.Abs(contentDir)
    if err != nil {
        c.JSON(500, gin.H{"error": "内部错误：无法解析内容目录"})
        return
    }
    
    if !strings.HasPrefix(absFullPath, absContentDir) {
        c.JSON(403, gin.H{"error": "禁止在此位置创建文件夹"})
        return
    }
    
    // 检查文件夹是否已存在
    if _, err := os.Stat(fullPath); err == nil {
        c.JSON(409, gin.H{"error": "文件夹已存在: " + cleanFolderName})
        return
    }
    
    // 创建文件夹
    if err := os.MkdirAll(fullPath, os.ModePerm); err != nil {
        c.JSON(500, gin.H{"error": "创建文件夹失败: " + err.Error()})
        return
    }
    
    // 如果提供了描述，创建一个README.md文件
    if request.Description != "" {
        readmePath := filepath.Join(fullPath, "README.md")
        readmeContent := fmt.Sprintf("# %s\n\n%s", cleanFolderName, request.Description)
        if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
            // README创建失败不影响整体操作，只记录日志
            fmt.Printf("Warning: 无法创建README文件: %v\n", err)
        }
    }
    
    c.JSON(200, gin.H{
        "message": "文件夹创建成功",
        "path":    relativePath,
        "name":    cleanFolderName,
    })
}

// 简单的Markdown转HTML（基础实现）
func convertMarkdownToHTML(markdown string) string {
    html := markdown
    
    // 处理标题
    lines := strings.Split(html, "\n")
    for i, line := range lines {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "# ") {
            lines[i] = "<h1>" + line[2:] + "</h1>"
        } else if strings.HasPrefix(line, "## ") {
            lines[i] = "<h2>" + line[3:] + "</h2>"
        } else if strings.HasPrefix(line, "### ") {
            lines[i] = "<h3>" + line[4:] + "</h3>"
        } else if strings.HasPrefix(line, "#### ") {
            lines[i] = "<h4>" + line[5:] + "</h4>"
        } else if strings.HasPrefix(line, "##### ") {
            lines[i] = "<h5>" + line[6:] + "</h5>"
        } else if strings.HasPrefix(line, "###### ") {
            lines[i] = "<h6>" + line[7:] + "</h6>"
        }
    }
    
    html = strings.Join(lines, "\n")
    
    // 处理段落（简单处理：用<br>替换单个换行，用<p>包装段落）
    paragraphs := strings.Split(html, "\n\n")
    var htmlParagraphs []string
    
    for _, paragraph := range paragraphs {
        paragraph = strings.TrimSpace(paragraph)
        if paragraph != "" {
            // 如果已经是HTML标签，直接使用
            if strings.HasPrefix(paragraph, "<h") || strings.HasPrefix(paragraph, "<ul") || 
               strings.HasPrefix(paragraph, "<ol") || strings.HasPrefix(paragraph, "<blockquote") {
                htmlParagraphs = append(htmlParagraphs, paragraph)
            } else {
                // 处理行内换行
                paragraph = strings.ReplaceAll(paragraph, "\n", "<br>")
                htmlParagraphs = append(htmlParagraphs, "<p>"+paragraph+"</p>")
            }
        }
    }
    
    return strings.Join(htmlParagraphs, "\n")
}