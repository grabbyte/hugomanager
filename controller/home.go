
package controller

import (
    "github.com/gin-gonic/gin"
    "hugo-manager-go/config"
    "io/fs"
    "path/filepath"
    "strings"
)

func Home(c *gin.Context) {
    var articles []string
    contentDir := config.GetContentDir()
    
    err := filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d != nil && !d.IsDir() && strings.HasSuffix(path, ".md") {
            rel, _ := filepath.Rel(contentDir, path)
            articles = append(articles, rel)
        }
        return nil
    })
    
    if err != nil {
        // If content directory doesn't exist, show empty list
        articles = []string{}
    }

    c.HTML(200, "home/index.html", gin.H{
        "Title":           "Hugo 博客管理器",
        "Articles":        articles,
        "HugoProjectPath": config.GetHugoProjectPath(),
    })
}
