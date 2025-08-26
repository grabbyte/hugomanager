package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"hugo-manager-go/config"
	"hugo-manager-go/utils"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 部署管理页面 - 多服务器部署功能
func DeployManager(c *gin.Context) {
	servers := config.GetServerConfigs()
	statuses := config.GetAllServerStatuses()

	// 获取Hugo serve状态
	hugoManager := utils.GetHugoServeManager()
	hugoStatus := hugoManager.GetStatus()

	// 处理Hugo状态，确保类型安全
	var status, url string
	if running, exists := hugoStatus["running"]; exists {
		if isRunning, ok := running.(bool); ok && isRunning {
			status = "running"
		} else {
			status = "stopped"
		}
	}
	if u, exists := hugoStatus["url"]; exists {
		if str, ok := u.(string); ok {
			url = str
		}
	}

	// 获取部署信息
	deployment := config.GetDeploymentInfo()

	c.HTML(200, "deploy/index.html", gin.H{
		"Title":          "部署管理",
		"Servers":        servers,
		"ServerStatuses": statuses,
		"Hugo": gin.H{
			"Status": status,
			"URL":    url,
		},
		"Deployment": deployment,
		"Page":       "deploy",
	})
}

// 获取SSH配置
func GetSSHConfig(c *gin.Context) {
	sshConfig := config.GetSSHConfig()
	// 不返回明文凭据到前端
	sshConfig.Username = ""
	sshConfig.Password = ""
	deploymentInfo := config.GetDeploymentInfo()

	c.JSON(200, gin.H{
		"ssh":                       sshConfig,
		"deployment":                deploymentInfo,
		"has_encrypted_credentials": config.HasEncryptedSSHCredentials(),
		"has_plaintext_credentials": config.HasPlaintextSSHCredentials(),
		"needs_decryption":          config.NeedsDecryption(),
		"is_decrypted":              config.IsDecryptionKeySet(),
	})
}

