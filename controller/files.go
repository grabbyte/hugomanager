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
    
    // 创建Front Matter
    frontMatter := utils.FrontMatter{
        Title:      request.Title,
        Author:     request.Author,
        Type:       request.Type,
        Date:       now.Format("2006-01-02T15:04:05+08:00"),
        Categories: request.Categories,
        Tags:       request.Tags,
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