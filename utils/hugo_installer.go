package utils

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	HugoVersion = "0.148.2"
	WindowsURL  = "https://github.com/gohugoio/hugo/releases/download/v0.148.2/hugo_extended_0.148.2_windows-amd64.zip"
	LinuxURL    = "https://github.com/gohugoio/hugo/releases/download/v0.148.2/hugo_extended_0.148.2_linux-amd64.tar.gz"
	MacOSURL    = "https://github.com/gohugoio/hugo/releases/download/v0.148.2/hugo_extended_0.148.2_darwin-universal.tar.gz"
)

// HugoInstaller 处理Hugo安装
type HugoInstaller struct {
	InstallDir string
}

// NewHugoInstaller 创建新的Hugo安装器
func NewHugoInstaller() *HugoInstaller {
	// 获取可执行文件目录作为安装目录
	execPath, err := os.Executable()
	if err != nil {
		// 如果获取失败，使用当前目录
		execPath, _ = os.Getwd()
	}
	installDir := filepath.Join(filepath.Dir(execPath), "hugo")
	
	return &HugoInstaller{
		InstallDir: installDir,
	}
}

// IsHugoInstalled 检查Hugo是否已安装
func (h *HugoInstaller) IsHugoInstalled() bool {
	// 先检查系统PATH中是否有hugo
	if _, err := exec.LookPath("hugo"); err == nil {
		return true
	}
	
	// 检查本地安装目录
	hugoPath := h.GetHugoPath()
	if _, err := os.Stat(hugoPath); err == nil {
		return true
	}
	
	return false
}

// GetHugoPath 获取Hugo可执行文件路径
func (h *HugoInstaller) GetHugoPath() string {
	var hugoExe string
	if runtime.GOOS == "windows" {
		hugoExe = "hugo.exe"
	} else {
		hugoExe = "hugo"
	}
	
	localPath := filepath.Join(h.InstallDir, hugoExe)
	
	// 如果本地存在，返回本地路径
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}
	
	// 否则返回系统路径
	return "hugo"
}

// GetHugoVersion 获取Hugo版本信息
func (h *HugoInstaller) GetHugoVersion() (string, error) {
	hugoPath := h.GetHugoPath()
	cmd := exec.Command(hugoPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	version := strings.TrimSpace(string(output))
	return version, nil
}

// InstallHugo 安装Hugo
func (h *HugoInstaller) InstallHugo() error {
	// 创建安装目录
	if err := os.MkdirAll(h.InstallDir, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %v", err)
	}
	
	// 获取下载URL
	downloadURL := h.getDownloadURL()
	if downloadURL == "" {
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	
	// 下载文件
	tempFile, err := h.downloadFile(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download Hugo: %v", err)
	}
	defer os.Remove(tempFile)
	
	// 解压并安装
	if err := h.extractAndInstall(tempFile); err != nil {
		return fmt.Errorf("failed to extract Hugo: %v", err)
	}
	
	return nil
}

// getDownloadURL 根据平台获取下载URL
func (h *HugoInstaller) getDownloadURL() string {
	switch runtime.GOOS {
	case "windows":
		return WindowsURL
	case "linux":
		return LinuxURL
	case "darwin":
		return MacOSURL
	default:
		return ""
	}
}

// downloadFile 下载文件到临时目录
func (h *HugoInstaller) downloadFile(url string) (string, error) {
	// 创建临时文件
	tempFile, err := os.CreateTemp("", "hugo-download-*")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()
	
	// 下载文件
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %s", resp.Status)
	}
	
	// 保存到临时文件
	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		return "", err
	}
	
	return tempFile.Name(), nil
}

// extractAndInstall 解压并安装Hugo
func (h *HugoInstaller) extractAndInstall(archivePath string) error {
	if runtime.GOOS == "windows" {
		return h.extractZip(archivePath)
	} else {
		return h.extractTarGz(archivePath)
	}
}

// extractZip 解压ZIP文件（Windows）
func (h *HugoInstaller) extractZip(zipPath string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	
	for _, file := range reader.File {
		if file.Name == "hugo.exe" {
			// 提取hugo.exe
			rc, err := file.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			
			// 创建目标文件
			targetPath := filepath.Join(h.InstallDir, "hugo.exe")
			targetFile, err := os.Create(targetPath)
			if err != nil {
				return err
			}
			defer targetFile.Close()
			
			// 复制文件内容
			if _, err := io.Copy(targetFile, rc); err != nil {
				return err
			}
			
			// 设置执行权限
			if err := os.Chmod(targetPath, 0755); err != nil {
				return err
			}
			
			break
		}
	}
	
	return nil
}

// extractTarGz 解压TAR.GZ文件（Linux/macOS）
func (h *HugoInstaller) extractTarGz(tarGzPath string) error {
	// 这里可以实现tar.gz解压逻辑
	// 为简化，目前只实现Windows版本
	return fmt.Errorf("tar.gz extraction not implemented yet")
}

// GetInstallStatus 获取安装状态信息
func (h *HugoInstaller) GetInstallStatus() map[string]interface{} {
	status := make(map[string]interface{})
	
	status["installed"] = h.IsHugoInstalled()
	status["install_dir"] = h.InstallDir
	status["hugo_path"] = h.GetHugoPath()
	status["platform"] = runtime.GOOS
	status["available_version"] = HugoVersion
	
	if h.IsHugoInstalled() {
		if version, err := h.GetHugoVersion(); err == nil {
			status["current_version"] = version
		} else {
			status["current_version"] = "unknown"
		}
	}
	
	return status
}