// 更新SSH配置
func UpdateSSHConfig(c *gin.Context) {
	var request struct {
		Host       string `json:"host"`
		Port       int    `json:"port"`
		Username   string `json:"username"`
		Password   string `json:"password"`
		KeyPath    string `json:"key_path"`
		RemotePath string `json:"remote_path"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "请求格式错误"})
		return
	}

	sshConfig := config.SSHConfig{
		Host:       request.Host,
		Port:       request.Port,
		Username:   request.Username,
		Password:   request.Password,
		KeyPath:    request.KeyPath,
		RemotePath: request.RemotePath,
	}

	config.SetSSHConfig(sshConfig)

	c.JSON(200, gin.H{
		"message": "SSH配置已更新",
	})
}

// Hugo构建
func BuildHugo(c *gin.Context) {
	projectPath := config.GetHugoProjectPath()

	// 检查项目目录是否存在
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		c.JSON(400, gin.H{"error": "Hugo项目目录不存在: " + projectPath})
		return
	}

	// 检查是否是Hugo项目
	configFile := filepath.Join(projectPath, "hugo.toml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		configFile = filepath.Join(projectPath, "config.toml")
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			configFile = filepath.Join(projectPath, "config.yaml")
			if _, err := os.Stat(configFile); os.IsNotExist(err) {
				c.JSON(400, gin.H{"error": "未找到Hugo配置文件，请确保这是一个Hugo项目"})
				return
			}
		}
	}

	// 广播构建开始
	utils.BroadcastBuildProgress("正在构建Hugo静态文件...", 0)

	// 更新构建状态
	config.UpdateDeploymentStatus("building", "正在构建Hugo静态文件...")

	// 广播构建进度
	utils.BroadcastBuildProgress("正在执行Hugo构建命令...", 50)

	// 执行Hugo构建
	cmd := exec.Command("hugo", "--source", projectPath)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		config.UpdateDeploymentStatus("failed", "Hugo构建失败: "+err.Error())
		utils.BroadcastError("build", "Hugo构建失败: "+err.Error())
		c.JSON(500, gin.H{
			"error":  "Hugo构建失败: " + err.Error(),
			"output": outputStr,
		})
		return
	}

	config.UpdateDeploymentStatus("success", "Hugo构建完成")
	utils.BroadcastComplete("build", "Hugo构建完成", 100)
	c.JSON(200, gin.H{
		"message": "Hugo构建成功",
		"output":  outputStr,
	})
}

// 部署到服务器
func DeployToServer(c *gin.Context) {
	sshConfig := config.GetSSHConfig()

	// 检查SSH配置
	if sshConfig.Host == "" || sshConfig.Username == "" || sshConfig.RemotePath == "" {
		c.JSON(400, gin.H{"error": "SSH配置不完整，请先配置SSH连接信息"})
		return
	}

	publicDir := config.GetPublicDir()

	// 检查public目录是否存在
	if _, err := os.Stat(publicDir); os.IsNotExist(err) {
		c.JSON(400, gin.H{"error": "public目录不存在，请先运行Hugo构建"})
		return
	}

	// 更新部署状态为开始部署
	config.UpdateDeploymentStatus("deploying", "正在部署文件到服务器...")

	// 广播部署开始
	utils.BroadcastDeployProgress("正在连接服务器...", 0, 100, 0, "")

	// 使用原生Go SSH进行部署
	result, err := utils.ExecuteDeployment(sshConfig, publicDir, sshConfig.RemotePath, false)
	if err != nil {
		config.UpdateDeploymentStatus("failed", "部署失败: "+err.Error())
		c.JSON(500, gin.H{
			"error":  "部署失败: " + err.Error(),
			"output": result.Output,
		})
		return
	}

	if !result.Success {
		config.UpdateDeploymentStatus("failed", result.Message)
		c.JSON(500, gin.H{
			"error":  result.Message,
			"output": result.Output,
		})
		return
	}

	// 更新部署统计和状态
	config.SetDeploymentStats(result.FilesDeployed, result.BytesTransferred)
	config.UpdateDeploymentStatus("success", fmt.Sprintf("部署完成，传输了 %d 个文件，共 %d 字节", result.FilesDeployed, result.BytesTransferred))

	c.JSON(200, gin.H{
		"message": "部署成功",
		"output":  result.Output,
		"stats": gin.H{
			"files_deployed":    result.FilesDeployed,
			"bytes_transferred": result.BytesTransferred,
		},
	})
}

// 检查工具是否可用
func checkToolAvailable(tool string) error {
	_, err := exec.LookPath(tool)
	return err
}

// 测试SSH连接
func TestSSHConnection(c *gin.Context) {
	sshConfig := config.GetSSHConfig()

	if sshConfig.Host == "" || sshConfig.Username == "" {
		c.JSON(400, gin.H{"error": "SSH配置不完整，请填写服务器地址和用户名"})
		return
	}

	// 检查认证方式
	if sshConfig.KeyPath == "" && sshConfig.Password == "" {
		c.JSON(400, gin.H{"error": "未配置SSH密码或密钥，请选择一种认证方式"})
		return
	}

	// 如果使用密钥认证，检查密钥文件
	if sshConfig.KeyPath != "" {
		if _, err := os.Stat(sshConfig.KeyPath); os.IsNotExist(err) {
			c.JSON(400, gin.H{
				"error":  "SSH密钥文件不存在: " + sshConfig.KeyPath,
				"output": "请检查密钥文件路径是否正确",
			})
			return
		}

		// 检查是否是 .ppk 格式（PuTTY格式）
		if strings.HasSuffix(strings.ToLower(sshConfig.KeyPath), ".ppk") {
			c.JSON(400, gin.H{
				"error": "不支持 .ppk 格式密钥",
				"output": "检测到PuTTY格式密钥文件。请转换为OpenSSH格式：\n" +
					"1. 使用PuTTYgen打开.ppk文件\n" +
					"2. 点击 Conversions → Export OpenSSH key\n" +
					"3. 保存为.pem格式文件\n" +
					"4. 或使用命令: puttygen " + sshConfig.KeyPath + " -O private-openssh -o newkey.pem",
			})
			return
		}
	}

	// 使用原生Go SSH库进行连接测试
	err := utils.TestSSHConnection(sshConfig)
	if err != nil {
		errorMsg := err.Error()

		// 提供更友好的错误信息
		if strings.Contains(errorMsg, "connection refused") {
			errorMsg = "连接被拒绝，请检查服务器是否运行SSH服务"
		} else if strings.Contains(errorMsg, "no route to host") {
			errorMsg = "无法到达主机，请检查服务器地址和网络连接"
		} else if strings.Contains(errorMsg, "permission denied") || strings.Contains(errorMsg, "authentication failed") {
			errorMsg = "认证失败，请检查用户名、密码或密钥是否正确"
		} else if strings.Contains(errorMsg, "timeout") {
			errorMsg = "连接超时，请检查服务器地址和端口是否正确"
		}

		c.JSON(500, gin.H{
			"error":  "SSH连接失败: " + errorMsg,
			"output": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"message": "SSH连接测试成功",
		"output":  "使用Go原生SSH库连接成功",
	})
}

// 一键构建和部署
func BuildAndDeploy(c *gin.Context) {
	// 首先构建
	projectPath := config.GetHugoProjectPath()

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		c.JSON(400, gin.H{"error": "Hugo项目目录不存在: " + projectPath})
		return
	}

	// 更新构建状态
	config.UpdateDeploymentStatus("building", "正在构建Hugo静态文件...")

	// 广播构建开始
	utils.BroadcastBuildProgress("正在构建Hugo静态文件...", 0)

	// 执行Hugo构建
	buildCmd := exec.Command("hugo", "--source", projectPath)
	buildOutput, err := buildCmd.CombinedOutput()
	buildOutputStr := string(buildOutput)

	if err != nil {
		config.UpdateDeploymentStatus("failed", "Hugo构建失败: "+err.Error())
		utils.BroadcastError("build", "Hugo构建失败: "+err.Error())
		c.JSON(500, gin.H{
			"error":  "Hugo构建失败: " + err.Error(),
			"output": buildOutputStr,
		})
		return
	}

	// 广播构建完成
	utils.BroadcastComplete("build", "Hugo构建完成", 100)

	// 然后部署
	sshConfig := config.GetSSHConfig()

	if sshConfig.Host == "" || sshConfig.Username == "" || sshConfig.RemotePath == "" {
		config.UpdateDeploymentStatus("failed", "SSH配置不完整")
		utils.BroadcastError("deploy", "SSH配置不完整")
		c.JSON(400, gin.H{
			"error":        "SSH配置不完整，请先配置SSH连接信息",
			"build_output": buildOutputStr,
		})
		return
	}

	// 更新部署状态
	config.UpdateDeploymentStatus("deploying", "正在部署文件到服务器...")

	// 广播部署开始
	utils.BroadcastDeployProgress("正在连接服务器...", 0, 100, 0, "")

	publicDir := config.GetPublicDir()

	// 使用原生Go SSH进行部署
	result, err := utils.ExecuteDeployment(sshConfig, publicDir, sshConfig.RemotePath, false)
	if err != nil {
		config.UpdateDeploymentStatus("failed", "部署失败: "+err.Error())
		c.JSON(500, gin.H{
			"error":         "部署失败: " + err.Error(),
			"build_output":  buildOutputStr,
			"deploy_output": result.Output,
		})
		return
	}

	if !result.Success {
		config.UpdateDeploymentStatus("failed", result.Message)
		c.JSON(500, gin.H{
			"error":         result.Message,
			"build_output":  buildOutputStr,
			"deploy_output": result.Output,
		})
		return
	}

	// 更新部署统计和状态
	config.SetDeploymentStats(result.FilesDeployed, result.BytesTransferred)
	config.UpdateDeploymentStatus("success", fmt.Sprintf("构建和部署完成，传输了 %d 个文件，共 %d 字节", result.FilesDeployed, result.BytesTransferred))

	c.JSON(200, gin.H{
		"message":       "构建和部署成功",
		"build_output":  buildOutputStr,
		"deploy_output": result.Output,
		"stats": gin.H{
			"files_deployed":    result.FilesDeployed,
			"bytes_transferred": result.BytesTransferred,
		},
	})
}

// 解析rsync统计输出
func parseRsyncStats(output string) (filesDeployed int, bytesTransferred int64) {
	// 查找文件传输统计
	// 例如: "sent 1,234,567 bytes  received 89 bytes  123,456 bytes/sec"
	re1 := regexp.MustCompile(`sent ([0-9,]+) bytes`)
	matches := re1.FindStringSubmatch(output)
	if len(matches) > 1 {
		bytesStr := strings.ReplaceAll(matches[1], ",", "")
		if bytes, err := strconv.ParseInt(bytesStr, 10, 64); err == nil {
			bytesTransferred = bytes
		}
	}

	// 查找文件数统计
	// 例如: "Number of files transferred: 123"
	re2 := regexp.MustCompile(`Number of files transferred: (\d+)`)
	matches = re2.FindStringSubmatch(output)
	if len(matches) > 1 {
		if files, err := strconv.Atoi(matches[1]); err == nil {
			filesDeployed = files
		}
	} else {
		// 如果没有找到明确的文件传输数，尝试从输出行数估算
		lines := strings.Split(output, "\n")
		fileCount := 0
		for _, line := range lines {
			line = strings.TrimSpace(line)
			// 跳过以特殊字符开头的行和空行
			if line != "" && !strings.HasPrefix(line, "sending") &&
				!strings.HasPrefix(line, "sent") && !strings.HasPrefix(line, "total") &&
				!strings.HasPrefix(line, "receiving") && !strings.Contains(line, "bytes/sec") {
				fileCount++
			}
		}
		if fileCount > 0 {
			filesDeployed = fileCount
		}
	}

	return filesDeployed, bytesTransferred
}

// 设置解密密钥
func SetDecryptionKey(c *gin.Context) {
	var request struct {
		MasterPassword string `json:"master_password"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "请求格式错误"})
		return
	}

	if request.MasterPassword == "" {
		c.JSON(400, gin.H{"error": "请输入主密码"})
		return
	}

	err := config.SetDecryptionKey(request.MasterPassword)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "解密密钥设置成功，可以安全使用加密的SSH密码了",
	})
}

