package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"hugo-manager-go/config"
)

// Language represents a language configuration
type Language struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// I18nManager manages translations
type I18nManager struct {
	currentLang string
	translations map[string]map[string]string
	mutex       sync.RWMutex
}

var i18nManager = &I18nManager{
	currentLang:  "en-US", // 默认语言，会在LoadTranslations中更新
	translations: make(map[string]map[string]string),
}

// GetI18nManager returns the singleton instance
func GetI18nManager() *I18nManager {
	return i18nManager
}

// SupportedLanguages returns the list of supported languages
func (i *I18nManager) SupportedLanguages() []Language {
	return []Language{
		{Code: "zh-CN", Name: "简体中文"},
		{Code: "zh-TW", Name: "繁體中文"},
		{Code: "ja-JP", Name: "日本語"},
		{Code: "en-US", Name: "English"},
		{Code: "ru-RU", Name: "Русский"},
		{Code: "ko-KR", Name: "한국어"},
	}
}

// SetLanguage sets the current language
func (i *I18nManager) SetLanguage(langCode string) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	// Validate language code
	supported := false
	for _, lang := range i.SupportedLanguages() {
		if lang.Code == langCode {
			supported = true
			break
		}
	}

	if !supported {
		return fmt.Errorf("unsupported language: %s", langCode)
	}

	i.currentLang = langCode
	
	// 保存到配置文件
	config.SetLanguage(langCode)
	
	return i.loadTranslations(langCode)
}

// SetLanguageInternal sets the current language internally without marking as user-set
func (i *I18nManager) SetLanguageInternal(langCode string) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	// Validate language code
	supported := false
	for _, lang := range i.SupportedLanguages() {
		if lang.Code == langCode {
			supported = true
			break
		}
	}

	if !supported {
		return fmt.Errorf("unsupported language: %s", langCode)
	}

	i.currentLang = langCode
	// 注意：不保存到配置文件，不标记为用户设置
	
	return i.loadTranslations(langCode)
}

// GetCurrentLanguage returns the current language code
func (i *I18nManager) GetCurrentLanguage() string {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.currentLang
}

// T translates a key to the current language
func (i *I18nManager) T(key string) string {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	if translations, exists := i.translations[i.currentLang]; exists {
		if translation, exists := translations[key]; exists {
			return translation
		}
	}

	// Fallback to English if current language translation doesn't exist
	if i.currentLang != "en-US" {
		if translations, exists := i.translations["en-US"]; exists {
			if translation, exists := translations[key]; exists {
				return translation
			}
		}
	}

	// Fallback to Chinese if English doesn't exist
	if i.currentLang != "zh-CN" {
		if translations, exists := i.translations["zh-CN"]; exists {
			if translation, exists := translations[key]; exists {
				return translation
			}
		}
	}

	// Return the key itself if no translation found
	return key
}

// TWithVars translates a key with variable substitution
func (i *I18nManager) TWithVars(key string, vars map[string]interface{}) string {
	template := i.T(key)
	result := template

	for k, v := range vars {
		placeholder := fmt.Sprintf("{{%s}}", k)
		replacement := fmt.Sprintf("%v", v)
		result = strings.ReplaceAll(result, placeholder, replacement)
	}

	return result
}

// LoadTranslations loads translations from JSON files
func (i *I18nManager) LoadTranslations() error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	// 从配置中读取用户选择的语言
	configLang := config.GetLanguage()
	if configLang != "" {
		i.currentLang = configLang
	}

	// Create translations directory if it doesn't exist
	translationsDir := "translations"
	if _, err := os.Stat(translationsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(translationsDir, 0755); err != nil {
			return fmt.Errorf("failed to create translations directory: %v", err)
		}
	}

	// Load translations for all supported languages
	for _, lang := range i.SupportedLanguages() {
		if err := i.loadTranslations(lang.Code); err != nil {
			fmt.Printf("Warning: Failed to load translations for %s: %v\n", lang.Code, err)
		}
	}

	return nil
}

