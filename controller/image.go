
package controller

import (
    "crypto/md5"
    "fmt"
    "github.com/gin-gonic/gin"
    "hugo-manager-go/config"
    "io"
    "os"
    "path/filepath"
)

func UploadImage(c *gin.Context) {
    file, err := c.FormFile("file")
    if err != nil {
        c.String(400, "上传失败: %v", err)
        return
    }

    src, err := file.Open()
    if err != nil {
        c.String(500, "无法读取上传内容")
        return
    }
    defer src.Close()

    h := md5.New()
    io.Copy(h, src)
    hash := fmt.Sprintf("%x", h.Sum(nil))

    ext := filepath.Ext(file.Filename)
    name := hash + ext

    uploadDir := config.GetImagesDir()
    os.MkdirAll(uploadDir, os.ModePerm)
    savePath := filepath.Join(uploadDir, name)

    src.Seek(0, io.SeekStart)
    out, _ := os.Create(savePath)
    defer out.Close()
    io.Copy(out, src)

    // 返回Hugo项目中的相对路径
    c.String(200, "/uploads/images/"+name)
}
