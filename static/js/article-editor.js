/**
 * Article Editor Module - 博客编辑器模块
 * Claude Prompt: 模块化博客编辑器JavaScript代码，实现KISS原则
 * 
 * 功能包括：
 * - CodeMirror编辑器初始化和管理
 * - Markdown语法高亮
 * - 图片上传（拖拽、粘贴、按钮）
 * - 文章保存和预览
 * - Front Matter处理
 */

class ArticleEditor {
    constructor() {
        this.codeMirrorEditor = null;
        this.hasUnsavedChanges = false;
        
        // 绑定方法到实例
        this.saveArticle = this.saveArticle.bind(this);
        this.previewArticle = this.previewArticle.bind(this);
        this.insertImage = this.insertImage.bind(this);
    }

    /**
     * 初始化编辑器
     */
    init() {
        document.addEventListener('DOMContentLoaded', () => {
            this.initCodeMirrorEditor();
            this.initDraftSwitch();
            this.initChangeTracking();
            this.initKeyboardShortcuts();
        });
    }

    /**
     * 初始化CodeMirror编辑器
     */
    initCodeMirrorEditor() {
        const textarea = document.getElementById('contentTextarea');
        const editorContainer = document.getElementById('contentEditor');
        
        if (!textarea || !editorContainer) return;
        
        // 创建CodeMirror编辑器
        this.codeMirrorEditor = CodeMirror.fromTextArea(textarea, {
            mode: 'gfm',
            theme: 'default',
            lineNumbers: true,
            lineWrapping: true,
            autofocus: false,
            placeholder: '在此输入文章的 Markdown 内容...支持拖拽图片和Ctrl+V粘贴图片',
            extraKeys: {
                'Ctrl-S': (cm) => {
                    this.saveArticle();
                }
            }
        });
        
        // 设置编辑器事件监听
        this.codeMirrorEditor.on('change', (cm, change) => {
            // 同步到隐藏的textarea
            textarea.value = cm.getValue();
            this.hasUnsavedChanges = true;
        });
        
        // 添加拖拽支持
        this.codeMirrorEditor.on('drop', (cm, event) => {
            this.handleCodeMirrorDrop(event);
        });
        
        // 添加粘贴支持
        this.codeMirrorEditor.on('paste', (cm, event) => {
            this.handleCodeMirrorPaste(event);
        });
        
        // 刷新编辑器
        setTimeout(() => {
            this.codeMirrorEditor.refresh();
        }, 100);
    }

    /**
     * 初始化草稿开关
     */
    initDraftSwitch() {
        const draftSwitch = document.getElementById('draftSwitch');
        const draftLabel = document.getElementById('draftLabel');
        const isDraftInput = document.getElementById('isDraftInput');
        
        if (!draftSwitch || !draftLabel || !isDraftInput) return;
        
        // 草稿开关变化事件
        draftSwitch.addEventListener('change', function() {
            const isDraft = this.checked;
            
            // 更新隐藏字段值
            isDraftInput.value = isDraft ? 'true' : 'false';
            
            // 更新标签文字和图标
            if (isDraft) {
                draftLabel.innerHTML = '<i class="bi bi-file-earmark-text me-1"></i>草稿';
                draftLabel.className = 'form-check-label text-warning';
            } else {
                draftLabel.innerHTML = '<i class="bi bi-file-earmark-check me-1"></i>已发布';
                draftLabel.className = 'form-check-label text-success';
            }
        });
        
        // 设置初始状态的颜色
        const currentIsDraft = draftSwitch.checked;
        if (currentIsDraft) {
            draftLabel.className = 'form-check-label text-warning';
        } else {
            draftLabel.className = 'form-check-label text-success';
        }
    }

    /**
     * 初始化变更追踪
     */
    initChangeTracking() {
        const inputs = [
            'articleTitle', 'articleAuthor', 'articleType', 'categories', 'tags', 
            'publishDate', 'articleURL'
        ].map(id => document.getElementById(id)).filter(el => el);
        
        inputs.forEach(element => {
            element.addEventListener('input', () => {
                this.hasUnsavedChanges = true;
            });
        });
    }