// 检查解密状态
func CheckDecryptionStatus(c *gin.Context) {
	isDecrypted := config.IsDecryptionKeySet()
	c.JSON(200, gin.H{
		"is_decrypted": isDecrypted,
	})
}

// 使用加密保存SSH配置
func UpdateSSHConfigWithEncryption(c *gin.Context) {
	var request struct {
		Host           string `json:"host"`
		Port           int    `json:"port"`
		Username       string `json:"username"`
		Password       string `json:"password"`
		KeyPath        string `json:"key_path"`
		RemotePath     string `json:"remote_path"`
		MasterPassword string `json:"master_password"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "请求格式错误"})
		return
	}

	sshConfig := config.SSHConfig{
		Host:       request.Host,
		Port:       request.Port,
		Username:   request.Username,
		Password:   request.Password,
		KeyPath:    request.KeyPath,
		RemotePath: request.RemotePath,
	}

	// 如果提供了密码，需要主密码来加密
	if request.Password != "" && request.MasterPassword == "" {
		c.JSON(400, gin.H{"error": "保存SSH密码需要提供主密码进行加密"})
		return
	}

	var err error
	if request.Password != "" {
		err = config.SetSSHConfigWithEncryption(sshConfig, request.MasterPassword)
		// 设置解密密钥以便立即可用
		config.SetDecryptionKey(request.MasterPassword)
	} else {
		config.SetSSHConfig(sshConfig)
	}

	if err != nil {
		c.JSON(500, gin.H{"error": "保存配置失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "SSH配置已保存",
	})
}

// 增量部署到服务器（只上传变化的文件）
func IncrementalDeployToServer(c *gin.Context) {
	sshConfig := config.GetSSHConfig()

	// 检查SSH配置
	if sshConfig.Host == "" || sshConfig.Username == "" || sshConfig.RemotePath == "" {
		c.JSON(400, gin.H{"error": "SSH配置不完整，请先配置SSH连接信息"})
		return
	}

	publicDir := config.GetPublicDir()

	// 检查public目录是否存在
	if _, err := os.Stat(publicDir); os.IsNotExist(err) {
		c.JSON(400, gin.H{"error": "public目录不存在，请先运行Hugo构建"})
		return
	}

	// 更新部署状态为开始增量部署
	config.UpdateDeploymentStatus("deploying", "正在进行增量部署，只传输变化的文件...")

	// 使用原生Go SSH进行增量部署
	result, err := utils.ExecuteDeployment(sshConfig, publicDir, sshConfig.RemotePath, true)
	if err != nil {
		config.UpdateDeploymentStatus("failed", "增量部署失败: "+err.Error())
		c.JSON(500, gin.H{
			"error":  "增量部署失败: " + err.Error(),
			"output": result.Output,
		})
		return
	}

	if !result.Success {
		config.UpdateDeploymentStatus("failed", result.Message)
		c.JSON(500, gin.H{
			"error":  result.Message,
			"output": result.Output,
		})
		return
	}

	// 更新部署统计和状态
	config.SetDeploymentStats(result.FilesDeployed, result.BytesTransferred)
	config.UpdateDeploymentStatus("success", fmt.Sprintf("增量部署完成，传输了 %d 个文件，共 %d 字节", result.FilesDeployed, result.BytesTransferred))

	c.JSON(200, gin.H{
		"message": "增量部署成功",
		"output":  result.Output,
		"stats": gin.H{
			"files_deployed":    result.FilesDeployed,
			"bytes_transferred": result.BytesTransferred,
		},
	})
}

// 增量构建和部署
func IncrementalBuildAndDeploy(c *gin.Context) {
	// 首先构建
	projectPath := config.GetHugoProjectPath()

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		c.JSON(400, gin.H{"error": "Hugo项目目录不存在: " + projectPath})
		return
	}

	// 更新构建状态
	config.UpdateDeploymentStatus("building", "正在构建Hugo静态文件...")

	// 执行Hugo构建
	buildCmd := exec.Command("hugo", "--source", projectPath)
	buildOutput, err := buildCmd.CombinedOutput()
	buildOutputStr := string(buildOutput)

	if err != nil {
		config.UpdateDeploymentStatus("failed", "Hugo构建失败: "+err.Error())
		c.JSON(500, gin.H{
			"error":  "Hugo构建失败: " + err.Error(),
			"output": buildOutputStr,
		})
		return
	}

	// 然后增量部署
	sshConfig := config.GetSSHConfig()

	if sshConfig.Host == "" || sshConfig.Username == "" || sshConfig.RemotePath == "" {
		config.UpdateDeploymentStatus("failed", "SSH配置不完整")
		c.JSON(400, gin.H{
			"error":        "SSH配置不完整，请先配置SSH连接信息",
			"build_output": buildOutputStr,
		})
		return
	}

	// 更新部署状态为增量部署
	config.UpdateDeploymentStatus("deploying", "正在进行增量部署，只传输变化的文件...")

	publicDir := config.GetPublicDir()

	// 使用原生Go SSH进行增量部署
	result, err := utils.ExecuteDeployment(sshConfig, publicDir, sshConfig.RemotePath, true)
	if err != nil {
		config.UpdateDeploymentStatus("failed", "增量部署失败: "+err.Error())
		c.JSON(500, gin.H{
			"error":         "增量部署失败: " + err.Error(),
			"build_output":  buildOutputStr,
			"deploy_output": result.Output,
		})
		return
	}

	if !result.Success {
		config.UpdateDeploymentStatus("failed", result.Message)
		c.JSON(500, gin.H{
			"error":         result.Message,
			"build_output":  buildOutputStr,
			"deploy_output": result.Output,
		})
		return
	}

	// 更新部署统计和状态
	config.SetDeploymentStats(result.FilesDeployed, result.BytesTransferred)
	config.UpdateDeploymentStatus("success", fmt.Sprintf("增量构建和部署完成，传输了 %d 个文件，共 %d 字节", result.FilesDeployed, result.BytesTransferred))

	c.JSON(200, gin.H{
		"message":       "增量构建和部署成功",
		"build_output":  buildOutputStr,
		"deploy_output": result.Output,
		"stats": gin.H{
			"files_deployed":    result.FilesDeployed,
			"bytes_transferred": result.BytesTransferred,
		},
	})
}

// 暂停部署
func PauseDeployment(c *gin.Context) {
	config.SetDeploymentPaused(true)

	c.JSON(200, gin.H{
		"message": "部署已暂停",
		"status":  "paused",
	})
}

// 继续部署
func ResumeDeployment(c *gin.Context) {
	sshConfig := config.GetSSHConfig()

	// 检查SSH配置
	if sshConfig.Host == "" || sshConfig.Username == "" || sshConfig.RemotePath == "" {
		c.JSON(400, gin.H{"error": "SSH配置不完整，请先配置SSH连接信息"})
		return
	}

	// 检查是否有待处理的任务
	pendingCount := config.GetPendingTasksCount()
	if pendingCount == 0 {
		c.JSON(400, gin.H{"error": "没有待处理的上传任务"})
		return
	}

	// 设置为非暂停状态
	config.SetDeploymentPaused(false)
	config.UpdateDeploymentStatus("deploying", fmt.Sprintf("继续上传，剩余 %d 个文件...", pendingCount))

	publicDir := config.GetPublicDir()

	// 使用原生Go SSH进行部署
	result, err := utils.ExecuteDeployment(sshConfig, publicDir, sshConfig.RemotePath, true)
	if err != nil {
		if strings.Contains(err.Error(), "暂停") {
			c.JSON(200, gin.H{
				"message": "部署已暂停",
				"status":  "paused",
			})
			return
		}

		config.UpdateDeploymentStatus("failed", "继续部署失败: "+err.Error())
		c.JSON(500, gin.H{
			"error":  "继续部署失败: " + err.Error(),
			"output": result.Output,
		})
		return
	}

	if !result.Success {
		config.UpdateDeploymentStatus("failed", result.Message)
		c.JSON(500, gin.H{
			"error":  result.Message,
			"output": result.Output,
		})
		return
	}

	// 更新部署统计和状态
	config.SetDeploymentStats(result.FilesDeployed, result.BytesTransferred)
	config.UpdateDeploymentStatus("success", fmt.Sprintf("部署完成，传输了 %d 个文件，共 %d 字节", result.FilesDeployed, result.BytesTransferred))

	c.JSON(200, gin.H{
		"message": "继续部署成功",
		"output":  result.Output,
		"stats": gin.H{
			"files_deployed":    result.FilesDeployed,
			"bytes_transferred": result.BytesTransferred,
		},
	})
}

// 获取部署状态和任务信息
func GetDeploymentStatus(c *gin.Context) {
	deploymentInfo := config.GetDeploymentInfo()
	pendingTasks := config.GetPendingTasksCount()

	c.JSON(200, gin.H{
		"deployment":    deploymentInfo,
		"pending_tasks": pendingTasks,
		"is_paused":     config.IsDeploymentPaused(),
	})
}

// 启动Hugo serve
func StartHugoServe(c *gin.Context) {
	var request struct {
		Port int `json:"port"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "请求格式错误"})
		return
	}

	// 默认端口
	if request.Port <= 0 {
		request.Port = 1313
	}

	hugoManager := utils.GetHugoServeManager()
	if err := hugoManager.Start(request.Port); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "Hugo serve启动成功",
		"status":  hugoManager.GetStatus(),
	})
}

