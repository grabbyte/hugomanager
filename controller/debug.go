package controller

import (
    "github.com/gin-gonic/gin"
    "hugo-manager-go/config"
    "os"
    "path/filepath"
)

// 调试文件路径信息
func DebugPath(c *gin.Context) {
    relativePath := c.Query("path")
    
    contentDir := config.GetContentDir()
    fullPath := filepath.Join(contentDir, relativePath)
    
    // 获取文件信息
    info := map[string]interface{}{
        "relative_path": relativePath,
        "content_dir":   contentDir,
        "full_path":     fullPath,
        "file_exists":   false,
        "is_dir":        false,
    }
    
    if stat, err := os.Stat(fullPath); err == nil {
        info["file_exists"] = true
        info["is_dir"] = stat.IsDir()
        info["file_size"] = stat.Size()
        info["mod_time"] = stat.ModTime()
    } else {
        info["error"] = err.Error()
    }
    
    // 检查目录是否存在
    dir := filepath.Dir(fullPath)
    if stat, err := os.Stat(dir); err == nil {
        info["dir_exists"] = true
        info["dir_is_dir"] = stat.IsDir()
    } else {
        info["dir_exists"] = false
        info["dir_error"] = err.Error()
    }
    
    // 列出目录内容
    if stat, err := os.Stat(dir); err == nil && stat.IsDir() {
        if entries, err := os.ReadDir(dir); err == nil {
            var files []string
            for _, entry := range entries {
                files = append(files, entry.Name())
            }
            info["dir_contents"] = files
        }
    }
    
    c.JSON(200, info)
}