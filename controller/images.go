package controller

import (
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"
    "github.com/gin-gonic/gin"
    "hugo-manager-go/config"
)

type ImageInfo struct {
    Name     string    `json:"name"`
    Path     string    `json:"path"`
    Size     int64     `json:"size"`
    ModTime  time.Time `json:"mod_time"`
    URL      string    `json:"url"`
    IsImage  bool      `json:"is_image"`
}

// 静态文件管理页面
func ImageManager(c *gin.Context) {
    c.HTML(200, "images/index.html", gin.H{
        "Title": "静态文件管理",
        "HugoProjectPath": config.GetHugoProjectPath(),
        "StaticDir": config.GetStaticDir(),
    })
}

// 获取图片列表
func GetImages(c *gin.Context) {
    // 获取查询参数中的目录路径
    relativePath := c.Query("path")
    
    staticDir := config.GetStaticDir()
    
    // 确保static目录存在
    if err := os.MkdirAll(staticDir, os.ModePerm); err != nil {
        c.JSON(500, gin.H{"error": "创建static目录失败: " + err.Error()})
        return
    }
    
    // 构建目标目录路径
    var targetDir string
    if relativePath == "" {
        targetDir = staticDir
    } else {
        targetDir = filepath.Join(staticDir, relativePath)
    }
    
    // 安全检查：确保路径在static目录内
    absTargetDir, err := filepath.Abs(targetDir)
    if err != nil {
        c.JSON(400, gin.H{"error": "无效的目录路径"})
        return
    }
    
    absStaticDir, err := filepath.Abs(staticDir)
    if err != nil {
        c.JSON(400, gin.H{"error": "static目录路径错误"})
        return
    }
    
    if !strings.HasPrefix(absTargetDir, absStaticDir) {
        c.JSON(400, gin.H{"error": "访问被拒绝：路径不在允许的目录内"})
        return
    }
    
    // 检查目录是否存在
    if _, err := os.Stat(targetDir); os.IsNotExist(err) {
        c.JSON(404, gin.H{"error": "目录不存在"})
        return
    }
    
    var images []ImageInfo
    
    // 只读取当前目录下的文件（不递归）
    entries, err := os.ReadDir(targetDir)
    if err != nil {
        c.JSON(500, gin.H{"error": "读取目录失败: " + err.Error()})
        return
    }
    
    for _, entry := range entries {
        if entry.IsDir() {
            continue // 跳过目录，只处理文件
        }
        
        info, err := entry.Info()
        if err != nil {
            continue
        }
        
        // 检查是否为图片文件或其他静态文件
        ext := strings.ToLower(filepath.Ext(info.Name()))
        isImage := ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" || ext == ".svg" || ext == ".bmp" || ext == ".ico"
        
        // 构建相对路径和URL路径
        var relPath string
        if relativePath == "" {
            relPath = info.Name()
        } else {
            relPath = filepath.Join(relativePath, info.Name())
        }
        
        // 生成相对于static的URL路径
        urlPath := "/static/" + filepath.ToSlash(relPath)
        
        images = append(images, ImageInfo{
            Name:    info.Name(),
            Path:    relPath,
            Size:    info.Size(),
            ModTime: info.ModTime(),
            URL:     urlPath,
            IsImage: isImage,
        })
    }
    
    // 按修改时间排序，最新的在前
    sort.Slice(images, func(i, j int) bool {
        return images[i].ModTime.After(images[j].ModTime)
    })
    
    c.JSON(200, gin.H{
        "images":      images,
        "count":       len(images),
        "current_path": relativePath,
    })
}

// 删除图片
func DeleteImage(c *gin.Context) {
    var request struct {
        Path string `json:"path"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": "请求格式错误"})
        return
    }
    
    if request.Path == "" {
        c.JSON(400, gin.H{"error": "图片路径不能为空"})
        return
    }
    
    // 安全检查：确保路径在static目录内
    staticDir := config.GetStaticDir()
    fullPath := filepath.Join(staticDir, request.Path)
    
    absFullPath, err := filepath.Abs(fullPath)
    if err != nil {
        c.JSON(400, gin.H{"error": "无效的文件路径"})
        return
    }
    
    absStaticDir, err := filepath.Abs(staticDir)
    if err != nil {
        c.JSON(400, gin.H{"error": "static目录路径错误"})
        return
    }
    
    if !strings.HasPrefix(absFullPath, absStaticDir) {
        c.JSON(400, gin.H{"error": "访问被拒绝：路径不在允许的目录内"})
        return
    }
    
    // 检查文件是否存在
    if _, err := os.Stat(fullPath); os.IsNotExist(err) {
        c.JSON(404, gin.H{"error": "图片不存在"})
        return
    }
    
    // 删除文件
    if err := os.Remove(fullPath); err != nil {
        c.JSON(500, gin.H{"error": "删除图片失败: " + err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"message": "图片删除成功"})
}

// 批量删除图片
func DeleteImages(c *gin.Context) {
    var request struct {
        Paths []string `json:"paths"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": "请求格式错误"})
        return
    }
    
    if len(request.Paths) == 0 {
        c.JSON(400, gin.H{"error": "未选择要删除的图片"})
        return
    }
    
    staticDir := config.GetStaticDir()
    absStaticDir, err := filepath.Abs(staticDir)
    if err != nil {
        c.JSON(400, gin.H{"error": "static目录路径错误"})
        return
    }
    
    var deleted []string
    var failed []string
    
    for _, path := range request.Paths {
        fullPath := filepath.Join(staticDir, path)
        absFullPath, err := filepath.Abs(fullPath)
        
        if err != nil || !strings.HasPrefix(absFullPath, absStaticDir) {
            failed = append(failed, path+": 路径无效")
            continue
        }
        
        if _, err := os.Stat(fullPath); os.IsNotExist(err) {
            failed = append(failed, path+": 文件不存在")
            continue
        }
        
        if err := os.Remove(fullPath); err != nil {
            failed = append(failed, path+": "+err.Error())
            continue
        }
        
        deleted = append(deleted, path)
    }
    
    c.JSON(200, gin.H{
        "message":      fmt.Sprintf("删除操作完成，成功删除 %d 个文件", len(deleted)),
        "deleted":      deleted,
        "failed":       failed,
        "deleted_count": len(deleted),
        "failed_count":  len(failed),
    })
}

