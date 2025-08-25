package controller

import (
	"github.com/gin-gonic/gin"
	"hugo-manager-go/utils"
	"hugo-manager-go/config"
)

// GetLanguages returns all supported languages
func GetLanguages(c *gin.Context) {
	i18nManager := utils.GetI18nManager()
	languages := i18nManager.SupportedLanguages()
	
	c.JSON(200, gin.H{
		"languages":          languages,
		"current":            i18nManager.GetCurrentLanguage(),
		"user_set_language":  config.IsUserSetLanguage(),
	})
}

// SetLanguage sets the current language
func SetLanguage(c *gin.Context) {
	var request struct {
		Language string `json:"language"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}
	
	i18nManager := utils.GetI18nManager()
	if err := i18nManager.SetLanguage(request.Language); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{
		"message":  "Language changed successfully",
		"language": request.Language,
	})
}

// GetTranslations returns translations for the current language
func GetTranslations(c *gin.Context) {
	i18nManager := utils.GetI18nManager()
	currentLang := i18nManager.GetCurrentLanguage()
	
	// Get all translation keys
	translations := make(map[string]string)
	
	// Common keys that should be available
	keys := []string{
		// Navigation
		"nav.home", "nav.files", "nav.images", "nav.deploy", "nav.trash", "nav.settings", "nav.back",
		
		// Common
		"common.save", "common.cancel", "common.delete", "common.edit", "common.create",
		"common.loading", "common.success", "common.error", "common.warning", "common.info",
		"common.confirm", "common.close", "common.refresh", "common.search", "common.filter",
		"common.all", "common.status", "common.port", "common.url", "common.start",
		"common.stop", "common.restart", "common.preview", "common.open",
		
		// Home page
		"home.title", "home.subtitle", "home.new.article", "home.manage.files", "home.articles.count",
		"home.search.articles", "home.search.placeholder", "home.search.button", "home.search.found", 
		"home.search.no.results", "home.search.try.different", "home.card.new.blog", "home.card.new.blog.desc",
		"home.card.file.management", "home.card.file.management.desc", "home.card.static.files", 
		"home.card.static.files.desc", "home.card.deploy", "home.card.deploy.desc", "home.card.trash",
		"home.card.trash.desc", "home.card.settings", "home.card.settings.desc", "home.recent.articles",
		"home.refresh", "home.root.directory", "home.edit", "home.no.articles", "home.no.articles.hint",
		"home.create.now", "home.current.hugo.project", "home.modal.new.article", "home.modal.article.title",
		"home.modal.title.placeholder", "home.modal.save.directory", "home.modal.posts.default",
		"home.modal.article.type", "home.modal.blog.article", "home.modal.static.page", "home.modal.cancel",
		"home.modal.create.and.edit", "home.footer.copyright", "home.articles.total", "home.delete.confirm",
		"home.delete.failed", "home.delete.success", "home.delete.error", "home.modal.title.required",
		"home.modal.create.failed", "home.modal.create.error",
		
		// Deploy
		"deploy.title", "deploy.config", "deploy.host", "deploy.username", "deploy.password",
		"deploy.keypath", "deploy.remotepath", "deploy.auth.method", "deploy.auth.password",
		"deploy.auth.key", "deploy.test", "deploy.build", "deploy.deploy", "deploy.build.desc",
		"deploy.deploy.desc", "deploy.incremental", "deploy.quick", "deploy.status",
		"deploy.last.sync", "deploy.files", "deploy.bytes",
		
		// Hugo Serve
		"serve.title", "serve.status", "serve.running", "serve.stopped", "serve.port",
		"serve.url", "serve.start", "serve.stop", "serve.restart", "serve.open",
		
		// Files
		"files.title", "files.editor", "files.create", "files.upload", "files.markdown",
		"files.metadata", "files.title.label", "files.author", "files.type", "files.date",
		"files.categories", "files.tags", "files.content", "files.preview", "files.split",
		
		// Messages
		"msg.save.success", "msg.deploy.success", "msg.build.success", "msg.serve.started",
		"msg.serve.stopped", "msg.connection.test", "msg.loading.config",
	}
	
	for _, key := range keys {
		translations[key] = i18nManager.T(key)
	}
	
	c.JSON(200, gin.H{
		"language":     currentLang,
		"translations": translations,
		"user_set":     config.IsUserSetLanguage(),
	})
}

// DetectBrowserLanguage detects and sets browser language if user hasn't set one
func DetectBrowserLanguage(c *gin.Context) {
	var request struct {
		BrowserLanguage string `json:"browser_language"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request format"})
		return
	}
	
	// 检查是否用户已经主动设置了语言
	if config.IsUserSetLanguage() {
		c.JSON(200, gin.H{
			"message": "User has set language, browser detection ignored",
			"language": config.GetLanguage(),
			"user_set": true,
		})
		return
	}
	
	// 映射浏览器语言到支持的语言代码
	browserLang := request.BrowserLanguage
	mappedLang := mapBrowserLanguage(browserLang)
	
	if mappedLang != "" {
		i18nManager := utils.GetI18nManager()
		config.SetBrowserLanguage(mappedLang)
		i18nManager.SetLanguageInternal(mappedLang) // 内部设置，不标记为用户设置
		
		c.JSON(200, gin.H{
			"message": "Browser language detected and set",
			"language": mappedLang,
			"user_set": false,
			"browser_detected": true,
		})
	} else {
		c.JSON(200, gin.H{
			"message": "Browser language not supported, using default",
			"language": config.GetLanguage(),
			"user_set": false,
		})
	}
}

