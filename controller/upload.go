package controller

import (
    "crypto/md5"
    "encoding/base64"
    "fmt"
    "github.com/gin-gonic/gin"
    "hugo-manager-go/config"
    "io"
    "os"
    "path/filepath"
    "strings"
)

// 处理粘贴上传的图片（base64格式）
func UploadImageBase64(c *gin.Context) {
    var request struct {
        ImageData string `json:"image_data"`
        Filename  string `json:"filename"`
    }
    
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": "请求格式错误"})
        return
    }
    
    // 解析base64数据
    parts := strings.Split(request.ImageData, ",")
    if len(parts) != 2 {
        c.JSON(400, gin.H{"error": "无效的图片数据"})
        return
    }
    
    // 获取图片格式
    header := parts[0]
    data := parts[1]
    
    var ext string
    if strings.Contains(header, "jpeg") {
        ext = ".jpg"
    } else if strings.Contains(header, "png") {
        ext = ".png"
    } else if strings.Contains(header, "gif") {
        ext = ".gif"
    } else if strings.Contains(header, "webp") {
        ext = ".webp"
    } else {
        ext = ".jpg" // 默认为jpg
    }
    
    // 解码base64
    imageData, err := base64.StdEncoding.DecodeString(data)
    if err != nil {
        c.JSON(400, gin.H{"error": "解码图片数据失败"})
        return
    }
    
    // 生成基于文件内容的MD5文件名
    h := md5.New()
    h.Write(imageData)
    hash := fmt.Sprintf("%x", h.Sum(nil))
    filename := hash + ext
    
    // 保存文件
    uploadDir := config.GetImagesDir()
    os.MkdirAll(uploadDir, os.ModePerm)
    savePath := filepath.Join(uploadDir, filename)
    
    if err := os.WriteFile(savePath, imageData, 0644); err != nil {
        c.JSON(500, gin.H{"error": "保存图片失败: " + err.Error()})
        return
    }
    
    // 返回Hugo项目中的相对路径
    c.JSON(200, gin.H{
        "url":      "/uploads/images/" + filename,
        "filename": filename,
        "size":     len(imageData),
    })
}

// 处理拖拽上传的图片
func UploadImageFile(c *gin.Context) {
    file, err := c.FormFile("file")
    if err != nil {
        c.JSON(400, gin.H{"error": "获取上传文件失败: " + err.Error()})
        return
    }
    
    // 检查文件类型
    allowedTypes := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
    ext := strings.ToLower(filepath.Ext(file.Filename))
    isAllowed := false
    for _, allowedType := range allowedTypes {
        if ext == allowedType {
            isAllowed = true
            break
        }
    }
    
    if !isAllowed {
        c.JSON(400, gin.H{"error": "不支持的图片格式，请上传 jpg, png, gif 或 webp 格式的图片"})
        return
    }
    
    // 读取文件内容生成hash
    src, err := file.Open()
    if err != nil {
        c.JSON(500, gin.H{"error": "无法读取上传文件"})
        return
    }
    defer src.Close()
    
    h := md5.New()
    io.Copy(h, src)
    hash := fmt.Sprintf("%x", h.Sum(nil))
    
    // 生成文件名
    filename := hash + ext
    
    // 保存文件
    uploadDir := config.GetImagesDir()
    os.MkdirAll(uploadDir, os.ModePerm)
    savePath := filepath.Join(uploadDir, filename)
    
    // 重置文件指针
    src.Seek(0, io.SeekStart)
    out, err := os.Create(savePath)
    if err != nil {
        c.JSON(500, gin.H{"error": "创建文件失败"})
        return
    }
    defer out.Close()
    
    size, err := io.Copy(out, src)
    if err != nil {
        c.JSON(500, gin.H{"error": "保存文件失败"})
        return
    }
    
    c.JSON(200, gin.H{
        "url":      "/uploads/images/" + filename,
        "filename": filename,
        "size":     size,
    })
}