    /**
     * 初始化快捷键
     */
    initKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            if (e.ctrlKey && e.key === 's') {
                e.preventDefault();
                this.saveArticle();
            }
        });
    }

    /**
     * CodeMirror拖拽处理
     */
    handleCodeMirrorDrop(event) {
        event.preventDefault();
        
        const files = event.dataTransfer.files;
        for (let file of files) {
            if (file.type.startsWith('image/')) {
                this.uploadImageFile(file);
            }
        }
    }

    /**
     * CodeMirror粘贴处理
     */
    handleCodeMirrorPaste(event) {
        const clipboardData = event.clipboardData || window.clipboardData;
        const items = clipboardData.items;
        
        for (let item of items) {
            if (item.type.indexOf('image') !== -1) {
                event.preventDefault();
                const file = item.getAsFile();
                const reader = new FileReader();
                
                reader.onload = (e) => {
                    this.uploadImageBase64(e.target.result, file.name || 'pasted-image');
                };
                
                reader.readAsDataURL(file);
                break;
            }
        }
    }

    /**
     * 保存文章
     */
    saveArticle() {
        const titleInput = document.getElementById('articleTitle');
        const authorInput = document.getElementById('articleAuthor');
        const typeInput = document.getElementById('articleType');
        const categoriesInput = document.getElementById('categories');
        const tagsInput = document.getElementById('tags');
        const dateInput = document.getElementById('publishDate');
        const urlInput = document.getElementById('articleURL');
        const isDraftInput = document.getElementById('isDraftInput');
        
        // 构建更新后的Front Matter和内容
        const title = titleInput ? titleInput.value.trim() : '';
        const author = authorInput ? authorInput.value.trim() : '';
        const type = typeInput ? typeInput.value.trim() : 'post';
        const categories = categoriesInput ? categoriesInput.value.trim() : '';
        const tags = tagsInput ? tagsInput.value.trim() : '';
        const publishDate = dateInput ? dateInput.value.trim() : '';
        const url = urlInput ? urlInput.value.trim() : '';
        const isDraft = isDraftInput ? isDraftInput.value === 'true' : false;
        
        // 从CodeMirror编辑器获取内容
        const bodyContent = this.codeMirrorEditor ? this.codeMirrorEditor.getValue() : '';
        
        // 构建完整的文档内容（Front Matter + 主体内容）
        const fullContent = this.buildFullContent({
            title: title,
            author: author,
            type: type,
            categories: categories,
            tags: tags,
            date: publishDate,
            url: url,
            draft: isDraft
        }, bodyContent);
        
        // 创建隐藏的input来提交完整内容
        const hiddenInput = document.createElement('input');
        hiddenInput.type = 'hidden';
        hiddenInput.name = 'content';
        hiddenInput.value = fullContent;
        
        // 将隐藏input添加到表单中
        const form = document.getElementById('articleForm');
        form.appendChild(hiddenInput);
        
        // 提交表单
        form.submit();
    }

    /**
     * 构建完整的Hugo文档（Front Matter + 主体内容）
     */
    buildFullContent(metadata, bodyContent) {
        const frontMatterLines = ['---'];
        
        // 添加作者
        if (metadata.author && metadata.author.trim()) {
            frontMatterLines.push(`author: ${metadata.author.trim()}`);
        }
        
        // 添加标题
        if (metadata.title && metadata.title.trim()) {
            frontMatterLines.push(`title: "${metadata.title.trim().replace(/"/g, '\\"')}"`);
        }
        
        // 添加类型
        if (metadata.type && metadata.type.trim()) {
            frontMatterLines.push(`type: ${metadata.type.trim()}`);
        }
        
        // 添加日期 - 确保Hugo兼容的RFC3339格式
        if (metadata.date && metadata.date.trim()) {
            const hugoDate = this.formatHugoDate(metadata.date.trim());
            frontMatterLines.push(`date: ${hugoDate}`);
        }
        
        // 添加分类
        if (metadata.categories && metadata.categories.trim()) {
            const cats = metadata.categories.split(',')
                .map(c => c.trim())
                .filter(c => c !== '')
                .map(c => `"${c.replace(/"/g, '\\"')}"`);
            if (cats.length > 0) {
                frontMatterLines.push(`categories: [${cats.join(', ')}]`);
            }
        }
        
        // 添加标签
        if (metadata.tags && metadata.tags.trim()) {
            const tags = metadata.tags.split(',')
                .map(t => t.trim())
                .filter(t => t !== '')
                .map(t => `"${t.replace(/"/g, '\\"')}"`);
            if (tags.length > 0) {
                frontMatterLines.push(`tags: [${tags.join(', ')}]`);
            }
        }
        
        // 添加URL
        if (metadata.url && metadata.url.trim()) {
            frontMatterLines.push(`url: ${metadata.url.trim()}`);
        }
        
        // 添加草稿状态
        frontMatterLines.push(`draft: ${metadata.draft}`);
        
        // 结束Front Matter
        frontMatterLines.push('---');
        frontMatterLines.push('');
        
        // 合并Front Matter和主体内容
        return frontMatterLines.join('\n') + bodyContent;
    }

    /**
     * 预览文章 - 自动检测并启动Hugo server
     */
    async previewArticle() {
        try {
            // 显示加载状态
            this.showPreviewLoading(true);
            
            // 检测并启动Hugo server
            const response = await fetch('/api/check-hugo-server', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                }
            });
            
            const result = await response.json();
            
            if (!result.running) {
                // Hugo server启动失败
                this.showPreviewLoading(false);
                alert(`Hugo server启动失败: ${result.error || result.message}`);
                return;
            }
            
            // 构建预览URL
            const previewUrl = this.buildPreviewUrl(result.url);
            
            // 调试信息
            console.log('预览URL:', previewUrl);
            
            // 隐藏加载状态
            this.showPreviewLoading(false);
            
            // 打开预览页面
            window.open(previewUrl, '_blank');
            
        } catch (error) {
            this.showPreviewLoading(false);
            console.error('预览失败:', error);
            alert('预览失败，请检查网络连接或Hugo配置');
        }
    }

    /**
     * 构建预览URL
     */
    buildPreviewUrl(baseUrl) {
        // 首先检查是否有自定义URL
        const urlInput = document.getElementById('articleURL');
        const customUrl = urlInput ? urlInput.value.trim() : '';
        
        // 如果有自定义URL，优先使用
        if (customUrl) {
            let fullUrl = baseUrl;
            if (!fullUrl.endsWith('/')) {
                fullUrl += '/';
            }
            
            // 处理自定义URL格式
            let cleanUrl = customUrl;
            
            // 移除开头的斜杠
            if (cleanUrl.startsWith('/')) {
                cleanUrl = cleanUrl.substring(1);
            }
            
            // 如果URL不包含文件扩展名且不以斜杠结尾，添加斜杠
            // 这处理了像 "asdfasd" 这样的简单URL
            if (!cleanUrl.endsWith('/') && !cleanUrl.includes('.html') && !cleanUrl.includes('.php')) {
                cleanUrl += '/';
            }
            
            console.log('使用自定义URL:', customUrl, '→', fullUrl + cleanUrl);
            return fullUrl + cleanUrl;
        }
        
        // 没有自定义URL时，使用文件路径生成
        const pathInput = document.querySelector('input[name="path"]');
        if (!pathInput || !pathInput.value) {
            return baseUrl;
        }
        
        const articlePath = pathInput.value;
        
        // 移除.md扩展名，转换路径格式
        const urlPath = articlePath.replace(/\.md$/, '').replace(/\\/g, '/');
        
        // 构建完整URL
        let fullUrl = baseUrl;
        if (!fullUrl.endsWith('/')) {
            fullUrl += '/';
        }
        
        if (urlPath) {
            if (!urlPath.startsWith('/')) {
                fullUrl += urlPath + '/';
            } else {
                fullUrl += urlPath.substring(1) + '/';
            }
        }
        
        return fullUrl;
    }

    /**
     * 格式化日期为Hugo兼容的RFC3339格式
     */
    formatHugoDate(dateStr) {
        try {
            let date;
            
            // 处理datetime-local格式 (YYYY-MM-DDTHH:mm)
            if (dateStr.match(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/)) {
                // 添加秒和时区信息
                date = new Date(dateStr + ':00');
            }
            // 处理其他格式
            else {
                date = new Date(dateStr);
            }
            
            // 检查日期是否有效
            if (isNaN(date.getTime())) {
                console.warn('Invalid date format:', dateStr);
                // 返回当前时间的RFC3339格式作为后备
                return new Date().toISOString();
            }
            
            // 返回RFC3339格式 (ISO8601)
            return date.toISOString();
            
        } catch (error) {
            console.error('Date formatting error:', error);
            // 返回当前时间的RFC3339格式作为后备
            return new Date().toISOString();
        }
    }

    /**
     * 显示/隐藏预览加载状态
     */
    showPreviewLoading(show) {
        const previewBtn = document.querySelector('button[onclick="previewArticle()"]');
        if (!previewBtn) return;
        
        if (show) {
            previewBtn.disabled = true;
            previewBtn.innerHTML = '<i class="bi bi-hourglass-split me-1"></i>启动Hugo中...';
        } else {
            previewBtn.disabled = false;
            previewBtn.innerHTML = '<i class="bi bi-eye me-1"></i>预览';
        }
    }

    /**
     * 插入图片
     */
    insertImage() {
        document.getElementById('imageUpload').click();
    }

    /**
     * 处理图片上传
     */
    handleImageUpload(event) {
        const file = event.target.files[0];
        if (!file) return;
        this.uploadImageFile(file);
    }

    /**
     * 上传图片文件
     */
    uploadImageFile(file) {
        const formData = new FormData();
        formData.append('file', file);

        fetch('/api/upload-image', {
            method: 'POST',
            body: formData
        })
        .then(response => response.json())
        .then(data => {
            if (data.error) {
                alert('上传失败: ' + data.error);
                return;
            }
            
            this.insertImageMarkdown(data.url, file.name);
        })
        .catch(error => {
            console.error('Error:', error);
            alert('上传失败');
        });
    }

    /**
     * 上传base64图片
     */
    uploadImageBase64(imageData, filename) {
        fetch('/api/upload-image-base64', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                image_data: imageData,
                filename: filename
            })
        })
        .then(response => response.json())
        .then(data => {
            if (data.error) {
                alert('上传失败: ' + data.error);
                return;
            }
            
            this.insertImageMarkdown(data.url, data.filename);
        })
        .catch(error => {
            console.error('Error:', error);
            alert('上传失败');
        });
    }

    /**
     * 插入图片Markdown
     */
    insertImageMarkdown(url, filename) {
        const imageMarkdown = `![${filename}](${url})`;
        
        if (this.codeMirrorEditor) {
            // 在CodeMirror编辑器中插入图片
            const cursor = this.codeMirrorEditor.getCursor();
            this.codeMirrorEditor.replaceSelection(imageMarkdown);
            
            // 设置光标位置到图片后面
            const newCursor = {
                line: cursor.line,
                ch: cursor.ch + imageMarkdown.length
            };
            this.codeMirrorEditor.setCursor(newCursor);
            this.codeMirrorEditor.focus();
        } else {
            // 兜底：使用原来的textarea方式
            const textarea = document.getElementById('contentTextarea');
            if (textarea) {
                const cursorPos = textarea.selectionStart;
                const textBefore = textarea.value.substring(0, cursorPos);
                const textAfter = textarea.value.substring(cursorPos);
                
                textarea.value = textBefore + imageMarkdown + textAfter;
                textarea.selectionStart = textarea.selectionEnd = cursorPos + imageMarkdown.length;
                textarea.focus();
            }
        }
    }
}

