package controller

import (
    "fmt"
    "github.com/gin-gonic/gin"
    "hugo-manager-go/config"
    "hugo-manager-go/utils"
    "os"
    "path/filepath"
)

// 修复文件名编码问题
func RepairFilenames(c *gin.Context) {
    contentDir := config.GetContentDir()
    
    var repaired []string
    var errors []string
    
    err := filepath.Walk(contentDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        if info.IsDir() {
            return nil
        }
        
        // 获取原始文件名
        originalName := info.Name()
        
        // 检查文件名是否需要修复
        if !utils.IsValidUTF8Filename(originalName) || !utils.ValidateFilename(originalName) {
            // 清理文件名
            cleanName := utils.CleanFilename(originalName)
            
            if cleanName == originalName {
                // 文件名没有改变，跳过
                return nil
            }
            
            // 构建新路径
            dir := filepath.Dir(path)
            newPath := filepath.Join(dir, cleanName)
            
            // 检查新文件是否已存在
            if _, err := os.Stat(newPath); err == nil {
                errors = append(errors, fmt.Sprintf("目标文件已存在: %s -> %s", originalName, cleanName))
                return nil
            }
            
            // 重命名文件
            if err := os.Rename(path, newPath); err != nil {
                errors = append(errors, fmt.Sprintf("重命名失败: %s -> %s (%v)", originalName, cleanName, err))
                return nil
            }
            
            repaired = append(repaired, fmt.Sprintf("%s -> %s", originalName, cleanName))
        }
        
        return nil
    })
    
    if err != nil {
        c.JSON(500, gin.H{"error": "扫描文件失败: " + err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "message":  "文件名修复完成",
        "repaired": repaired,
        "errors":   errors,
        "repaired_count": len(repaired),
        "error_count":    len(errors),
    })
}