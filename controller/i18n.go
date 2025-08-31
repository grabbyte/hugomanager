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
		"nav.home", "nav.articles", "nav.files", "nav.images", "nav.deploy", "nav.tools", "nav.books", "nav.wiki", "nav.trash", "nav.settings", "nav.back",
		
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
		"files.directory.structure", "files.project", "files.select.directory", "files.loading",
		"files.new.article", "files.refresh", "files.current.path", "files.empty.directory",
		"files.modal.new.article", "files.modal.article.title", "files.modal.title.placeholder",
		"files.modal.save.directory", "files.modal.posts.default", "files.modal.author",
		"files.modal.author.placeholder", "files.modal.article.type", "files.modal.blog.article",
		"files.modal.static.page", "files.modal.categories", "files.modal.categories.placeholder",
		"files.modal.tags", "files.modal.tags.placeholder", "files.modal.cancel", "files.modal.create",
		"files.load.tree.failed", "files.load.files.failed", "files.create.failed", "files.create.success",
		"files.filename.generated", "files.title.required",
		
		// Settings
		"settings.title", "settings.project.settings", "settings.hugo.project.path", 
		"settings.hugo.project.root", "settings.browse", "settings.path.description",
		"settings.save.settings", "settings.folder.browser.title", "settings.select.hugo.folder",
		"settings.loading", "settings.current.path", "settings.folder.name", "settings.type", 
		"settings.actions", "settings.cancel", "settings.select.current.folder", "settings.description",
		"settings.config.description", "settings.articles.from", "settings.images.to", 
		"settings.reload.homepage", "settings.hugo.config.title", "settings.hugo.config.description",
		"settings.site.title", "settings.site.title.placeholder", "settings.base.url",
		"settings.base.url.placeholder", "settings.language.code", "settings.theme.name",
		"settings.theme.placeholder", "settings.timezone", "settings.posts.per.page",
		"settings.site.description", "settings.site.description.placeholder", "settings.author.name",
		"settings.author.name.placeholder", "settings.author.email", "settings.author.email.placeholder",
		"settings.build.drafts", "settings.build.future", "settings.save.hugo.config",
		"settings.reload.config", "settings.preview.config", "settings.filename.repair.title",
		"settings.filename.repair.description", "settings.repair.filenames", "settings.root.directory",
		"settings.enter", "settings.select", "settings.empty.directory", "settings.hugo.project",
		"settings.normal.folder", "settings.error", "settings.load.folder.failed", 
		"settings.required.fields", "settings.save.failed", "settings.hugo.config.saved",
		"settings.preview.failed", "settings.config.preview", "settings.copy.config",
		"settings.config.copied", "settings.repairing.filenames", "settings.repair.success",
		"settings.repaired.files", "settings.no.files.to.repair", "settings.repair.failed.files",
		"settings.repair.failed", "settings.category.management", "settings.category.description", 
		"settings.manage.categories",
		
		// Hugo Installation
		"hugo.install.title", "hugo.install.description", "hugo.install.status", "hugo.install.installed",
		"hugo.install.not.installed", "hugo.install.version", "hugo.install.install.button", 
		"hugo.install.installing", "hugo.install.success", "hugo.install.failed", "hugo.install.already.installed",
		"hugo.install.checking", "hugo.install.download", "hugo.install.extract", "hugo.install.verify",
		"hugo.install.location", "hugo.install.platform", "hugo.install.available.version",
		
		// Articles
		"articles.title", "articles.subtitle", "articles.total.count", "articles.filter", "articles.filter.year", 
		"articles.filter.month", "articles.filter.all.years", "articles.filter.all.months", "articles.month.1", 
		"articles.month.2", "articles.month.3", "articles.month.4", "articles.month.5", "articles.month.6", 
		"articles.month.7", "articles.month.8", "articles.month.9", "articles.month.10", "articles.month.11", 
		"articles.month.12", "articles.actions", "articles.filter.apply", "articles.filter.clear", "articles.list", 
		"articles.showing", "articles.items", "articles.root.directory", "articles.edit", "articles.no.articles", 
		"articles.no.articles.hint", "articles.back.home", "articles.current.project", "articles.delete.confirm", 
		"articles.delete.failed", "articles.delete.success", "articles.delete.error", "articles.pagination.first", 
		"articles.pagination.prev", "articles.pagination.next", "articles.pagination.last", "articles.tab.published", 
		"articles.tab.drafts", "articles.tab.issues", "articles.draft.badge", "articles.search.filter", 
		"articles.search", "articles.search.placeholder", "articles.search.actions", "articles.search.button", 
		"articles.search.clear", "articles.filter.actions", "articles.filter.clear.all", "articles.search.results", 
		"articles.search.found", "articles.search.articles", "articles.issues.detected", "articles.path.not.exist", 
		"articles.publish.date", "articles.publish.date.help", "articles.editor.title", "articles.editor.content", 
		"articles.editor.metadata", "articles.editor.settings", "articles.editor.markdown.content", 
		"articles.editor.insert.image", "articles.editor.title.label", "articles.editor.author", 
		"articles.editor.type", "articles.editor.categories", "articles.editor.tags", "articles.editor.url", 
		"articles.new.article", "articles.back.to.list", "articles.preview", "articles.preview.title", 
		"articles.preview.loading",
		
		// Deploy (extended)
		"deploy.multi.title", "deploy.multi.subtitle", "deploy.server.name", "deploy.server.domain", 
		"deploy.server.host", "deploy.server.status", "deploy.server.progress", "deploy.server.actions", 
		"deploy.server.add", "deploy.server.config", "deploy.server.deploy", "deploy.server.build.deploy", 
		"deploy.server.pause", "deploy.server.resume", "deploy.server.stop", "deploy.server.delete", 
		"deploy.server.test", "deploy.status.idle", "deploy.status.building", "deploy.status.deploying", 
		"deploy.status.success", "deploy.status.failed", "deploy.status.paused", "deploy.modal.server.config", 
		"deploy.modal.server.name.placeholder", "deploy.modal.domain.placeholder", "deploy.modal.enabled", 
		"deploy.modal.save", "deploy.modal.cancel", "deploy.confirm.delete", "deploy.no.servers", 
		"deploy.no.servers.hint", "deploy.quick.operations", "deploy.build.control", "deploy.build.action", 
		"deploy.clean.action", "deploy.build.waiting", "deploy.preview.service", "deploy.batch.operations", 
		"deploy.full.deploy", "deploy.incremental.deploy", "deploy.build.required", "deploy.sync.status", 
		"deploy.sync.success", "deploy.sync.failed", "deploy.sync.pending", "deploy.statistics", 
		"deploy.files.count", "deploy.transfer.size", "deploy.no.statistics", "deploy.server.list", 
		"deploy.server.disabled", "common.not.set", "deploy.waiting.deploy", "deploy.config.server", 
		"deploy.test.connection", "deploy.operation.log", "deploy.clear.log", "deploy.log.ready", 
		"deploy.domain.optional", "deploy.host.placeholder",
		
		// Build
		"build.hugo.title", "build.build.site", "build.clean.build", "build.description",
		
		// Serve (extended)  
		"serve.status.running", "serve.status.stopped", "serve.start.action", "serve.stop.action", 
		"serve.restart.action", "serve.not.running",
		
		// Settings (extended)
		"settings.hugo.path.not.set", "settings.title",
		
		// DateTime
		"datetime.repair.title", "datetime.repair.check", "datetime.repair.batch", "datetime.repair.checking", 
		"datetime.repair.total.files", "datetime.repair.valid.count", "datetime.repair.invalid.count", 
		"datetime.repair.success.rate", "datetime.repair.invalid.files", "datetime.repair.problem.type", 
		"datetime.repair.current.format", "datetime.repair.suggested.format", "datetime.repair.action", 
		"datetime.repair.missing.date", "datetime.repair.invalid.format", "datetime.repair.read.error", 
		"datetime.repair.cannot.fix", "datetime.repair.single.fix", "datetime.repair.all.success", 
		"datetime.repair.all.valid", "datetime.repair.confirm", "datetime.repair.repairing", 
		"datetime.repair.completed", "datetime.repair.success.single", "datetime.format.warning", 
		"datetime.current.format", "datetime.suggested.format",
		
		// Categories
		"categories.title", "categories.subtitle", "categories.create", "categories.add", 
		"categories.modules.tools", "categories.modules.books", "categories.modules.wiki", 
		"categories.form.name", "categories.form.name.placeholder", "categories.form.icon", 
		"categories.form.icon.placeholder", "categories.form.icon.help", "categories.form.color", 
		"categories.form.description", "categories.form.description.placeholder", "categories.form.enabled", 
		"categories.form.sort_order", "tools.manage.categories", "books.manage.categories", 
		"wiki.manage.categories",
		
		// Tools, Books, Wiki
		"tools.title", "tools.subtitle", "tools.add.tool", "tools.form.name", "tools.form.category", 
		"tools.form.icon", "tools.form.url", "tools.form.description", "tools.form.tags", "tools.form.favorite", 
		"books.title", "books.subtitle", "books.add.book", "books.form.title", "books.form.category", 
		"books.form.author", "books.form.publisher", "books.form.url", "books.form.cover", 
		"books.form.description", "books.form.rating", "books.form.status", "books.form.tags", 
		"books.status.want", "books.status.reading", "books.status.completed", "books.status.reference", 
		"books.rating.unrated", "books.rating.5", "books.rating.4", "books.rating.3", "books.rating.2", 
		"books.rating.1", "wiki.title", "wiki.subtitle", "wiki.add.entry", "wiki.form.title", 
		"wiki.form.category", "wiki.form.type", "wiki.form.difficulty", "wiki.form.url", 
		"wiki.form.description", "wiki.form.tags", "wiki.form.keywords", "wiki.form.official", 
		"wiki.form.favorite", "wiki.form.frequent", "wiki.search.title", "wiki.type.guide", 
		"wiki.type.reference", "wiki.type.tutorial", "wiki.type.glossary", "wiki.type.example", 
		"wiki.difficulty.beginner", "wiki.difficulty.intermediate", "wiki.difficulty.advanced", 
		"wiki.difficulty.expert", "wiki.category.ai", "wiki.category.claude", "wiki.category.mcp", 
		"wiki.category.terms", "wiki.category.examples",
		
		// Images
		"images.title", "images.subtitle", "images.upload.files", "images.filter.files", 
		"images.select.all", "images.clear.selection", "images.download.selected", "images.delete.selected", 
		"images.file.list", "images.file.name", "images.file.size", "images.file.type", 
		"images.file.modified", "images.file.actions", "images.no.files", "images.no.files.hint", 
		"images.current.hugo.project", "images.directory.title", "images.files.count", "images.total.size", 
		"images.stats.template", "images.create.folder", "images.static.files", "images.total.files.count", 
		"images.total.capacity", "images.image.files", "images.image.capacity", "images.no.images", 
		"images.upload.hint", "images.upload.now", "images.upload.images", "images.upload.drop.hint", 
		"images.upload.formats", "images.new.folder",
		
		// Trash
		"trash.title", "trash.subtitle", "trash.restore.selected", "trash.empty.trash", 
		"trash.files.count", "trash.files.list", "trash.select.all", "trash.clear.selection", 
		"trash.permanent.delete", "trash.empty.title", "trash.empty.message", "trash.current.hugo.project", 
		"trash.load.failed", "trash.original.path", "trash.delete.time", "trash.restore",
		
		// Footer
		"footer.copyright",
		
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
			if userSetCookie == "true" {
				// 用户在Cookie中设置了语言，优先使用
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