// 创建文件夹
func CreateImageFolder(c *gin.Context) {
    var request struct {
        FolderName string `json:"folder_name"`
        ParentPath string `json:"parent_path"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": "请求格式错误"})
        return
    }
    
    if request.FolderName == "" {
        c.JSON(400, gin.H{"error": "文件夹名称不能为空"})
        return
    }
    
    // 清理文件夹名称
    folderName := strings.TrimSpace(request.FolderName)
    if folderName != request.FolderName || strings.Contains(folderName, "/") || strings.Contains(folderName, "\\") {
        c.JSON(400, gin.H{"error": "文件夹名称包含非法字符"})
        return
    }
    
    staticDir := config.GetStaticDir()
    var folderPath string
    
    if request.ParentPath == "" {
        folderPath = filepath.Join(staticDir, folderName)
    } else {
        folderPath = filepath.Join(staticDir, request.ParentPath, folderName)
    }
    
    // 安全检查
    absFolderPath, err := filepath.Abs(folderPath)
    if err != nil {
        c.JSON(400, gin.H{"error": "路径处理失败"})
        return
    }
    
    absStaticDir, err := filepath.Abs(staticDir)
    if err != nil {
        c.JSON(400, gin.H{"error": "static目录路径错误"})
        return
    }
    
    if !strings.HasPrefix(absFolderPath, absStaticDir) {
        c.JSON(400, gin.H{"error": "访问被拒绝：路径不在允许的目录内"})
        return
    }
    
    // 检查文件夹是否已存在
    if _, err := os.Stat(folderPath); err == nil {
        c.JSON(409, gin.H{"error": "文件夹已存在"})
        return
    }
    
    // 创建文件夹
    if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
        c.JSON(500, gin.H{"error": "创建文件夹失败: " + err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "message":     "文件夹创建成功",
        "folder_name": folderName,
        "folder_path": filepath.ToSlash(filepath.Join(request.ParentPath, folderName)),
    })
}

// 获取图片目录树
func GetImageDirectories(c *gin.Context) {
    staticDir := config.GetStaticDir()
    
    // 确保static目录存在
    if err := os.MkdirAll(staticDir, os.ModePerm); err != nil {
        c.JSON(500, gin.H{"error": "创建static目录失败: " + err.Error()})
        return
    }
    
    type DirNode struct {
        Name     string     `json:"name"`
        Path     string     `json:"path"`
        Children []*DirNode `json:"children,omitempty"`
    }
    
    var buildDirTree func(basePath, relativePath string) (*DirNode, error)
    buildDirTree = func(basePath, relativePath string) (*DirNode, error) {
        fullPath := filepath.Join(basePath, relativePath)
        info, err := os.Stat(fullPath)
        if err != nil {
            return nil, err
        }
        
        name := info.Name()
        if relativePath == "" {
            name = "static"
        }
        
        node := &DirNode{
            Name: name,
            Path: filepath.ToSlash(relativePath),
        }
        
        if info.IsDir() {
            entries, err := os.ReadDir(fullPath)
            if err != nil {
                return node, nil
            }
            
            for _, entry := range entries {
                if entry.IsDir() {
                    childPath := filepath.Join(relativePath, entry.Name())
                    if child, err := buildDirTree(basePath, childPath); err == nil {
                        node.Children = append(node.Children, child)
                    }
                }
            }
            
            // 按名称排序
            sort.Slice(node.Children, func(i, j int) bool {
                return node.Children[i].Name < node.Children[j].Name
            })
        }
        
        return node, nil
    }
    
    tree, err := buildDirTree(staticDir, "")
    if err != nil {
        c.JSON(500, gin.H{"error": "构建目录树失败: " + err.Error()})
        return
    }
    
    c.JSON(200, tree)
}

// 获取静态文件统计信息
func GetImageStats(c *gin.Context) {
    staticDir := config.GetStaticDir()
    
    var totalFiles int
    var totalSize int64
    var imageCount int
    var imageSize int64
    var typeCount = make(map[string]int)
    var typeSizes = make(map[string]int64)
    
    err := filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        if info.IsDir() {
            return nil
        }
        
        // 统计所有文件
        totalFiles++
        totalSize += info.Size()
        
        ext := strings.ToLower(filepath.Ext(info.Name()))
        isImage := ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" || ext == ".svg" || ext == ".bmp" || ext == ".ico"
        
        // 按类型统计
        typeCount[ext]++
        typeSizes[ext] += info.Size()
        
        // 单独统计图片
        if isImage {
            imageCount++
            imageSize += info.Size()
        }
        
        return nil
    })
    
    if err != nil {
        c.JSON(500, gin.H{"error": "统计静态文件信息失败: " + err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "total_files": totalFiles,
        "total_size":  totalSize,
        "image_count": imageCount,
        "image_size":  imageSize,
        "type_count":  typeCount,
        "type_sizes":  typeSizes,
    })
}