// 停止Hugo serve
func StopHugoServe(c *gin.Context) {
	hugoManager := utils.GetHugoServeManager()
	if err := hugoManager.Stop(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "Hugo serve已停止",
		"status":  hugoManager.GetStatus(),
	})
}

// 重启Hugo serve
func RestartHugoServe(c *gin.Context) {
	hugoManager := utils.GetHugoServeManager()
	if err := hugoManager.Restart(); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "Hugo serve重启成功",
		"status":  hugoManager.GetStatus(),
	})
}

// 获取Hugo serve状态
func GetHugoServeStatus(c *gin.Context) {
	hugoManager := utils.GetHugoServeManager()
	c.JSON(200, gin.H{
		"status": hugoManager.GetStatus(),
	})
}

// 加密现有的明文凭据
func EncryptPlaintextCredentials(c *gin.Context) {
	var request struct {
		MasterPassword string `json:"master_password"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "请求格式错误"})
		return
	}

	if request.MasterPassword == "" {
		c.JSON(400, gin.H{"error": "主密码不能为空"})
		return
	}

	if err := config.EncryptExistingCredentials(request.MasterPassword); err != nil {
		c.JSON(500, gin.H{"error": "加密失败: " + err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message":   "凭据加密成功",
		"encrypted": true,
	})
}

// 更新主密码（重新加密所有凭据）
func UpdateMasterPassword(c *gin.Context) {
	var request struct {
		OldMasterPassword string `json:"old_master_password"`
		NewMasterPassword string `json:"new_master_password"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "请求格式错误"})
		return
	}

	if request.OldMasterPassword == "" || request.NewMasterPassword == "" {
		c.JSON(400, gin.H{"error": "旧密码和新密码都不能为空"})
		return
	}

	// 首先用旧密码解密
	if err := config.SetDecryptionKey(request.OldMasterPassword); err != nil {
		c.JSON(400, gin.H{"error": "旧主密码错误"})
		return
	}

	// 获取解密后的凭据
	sshConfig := config.GetSSHConfig()

	// 用新密码重新加密
	if err := config.SetSSHConfigWithEncryption(sshConfig, request.NewMasterPassword); err != nil {
		c.JSON(500, gin.H{"error": "重新加密失败: " + err.Error()})
		return
	}

	// 设置新的解密密钥
	config.SetDecryptionKey(request.NewMasterPassword)

	c.JSON(200, gin.H{
		"message": "主密码更新成功",
		"updated": true,
	})
}