// mapBrowserLanguage maps browser language codes to supported language codes
func mapBrowserLanguage(browserLang string) string {
	// 处理常见的浏览器语言代码
	switch {
	case browserLang == "zh-CN" || browserLang == "zh" || browserLang == "zh-Hans":
		return "zh-CN"
	case browserLang == "zh-TW" || browserLang == "zh-Hant":
		return "zh-TW"
	case browserLang == "ja" || browserLang == "ja-JP":
		return "ja-JP"
	case browserLang == "en" || browserLang == "en-US" || browserLang == "en-GB":
		return "en-US"
	case browserLang == "ru" || browserLang == "ru-RU":
		return "ru-RU"
	case browserLang == "ko" || browserLang == "ko-KR":
		return "ko-KR"
	default:
		// 尝试匹配语言前缀
		if len(browserLang) >= 2 {
			prefix := browserLang[:2]
			switch prefix {
			case "zh":
				return "zh-CN" // 默认简体中文
			case "ja":
				return "ja-JP"
			case "en":
				return "en-US"
			case "ru":
				return "ru-RU"
			case "ko":
				return "ko-KR"
			}
		}
		return "" // 不支持的语言
	}
}

// InitializeI18n initializes the i18n system and middleware
func InitializeI18n() gin.HandlerFunc {
	return func(c *gin.Context) {
		i18nManager := utils.GetI18nManager()
		
		// Load translations on first request
		i18nManager.LoadTranslations()
		
		// 检查Cookie中的语言设置（优先级最高）
		if cookie, err := c.Cookie("user_language"); err == nil && cookie != "" {
			userSetCookie, _ := c.Cookie("user_set_language")
			fmt.Printf("Cookie检测: user_language=%s, user_set_language=%s\n", cookie, userSetCookie)
			if userSetCookie == "true" {
				// 用户在Cookie中设置了语言，优先使用
				fmt.Printf("使用Cookie语言设置: %s\n", cookie)
				if err := i18nManager.SetLanguage(cookie); err == nil {
					c.Set("T", i18nManager.T)
					c.Set("i18n", i18nManager)
					c.Next()
					return
				}
			}
		}
		
		// 只有在用户未设置语言时，才检查浏览器语言偏好
		if !config.IsUserSetLanguage() {
			acceptLang := c.GetHeader("Accept-Language")
			if acceptLang != "" {
				// Simple language detection - get first language code
				if len(acceptLang) >= 2 {
					langCode := acceptLang[:2]
					switch langCode {
					case "zh":
						i18nManager.SetLanguageInternal("zh-CN") // 使用内部方法，不标记为用户设置
					case "ja":
						i18nManager.SetLanguageInternal("ja-JP")
					case "en":
						i18nManager.SetLanguageInternal("en-US")
					case "ru":
						i18nManager.SetLanguageInternal("ru-RU")
					case "ko":
						i18nManager.SetLanguageInternal("ko-KR")
					}
				}
			}
		}
		
		// Add translation function to context
		c.Set("T", i18nManager.T)
		c.Set("i18n", i18nManager)
		c.Next()
	}
}

// Helper function to get translated page data
func GetPageData(c *gin.Context, pageKey string) gin.H {
	i18nManager := utils.GetI18nManager()
	
	data := gin.H{
		"Language":     i18nManager.GetCurrentLanguage(),
		"Languages":    i18nManager.SupportedLanguages(),
		"T": func(key string) string {
			return i18nManager.T(key)
		},
	}
	
	// Add page-specific title
	if pageKey != "" {
		data["Title"] = i18nManager.T(pageKey)
	}
	
	return data
}