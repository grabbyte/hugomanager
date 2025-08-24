package controller

import (
    "github.com/gin-gonic/gin"
    "hugo-manager-go/config"
    "os"
)

func Settings(c *gin.Context) {
    c.HTML(200, "settings/index.html", gin.H{
        "Title":           "项目设置",
        "HugoProjectPath": config.GetHugoProjectPath(),
    })
}

func UpdateSettings(c *gin.Context) {
    newPath := c.PostForm("hugo_project_path")
    if newPath == "" {
        c.String(400, "Hugo项目路径不能为空")
        return
    }

    // 验证路径是否存在
    if _, err := os.Stat(newPath); os.IsNotExist(err) {
        c.String(400, "指定的路径不存在: %s", newPath)
        return
    }

    // 检查是否是Hugo项目（检查是否有config.toml或config.yaml等）
    configFiles := []string{"config.toml", "config.yaml", "config.yml", "hugo.toml", "hugo.yaml", "hugo.yml"}
    isHugoProject := false
    for _, configFile := range configFiles {
        if _, err := os.Stat(newPath + "/" + configFile); err == nil {
            isHugoProject = true
            break
        }
    }

    if !isHugoProject {
        c.String(400, "指定的路径不是Hugo项目目录（未找到配置文件）")
        return
    }

    config.SetHugoProjectPath(newPath)
    c.Redirect(302, "/settings")
}