// 创建全局实例
const articleEditor = new ArticleEditor();

// 初始化编辑器
articleEditor.init();

// 导出全局函数供HTML调用
window.saveArticle = articleEditor.saveArticle;
window.previewArticle = articleEditor.previewArticle;
window.insertImage = articleEditor.insertImage;

// Claude Prompt: 添加时间格式检测和修复功能到编辑器
/**
 * 检查时间格式是否符合Hugo要求
 * @param {string} dateStr 时间字符串
 * @returns {boolean} 是否符合Hugo格式
 */
function isValidHugoDate(dateStr) {
    if (!dateStr) return false;
    
    // Hugo推荐的RFC3339格式
    const hugoFormats = [
        /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/,                    // 2006-01-02T15:04:05Z
        /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3,9}Z$/,           // 2006-01-02T15:04:05.999999999Z
        /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$/,      // 2006-01-02T15:04:05+08:00
    ];
    
    return hugoFormats.some(format => format.test(dateStr));
}

/**
 * 修复时间格式为Hugo兼容格式
 * @param {string} dateStr 时间字符串
 * @returns {string} 修复后的时间字符串
 */
function fixDateFormat(dateStr) {
    if (!dateStr) return '';
    
    try {
        let date;
        
        // 处理datetime-local格式 (YYYY-MM-DDTHH:mm)
        if (dateStr.match(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/)) {
            // 添加秒信息
            date = new Date(dateStr + ':00');
        } else {
            date = new Date(dateStr);
        }
        
        // 检查日期是否有效
        if (isNaN(date.getTime())) {
            console.warn('Invalid date format:', dateStr);
            return '';
        }
        
        // 返回Hugo标准的RFC3339格式
        return date.toISOString();
        
    } catch (error) {
        console.error('Date fixing error:', error);
        return '';
    }
}