// ======== 多服务器部署API ========

// 获取所有服务器配置
func GetMultiServerConfigs(c *gin.Context) {
	servers := config.GetServerConfigs()
	c.JSON(200, gin.H{
		"servers": servers,
	})
}

// 获取单个服务器配置
func GetMultiServerConfig(c *gin.Context) {
	serverID := c.Param("server_id")
	server, err := config.GetServerConfig(serverID)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"server": server,
	})
}

// 添加服务器配置
func AddMultiServerConfig(c *gin.Context) {
	var request config.ServerConfig

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "请求格式错误: " + err.Error()})
		return
	}

	// 验证必填字段
	if request.Name == "" || request.Host == "" || request.Username == "" || request.RemotePath == "" {
		c.JSON(400, gin.H{"error": "服务器名称、地址、用户名和远程路径不能为空"})
		return
	}

	// 添加服务器
	config.AddServerConfig(request)

	c.JSON(200, gin.H{
		"message": "服务器配置已添加",
	})
}

// 更新服务器配置
func UpdateMultiServerConfig(c *gin.Context) {
	serverID := c.Param("server_id")
	var request config.ServerConfig

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, gin.H{"error": "请求格式错误: " + err.Error()})
		return
	}

	// 验证必填字段
	if request.Name == "" || request.Host == "" || request.Username == "" || request.RemotePath == "" {
		c.JSON(400, gin.H{"error": "服务器名称、地址、用户名和远程路径不能为空"})
		return
	}

	// 更新服务器
	err := config.UpdateServerConfig(serverID, request)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "服务器配置已更新",
	})
}

