# CLAUDE.md – Claude 协作编码规范（KISS + 多语言 版）

本项目为基于 **Go + Bootstrap + Claude AI 协作开发** 的轻量级系统。为保证代码整洁、统一风格、便于维护和 AI 接力开发，制定以下规范。

---

## 🧠 开发哲学：KISS 原则

> **KISS = Keep It Simple, Stupid**

Claude 生成的代码必须满足以下理念：

* 简洁优先，不搞花派
* 不重复造轮子，能复用就复用
* 函数专一、职责单一
* 每一行代码都能“自解释”或注释清楚
* 所有逻辑都应明确可读、方便调试

---

## 📁 项目结构说明（已完成 90%）

```
K:.
├─.claude              # Claude AI 缓存/上下文
├─.idea                # IDE 配置（可忽略）
├─config               # 配置文件（如 config.yaml）
├─controller           # 控制器逻辑（每个模块一个文件）
├─docs                 # 开发文档
│  └─images            # 项目文档截图、示意图
├─router               # 路由初始化
├─static               # 静态资源（Bootstrap、JS、图片）
│  ├─css
│  ├─js
│  └─uploads/images    # 用户上传图像，自动按内容重命名
├─translations         # 多语言资源包（如 zh-CN.json）
├─utils                # 工具函数（时间、错误包装等）
└─view                 # Go模板视图结构（按模块划分）
    ├─article
    ├─deploy
    ├─files
    ├─home
    ├─images
    ├─layout
    ├─settings
    └─trash
```

---

## 🔠 命名风格统一（Go）

| 类型       | 命名风格        | 示例                       |
| -------- | ----------- | ------------------------ |
| 包/文件名    | 小写+下划线      | `user_handler.go`        |
| 函数名      | camelCase   | `getUserList()`          |
| 结构体      | PascalCase  | `UserInfo`               |
| JSON tag | snake\_case | `json:"user_name"`       |
| 模板文件名    | 小写短名        | `list.html`, `edit.html` |

---

## ✏️ Claude Code 提示规范

Claude 所生成的函数或文件顶部，必须注明使用的提示词（prompt）来源：

```go
// Claude Prompt: 实现一个 GET /images 页面，展示所有图片缩略图
func GetImageList(c *gin.Context) {
    ...
}
```

生成代码必须满足：

* 不生成无用代码
* 没有语法错误
* 明确归属功能模块
* 包含适当注释

---

## 🌐 前端模板约定（view）

* 所有页面模板放入 `view/` 下的子模块中
* 使用 Go 模板 + Bootstrap 编写，风格统一
* 通用布局放在 `view/layout/`（如 `header.html`, `footer.html`）
* 模板文件命名：`list.html`, `form.html`, `detail.html`
* 页面如有 JS，放在 `static/js/<模块>.js`

---

## 📦 路由与控制器结构（router + controller）

* 所有路由统一在 `router/router.go` 注册
* 控制器以模块名划分，每个控制器一个文件，如：

  * `controller/article.go`
  * `controller/settings.go`
* 每个函数只处理一个任务（例如：查询 / 保存 / 删除）

```go
func GetArticleList(c *gin.Context)    // 查询列表
func CreateArticle(c *gin.Context)     // 创建新项
func DeleteArticle(c *gin.Context)     // 删除项
```

---

## 🧰 工具函数（utils/）

* 通用逻辑封装在 `utils/` 包内
* 所有工具函数必须简洁、通用、可测试
* 常用示例：`ResponseJSON()`, `GetClientIP()`, `FormatTime()`

---

## 🌚 多语言支持说明（系统核心需求）

本系统支持中英文等多语言工作模式，以下为 Claude 生成代码时需遵循的约定：

### 1. 模板语言调用

所有页面显示文字必须通过 `i18n` 模板函数调用：

```html
<!-- ❌ 错误做法 -->
<h1>欢迎使用本系统</h1>

<!-- ✅ 正确做法 -->
<h1>{{ i18n "home.welcome" }}</h1>
```

### 2. 后端提示回复国际化

所有后端 JSON 响应提示语，也应调用翻译函数，例如：

```go
c.JSON(http.StatusOK, gin.H{
    "message": i18n.T("user.create_success"),
})
```

### 3. 翻译文件结构

翻译文件统一存放于：

```
/translations
├─ zh-CN.json
└─ en-US.json
```

* 格式：JSON / YAML（默认 JSON）
* 命名风格：`模块名.键名`，如：`login.button_text`

### 4. Claude 编码行为要求

Claude 生成任何含“文字”的部分时，必须：

* 调用 `i18n` 机制
* 使用已有 key 或生成规范 key
* 不允许确码英文、中文提示语
* 所有输入都要做空值验证，非法输入验证
* 所有输出都要做空值验证，避免出现模板空值报错
* 所有功能模块，按模块化写代码，单文件代码不能超过800行 
---

## 📄 配置与启动（config/）

* 配置文件使用 `YAML` 格式：如 `config/app.yaml`
* 建议用 Viper 加载，支持环境变量覆盖
* 默认端口：`8080`，端口如果被占用自动跳转到下一个端口启动。

---

## 🧪 测试建议

* 可选使用 `xxx_test.go` 添加基本单元测试 只有在BUG无法找到情况下才写，用完删除。
* 单元测试遵循 KISS 原则：测试场景最小可用即可

---

## 📜 附加建议




* 禁止生成“装饰性代码”或“AI猜测但未验证的逻辑”：Claude 不应自行生成意图不明或策略不明确的逻辑，需要输出体系化、规范化的结果。

* 所有数据操作必须有容错处理：避免 panic，根据故障返回明确的错误信息

* Claude 未知行为必须标注 TODO / FIXME：如果无法确定逻辑或需要律合检查，需添加注释标记

* Claude 代码要给出根据而非描述： 如非线程实现、未确认接口、未确定动作不输出代码，而是指导、表格、区分 TODO 等

* Claude 生成的文件/函数必须能被直接使用，不允许写半抽象代码

## Claude 开发察根列表

Claude 不应生成模糊/未验证逻辑

所有数据操作需有容错处理

必须标注 TODO/FIXME

鼓励使用 Makefile 与 .env

提倡 Claude 生成系统化输出（而非“建议性的描述”）