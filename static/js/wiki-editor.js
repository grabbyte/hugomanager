/**
 * Wiki Editor Module - Wiki编辑器模块
 * Claude Prompt: 基于博客编辑器的Wiki知识条目编辑器，支持全功能Markdown编辑
 * 
 * 功能包括：
 * - CodeMirror编辑器初始化和管理
 * - Markdown语法高亮
 * - 图片上传（拖拽、粘贴、按钮）
 * - Wiki条目保存和预览
 * - 元数据管理
 */

class WikiEditor {
    constructor() {
        this.codeMirrorEditor = null;
        this.hasUnsavedChanges = false;
        this.currentEntryId = null;
        
        // 绑定方法到实例
        this.saveWikiEntry = this.saveWikiEntry.bind(this);
        this.previewWiki = this.previewWiki.bind(this);
        this.insertImage = this.insertImage.bind(this);
    }

    /**
     * 初始化编辑器
     */
    init() {
        document.addEventListener('DOMContentLoaded', () => {
            this.initCodeMirrorEditor();
            this.initChangeTracking();
            this.initKeyboardShortcuts();
            this.loadCategoryOptions();
            this.getCurrentEntryId();
        });
    }

    /**
     * 获取当前编辑的条目ID
     */
    getCurrentEntryId() {
        const entryIdInput = document.getElementById('wikiEntryId');
        if (entryIdInput && entryIdInput.value) {
            this.currentEntryId = entryIdInput.value;
        }
    }

