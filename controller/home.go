
package controller

import (
    "github.com/gin-gonic/gin"
    "hugo-manager-go/config"
)

func Home(c *gin.Context) {
    // 获取所有文章信息
    articleInfos, err := getAllArticles()
    if err != nil {
        articleInfos = []ArticleInfo{}
    }

    // 限制显示最近20条记录
    var articles []string
    maxArticles := 20
    for i, articleInfo := range articleInfos {
        if i >= maxArticles {
            break
        }
        articles = append(articles, articleInfo.Path)
    }

    c.HTML(200, "home/index.html", gin.H{
        "Title":           "Hugo 博客管理器",
        "Articles":        articles,
        "TotalArticles":   len(articleInfos),
        "HugoProjectPath": config.GetHugoProjectPath(),
    })
}
