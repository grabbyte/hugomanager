// 全局多语言支持脚本

// 确保变量不会重复声明
window.currentTranslations = window.currentTranslations || {};

// 语言与显示名称的映射
const languageNames = {
    'zh-CN': '简体中文',
    'zh-TW': '繁體中文', 
    'ja-JP': '日本語',
    'en-US': 'English',
    'ru-RU': 'Русский'
};

// 切换语言函数 - 立即定义并暴露
window.changeLanguage = async function(langCode) {
    try {
        const response = await fetch('/api/set-language', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                language: langCode
            })
        });

        const data = await response.json();
        if (data.error) {
            console.error('Failed to change language:', data.error);
            return;
        }

        // 重新加载翻译
        const translationsResponse = await fetch('/api/translations');
        const translationsData = await translationsResponse.json();
        
        if (translationsData.error) {
            console.error('Failed to load translations:', translationsData.error);
            return;
        }
        
        window.currentTranslations = translationsData.translations || {};
        
        // 更新当前语言显示
        updateCurrentLanguageDisplay(langCode);
        
        // 更新页面文本
        updatePageText();
        
    } catch (error) {
        console.error('Error changing language:', error);
    }
};

// 获取翻译文本函数 - 立即定义并暴露
window.getTranslation = function(key, defaultText = '') {
    return window.currentTranslations[key] || defaultText;
};

// 页面加载时初始化多语言功能
document.addEventListener('DOMContentLoaded', function() {
    initializeI18n();
});

// 初始化国际化
async function initializeI18n() {
    try {
        // 获取当前语言和翻译
        const response = await fetch('/api/translations');
        const data = await response.json();
        
        if (data.error) {
            console.error('Failed to load translations:', data.error);
            return;
        }
        
        window.currentTranslations = data.translations || {};
        const currentLanguage = data.language || 'en-US';
        
        // 更新当前语言显示
        updateCurrentLanguageDisplay(currentLanguage);
        
        // 更新页面文本
        updatePageText();
        
        // 检测浏览器语言（仅在用户未主动设置语言时）
        if (!data.user_set) {
            detectBrowserLanguage();
        }
    } catch (error) {
        console.error('Error initializing i18n:', error);
    }
}

// 更新当前语言显示
function updateCurrentLanguageDisplay(language) {
    const currentLanguageNameEl = document.getElementById('currentLanguageName');
    if (currentLanguageNameEl) {
        currentLanguageNameEl.textContent = languageNames[language] || 'English';
    }
}

// 更新页面文本
function updatePageText() {
    // 更新所有带有 data-i18n 属性的元素
    document.querySelectorAll('[data-i18n]').forEach(element => {
        const key = element.getAttribute('data-i18n');
        if (window.currentTranslations[key]) {
            element.textContent = window.currentTranslations[key];
        }
    });
    
    // 更新所有带有 data-i18n-placeholder 属性的输入框
    document.querySelectorAll('[data-i18n-placeholder]').forEach(element => {
        const key = element.getAttribute('data-i18n-placeholder');
        if (window.currentTranslations[key]) {
            element.placeholder = window.currentTranslations[key];
        }
    });
    
    // 更新所有带有 data-i18n-title 属性的元素
    document.querySelectorAll('[data-i18n-title]').forEach(element => {
        const key = element.getAttribute('data-i18n-title');
        if (window.currentTranslations[key]) {
            element.title = window.currentTranslations[key];
        }
    });
}


// 检测浏览器语言
async function detectBrowserLanguage() {
    try {
        const browserLanguage = navigator.language || navigator.languages[0] || 'en-US';
        
        const response = await fetch('/api/detect-browser-language', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                browser_language: browserLanguage
            })
        });

        const data = await response.json();
        
        if (data.error) {
            console.log('Browser language detection failed:', data.error);
            return;
        }
        
        if (data.language_changed) {
            // 重新加载翻译
            const translationsResponse = await fetch('/api/translations');
            const translationsData = await translationsResponse.json();
            
            if (!translationsData.error) {
                window.currentTranslations = translationsData.translations || {};
                updateCurrentLanguageDisplay(data.language);
                updatePageText();
            }
        }
    } catch (error) {
        console.error('Error detecting browser language:', error);
    }
}

// 导出函数供其他模块使用
window.i18n = {
    getTranslation: window.getTranslation,
    changeLanguage: window.changeLanguage,
    updatePageText,
    currentTranslations: () => window.currentTranslations
};

// 立即执行，确保函数在页面加载时就可用
console.log('i18n.js loaded, changeLanguage available:', typeof window.changeLanguage);