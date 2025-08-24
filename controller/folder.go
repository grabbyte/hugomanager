package controller

import (
    "github.com/gin-gonic/gin"
    "os"
    "path/filepath"
    "strings"
)

type FolderItem struct {
    Name     string `json:"name"`
    Path     string `json:"path"`
    IsDir    bool   `json:"is_dir"`
    IsHugo   bool   `json:"is_hugo"`
}

func BrowseFolders(c *gin.Context) {
    currentPath := c.Query("path")
    if currentPath == "" {
        // 获取用户主目录作为默认起始点
        homeDir, err := os.UserHomeDir()
        if err != nil {
            // Windows默认显示C盘，Linux显示根目录
            if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
                currentPath = "C:\\"
            } else {
                currentPath = "/"
            }
        } else {
            currentPath = homeDir
        }
    }

    // 确保路径存在
    if _, err := os.Stat(currentPath); os.IsNotExist(err) {
        c.JSON(400, gin.H{"error": "路径不存在"})
        return
    }

    entries, err := os.ReadDir(currentPath)
    if err != nil {
        c.JSON(500, gin.H{"error": "无法读取目录"})
        return
    }

    var items []FolderItem
    
    // 添加上级目录选项（除非是根目录）
    if currentPath != "/" && currentPath != "" {
        parent := filepath.Dir(currentPath)
        items = append(items, FolderItem{
            Name:  "..",
            Path:  parent,
            IsDir: true,
            IsHugo: false,
        })
    }

    // 添加子目录
    for _, entry := range entries {
        if entry.IsDir() {
            fullPath := filepath.Join(currentPath, entry.Name())
            
            // 跳过隐藏目录
            if strings.HasPrefix(entry.Name(), ".") {
                continue
            }
            
            // 检查是否是Hugo项目
            isHugo := isHugoProject(fullPath)
            
            items = append(items, FolderItem{
                Name:  entry.Name(),
                Path:  fullPath,
                IsDir: true,
                IsHugo: isHugo,
            })
        }
    }

    c.JSON(200, gin.H{
        "current_path": currentPath,
        "items":       items,
    })
}

func isHugoProject(path string) bool {
    configFiles := []string{"config.toml", "config.yaml", "config.yml", "hugo.toml", "hugo.yaml", "hugo.yml"}
    for _, configFile := range configFiles {
        if _, err := os.Stat(filepath.Join(path, configFile)); err == nil {
            return true
        }
    }
    return false
}