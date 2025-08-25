package controller

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"hugo-manager-go/config"
)

// 处理静态文件访问
func ServeStaticFile(c *gin.Context) {
	// 获取请求的文件路径，去掉 /static/ 前缀
	requestPath := c.Param("filepath")
	if requestPath == "" {
		c.JSON(404, gin.H{"error": "文件路径为空"})
		return
	}

	var staticDir string
	
	// 判断是应用程序静态文件还是Hugo项目静态文件
	if strings.HasPrefix(requestPath, "/js/") || strings.HasPrefix(requestPath, "/css/") {
		// 应用程序自己的静态文件 (js, css等)
		staticDir = "./static"
	} else {
		// Hugo项目的静态文件 (用户上传的图片等)
		staticDir = config.GetStaticDir()
	}
	
	// 构建完整的文件路径
	fullPath := filepath.Join(staticDir, requestPath)
	
	// 安全检查：确保路径在static目录内
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
		c.JSON(403, gin.H{"error": "访问被拒绝：路径不在允许的目录内"})
		return
	}
	
	// 检查文件是否存在
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.JSON(404, gin.H{"error": "文件不存在"})
		return
	}
	
	// 直接服务文件
	c.File(fullPath)
}

// 处理Hugo项目上传的静态文件访问
func ServeHugoStaticFile(c *gin.Context) {
	// 获取请求的文件路径，去掉 /uploads/ 前缀
	requestPath := c.Param("filepath")
	if requestPath == "" {
		c.JSON(404, gin.H{"error": "文件路径为空"})
		return
	}

	// Hugo项目的static目录
	staticDir := config.GetStaticDir()
	
	// 构建完整的文件路径，添加uploads前缀
	fullPath := filepath.Join(staticDir, "uploads", requestPath)
	
	// 安全检查：确保路径在static目录内
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
		c.JSON(403, gin.H{"error": "访问被拒绝：路径不在允许的目录内"})
		return
	}
	
	// 检查文件是否存在
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.JSON(404, gin.H{"error": "文件不存在"})
		return
	}
	
	// 直接服务文件
	c.File(fullPath)
}