// 删除服务器配置
func DeleteMultiServerConfig(c *gin.Context) {
	serverID := c.Param("server_id")

	err := config.DeleteServerConfig(serverID)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"message": "服务器配置已删除",
	})
}

// 测试服务器连接
func TestMultiServerConnection(c *gin.Context) {
	serverID := c.Param("server_id")

	server, err := config.GetServerConfig(serverID)
	if err != nil {
		c.JSON(404, gin.H{"error": "服务器不存在"})
		return
	}

	// 转换为SSH配置格式进行测试
	sshConfig := config.SSHConfig{
		Host:       server.Host,
		Port:       server.Port,
		Username:   server.Username,
		Password:   server.Password,
		KeyPath:    server.KeyPath,
		RemotePath: server.RemotePath,
	}

	err = utils.TestSSHConnection(sshConfig)
	if err != nil {
		c.JSON(500, gin.H{
			"error": "SSH连接失败: " + err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"message": "SSH连接测试成功",
	})
}

// 部署到指定服务器
func DeployToMultiServer(c *gin.Context) {
	serverID := c.Param("server_id")

	server, err := config.GetServerConfig(serverID)
	if err != nil {
		c.JSON(404, gin.H{"error": "服务器不存在"})
		return
	}

	if !server.Enabled {
		c.JSON(400, gin.H{"error": "服务器已禁用"})
		return
	}

	// 更新服务器状态为部署中
	config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
		Status:   "deploying",
		Message:  "正在部署到 " + server.Name,
		Progress: 0,
		CanPause: true,
		CanStop:  true,
	})

	// 广播部署开始消息
	utils.BroadcastMultiServerDeployProgress(serverID, server.Name, "开始部署到 "+server.Name, 0, 100, 0, "")

	// 启动部署（异步）
	go func() {
		publicDir := config.GetPublicDir()

		// 检查public目录
		if _, err := os.Stat(publicDir); os.IsNotExist(err) {
			config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
				Status:  "failed",
				Message: "public目录不存在，请先运行Hugo构建",
			})
			utils.BroadcastMultiServerError(serverID, server.Name, "deploy", "public目录不存在，请先运行Hugo构建")
			return
		}

		// 转换为SSH配置格式
		sshConfig := config.SSHConfig{
			Host:       server.Host,
			Port:       server.Port,
			Username:   server.Username,
			Password:   server.Password,
			KeyPath:    server.KeyPath,
			RemotePath: server.RemotePath,
		}

		// 执行部署
		result, err := utils.ExecuteDeployment(sshConfig, publicDir, server.RemotePath, false)

		if err != nil || !result.Success {
			config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
				Status:  "failed",
				Message: "部署失败: " + result.Message,
			})
			utils.BroadcastMultiServerError(serverID, server.Name, "deploy", "部署失败: "+result.Message)
			return
		}

		// 更新成功状态
		config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
			Status:           "success",
			Message:          fmt.Sprintf("部署完成，传输了 %d 个文件", result.FilesDeployed),
			Progress:         100,
			FilesDeployed:    result.FilesDeployed,
			BytesTransferred: result.BytesTransferred,
		})

		// 广播部署完成消息
		utils.BroadcastMultiServerComplete(serverID, server.Name, "deploy", fmt.Sprintf("部署完成，传输了 %d 个文件", result.FilesDeployed), result.FilesDeployed)

		// 更新服务器的最后部署时间
		now := time.Now()
		server.LastDeployment = &now
		config.UpdateServerConfig(serverID, server)
	}()

	c.JSON(200, gin.H{
		"message": "开始部署到 " + server.Name,
	})
}