/**
 * 从datetime-local值转换为Hugo格式显示
 * @param {string} datetimeLocalValue datetime-local输入的值
 * @returns {string} 对应的Hugo格式时间
 */
function datetimeLocalToHugoFormat(datetimeLocalValue) {
    if (!datetimeLocalValue) return '';
    
    try {
        // datetime-local格式: YYYY-MM-DDTHH:mm
        // 需要添加秒和时区信息
        const date = new Date(datetimeLocalValue + ':00');
        if (isNaN(date.getTime())) return '';
        
        return date.toISOString();
    } catch (error) {
        console.error('Datetime-local conversion error:', error);
        return '';
    }
}

/**
 * 从Hugo格式转换为datetime-local值
 * @param {string} hugoDate Hugo格式的时间字符串
 * @returns {string} datetime-local输入需要的值
 */
function hugoFormatToDatetimeLocal(hugoDate) {
    if (!hugoDate) return '';
    
    try {
        const date = new Date(hugoDate);
        if (isNaN(date.getTime())) return '';
        
        // 返回YYYY-MM-DDTHH:mm格式
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        const hours = String(date.getHours()).padStart(2, '0');
        const minutes = String(date.getMinutes()).padStart(2, '0');
        
        return `${year}-${month}-${day}T${hours}:${minutes}`;
    } catch (error) {
        console.error('Hugo format conversion error:', error);
        return '';
    }
}

