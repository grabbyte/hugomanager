
package controller

import (
    "github.com/gin-gonic/gin"
    "hugo-manager-go/config"
    "os"
    "path/filepath"
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