    /**
     * 初始化CodeMirror编辑器
     */
    initCodeMirrorEditor() {
        const textarea = document.getElementById('wikiContentTextarea');
        const editorContainer = document.getElementById('wikiContentEditor');
        
        if (!textarea || !editorContainer) return;
        
        // 创建CodeMirror编辑器
        this.codeMirrorEditor = CodeMirror.fromTextArea(textarea, {
            mode: 'gfm',
            theme: 'default',
            lineNumbers: true,
            lineWrapping: true,
            autofocus: false,
            placeholder: '在此输入Wiki条目的 Markdown 内容...支持拖拽图片和Ctrl+V粘贴图片',
            extraKeys: {
                'Ctrl-S': (cm) => {
                    this.saveWikiEntry();
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
     * 初始化变更追踪
     */
    initChangeTracking() {
        const inputs = [
            'wikiTitle', 'wikiCategory', 'wikiType', 'wikiDifficulty', 'wikiUrl',
            'wikiDescription', 'wikiTags', 'wikiKeywords', 'wikiSource', 'wikiVersion'
        ].map(id => document.getElementById(id)).filter(el => el);
        
        const checkboxes = [
            'wikiOfficial', 'wikiFavorite', 'wikiFrequent'
        ].map(id => document.getElementById(id)).filter(el => el);
        
        inputs.forEach(element => {
            element.addEventListener('input', () => {
                this.hasUnsavedChanges = true;
            });
        });
        
        checkboxes.forEach(element => {
            element.addEventListener('change', () => {
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
                this.saveWikiEntry();
            }
        });
    }

    /**
     * 加载分类选项
     */
    async loadCategoryOptions() {
        try {
            const response = await fetch('/api/categories/active?module_type=wiki');
            const data = await response.json();
            
            if (!data.error) {
                const select = document.getElementById('wikiCategory');
                if (select) {
                    const currentValue = select.dataset.value || select.value;
                    select.innerHTML = '';
                    
                    (data.categories || []).forEach(category => {
                        const option = document.createElement('option');
                        option.value = category.id;
                        option.textContent = category.name;
                        if (category.id === currentValue) {
                            option.selected = true;
                        }
                        select.appendChild(option);
                    });
                }
            }
        } catch (error) {
            console.error('Error loading category options:', error);
        }
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
     * 保存Wiki条目
     */
    async saveWikiEntry() {
        const title = document.getElementById('wikiTitle')?.value.trim() || '';
        const category = document.getElementById('wikiCategory')?.value || '';
        const type = document.getElementById('wikiType')?.value || 'guide';
        const difficulty = document.getElementById('wikiDifficulty')?.value || 'beginner';
        const url = document.getElementById('wikiUrl')?.value.trim() || '';
        const description = document.getElementById('wikiDescription')?.value.trim() || '';
        const tags = document.getElementById('wikiTags')?.value.trim() || '';
        const keywords = document.getElementById('wikiKeywords')?.value.trim() || '';
        const source = document.getElementById('wikiSource')?.value.trim() || '';
        const version = document.getElementById('wikiVersion')?.value.trim() || '';
        const official = document.getElementById('wikiOfficial')?.checked || false;
        const favorite = document.getElementById('wikiFavorite')?.checked || false;
        const frequent = document.getElementById('wikiFrequent')?.checked || false;
        
        // 从CodeMirror编辑器获取内容
        const content = this.codeMirrorEditor ? this.codeMirrorEditor.getValue() : '';
        
        if (!title) {
            alert('请填写条目标题');
            return;
        }
        
        const requestData = {
            title: title,
            category: category,
            type: type,
            difficulty: difficulty,
            url: url,
            description: description,
            tags: tags,
            keywords: keywords,
            source: source,
            version: version,
            official: official,
            favorite: favorite,
            frequent: frequent,
            content: content
        };
        
        try {
            let response;
            
            if (this.currentEntryId) {
                // 编辑模式 - 更新条目
                response = await fetch(`/api/wiki/content/${this.currentEntryId}`, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(requestData)
                });
            } else {
                // 新建模式 - 创建条目
                response = await fetch('/api/wiki/content', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify(requestData)
                });
            }
            
            const data = await response.json();
            
            if (data.error) {
                alert(`保存失败: ${data.error}`);
                return;
            }
            
            // 保存成功
            this.hasUnsavedChanges = false;
            alert(`Wiki条目${this.currentEntryId ? '更新' : '创建'}成功！`);
            
            // 如果是新建，更新当前条目ID
            if (!this.currentEntryId && data.id) {
                this.currentEntryId = data.id;
                const entryIdInput = document.getElementById('wikiEntryId');
                if (entryIdInput) {
                    entryIdInput.value = data.id;
                }
                
                // 更新浏览器URL为编辑模式
                if (window.history && window.history.pushState) {
                    window.history.pushState(null, null, `/wiki/edit/${data.id}`);
                }
            }
            
            // 3秒后自动跳转回wiki列表页面
            setTimeout(() => {
                window.location.href = '/wiki';
            }, 3000);
            
        } catch (error) {
            console.error('Error saving wiki entry:', error);
            alert('保存失败: ' + error.message);
        }
    }

    /**
     * 预览Wiki条目
     */
    async previewWiki() {
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
            
            console.log('Wiki预览URL:', previewUrl);
            
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
        const title = document.getElementById('wikiTitle')?.value.trim() || '';
        
        if (!title) {
            return baseUrl + 'wiki/';
        }
        
        // 生成基于标题的URL
        const urlPath = this.sanitizeUrlPath(title);
        
        let fullUrl = baseUrl;
        if (!fullUrl.endsWith('/')) {
            fullUrl += '/';
        }
        
        return fullUrl + 'wiki/' + urlPath + '/';
    }

    /**
     * 清理URL路径
     */
    sanitizeUrlPath(title) {
        return title.toLowerCase()
            .replace(/[^\w\s-]/g, '') // 移除特殊字符
            .replace(/\s+/g, '-')     // 空格替换为连字符
            .replace(/-+/g, '-')      // 多个连字符合并为一个
            .replace(/^-|-$/g, '');   // 移除首尾连字符
    }

    /**
     * 显示/隐藏预览加载状态
     */
    showPreviewLoading(show) {
        const previewBtn = document.querySelector('button[onclick="previewWiki()"]');
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
            const textarea = document.getElementById('wikiContentTextarea');
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
const wikiEditor = new WikiEditor();

// 初始化编辑器
wikiEditor.init();

// 导出全局函数供HTML调用
window.saveWikiEntry = wikiEditor.saveWikiEntry;
window.previewWiki = wikiEditor.previewWiki;
window.insertImage = wikiEditor.insertImage;
window.handleImageUpload = wikiEditor.handleImageUpload.bind(wikiEditor);