// loadTranslations loads translations for a specific language
func (i *I18nManager) loadTranslations(langCode string) error {
	translationFile := filepath.Join("translations", langCode+".json")
	
	// Create default translation file if it doesn't exist
	if _, err := os.Stat(translationFile); os.IsNotExist(err) {
		if err := i.createDefaultTranslation(langCode); err != nil {
			return fmt.Errorf("failed to create default translation for %s: %v", langCode, err)
		}
	}

	// Read translation file
	data, err := os.ReadFile(translationFile)
	if err != nil {
		return fmt.Errorf("failed to read translation file %s: %v", translationFile, err)
	}

	// Parse JSON
	var translations map[string]string
	if err := json.Unmarshal(data, &translations); err != nil {
		return fmt.Errorf("failed to parse translation file %s: %v", translationFile, err)
	}

	// Store translations
	if i.translations == nil {
		i.translations = make(map[string]map[string]string)
	}
	i.translations[langCode] = translations

	return nil
}

// createDefaultTranslation creates a default translation file
func (i *I18nManager) createDefaultTranslation(langCode string) error {
	translations := i.getDefaultTranslations(langCode)
	
	data, err := json.MarshalIndent(translations, "", "    ")
	if err != nil {
		return err
	}

	translationFile := filepath.Join("translations", langCode+".json")
	return os.WriteFile(translationFile, data, 0644)
}

