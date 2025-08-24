package router

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"hugo-manager-go/controller"
	"hugo-manager-go/utils"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

func Start() {
	r := gin.Default()
	// 移除固定的静态文件路由，使用动态路由
	// r.Static("/static", "./static")
	r.LoadHTMLGlob("view/**/*.html")

	// 启动WebSocket管理器
	utils.Manager.Start()

	// 初始化多语言支持
	r.Use(controller.InitializeI18n())

	// 动态静态文件服务路由
	r.GET("/static/*filepath", controller.ServeStaticFile)

	r.GET("/", controller.Home)
	r.POST("/upload", controller.UploadImage)
	r.GET("/article/edit", controller.EditArticle)
	r.POST("/article/save", controller.SaveArticle)
	r.GET("/settings", controller.Settings)
	r.POST("/settings/update", controller.UpdateSettings)
	r.GET("/api/browse-folders", controller.BrowseFolders)

	// Hugo配置管理相关路由
	r.GET("/api/hugo-config", controller.GetHugoConfig)
	r.POST("/api/hugo-config", controller.SaveHugoConfig)
	r.GET("/api/hugo-config/preview", controller.PreviewHugoConfig)

	// 部署管理相关路由
	r.GET("/deploy", controller.DeployManager)
	r.GET("/api/ssh-config", controller.GetSSHConfig)
	r.POST("/api/ssh-config", controller.UpdateSSHConfig)
	r.POST("/api/ssh-config-encrypted", controller.UpdateSSHConfigWithEncryption)
	r.POST("/api/set-decryption-key", controller.SetDecryptionKey)
	r.GET("/api/check-decryption-status", controller.CheckDecryptionStatus)
	r.POST("/api/encrypt-credentials", controller.EncryptPlaintextCredentials)
	r.POST("/api/update-master-password", controller.UpdateMasterPassword)
	r.POST("/api/test-ssh", controller.TestSSHConnection)
	r.POST("/api/build-hugo", controller.BuildHugo)
	r.POST("/api/deploy", controller.DeployToServer)
	r.POST("/api/incremental-deploy", controller.IncrementalDeployToServer)
	r.POST("/api/build-and-deploy", controller.BuildAndDeploy)
	r.POST("/api/incremental-build-and-deploy", controller.IncrementalBuildAndDeploy)
	r.POST("/api/pause-deployment", controller.PauseDeployment)
	r.POST("/api/resume-deployment", controller.ResumeDeployment)
	r.GET("/api/deployment-status", controller.GetDeploymentStatus)
	
	// Hugo serve相关路由
	r.POST("/api/hugo-serve/start", controller.StartHugoServe)
	r.POST("/api/hugo-serve/stop", controller.StopHugoServe)
	r.POST("/api/hugo-serve/restart", controller.RestartHugoServe)
	r.GET("/api/hugo-serve/status", controller.GetHugoServeStatus)
	
	// WebSocket进度监控路由
	r.GET("/ws/progress", gin.WrapH(http.HandlerFunc(utils.HandleWebSocketConnection)))

	// 图片管理相关路由
	r.GET("/images", controller.ImageManager)
	r.GET("/api/images", controller.GetImages)
	r.POST("/api/delete-image", controller.DeleteImage)
	r.POST("/api/delete-images", controller.DeleteImages)
	r.POST("/api/create-image-folder", controller.CreateImageFolder)
	r.GET("/api/image-directories", controller.GetImageDirectories)
	r.GET("/api/image-stats", controller.GetImageStats)

	// 回收站相关路由
	r.GET("/trash", controller.TrashManager)
	r.GET("/api/trash", controller.GetTrashItems)
	r.POST("/api/delete-article", controller.DeleteArticle)
	r.POST("/api/restore-from-trash", controller.RestoreFromTrash)
	r.POST("/api/permanent-delete", controller.PermanentDelete)
	r.POST("/api/empty-trash", controller.EmptyTrash)

	// 文件管理相关路由
	r.GET("/files", controller.FileManager)
	r.GET("/files/edit", controller.FileEditor)
	r.GET("/api/directory-tree", controller.GetDirectoryTree)
	r.GET("/api/files", controller.GetFiles)
	r.GET("/api/file-content", controller.GetFileContent)
	r.POST("/api/save-file", controller.SaveFileContent)
	r.POST("/api/upload-image", controller.UploadImageFile)
	r.POST("/api/upload-image-base64", controller.UploadImageBase64)
	r.POST("/api/create-article", controller.CreateNewArticle)
	r.POST("/api/repair-filenames", controller.RepairFilenames)
	r.GET("/api/debug-path", controller.DebugPath)

	// 多语言相关路由
	r.GET("/api/languages", controller.GetLanguages)
	r.POST("/api/set-language", controller.SetLanguage)
	r.POST("/api/detect-browser-language", controller.DetectBrowserLanguage)
	r.GET("/api/translations", controller.GetTranslations)

	// 自动选择可用端口
	port := findAvailablePort(80)
	if port == -1 {
		fmt.Printf("无法找到可用端口，请检查系统资源\n")
		return
	}

	address := ":" + strconv.Itoa(port)
	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Printf("Hugo Manager 正在启动，访问地址: %s\n", url)

	// 延迟1秒后自动打开网页
	go func() {
		time.Sleep(1 * time.Second)
		openBrowser(url)
	}()

	r.Run(address)
}

// 查找可用端口，从指定端口开始递增查找
func findAvailablePort(startPort int) int {
	// 定义端口查找顺序：8080 -> 8081 -> 8082 ...
	portsToTry := []int{8080}

	// 添加8080-8099范围的端口
	for i := 8080; i <= 8099; i++ {
		portsToTry = append(portsToTry, i)
	}

	// 如果还没找到，继续添加3000-3099范围的端口
	for i := 3000; i <= 3099; i++ {
		portsToTry = append(portsToTry, i)
	}

	for _, port := range portsToTry {
		if isPortAvailable(port) {
			return port
		}
	}

	return -1 // 未找到可用端口
}

// 检查端口是否可用
func isPortAvailable(port int) bool {
	address := ":" + strconv.Itoa(port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	defer listener.Close()
	return true
}

// 跨平台自动打开浏览器
func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin": // macOS
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	default:
		fmt.Printf("无法在当前操作系统上自动打开浏览器，请手动访问: %s\n", url)
		return
	}

	err := exec.Command(cmd, args...).Start()
	if err != nil {
		fmt.Printf("自动打开浏览器失败，请手动访问: %s\n", url)
	} else {
		fmt.Printf("正在打开默认浏览器...\n")
	}
}