// 构建并部署到指定服务器
func BuildAndDeployToMultiServer(c *gin.Context) {
	serverID := c.Param("server_id")

	server, err := config.GetServerConfig(serverID)
	if err != nil {
		c.JSON(404, gin.H{"error": "服务器不存在"})
		return
	}

	if !server.Enabled {
		c.JSON(400, gin.H{"error": "服务器已禁用"})
		return
	}

	// 启动构建和部署（异步）
	go func() {
		// 1. 构建阶段
		config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
			Status:   "building",
			Message:  "正在构建Hugo站点...",
			Progress: 0,
		})

		// 广播构建开始消息
		utils.BroadcastMultiServerBuildProgress(serverID, server.Name, "开始构建Hugo站点...", 0)

		projectPath := config.GetHugoProjectPath()
		buildCmd := exec.Command("hugo", "--source", projectPath)
		_, err := buildCmd.CombinedOutput()

		if err != nil {
			config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
				Status:  "failed",
				Message: "Hugo构建失败: " + err.Error(),
			})
			utils.BroadcastMultiServerError(serverID, server.Name, "build", "Hugo构建失败: "+err.Error())
			return
		}

		// 广播构建完成消息
		utils.BroadcastMultiServerBuildProgress(serverID, server.Name, "Hugo构建完成", 100)

		// 2. 部署阶段
		config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
			Status:   "deploying",
			Message:  "正在部署到 " + server.Name,
			Progress: 50,
		})

		// 广播部署开始消息
		utils.BroadcastMultiServerDeployProgress(serverID, server.Name, "开始部署到 "+server.Name, 50, 100, 50, "")

		publicDir := config.GetPublicDir()

		// 转换为SSH配置格式
		sshConfig := config.SSHConfig{
			Host:       server.Host,
			Port:       server.Port,
			Username:   server.Username,
			Password:   server.Password,
			KeyPath:    server.KeyPath,
			RemotePath: server.RemotePath,
		}

		// 执行部署
		result, err := utils.ExecuteDeployment(sshConfig, publicDir, server.RemotePath, false)

		if err != nil || !result.Success {
			config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
				Status:  "failed",
				Message: "部署失败: " + result.Message,
			})
			utils.BroadcastMultiServerError(serverID, server.Name, "deploy", "部署失败: "+result.Message)
			return
		}

		// 更新成功状态
		config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
			Status:           "success",
			Message:          fmt.Sprintf("构建和部署完成，传输了 %d 个文件", result.FilesDeployed),
			Progress:         100,
			FilesDeployed:    result.FilesDeployed,
			BytesTransferred: result.BytesTransferred,
		})

		// 广播部署完成消息
		utils.BroadcastMultiServerComplete(serverID, server.Name, "deploy", fmt.Sprintf("构建和部署完成，传输了 %d 个文件", result.FilesDeployed), result.FilesDeployed)

		// 更新服务器的最后部署时间
		now := time.Now()
		server.LastDeployment = &now
		config.UpdateServerConfig(serverID, server)
	}()

	c.JSON(200, gin.H{
		"message": "开始构建并部署到 " + server.Name,
	})
}

// 暂停服务器部署
func PauseMultiServerDeployment(c *gin.Context) {
	serverID := c.Param("server_id")

	server, err := config.GetServerConfig(serverID)
	if err != nil {
		c.JSON(404, gin.H{"error": "服务器不存在"})
		return
	}

	config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
		Status:    "paused",
		Message:   "部署已暂停",
		CanResume: true,
	})

	// 广播暂停消息
	utils.BroadcastMultiServerPause(serverID, server.Name, "部署已暂停", 0, 0, 0)

	c.JSON(200, gin.H{
		"message": "部署已暂停",
	})
}

// 继续服务器部署
func ResumeMultiServerDeployment(c *gin.Context) {
	serverID := c.Param("server_id")

	server, err := config.GetServerConfig(serverID)
	if err != nil {
		c.JSON(404, gin.H{"error": "服务器不存在"})
		return
	}

	config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
		Status:   "deploying",
		Message:  "继续部署到 " + server.Name,
		CanPause: true,
		CanStop:  true,
	})

	// 广播继续部署消息
	utils.BroadcastMultiServerDeployProgress(serverID, server.Name, "继续部署到 "+server.Name, 0, 100, 0, "")

	c.JSON(200, gin.H{
		"message": "继续部署到 " + server.Name,
	})
}

// 停止服务器部署
func StopMultiServerDeployment(c *gin.Context) {
	serverID := c.Param("server_id")

	server, err := config.GetServerConfig(serverID)
	if err != nil {
		c.JSON(404, gin.H{"error": "服务器不存在"})
		return
	}

	config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
		Status:  "idle",
		Message: "部署已停止",
	})

	// 广播停止部署消息
	utils.BroadcastMultiServerProgress(serverID, server.Name, "deploy", "idle", "部署已停止", 0, 0, 0, "")

	c.JSON(200, gin.H{
		"message": "部署已停止",
	})
}

