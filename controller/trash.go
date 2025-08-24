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

type TrashItem struct {
    Name         string    `json:"name"`
    OriginalPath string    `json:"original_path"`
    TrashPath    string    `json:"trash_path"`
    Size         int64     `json:"size"`
    DeletedTime  time.Time `json:"deleted_time"`
    IsMarkdown   bool      `json:"is_markdown"`
}

// 回收站页面
func TrashManager(c *gin.Context) {
    c.HTML(200, "trash/index.html", gin.H{
        "Title": "回收站",
        "HugoProjectPath": config.GetHugoProjectPath(),
    })
}

// 获取回收站目录路径
func getTrashDir() string {
    return filepath.Join(config.GetHugoProjectPath(), ".trash")
}

// 获取回收站文件列表
func GetTrashItems(c *gin.Context) {
    trashDir := getTrashDir()
    
    // 确保回收站目录存在
    if err := os.MkdirAll(trashDir, os.ModePerm); err != nil {
        c.JSON(500, gin.H{"error": "创建回收站目录失败: " + err.Error()})
        return
    }
    
    var items []TrashItem
    
    err := filepath.Walk(trashDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        if info.IsDir() {
            return nil
        }
        
        // 解析文件名以获取原始路径和删除时间
        rel, _ := filepath.Rel(trashDir, path)
        originalPath, deletedTime := parseTrashFileName(rel)
        
        isMarkdown := strings.HasSuffix(strings.ToLower(info.Name()), ".md")
        
        items = append(items, TrashItem{
            Name:         info.Name(),
            OriginalPath: originalPath,
            TrashPath:    rel,
            Size:         info.Size(),
            DeletedTime:  deletedTime,
            IsMarkdown:   isMarkdown,
        })
        
        return nil
    })
    
    if err != nil {
        c.JSON(500, gin.H{"error": "读取回收站失败: " + err.Error()})
        return
    }
    
    // 按删除时间排序，最新的在前
    sort.Slice(items, func(i, j int) bool {
        return items[i].DeletedTime.After(items[j].DeletedTime)
    })
    
    c.JSON(200, gin.H{
        "items": items,
        "count": len(items),
    })
}

// 删除文章到回收站
func DeleteArticle(c *gin.Context) {
    var request struct {
        Path string `json:"path"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": "请求格式错误"})
        return
    }
    
    if request.Path == "" {
        c.JSON(400, gin.H{"error": "文件路径不能为空"})
        return
    }
    
    // 构建文件的完整路径
    contentDir := config.GetContentDir()
    fullPath := filepath.Join(contentDir, request.Path)
    
    // 安全检查
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
    
    // 检查文件是否存在
    if _, err := os.Stat(fullPath); os.IsNotExist(err) {
        c.JSON(404, gin.H{"error": "文件不存在"})
        return
    }
    
    // 确保回收站目录存在
    trashDir := getTrashDir()
    if err := os.MkdirAll(trashDir, os.ModePerm); err != nil {
        c.JSON(500, gin.H{"error": "创建回收站目录失败: " + err.Error()})
        return
    }
    
    // 生成回收站文件名（包含原始路径和时间戳）
    trashFileName := generateTrashFileName(request.Path, time.Now())
    trashPath := filepath.Join(trashDir, trashFileName)
    
    // 移动文件到回收站
    if err := os.Rename(fullPath, trashPath); err != nil {
        c.JSON(500, gin.H{"error": "移动文件到回收站失败: " + err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "message":      "文件已移至回收站",
        "original_path": request.Path,
        "trash_path":   trashFileName,
    })
}

// 从回收站恢复文件
func RestoreFromTrash(c *gin.Context) {
    var request struct {
        TrashPath string `json:"trash_path"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": "请求格式错误"})
        return
    }
    
    if request.TrashPath == "" {
        c.JSON(400, gin.H{"error": "回收站路径不能为空"})
        return
    }
    
    trashDir := getTrashDir()
    trashFullPath := filepath.Join(trashDir, request.TrashPath)
    
    // 安全检查
    absTrashPath, err := filepath.Abs(trashFullPath)
    if err != nil {
        c.JSON(400, gin.H{"error": "无效的回收站路径"})
        return
    }
    
    absTrashDir, err := filepath.Abs(trashDir)
    if err != nil {
        c.JSON(400, gin.H{"error": "回收站目录路径错误"})
        return
    }
    
    if !strings.HasPrefix(absTrashPath, absTrashDir) {
        c.JSON(400, gin.H{"error": "访问被拒绝：路径不在回收站目录内"})
        return
    }
    
    // 检查回收站文件是否存在
    if _, err := os.Stat(trashFullPath); os.IsNotExist(err) {
        c.JSON(404, gin.H{"error": "回收站中的文件不存在"})
        return
    }
    
    // 解析原始路径
    originalPath, _ := parseTrashFileName(request.TrashPath)
    
    // 构建恢复路径
    contentDir := config.GetContentDir()
    restorePath := filepath.Join(contentDir, originalPath)
    
    // 检查目标位置是否已存在文件
    if _, err := os.Stat(restorePath); err == nil {
        // 文件已存在，生成新名称
        dir := filepath.Dir(restorePath)
        ext := filepath.Ext(restorePath)
        name := strings.TrimSuffix(filepath.Base(restorePath), ext)
        timestamp := time.Now().Format("20060102_150405")
        newName := fmt.Sprintf("%s_recovered_%s%s", name, timestamp, ext)
        restorePath = filepath.Join(dir, newName)
    }
    
    // 确保目标目录存在
    restoreDir := filepath.Dir(restorePath)
    if err := os.MkdirAll(restoreDir, os.ModePerm); err != nil {
        c.JSON(500, gin.H{"error": "创建恢复目录失败: " + err.Error()})
        return
    }
    
    // 移动文件回原位置
    if err := os.Rename(trashFullPath, restorePath); err != nil {
        c.JSON(500, gin.H{"error": "恢复文件失败: " + err.Error()})
        return
    }
    
    // 获取相对于content目录的路径
    relRestorePath, _ := filepath.Rel(contentDir, restorePath)
    
    c.JSON(200, gin.H{
        "message":      "文件恢复成功",
        "restore_path": filepath.ToSlash(relRestorePath),
    })
}