// getDefaultTranslations returns default translations for a language
func (i *I18nManager) getDefaultTranslations(langCode string) map[string]string {
	switch langCode {
	case "zh-TW":
		return map[string]string{
			// Navigation
			"nav.home":         "首頁",
			"nav.files":        "檔案管理",
			"nav.images":       "靜態檔案",
			"nav.deploy":       "部署管理",
			"nav.trash":        "資源回收筒",
			"nav.settings":     "設定",
			"nav.back":         "返回",
			
			// Common
			"common.save":      "儲存",
			"common.cancel":    "取消",
			"common.delete":    "刪除",
			"common.edit":      "編輯",
			"common.create":    "建立",
			"common.loading":   "載入中...",
			"common.success":   "成功",
			"common.error":     "錯誤",
			"common.warning":   "警告",
			"common.info":      "資訊",
			"common.confirm":   "確認",
			"common.close":     "關閉",
			"common.refresh":   "重新整理",
			"common.search":    "搜尋",
			"common.filter":    "篩選",
			"common.all":       "全部",
			"common.status":    "狀態",
			"common.port":      "連接埠",
			"common.url":       "網址",
			"common.start":     "啟動",
			"common.stop":      "停止",
			"common.restart":   "重新啟動",
			"common.preview":   "預覽",
			"common.open":      "開啟",
			
			// Home page
			"home.title":       "Hugo 管理器",
			"home.subtitle":    "輕鬆管理您的 Hugo 部落格",
			"home.articles":    "文章",
			"home.images":      "圖片",
			"home.deploy":      "部署",
			
			// Deploy
			"deploy.title":        "部署管理",
			"deploy.config":       "SSH 配置",
			"deploy.host":         "伺服器位址",
			"deploy.username":     "使用者名稱",
			"deploy.password":     "密碼",
			"deploy.keypath":      "私鑰路徑",
			"deploy.remotepath":   "遠端路徑",
			"deploy.auth.method":  "驗證方式",
			"deploy.auth.password": "密碼",
			"deploy.auth.key":     "私鑰",
			"deploy.test":         "測試連線",
			"deploy.build":        "建立網站",
			"deploy.deploy":       "部署",
			"deploy.build.desc":   "使用Hugo生成靜態HTML檔案",
			"deploy.deploy.desc":  "上傳檔案到遠端伺服器",
			"deploy.incremental":  "增量部署",
			"deploy.quick":        "快速建立+部署",
			"deploy.status":       "部署狀態",
			"deploy.last.sync":    "最後同步時間",
			"deploy.files":        "傳輸檔案數",
			"deploy.bytes":        "傳輸資料量",
			
			// Hugo Serve
			"serve.title":       "Hugo 預覽服務",
			"serve.status":      "狀態",
			"serve.running":     "執行中",
			"serve.stopped":     "未執行",
			"serve.port":        "服務連接埠",
			"serve.url":         "預覽位址",
			"serve.start":       "啟動",
			"serve.stop":        "停止",
			"serve.restart":     "重新啟動",
			"serve.open":        "開啟預覽",
			
			// Files
			"files.title":       "檔案管理",
			"files.editor":      "檔案編輯器",
			"files.create":      "建立文章",
			"files.upload":      "上傳檔案",
			"files.markdown":    "Markdown編輯器",
			"files.metadata":    "文章資訊",
			"files.title.label": "標題",
			"files.author":      "作者",
			"files.type":        "類型",
			"files.date":        "發布時間",
			"files.categories":  "分類",
			"files.tags":        "標籤",
			"files.content":     "內容",
			"files.preview":     "預覽",
			"files.split":       "分屏",
			
			// Messages
			"msg.save.success":     "儲存成功",
			"msg.deploy.success":   "部署完成",
			"msg.build.success":    "建立完成",
			"msg.serve.started":    "Hugo serve啟動成功",
			"msg.serve.stopped":    "Hugo serve已停止",
			"msg.connection.test":  "正在測試連線...",
			"msg.loading.config":   "正在載入配置...",
		}
	case "ru-RU":
		return map[string]string{
			// Navigation
			"nav.home":         "Главная",
			"nav.files":        "Файлы",
			"nav.images":       "Изображения",
			"nav.deploy":       "Развертывание",
			"nav.trash":        "Корзина",
			"nav.settings":     "Настройки",
			"nav.back":         "Назад",
			
			// Common
			"common.save":      "Сохранить",
			"common.cancel":    "Отмена",
			"common.delete":    "Удалить",
			"common.edit":      "Редактировать",
			"common.create":    "Создать",
			"common.loading":   "Загрузка...",
			"common.success":   "Успешно",
			"common.error":     "Ошибка",
			"common.warning":   "Предупреждение",
			"common.info":      "Информация",
			"common.confirm":   "Подтвердить",
			"common.close":     "Закрыть",
			"common.refresh":   "Обновить",
			"common.search":    "Поиск",
			"common.filter":    "Фильтр",
			"common.all":       "Все",
			"common.status":    "Статус",
			"common.port":      "Порт",
			"common.url":       "URL",
			"common.start":     "Запустить",
			"common.stop":      "Остановить",
			"common.restart":   "Перезапустить",
			"common.preview":   "Предпросмотр",
			"common.open":      "Открыть",
			
			// Home page
			"home.title":       "Hugo Менеджер",
			"home.subtitle":    "Легко управляйте своим Hugo блогом",
			"home.articles":    "Статьи",
			"home.images":      "Изображения",
			"home.deploy":      "Развертывание",
			
			// Deploy
			"deploy.title":        "Управление развертыванием",
			"deploy.config":       "Конфигурация SSH",
			"deploy.host":         "Адрес сервера",
			"deploy.username":     "Имя пользователя",
			"deploy.password":     "Пароль",
			"deploy.keypath":      "Путь к приватному ключу",
			"deploy.remotepath":   "Удаленный путь",
			"deploy.auth.method":  "Метод аутентификации",
			"deploy.auth.password": "Пароль",
			"deploy.auth.key":     "Приватный ключ",
			"deploy.test":         "Тест соединения",
			"deploy.build":        "Сборка сайта",
			"deploy.deploy":       "Развертывание",
			"deploy.build.desc":   "Генерация статических HTML файлов с помощью Hugo",
			"deploy.deploy.desc":  "Загрузка файлов на удаленный сервер",
			"deploy.incremental":  "Инкрементальное развертывание",
			"deploy.quick":        "Быстрая сборка+развертывание",
			"deploy.status":       "Статус развертывания",
			"deploy.last.sync":    "Последняя синхронизация",
			"deploy.files":        "Развернуто файлов",
			"deploy.bytes":        "Передано байт",
			
			// Hugo Serve
			"serve.title":       "Сервис предпросмотра Hugo",
			"serve.status":      "Статус",
			"serve.running":     "Запущен",
			"serve.stopped":     "Остановлен",
			"serve.port":        "Порт сервиса",
			"serve.url":         "URL предпросмотра",
			"serve.start":       "Запустить",
			"serve.stop":        "Остановить",
			"serve.restart":     "Перезапустить",
			"serve.open":        "Открыть предпросмотр",
			
			// Files
			"files.title":       "Управление файлами",
			"files.editor":      "Редактор файлов",
			"files.create":      "Создать статью",
			"files.upload":      "Загрузить файл",
			"files.markdown":    "Markdown редактор",
			"files.metadata":    "Информация о статье",
			"files.title.label": "Заголовок",
			"files.author":      "Автор",
			"files.type":        "Тип",
			"files.date":        "Дата публикации",
			"files.categories":  "Категории",
			"files.tags":        "Теги",
			"files.content":     "Содержимое",
			"files.preview":     "Предпросмотр",
			"files.split":       "Разделенный вид",
			
			// Messages
			"msg.save.success":     "Успешно сохранено",
			"msg.deploy.success":   "Развертывание завершено успешно",
			"msg.build.success":    "Сборка завершена успешно",
			"msg.serve.started":    "Hugo serve успешно запущен",
			"msg.serve.stopped":    "Hugo serve остановлен",
			"msg.connection.test":  "Тестирование соединения...",
			"msg.loading.config":   "Загрузка конфигурации...",
		}
	case "ko-KR":
		return map[string]string{
			// Navigation
			"nav.home":         "홈",
			"nav.files":        "파일",
			"nav.images":       "이미지",
			"nav.deploy":       "배포",
			"nav.trash":        "휴지통",
			"nav.settings":     "설정",
			"nav.back":         "뒤로",
			
			// Common
			"common.save":      "저장",
			"common.cancel":    "취소",
			"common.delete":    "삭제",
			"common.edit":      "편집",
			"common.create":    "만들기",
			"common.loading":   "로딩 중...",
			"common.success":   "성공",
			"common.error":     "오류",
			"common.warning":   "경고",
			"common.info":      "정보",
			"common.confirm":   "확인",
			"common.close":     "닫기",
			"common.refresh":   "새로 고침",
			"common.search":    "검색",
			"common.filter":    "필터",
			"common.all":       "전체",
			"common.status":    "상태",
			"common.port":      "포트",
			"common.url":       "URL",
			"common.start":     "시작",
			"common.stop":      "중지",
			"common.restart":   "재시작",
			"common.preview":   "미리보기",
			"common.open":      "열기",
			
			// Home page
			"home.title":       "Hugo 매니저",
			"home.subtitle":    "Hugo 블로그를 쉽게 관리하세요",
			"home.articles":    "글",
			"home.images":      "이미지",
			"home.deploy":      "배포",
			
			// Deploy
			"deploy.title":        "배포 관리",
			"deploy.config":       "SSH 구성",
			"deploy.host":         "서버 주소",
			"deploy.username":     "사용자명",
			"deploy.password":     "비밀번호",
			"deploy.keypath":      "개인키 경로",
			"deploy.remotepath":   "원격 경로",
			"deploy.auth.method":  "인증 방식",
			"deploy.auth.password": "비밀번호",
			"deploy.auth.key":     "개인키",
			"deploy.test":         "연결 테스트",
			"deploy.build":        "사이트 빌드",
			"deploy.deploy":       "배포",
			"deploy.build.desc":   "Hugo로 정적 HTML 파일 생성",
			"deploy.deploy.desc":  "원격 서버에 파일 업로드",
			"deploy.incremental":  "증분 배포",
			"deploy.quick":        "빠른 빌드+배포",
			"deploy.status":       "배포 상태",
			"deploy.last.sync":    "마지막 동기화",
			"deploy.files":        "배포된 파일 수",
			"deploy.bytes":        "전송된 바이트",
			
			// Hugo Serve
			"serve.title":       "Hugo 미리보기 서비스",
			"serve.status":      "상태",
			"serve.running":     "실행 중",
			"serve.stopped":     "중지됨",
			"serve.port":        "서비스 포트",
			"serve.url":         "미리보기 URL",
			"serve.start":       "시작",
			"serve.stop":        "중지",
			"serve.restart":     "재시작",
			"serve.open":        "미리보기 열기",
			
			// Files
			"files.title":       "파일 관리",
			"files.editor":      "파일 편집기",
			"files.create":      "글 작성",
			"files.upload":      "파일 업로드",
			"files.markdown":    "Markdown 편집기",
			"files.metadata":    "글 정보",
			"files.title.label": "제목",
			"files.author":      "작성자",
			"files.type":        "유형",
			"files.date":        "발행 날짜",
			"files.categories":  "카테고리",
			"files.tags":        "태그",
			"files.content":     "내용",
			"files.preview":     "미리보기",
			"files.split":       "분할 보기",
			
			// Messages
			"msg.save.success":     "성공적으로 저장됨",
			"msg.deploy.success":   "배포가 성공적으로 완료됨",
			"msg.build.success":    "빌드가 성공적으로 완료됨",
			"msg.serve.started":    "Hugo serve가 성공적으로 시작됨",
			"msg.serve.stopped":    "Hugo serve가 중지됨",
			"msg.connection.test":  "연결 테스트 중...",
			"msg.loading.config":   "구성 로딩 중...",
		}
	case "en-US":
		return map[string]string{
			// Navigation
			"nav.home":         "Home",
			"nav.files":        "Files",
			"nav.images":       "Images",
			"nav.deploy":       "Deploy",
			"nav.trash":        "Trash",
			"nav.settings":     "Settings",
			"nav.back":         "Back",
			
			// Common
			"common.save":      "Save",
			"common.cancel":    "Cancel",
			"common.delete":    "Delete",
			"common.edit":      "Edit",
			"common.create":    "Create",
			"common.loading":   "Loading...",
			"common.success":   "Success",
			"common.error":     "Error",
			"common.warning":   "Warning",
			"common.info":      "Info",
			"common.confirm":   "Confirm",
			"common.close":     "Close",
			"common.refresh":   "Refresh",
			"common.search":    "Search",
			"common.filter":    "Filter",
			"common.all":       "All",
			"common.status":    "Status",
			"common.port":      "Port",
			"common.url":       "URL",
			"common.start":     "Start",
			"common.stop":      "Stop",
			"common.restart":   "Restart",
			"common.preview":   "Preview",
			"common.open":      "Open",
			
			// Home page
			"home.title":       "Hugo Manager",
			"home.subtitle":    "Manage your Hugo blog easily",
			"home.articles":    "Articles",
			"home.images":      "Images",
			"home.deploy":      "Deploy",
			
			// Deploy
			"deploy.title":        "Deployment Management",
			"deploy.config":       "SSH Configuration",
			"deploy.host":         "Server Host",
			"deploy.username":     "Username",
			"deploy.password":     "Password",
			"deploy.keypath":      "Private Key Path",
			"deploy.remotepath":   "Remote Path",
			"deploy.auth.method":  "Authentication Method",
			"deploy.auth.password": "Password",
			"deploy.auth.key":     "Private Key",
			"deploy.test":         "Test Connection",
			"deploy.build":        "Build Site",
			"deploy.deploy":       "Deploy",
			"deploy.build.desc":   "Generate static HTML files using Hugo",
			"deploy.deploy.desc":  "Upload files to remote server",
			"deploy.incremental":  "Incremental Deploy",
			"deploy.quick":        "Quick Build+Deploy",
			"deploy.status":       "Deployment Status",
			"deploy.last.sync":    "Last Sync",
			"deploy.files":        "Files Deployed",
			"deploy.bytes":        "Bytes Transferred",
			
			// Hugo Serve
			"serve.title":       "Hugo Preview Service",
			"serve.status":      "Status",
			"serve.running":     "Running",
			"serve.stopped":     "Stopped",
			"serve.port":        "Service Port",
			"serve.url":         "Preview URL",
			"serve.start":       "Start",
			"serve.stop":        "Stop",
			"serve.restart":     "Restart",
			"serve.open":        "Open Preview",
			
			// Files
			"files.title":       "File Management",
			"files.editor":      "File Editor",
			"files.create":      "Create Article",
			"files.upload":      "Upload File",
			"files.markdown":    "Markdown Editor",
			"files.metadata":    "Article Info",
			"files.title.label": "Title",
			"files.author":      "Author",
			"files.type":        "Type",
			"files.date":        "Publish Date",
			"files.categories":  "Categories",
			"files.tags":        "Tags",
			"files.content":     "Content",
			"files.preview":     "Preview",
			"files.split":       "Split View",
			
			// Messages
			"msg.save.success":     "Saved successfully",
			"msg.deploy.success":   "Deployment completed successfully",
			"msg.build.success":    "Build completed successfully",
			"msg.serve.started":    "Hugo serve started successfully",
			"msg.serve.stopped":    "Hugo serve stopped",
			"msg.connection.test":  "Testing connection...",
			"msg.loading.config":   "Loading configuration...",
		}
	case "ja-JP":
		return map[string]string{
			// Navigation
			"nav.home":         "ホーム",
			"nav.files":        "ファイル",
			"nav.images":       "画像",
			"nav.deploy":       "デプロイ",
			"nav.trash":        "ゴミ箱",
			"nav.settings":     "設定",
			"nav.back":         "戻る",
			
			// Common
			"common.save":      "保存",
			"common.cancel":    "キャンセル",
			"common.delete":    "削除",
			"common.edit":      "編集",
			"common.create":    "作成",
			"common.loading":   "読み込み中...",
			"common.success":   "成功",
			"common.error":     "エラー",
			"common.warning":   "警告",
			"common.info":      "情報",
			"common.confirm":   "確認",
			"common.close":     "閉じる",
			"common.refresh":   "更新",
			"common.search":    "検索",
			"common.filter":    "フィルター",
			"common.all":       "すべて",
			"common.status":    "ステータス",
			"common.port":      "ポート",
			"common.url":       "URL",
			"common.start":     "開始",
			"common.stop":      "停止",
			"common.restart":   "再開",
			"common.preview":   "プレビュー",
			"common.open":      "開く",
			
			// Home page
			"home.title":       "Hugo Manager",
			"home.subtitle":    "Hugoブログを簡単に管理",
			"home.articles":    "記事",
			"home.images":      "画像",
			"home.deploy":      "デプロイ",
			
			// Deploy
			"deploy.title":        "デプロイ管理",
			"deploy.config":       "SSH設定",
			"deploy.host":         "サーバーホスト",
			"deploy.username":     "ユーザー名",
			"deploy.password":     "パスワード",
			"deploy.keypath":      "秘密鍵パス",
			"deploy.remotepath":   "リモートパス",
			"deploy.auth.method":  "認証方式",
			"deploy.auth.password": "パスワード",
			"deploy.auth.key":     "秘密鍵",
			"deploy.test":         "接続テスト",
			"deploy.build":        "サイト構築",
			"deploy.deploy":       "デプロイ",
			"deploy.build.desc":   "Hugoで静的HTMLファイルを生成",
			"deploy.deploy.desc":  "リモートサーバーにファイルをアップロード",
			"deploy.incremental":  "増分デプロイ",
			"deploy.quick":        "高速構築+デプロイ",
			"deploy.status":       "デプロイステータス",
			"deploy.last.sync":    "最終同期",
			"deploy.files":        "デプロイファイル数",
			"deploy.bytes":        "転送バイト数",
			
			// Hugo Serve
			"serve.title":       "Hugoプレビューサービス",
			"serve.status":      "ステータス",
			"serve.running":     "実行中",
			"serve.stopped":     "停止中",
			"serve.port":        "サービスポート",
			"serve.url":         "プレビューURL",
			"serve.start":       "開始",
			"serve.stop":        "停止",
			"serve.restart":     "再開",
			"serve.open":        "プレビューを開く",
			
			// Files
			"files.title":       "ファイル管理",
			"files.editor":      "ファイルエディター",
			"files.create":      "記事作成",
			"files.upload":      "ファイルアップロード",
			"files.markdown":    "Markdownエディター",
			"files.metadata":    "記事情報",
			"files.title.label": "タイトル",
			"files.author":      "著者",
			"files.type":        "タイプ",
			"files.date":        "公開日",
			"files.categories":  "カテゴリー",
			"files.tags":        "タグ",
			"files.content":     "コンテンツ",
			"files.preview":     "プレビュー",
			"files.split":       "分割表示",
			
			// Messages
			"msg.save.success":     "正常に保存されました",
			"msg.deploy.success":   "デプロイが正常に完了しました",
			"msg.build.success":    "構築が正常に完了しました",
			"msg.serve.started":    "Hugo serveが正常に開始されました",
			"msg.serve.stopped":    "Hugo serveが停止されました",
			"msg.connection.test":  "接続をテスト中...",
			"msg.loading.config":   "設定を読み込み中...",
		}
	default: // zh-CN
		return map[string]string{
			// Navigation
			"nav.home":         "首页",
			"nav.files":        "文件管理",
			"nav.images":       "静态文件",
			"nav.deploy":       "部署管理",
			"nav.trash":        "回收站",
			"nav.settings":     "设置",
			"nav.back":         "返回",
			
			// Common
			"common.save":      "保存",
			"common.cancel":    "取消",
			"common.delete":    "删除",
			"common.edit":      "编辑",
			"common.create":    "创建",
			"common.loading":   "加载中...",
			"common.success":   "成功",
			"common.error":     "错误",
			"common.warning":   "警告",
			"common.info":      "信息",
			"common.confirm":   "确认",
			"common.close":     "关闭",
			"common.refresh":   "刷新",
			"common.search":    "搜索",
			"common.filter":    "筛选",
			"common.all":       "全部",
			"common.status":    "状态",
			"common.port":      "端口",
			"common.url":       "网址",
			"common.start":     "启动",
			"common.stop":      "停止",
			"common.restart":   "重启",
			"common.preview":   "预览",
			"common.open":      "打开",
			
			// Home page
			"home.title":       "Hugo 管理器",
			"home.subtitle":    "轻松管理您的 Hugo 博客",
			"home.articles":    "文章",
			"home.images":      "图片",
			"home.deploy":      "部署",
			
			// Deploy
			"deploy.title":        "部署管理",
			"deploy.config":       "SSH 配置",
			"deploy.host":         "服务器地址",
			"deploy.username":     "用户名",
			"deploy.password":     "密码",
			"deploy.keypath":      "私钥路径",
			"deploy.remotepath":   "远程路径",
			"deploy.auth.method":  "认证方式",
			"deploy.auth.password": "密码",
			"deploy.auth.key":     "私钥",
			"deploy.test":         "测试连接",
			"deploy.build":        "构建网站",
			"deploy.deploy":       "部署",
			"deploy.build.desc":   "使用Hugo生成静态HTML文件",
			"deploy.deploy.desc":  "上传文件到远程服务器",
			"deploy.incremental":  "增量部署",
			"deploy.quick":        "快速构建+部署",
			"deploy.status":       "部署状态",
			"deploy.last.sync":    "最后同步时间",
			"deploy.files":        "传输文件数",
			"deploy.bytes":        "传输数据量",
			
			// Hugo Serve
			"serve.title":       "Hugo 预览服务",
			"serve.status":      "状态",
			"serve.running":     "运行中",
			"serve.stopped":     "未运行",
			"serve.port":        "服务端口",
			"serve.url":         "预览地址",
			"serve.start":       "启动",
			"serve.stop":        "停止",
			"serve.restart":     "重启",
			"serve.open":        "打开预览",
			
			// Files
			"files.title":       "文件管理",
			"files.editor":      "文件编辑器",
			"files.create":      "创建文章",
			"files.upload":      "上传文件",
			"files.markdown":    "Markdown编辑器",
			"files.metadata":    "文章信息",
			"files.title.label": "标题",
			"files.author":      "作者",
			"files.type":        "类型",
			"files.date":        "发布时间",
			"files.categories":  "分类",
			"files.tags":        "标签",
			"files.content":     "内容",
			"files.preview":     "预览",
			"files.split":       "分屏",
			
			// Messages
			"msg.save.success":     "保存成功",
			"msg.deploy.success":   "部署完成",
			"msg.build.success":    "构建完成",
			"msg.serve.started":    "Hugo serve启动成功",
			"msg.serve.stopped":    "Hugo serve已停止",
			"msg.connection.test":  "正在测试连接...",
			"msg.loading.config":   "正在加载配置...",
		}
	}
}