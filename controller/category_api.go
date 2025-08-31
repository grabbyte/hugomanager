package controller

import (
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
)

// Claude Prompt: 实现分类管理相关的API接口

// 获取所有分类
func GetCategories(c *gin.Context) {
    moduleType := c.Query("module_type")
    if moduleType == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "模块类型参数不能为空"})
        return
    }
    
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    categories := collections.Categories[moduleType]
    if categories == nil {
        categories = []Category{}
    }
    
    c.JSON(http.StatusOK, gin.H{"categories": categories})
}

// 创建新分类
func CreateCategory(c *gin.Context) {
    var request struct {
        Name        string `json:"name"`
        Icon        string `json:"icon"`
        Color       string `json:"color"`
        Description string `json:"description"`
        ModuleType  string `json:"module_type"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
        return
    }
    
    if request.Name == "" || request.ModuleType == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "分类名称和模块类型不能为空"})
        return
    }
    
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // 检查分类名称是否已存在
    for _, category := range collections.Categories[request.ModuleType] {
        if category.Name == request.Name {
            c.JSON(http.StatusBadRequest, gin.H{"error": "分类名称已存在"})
            return
        }
    }
    
    // 生成ID和时间
    now := time.Now()
    id := fmt.Sprintf("%s_%d", request.ModuleType, now.UnixNano())
    
    // 获取下一个排序号
    maxSortOrder := 0
    for _, category := range collections.Categories[request.ModuleType] {
        if category.SortOrder > maxSortOrder {
            maxSortOrder = category.SortOrder
        }
    }
    
    // 创建新分类
    category := Category{
        ID:          id,
        Name:        request.Name,
        Icon:        request.Icon,
        Color:       request.Color,
        Description: request.Description,
        ModuleType:  request.ModuleType,
        Enabled:     true,
        SortOrder:   maxSortOrder + 1,
        IsDefault:   false,
        CreatedAt:   now,
        UpdatedAt:   now,
    }
    
    // 添加到分类列表
    if collections.Categories[request.ModuleType] == nil {
        collections.Categories[request.ModuleType] = []Category{}
    }
    collections.Categories[request.ModuleType] = append(collections.Categories[request.ModuleType], category)
    
    // 保存数据
    if err := saveCollections(collections); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "message": "分类创建成功",
        "category": category,
    })
}

// 更新分类
func UpdateCategory(c *gin.Context) {
    categoryID := c.Param("id")
    moduleType := c.Query("module_type")
    
    if categoryID == "" || moduleType == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "分类ID和模块类型不能为空"})
        return
    }
    
    var request struct {
        Name        string `json:"name"`
        Icon        string `json:"icon"`
        Color       string `json:"color"`
        Description string `json:"description"`
        Enabled     *bool  `json:"enabled"`
        SortOrder   *int   `json:"sort_order"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
        return
    }
    
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // 查找并更新分类
    found := false
    for i, category := range collections.Categories[moduleType] {
        if category.ID == categoryID {
            if request.Name != "" {
                category.Name = request.Name
            }
            if request.Icon != "" {
                category.Icon = request.Icon
            }
            if request.Color != "" {
                category.Color = request.Color
            }
            if request.Description != "" {
                category.Description = request.Description
            }
            if request.Enabled != nil {
                category.Enabled = *request.Enabled
            }
            if request.SortOrder != nil {
                category.SortOrder = *request.SortOrder
            }
            category.UpdatedAt = time.Now()
            
            collections.Categories[moduleType][i] = category
            found = true
            break
        }
    }
    
    if !found {
        c.JSON(http.StatusNotFound, gin.H{"error": "分类不存在"})
        return
    }
    
    // 保存数据
    if err := saveCollections(collections); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "分类更新成功"})
}

// 删除分类
func DeleteCategory(c *gin.Context) {
    categoryID := c.Param("id")
    moduleType := c.Query("module_type")
    
    if categoryID == "" || moduleType == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "分类ID和模块类型不能为空"})
        return
    }
    
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // 检查是否为默认分类，默认分类不允许删除
    var categoryToDelete Category
    found := false
    for _, category := range collections.Categories[moduleType] {
        if category.ID == categoryID {
            categoryToDelete = category
            found = true
            break
        }
    }
    
    if !found {
        c.JSON(http.StatusNotFound, gin.H{"error": "分类不存在"})
        return
    }
    
    if categoryToDelete.IsDefault {
        c.JSON(http.StatusBadRequest, gin.H{"error": "默认分类不允许删除"})
        return
    }
    
    // 检查该分类下是否有收藏项
    var collectionMap map[string][]CollectionItem
    switch moduleType {
    case "tools":
        collectionMap = collections.Tools
    case "books":
        collectionMap = collections.Books
    case "wiki":
        collectionMap = collections.Wiki
    case "ai-resources":
        collectionMap = collections.AIResources
    default:
        c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模块类型"})
        return
    }
    
    if items, exists := collectionMap[categoryID]; exists && len(items) > 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "该分类下存在收藏项，无法删除"})
        return
    }
    
    // 从分类列表中删除
    for i, category := range collections.Categories[moduleType] {
        if category.ID == categoryID {
            collections.Categories[moduleType] = append(
                collections.Categories[moduleType][:i],
                collections.Categories[moduleType][i+1:]...,
            )
            break
        }
    }
    
    // 保存数据
    if err := saveCollections(collections); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{"message": "分类删除成功"})
}

// 获取模块的可用分类列表（仅启用的）
func GetActiveCategories(c *gin.Context) {
    moduleType := c.Query("module_type")
    if moduleType == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "模块类型参数不能为空"})
        return
    }
    
    collections, err := loadCollections()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    
    // 过滤出启用的分类并按排序号排序
    var activeCategories []Category
    for _, category := range collections.Categories[moduleType] {
        if category.Enabled {
            activeCategories = append(activeCategories, category)
        }
    }
    
    // 按 SortOrder 排序
    for i := 0; i < len(activeCategories)-1; i++ {
        for j := i + 1; j < len(activeCategories); j++ {
            if activeCategories[i].SortOrder > activeCategories[j].SortOrder {
                activeCategories[i], activeCategories[j] = activeCategories[j], activeCategories[i]
            }
        }
    }
    
    c.JSON(http.StatusOK, gin.H{"categories": activeCategories})
}