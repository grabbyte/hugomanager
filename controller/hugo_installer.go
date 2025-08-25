package controller

import (
	"github.com/gin-gonic/gin"
	"hugo-manager-go/utils"
)

// GetHugoStatus 获取Hugo安装状态
func GetHugoStatus(c *gin.Context) {
	installer := utils.NewHugoInstaller()
	status := installer.GetInstallStatus()
	
	c.JSON(200, gin.H{
		"success": true,
		"data":    status,
	})
}

// InstallHugo 安装Hugo
func InstallHugo(c *gin.Context) {
	installer := utils.NewHugoInstaller()
	
	// 检查是否已安装
	if installer.IsHugoInstalled() {
		c.JSON(200, gin.H{
			"success": true,
			"message": "Hugo is already installed",
		})
		return
	}
	
	// 安装Hugo
	if err := installer.InstallHugo(); err != nil {
		c.JSON(500, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	
	// 验证安装
	if !installer.IsHugoInstalled() {
		c.JSON(500, gin.H{
			"success": false,
			"error":   "Hugo installation verification failed",
		})
		return
	}
	
	// 获取版本信息
	version, _ := installer.GetHugoVersion()
	
	c.JSON(200, gin.H{
		"success": true,
		"message": "Hugo installed successfully",
		"data": gin.H{
			"version":     version,
			"install_dir": installer.InstallDir,
			"hugo_path":   installer.GetHugoPath(),
		},
	})
}