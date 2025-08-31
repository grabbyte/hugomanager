package router

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"html/template"
	"hugo-manager-go/controller"
	"hugo-manager-go/utils"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func Start() {
	r := gin.Default()
	// 移除固定的静态文件路由，使用动态路由
	// r.Static("/static", "./static")

	// 添加模板函数
	r.SetFuncMap(template.FuncMap{
		"formatFileSize": func(bytes int64) string {
			if bytes == 0 {
				return "0 B"
			}
			const unit = 1024
			if bytes < unit {
				return fmt.Sprintf("%d B", bytes)
			}
			div, exp := int64(unit), 0
			for n := bytes / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
		},
		"formatBytes": func(bytes int64) string {
			if bytes == 0 {
				return "0 B"
			}
			const unit = 1024
			if bytes < unit {
				return fmt.Sprintf("%d B", bytes)
			}
			div, exp := int64(unit), 0
			for n := bytes / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
		},
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"mul": func(a, b int) int {
			return a * b
		},
		"gt": func(a, b int) bool {
			return a > b
		},
		"lt": func(a, b int) bool {
			return a < b
		},
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(s, substr)
		},
		"split": func(s, sep string) []string {
			return strings.Split(s, sep)
		},
		"index": func(slice []string, i int) string {
			if i < 0 || i >= len(slice) {
				return ""
			}
			return slice[i]
		},
		"len": func(v interface{}) int {
			switch val := v.(type) {
			case []interface{}:
				return len(val)
			case map[string]interface{}:
				return len(val)
			case string:
				return len(val)
			default:
				return 0
			}
		},
		"buildPageURL": func(page int, year string, month string) string {
			url := fmt.Sprintf("/articles?page=%d", page)
			if year != "" {
				url += "&year=" + year
			}
			if month != "" {
				url += "&month=" + month
			}
			return url
		},
		"seq": func(start, end int) []int {
			if start > end {
				return nil
			}
			result := make([]int, end-start+1)
			for i := range result {
				result[i] = start + i
			}
			return result
		},
		"hasServerStatus": func(m map[string]interface{}, key string) bool {
			if m == nil {
				return false
			}
			_, exists := m[key]
			return exists
		},
		"getServerStatus": func(m map[string]interface{}, key string) interface{} {
			if m == nil {
				// 返回默认状态
				return map[string]interface{}{
					"Status": "idle",
					"Message": "等待初始化",
					"Progress": 0,
					"FilesDeployed": 0,
					"BytesTransferred": int64(0),
					"Speed": "",
				}
			}
			value, exists := m[key]
			if !exists {
				// 返回默认状态
				return map[string]interface{}{
					"Status": "idle", 
					"Message": "等待初始化",
					"Progress": 0,
					"FilesDeployed": 0,
					"BytesTransferred": int64(0),
					"Speed": "",
				}
			}
			return value
		},
	})

	r.LoadHTMLGlob("view/**/*.html")

	// 启动WebSocket管理器
	utils.Manager.Start()

	// 初始化多语言支持
	r.Use(controller.InitializeI18n())

	// 动态静态文件服务路由
	r.GET("/static/*filepath", controller.ServeStaticFile)
	// Hugo项目上传的图片访问路由
	r.GET("/uploads/*filepath", controller.ServeHugoStaticFile)

	r.GET("/", controller.Home)
	r.POST("/upload", controller.UploadImage)
	r.GET("/articles", controller.ArticleList)
	r.GET("/api/articles", controller.GetArticlesAPI)
	r.GET("/api/articles/stats", controller.GetArticleStatsAPI)
	r.GET("/article/edit", controller.EditArticle)
	r.POST("/article/save", controller.SaveArticle)
	
	// Hugo server检测和启动API
	r.POST("/api/check-hugo-server", controller.CheckHugoServerAPI)
	r.GET("/api/hugo-server/status", controller.GetHugoServerStatusAPI)
	r.GET("/settings", controller.Settings)
	r.GET("/settings/categories", controller.CategoriesPage)
	r.POST("/settings/update", controller.UpdateSettings)
	r.GET("/api/browse-folders", controller.BrowseFolders)

	// Hugo配置管理相关路由
	r.GET("/api/hugo-config", controller.GetHugoConfig)
	r.POST("/api/hugo-config", controller.SaveHugoConfig)
	r.GET("/api/hugo-config/preview", controller.PreviewHugoConfig)

	// 部署管理相关路由（单服务器，保持兼容）
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

	// 多服务器部署相关路由
	r.GET("/api/multi-deploy/servers", controller.GetMultiServerConfigs)
	r.GET("/api/multi-deploy/server/:server_id", controller.GetMultiServerConfig)
	r.POST("/api/multi-deploy/server", controller.AddMultiServerConfig)
	r.PUT("/api/multi-deploy/server/:server_id", controller.UpdateMultiServerConfig)
	r.DELETE("/api/multi-deploy/server/:server_id", controller.DeleteMultiServerConfig)
	r.POST("/api/multi-deploy/test/:server_id", controller.TestMultiServerConnection)
	r.POST("/api/multi-deploy/deploy/:server_id", controller.DeployToMultiServer)
	r.POST("/api/multi-deploy/incremental-deploy/:server_id", controller.IncrementalDeployToMultiServer)
	r.POST("/api/multi-deploy/build-deploy/:server_id", controller.BuildAndDeployToMultiServer)
	r.POST("/api/multi-deploy/incremental-build-deploy/:server_id", controller.IncrementalBuildAndDeployToMultiServer)
	r.POST("/api/multi-deploy/pause/:server_id", controller.PauseMultiServerDeployment)
	r.POST("/api/multi-deploy/resume/:server_id", controller.ResumeMultiServerDeployment)
	r.POST("/api/multi-deploy/stop/:server_id", controller.StopMultiServerDeployment)
	r.GET("/api/multi-deploy/statuses", controller.GetMultiServerStatuses)

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
	r.GET("/api/article/preview", controller.PreviewArticle)
	r.POST("/api/save-file", controller.SaveFileContent)
	r.POST("/api/upload-image", controller.UploadImageFile)
	r.POST("/api/upload-image-base64", controller.UploadImageBase64)
	r.POST("/api/create-article", controller.CreateNewArticle)
	r.POST("/api/create-folder", controller.CreateFolder)
	r.POST("/api/repair-filenames", controller.RepairFilenames)
	
	// 时间格式修复相关API
	r.POST("/api/repair-all-dates", controller.RepairAllArticleDates)
	r.POST("/api/repair-single-date", controller.RepairSingleArticleDate)
	r.GET("/api/check-date-formats", controller.CheckDateFormats)
	r.GET("/api/debug-path", controller.DebugPath)

	// 收藏管理页面路由
	r.GET("/tools", controller.ToolsPage)
	r.GET("/books", controller.BooksPage) 
	r.GET("/wiki", controller.WikiPage)
	r.GET("/wiki/new", controller.WikiEditorPage)
	r.GET("/wiki/edit/:id", controller.WikiEditorPage)
	
	// 收藏管理API路由
	r.GET("/api/tools", controller.GetTools)
	r.POST("/api/tools", controller.AddTool)
	r.PUT("/api/tools/:id", controller.UpdateTool)
	r.DELETE("/api/tools/:id", func(c *gin.Context) {
		c.Set("type", "tools")
		controller.DeleteCollectionItem(c)
	})
	
	r.GET("/api/books", controller.GetBooks)
	r.POST("/api/books", controller.AddBook)
	r.PUT("/api/books/:id", controller.UpdateBook)
	r.DELETE("/api/books/:id", func(c *gin.Context) {
		c.Set("type", "books")
		controller.DeleteCollectionItem(c)
	})
	
	r.GET("/api/wiki", controller.GetWikiEntries)
	r.POST("/api/wiki", controller.AddWikiEntry)
	r.PUT("/api/wiki/:id", controller.UpdateWikiEntry)
	r.GET("/api/wiki/search", controller.SearchWikiEntries)
	r.POST("/api/wiki/content", controller.SaveWikiContent)
	r.PUT("/api/wiki/content/:id", controller.SaveWikiContent)
	r.DELETE("/api/wiki/:id", func(c *gin.Context) {
		c.Set("type", "wiki")
		controller.DeleteCollectionItem(c)
	})
	
	// 分类管理API路由
	r.GET("/api/categories", controller.GetCategories)
	r.GET("/api/categories/active", controller.GetActiveCategories)
	r.POST("/api/categories", controller.CreateCategory)
	r.PUT("/api/categories/:id", controller.UpdateCategory)
	r.DELETE("/api/categories/:id", controller.DeleteCategory)

	// 多语言相关路由
	r.GET("/api/languages", controller.GetLanguages)
	r.POST("/api/set-language", controller.SetLanguage)
	r.POST("/api/detect-browser-language", controller.DetectBrowserLanguage)
	r.GET("/api/translations", controller.GetTranslations)

	// Hugo安装相关路由
	r.GET("/api/hugo-status", controller.GetHugoStatus)
	r.POST("/api/install-hugo", controller.InstallHugo)

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