// 增量部署到指定服务器
func IncrementalDeployToMultiServer(c *gin.Context) {
	serverID := c.Param("server_id")

	server, err := config.GetServerConfig(serverID)
	if err != nil {
		c.JSON(404, gin.H{"error": "服务器不存在"})
		return
	}

	if !server.Enabled {
		c.JSON(400, gin.H{"error": "服务器已禁用"})
		return
	}

	// 更新服务器状态为增量部署中
	config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
		Status:   "deploying",
		Message:  "正在增量部署到 " + server.Name,
		Progress: 0,
		CanPause: true,
		CanStop:  true,
	})

	// 广播增量部署开始消息
	utils.BroadcastMultiServerDeployProgress(serverID, server.Name, "开始增量部署到 "+server.Name, 0, 100, 0, "")

	// 启动增量部署（异步）
	go func() {
		publicDir := config.GetPublicDir()

		// 检查public目录
		if _, err := os.Stat(publicDir); os.IsNotExist(err) {
			config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
				Status:  "failed",
				Message: "public目录不存在，请先运行Hugo构建",
			})
			return
		}

		// 转换为SSH配置格式
		sshConfig := config.SSHConfig{
			Host:       server.Host,
			Port:       server.Port,
			Username:   server.Username,
			Password:   server.Password,
			KeyPath:    server.KeyPath,
			RemotePath: server.RemotePath,
		}

		// 执行增量部署
		result, err := utils.ExecuteDeployment(sshConfig, publicDir, server.RemotePath, true)

		if err != nil || !result.Success {
			config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
				Status:  "failed",
				Message: "增量部署失败: " + result.Message,
			})
			utils.BroadcastMultiServerError(serverID, server.Name, "deploy", "增量部署失败: "+result.Message)
			return
		}

		// 更新成功状态
		config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
			Status:           "success",
			Message:          fmt.Sprintf("增量部署完成，传输了 %d 个文件", result.FilesDeployed),
			Progress:         100,
			FilesDeployed:    result.FilesDeployed,
			BytesTransferred: result.BytesTransferred,
		})

		// 广播增量部署完成消息
		utils.BroadcastMultiServerComplete(serverID, server.Name, "deploy", fmt.Sprintf("增量部署完成，传输了 %d 个文件", result.FilesDeployed), result.FilesDeployed)

		// 更新服务器的最后部署时间
		now := time.Now()
		server.LastDeployment = &now
		config.UpdateServerConfig(serverID, server)
	}()

	c.JSON(200, gin.H{
		"message": "开始增量部署到 " + server.Name,
	})
}

// 增量构建并部署到指定服务器
func IncrementalBuildAndDeployToMultiServer(c *gin.Context) {
	serverID := c.Param("server_id")

	server, err := config.GetServerConfig(serverID)
	if err != nil {
		c.JSON(404, gin.H{"error": "服务器不存在"})
		return
	}

	if !server.Enabled {
		c.JSON(400, gin.H{"error": "服务器已禁用"})
		return
	}

	// 启动增量构建和部署（异步）
	go func() {
		// 1. 构建阶段
		config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
			Status:   "building",
			Message:  "正在构建Hugo站点...",
			Progress: 0,
		})

		// 广播增量构建开始消息
		utils.BroadcastMultiServerBuildProgress(serverID, server.Name, "开始增量构建Hugo站点...", 0)

		projectPath := config.GetHugoProjectPath()
		buildCmd := exec.Command("hugo", "--source", projectPath)
		_, err := buildCmd.CombinedOutput()

		if err != nil {
			config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
				Status:  "failed",
				Message: "Hugo构建失败: " + err.Error(),
			})
			utils.BroadcastMultiServerError(serverID, server.Name, "build", "Hugo构建失败: "+err.Error())
			return
		}

		// 广播构建完成消息
		utils.BroadcastMultiServerBuildProgress(serverID, server.Name, "Hugo构建完成", 100)

		// 2. 增量部署阶段
		config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
			Status:   "deploying",
			Message:  "正在增量部署到 " + server.Name,
			Progress: 50,
		})

		// 广播部署开始消息
		utils.BroadcastMultiServerDeployProgress(serverID, server.Name, "开始增量部署到 "+server.Name, 50, 100, 50, "")

		publicDir := config.GetPublicDir()

		// 转换为SSH配置格式
		sshConfig := config.SSHConfig{
			Host:       server.Host,
			Port:       server.Port,
			Username:   server.Username,
			Password:   server.Password,
			KeyPath:    server.KeyPath,
			RemotePath: server.RemotePath,
		}

		// 执行增量部署
		result, err := utils.ExecuteDeployment(sshConfig, publicDir, server.RemotePath, true)

		if err != nil || !result.Success {
			config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
				Status:  "failed",
				Message: "增量部署失败: " + result.Message,
			})
			utils.BroadcastMultiServerError(serverID, server.Name, "deploy", "增量部署失败: "+result.Message)
			return
		}

		// 更新成功状态
		config.UpdateServerDeploymentStatus(serverID, config.ServerDeploymentStatus{
			Status:           "success",
			Message:          fmt.Sprintf("增量构建和部署完成，传输了 %d 个文件", result.FilesDeployed),
			Progress:         100,
			FilesDeployed:    result.FilesDeployed,
			BytesTransferred: result.BytesTransferred,
		})

		// 广播增量构建和部署完成消息
		utils.BroadcastMultiServerComplete(serverID, server.Name, "deploy", fmt.Sprintf("增量构建和部署完成，传输了 %d 个文件", result.FilesDeployed), result.FilesDeployed)

		// 更新服务器的最后部署时间
		now := time.Now()
		server.LastDeployment = &now
		config.UpdateServerConfig(serverID, server)
	}()

	c.JSON(200, gin.H{
		"message": "开始增量构建并部署到 " + server.Name,
	})
}

// 获取所有服务器状态
func GetMultiServerStatuses(c *gin.Context) {
	statuses := config.GetAllServerStatuses()
	c.JSON(200, gin.H{
		"statuses": statuses,
	})
}