/**
 * 检查并修复发布日期格式
 */
function checkAndFixDate() {
    const publishDateInput = document.getElementById('publishDate');
    const fixDateBtn = document.getElementById('fixDateBtn');
    const dateFormatAlert = document.getElementById('dateFormatAlert');
    
    if (!publishDateInput) return;
    
    const currentValue = publishDateInput.value;
    if (!currentValue) return;
    
    // 将datetime-local格式转换为Hugo格式
    const hugoFormat = datetimeLocalToHugoFormat(currentValue);
    
    if (hugoFormat && isValidHugoDate(hugoFormat)) {
        // 格式已正确，隐藏修复按钮和警告
        if (fixDateBtn) fixDateBtn.style.display = 'none';
        if (dateFormatAlert) dateFormatAlert.style.display = 'none';
        
        // 显示成功提示
        showDateStatusMessage('时间格式正确', 'success');
    } else {
        // 尝试修复格式
        const fixedFormat = fixDateFormat(currentValue);
        if (fixedFormat) {
            // 更新输入框的值
            const newDatetimeLocalValue = hugoFormatToDatetimeLocal(fixedFormat);
            if (newDatetimeLocalValue) {
                publishDateInput.value = newDatetimeLocalValue;
                
                // 隐藏修复按钮和警告
                if (fixDateBtn) fixDateBtn.style.display = 'none';
                if (dateFormatAlert) dateFormatAlert.style.display = 'none';
                
                // 显示修复成功提示
                showDateStatusMessage('时间格式已修复', 'success');
            }
        } else {
            showDateStatusMessage('无法修复时间格式，请手动输入正确的时间', 'danger');
        }
    }
}