// 永久删除回收站文件
func PermanentDelete(c *gin.Context) {
    var request struct {
        TrashPaths []string `json:"trash_paths"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": "请求格式错误"})
        return
    }
    
    if len(request.TrashPaths) == 0 {
        c.JSON(400, gin.H{"error": "未选择要删除的文件"})
        return
    }
    
    trashDir := getTrashDir()
    absTrashDir, err := filepath.Abs(trashDir)
    if err != nil {
        c.JSON(400, gin.H{"error": "回收站目录路径错误"})
        return
    }
    
    var deleted []string
    var failed []string
    
    for _, trashPath := range request.TrashPaths {
        fullPath := filepath.Join(trashDir, trashPath)
        absFullPath, err := filepath.Abs(fullPath)
        
        if err != nil || !strings.HasPrefix(absFullPath, absTrashDir) {
            failed = append(failed, trashPath+": 路径无效")
            continue
        }
        
        if _, err := os.Stat(fullPath); os.IsNotExist(err) {
            failed = append(failed, trashPath+": 文件不存在")
            continue
        }
        
        if err := os.Remove(fullPath); err != nil {
            failed = append(failed, trashPath+": "+err.Error())
            continue
        }
        
        deleted = append(deleted, trashPath)
    }
    
    c.JSON(200, gin.H{
        "message":       fmt.Sprintf("永久删除操作完成，成功删除 %d 个文件", len(deleted)),
        "deleted":       deleted,
        "failed":        failed,
        "deleted_count": len(deleted),
        "failed_count":  len(failed),
    })
}

// 清空回收站
func EmptyTrash(c *gin.Context) {
    trashDir := getTrashDir()
    
    // 删除回收站目录及其所有内容
    if err := os.RemoveAll(trashDir); err != nil {
        c.JSON(500, gin.H{"error": "清空回收站失败: " + err.Error()})
        return
    }
    
    // 重新创建空的回收站目录
    if err := os.MkdirAll(trashDir, os.ModePerm); err != nil {
        c.JSON(500, gin.H{"error": "重建回收站目录失败: " + err.Error()})
        return
    }
    
    c.JSON(200, gin.H{"message": "回收站已清空"})
}

// 生成回收站文件名（包含原始路径和时间戳信息）
func generateTrashFileName(originalPath string, deleteTime time.Time) string {
    // 将路径中的分隔符替换为下划线，避免创建子目录
    safePath := strings.ReplaceAll(originalPath, "/", "_")
    safePath = strings.ReplaceAll(safePath, "\\", "_")
    
    // 添加时间戳
    timestamp := deleteTime.Format("20060102_150405")
    
    // 分离文件名和扩展名
    ext := filepath.Ext(safePath)
    nameWithoutExt := strings.TrimSuffix(safePath, ext)
    
    return fmt.Sprintf("%s_%s%s", nameWithoutExt, timestamp, ext)
}

// 解析回收站文件名以获取原始路径和删除时间
func parseTrashFileName(trashFileName string) (string, time.Time) {
    // 尝试解析时间戳
    parts := strings.Split(trashFileName, "_")
    if len(parts) < 3 {
        // 无法解析，返回默认值
        originalPath := strings.ReplaceAll(trashFileName, "_", "/")
        return originalPath, time.Time{}
    }
    
    // 最后两部分应该是日期和时间
    dateStr := parts[len(parts)-2]
    timeStr := parts[len(parts)-1]
    
    // 移除扩展名
    ext := filepath.Ext(timeStr)
    timeStr = strings.TrimSuffix(timeStr, ext)
    
    timestampStr := dateStr + "_" + timeStr
    deleteTime, err := time.Parse("20060102_150405", timestampStr)
    if err != nil {
        deleteTime = time.Time{}
    }
    
    // 重建原始路径（移除时间戳部分）
    originalParts := parts[:len(parts)-2]
    originalPath := strings.Join(originalParts, "_")
    if ext != "" {
        originalPath += ext
    }
    originalPath = strings.ReplaceAll(originalPath, "_", "/")
    
    return originalPath, deleteTime
}