/**
 * 检查时间格式并显示提示
 */
function validateDateFormat() {
    const publishDateInput = document.getElementById('publishDate');
    const fixDateBtn = document.getElementById('fixDateBtn');
    const dateFormatAlert = document.getElementById('dateFormatAlert');
    const currentDateFormat = document.getElementById('currentDateFormat');
    const suggestedDateFormat = document.getElementById('suggestedDateFormat');
    
    if (!publishDateInput) return;
    
    const currentValue = publishDateInput.value;
    if (!currentValue) {
        // 隐藏所有提示
        if (fixDateBtn) fixDateBtn.style.display = 'none';
        if (dateFormatAlert) dateFormatAlert.style.display = 'none';
        return;
    }
    
    // 将datetime-local格式转换为Hugo格式用于检查
    const hugoFormat = datetimeLocalToHugoFormat(currentValue);
    
    if (hugoFormat && isValidHugoDate(hugoFormat)) {
        // 格式正确，隐藏警告
        if (fixDateBtn) fixDateBtn.style.display = 'none';
        if (dateFormatAlert) dateFormatAlert.style.display = 'none';
    } else {
        // 格式不正确，显示警告和修复按钮
        if (fixDateBtn) fixDateBtn.style.display = 'block';
        if (dateFormatAlert) {
            dateFormatAlert.style.display = 'block';
            
            // 显示当前格式和建议格式
            if (currentDateFormat) {
                currentDateFormat.textContent = currentValue + ' (datetime-local)';
            }
            if (suggestedDateFormat) {
                const suggestedFormat = fixDateFormat(currentValue);
                suggestedDateFormat.textContent = suggestedFormat || '无法自动生成';
            }
        }
    }
}

/**
 * 显示日期状态消息
 * @param {string} message 消息内容
 * @param {string} type 消息类型 (success, warning, danger)
 */
function showDateStatusMessage(message, type = 'info') {
    const helpDiv = document.getElementById('publishDateHelp');
    if (!helpDiv) return;
    
    // 移除之前的状态类
    helpDiv.className = 'form-text';
    
    // 添加新的状态类和消息
    if (type === 'success') {
        helpDiv.className = 'form-text text-success';
        helpDiv.innerHTML = `<i class="bi bi-check-circle me-1"></i>${message}`;
    } else if (type === 'danger') {
        helpDiv.className = 'form-text text-danger';
        helpDiv.innerHTML = `<i class="bi bi-exclamation-triangle me-1"></i>${message}`;
    } else if (type === 'warning') {
        helpDiv.className = 'form-text text-warning';
        helpDiv.innerHTML = `<i class="bi bi-exclamation-triangle me-1"></i>${message}`;
    } else {
        helpDiv.innerHTML = `<span data-i18n="articles.publish.date.help">设置文章的发布时间</span>`;
    }
    
    // 3秒后恢复原始状态
    if (type !== 'info') {
        setTimeout(() => {
            if (helpDiv) {
                helpDiv.className = 'form-text';
                helpDiv.innerHTML = `<span data-i18n="articles.publish.date.help">设置文章的发布时间</span>`;
            }
        }, 3000);
    }
}

// 初始化时间格式检查
document.addEventListener('DOMContentLoaded', function() {
    const publishDateInput = document.getElementById('publishDate');
    if (publishDateInput) {
        // 监听时间输入变化
        publishDateInput.addEventListener('input', validateDateFormat);
        publishDateInput.addEventListener('change', validateDateFormat);
        
        // 页面加载时检查一次
        setTimeout(validateDateFormat, 100);
    }
});

// 导出时间相关函数
window.checkAndFixDate = checkAndFixDate;
window.validateDateFormat = validateDateFormat;
window.handleImageUpload = articleEditor.handleImageUpload.bind(articleEditor);