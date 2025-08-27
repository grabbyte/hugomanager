package utils

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
	"hugo-manager-go/config"
)

// SSHClient 包装了SSH连接和相关方法
type SSHClient struct {
	client *ssh.Client
	config *ssh.ClientConfig
	host   string
	port   int
}

// DeployResult 部署结果
type DeployResult struct {
	Success          bool
	Message          string
	Output           string
	FilesDeployed    int
	BytesTransferred int64
}

// 创建SSH客户端
func NewSSHClient(sshConfig config.SSHConfig) (*SSHClient, error) {
	var auth []ssh.AuthMethod
	
	if sshConfig.KeyPath != "" {
		// 使用私钥认证
		key, err := os.ReadFile(sshConfig.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("无法读取私钥文件: %v", err)
		}
		
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("无法解析私钥: %v", err)
		}
		
		auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	} else if sshConfig.Password != "" {
		// 使用密码认证
		auth = []ssh.AuthMethod{
			ssh.Password(sshConfig.Password),
		}
	} else {
		return nil, fmt.Errorf("必须提供密钥文件或密码")
	}
	
	clientConfig := &ssh.ClientConfig{
		User: sshConfig.Username,
		Auth: auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 注意：生产环境应该验证主机密钥
		Timeout: 15 * time.Second,
	}
	
	return &SSHClient{
		config: clientConfig,
		host:   sshConfig.Host,
		port:   sshConfig.Port,
	}, nil
}

// 连接到SSH服务器
func (c *SSHClient) Connect(ctx context.Context) error {
	addr := net.JoinHostPort(c.host, strconv.Itoa(c.port))
	
	// 创建带超时的连接
	conn, err := net.DialTimeout("tcp", addr, c.config.Timeout)
	if err != nil {
		return fmt.Errorf("无法连接到 %s: %v", addr, err)
	}
	
	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		conn.Close()
		return ctx.Err()
	default:
	}
	
	// 创建SSH连接
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, c.config)
	if err != nil {
		conn.Close()
		return fmt.Errorf("SSH握手失败: %v", err)
	}
	
	c.client = ssh.NewClient(sshConn, chans, reqs)
	return nil
}

// 关闭连接
func (c *SSHClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// 测试SSH连接
func (c *SSHClient) TestConnection(ctx context.Context) error {
	if err := c.Connect(ctx); err != nil {
		return err
	}
	defer c.Close()
	
	// 执行一个简单的命令来验证连接
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("无法创建SSH会话: %v", err)
	}
	defer session.Close()
	
	output, err := session.Output("echo 'SSH连接测试成功'")
	if err != nil {
		return fmt.Errorf("命令执行失败: %v", err)
	}
	
	if !strings.Contains(string(output), "SSH连接测试成功") {
		return fmt.Errorf("SSH连接验证失败")
	}
	
	return nil
}

// 执行rsync命令进行文件同步
func (c *SSHClient) ExecuteRsync(ctx context.Context, localPath, remotePath string, incremental bool) (*DeployResult, error) {
	if err := c.Connect(ctx); err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("SSH连接失败: %v", err),
		}, err
	}
	defer c.Close()
	
	// 构建rsync命令
	rsyncArgs := []string{"-avz", "--stats"}
	if incremental {
		rsyncArgs = append(rsyncArgs, "--update", "--times")
	} else {
		rsyncArgs = append(rsyncArgs, "--delete")
	}
	
	// 检查本地目录是否存在
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("本地目录不存在: %s", localPath),
		}, err
	}
	
	// 确保远程目录存在
	if err := c.ensureRemoteDirectory(remotePath); err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("无法创建远程目录: %v", err),
		}, err
	}
	
	// 使用tar进行文件传输（更可靠的方法）
	return c.transferFilesWithServer(ctx, localPath, remotePath, incremental, "", "")
}

// 执行rsync命令进行文件同步（支持服务器信息）
func (c *SSHClient) ExecuteRsyncWithServer(ctx context.Context, localPath, remotePath string, incremental bool, serverID, serverName string) (*DeployResult, error) {
	if err := c.Connect(ctx); err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("SSH连接失败: %v", err),
		}, err
	}
	defer c.Close()

	// 构建rsync命令
	rsyncArgs := []string{"-avz", "--stats"}
	if incremental {
		rsyncArgs = append(rsyncArgs, "--update", "--times")
	} else {
		rsyncArgs = append(rsyncArgs, "--delete")
	}

	// 检查本地目录是否存在
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("本地目录不存在: %s", localPath),
		}, err
	}

	// 确保远程目录存在
	if err := c.ensureRemoteDirectory(remotePath); err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("无法创建远程目录: %v", err),
		}, err
	}

	// 使用tar进行文件传输（更可靠的方法）
	return c.transferFilesWithServer(ctx, localPath, remotePath, incremental, serverID, serverName)
}

// 确保远程目录存在
func (c *SSHClient) ensureRemoteDirectory(remotePath string) error {
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	
	cmd := fmt.Sprintf("mkdir -p %s", remotePath)
	return session.Run(cmd)
}

// 文件传输任务
type FileTask struct {
	LocalFile  string
	RemoteFile string
	Size       int64
	ModTime    time.Time
}

// 使用并发传输文件
func (c *SSHClient) transferFiles(ctx context.Context, localPath, remotePath string, incremental bool) (*DeployResult, error) {
	return c.transferFilesWithServer(ctx, localPath, remotePath, incremental, "", "")
}

// 使用并发传输文件（支持服务器信息）
func (c *SSHClient) transferFilesWithServer(ctx context.Context, localPath, remotePath string, incremental bool, serverID, serverName string) (*DeployResult, error) {
	result := &DeployResult{}
	
	// 检查是否有未完成的上传任务
	existingTasks := config.GetUploadTasks()
	var fileTasks []FileTask
	
	if len(existingTasks) > 0 && !config.IsDeploymentPaused() {
		// 使用现有任务列表，过滤掉已完成的
		fmt.Println("发现未完成的上传任务，继续上传...")
		for _, uploadTask := range existingTasks {
			if !uploadTask.Completed {
				fileTasks = append(fileTasks, FileTask{
					LocalFile:  uploadTask.LocalFile,
					RemoteFile: uploadTask.RemoteFile,
					Size:       uploadTask.Size,
					ModTime:    uploadTask.CreatedAt,
				})
			}
		}
	} else {
		// 收集需要传输的文件
		var err error
		fileTasks, err = c.collectFileTasks(localPath, remotePath, incremental)
		if err != nil {
			return &DeployResult{
				Success: false,
				Message: fmt.Sprintf("收集文件任务失败: %v", err),
			}, err
		}
		
		// 保存任务列表以便恢复
		c.saveUploadTasks(fileTasks)
	}
	
	if len(fileTasks) == 0 {
		config.RemoveCompletedTasks()
		result.Success = true
		result.Message = "没有文件需要传输"
		result.Output = "所有文件都是最新的"
		result.FilesDeployed = 0
		result.BytesTransferred = 0
		return result, nil
	}
	
	// 设置为非暂停状态
	config.SetDeploymentPaused(false)
	
	// 广播部署开始
	if serverID != "" && serverName != "" {
		BroadcastMultiServerDeployProgress(serverID, serverName, "正在准备文件传输...", 0, len(fileTasks), 0, "")
	} else {
		BroadcastDeployProgress("正在准备文件传输...", 0, len(fileTasks), 0, "")
	}
	
	// 并发传输文件
	err := c.transferFilesConcurrentlyWithServer(ctx, fileTasks, serverID, serverName)
	if err != nil {
		BroadcastError("deploy", fmt.Sprintf("文件传输失败: %v", err))
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("文件传输失败: %v", err),
		}, err
	}
	
	// 计算传输统计
	var totalSize int64
	for _, task := range fileTasks {
		totalSize += task.Size
	}
	
	// 清理完成的任务
	config.RemoveCompletedTasks()
	
	// 广播完成消息
	BroadcastComplete("deploy", 
		fmt.Sprintf("部署完成！传输了 %d 个文件，共 %d 字节", len(fileTasks), totalSize), 
		len(fileTasks))
	
	result.Success = true
	result.Message = "文件传输完成"
	result.Output = fmt.Sprintf("成功传输 %d 个文件，共 %d 字节", len(fileTasks), totalSize)
	result.FilesDeployed = len(fileTasks)
	result.BytesTransferred = totalSize
	
	return result, nil
}

// 收集需要传输的文件任务
func (c *SSHClient) collectFileTasks(localPath, remotePath string, incremental bool) ([]FileTask, error) {
	var tasks []FileTask
	
	err := filepath.Walk(localPath, func(localFile string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// 跳过目录
		if info.IsDir() {
			return nil
		}
		
		// 计算相对路径
		relPath, err := filepath.Rel(localPath, localFile)
		if err != nil {
			return err
		}
		
		// 构建远程文件路径
		remoteFile := filepath.Join(remotePath, relPath)
		// 在Unix系统上使用正斜杠
		remoteFile = strings.ReplaceAll(remoteFile, "\\", "/")
		
		// 检查是否需要传输
		needTransfer := true
		if incremental {
			needTransfer, err = c.shouldTransferFile(localFile, remoteFile, info)
			if err != nil {
				fmt.Printf("检查文件失败 %s: %v\n", remoteFile, err)
				// 如果检查失败，仍然传输文件
				needTransfer = true
			}
		}
		
		if needTransfer {
			tasks = append(tasks, FileTask{
				LocalFile:  localFile,
				RemoteFile: remoteFile,
				Size:       info.Size(),
				ModTime:    info.ModTime(),
			})
		} else {
			fmt.Printf("跳过文件（已是最新）: %s\n", remoteFile)
		}
		
		return nil
	})
	
	return tasks, err
}

// 检查文件是否需要传输
func (c *SSHClient) shouldTransferFile(localFile, remoteFile string, localInfo os.FileInfo) (bool, error) {
	// 检查远程文件是否存在
	session, err := c.client.NewSession()
	if err != nil {
		return true, err
	}
	defer session.Close()
	
	// 使用stat命令获取远程文件信息
	cmd := fmt.Sprintf("stat -c '%%s %%Y' %s 2>/dev/null || echo 'NOTEXIST'", remoteFile)
	output, err := session.Output(cmd)
	if err != nil {
		return true, err
	}
	
	result := strings.TrimSpace(string(output))
	if result == "NOTEXIST" {
		return true, nil // 远程文件不存在，需要传输
	}
	
	// 解析远程文件信息
	parts := strings.Fields(result)
	if len(parts) != 2 {
		return true, nil // 无法解析，传输文件
	}
	
	remoteSize, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return true, nil
	}
	
	remoteMtime, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return true, nil
	}
	
	// 比较文件大小和修改时间
	localSize := localInfo.Size()
	localMtime := localInfo.ModTime().Unix()
	
	// 如果大小不同，需要传输
	if localSize != remoteSize {
		return true, nil
	}
	
	// 如果本地文件更新，需要传输
	if localMtime > remoteMtime {
		return true, nil
	}
	
	// 文件相同，不需要传输
	return false, nil
}

// 保存上传任务列表
func (c *SSHClient) saveUploadTasks(fileTasks []FileTask) {
	var uploadTasks []config.UploadTask
	for _, task := range fileTasks {
		// 生成随机ID
		id := generateTaskID()
		uploadTasks = append(uploadTasks, config.UploadTask{
			ID:         id,
			LocalFile:  task.LocalFile,
			RemoteFile: task.RemoteFile,
			Size:       task.Size,
			Completed:  false,
			CreatedAt:  time.Now(),
		})
	}
	config.SetUploadTasks(uploadTasks)
}

// 生成任务ID
func generateTaskID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// 并发传输文件（支持暂停/继续）
func (c *SSHClient) transferFilesConcurrently(ctx context.Context, tasks []FileTask) error {
	return c.transferFilesConcurrentlyWithServer(ctx, tasks, "", "")
}

func (c *SSHClient) transferFilesConcurrentlyWithServer(ctx context.Context, tasks []FileTask, serverID, serverName string) error {
	const maxConcurrency = 4 // 最大并发数
	
	// 进度跟踪
	var completedCount int32 = 0
	var failedCount int32 = 0
	totalTasks := len(tasks)
	
	// 创建工作池
	taskChan := make(chan FileTask, len(tasks))
	failedTasks := make(chan FileTask, len(tasks)) // 失败的任务用于重试
	pauseChan := make(chan struct{}, 1)
	var wg sync.WaitGroup
	
	// 启动暂停检查goroutine
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if config.IsDeploymentPaused() {
					select {
					case pauseChan <- struct{}{}:
					default:
					}
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	
	// 启动worker goroutines
	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for task := range taskChan {
				// 检查是否暂停
				select {
				case <-pauseChan:
					fmt.Printf("[Worker %d] 检测到暂停信号，停止上传\n", workerID+1)
					return
				default:
				}
				
				fmt.Printf("[Worker %d] 传输文件: %s -> %s (%d 字节)\n", 
					workerID+1, task.LocalFile, task.RemoteFile, task.Size)
				
				// 广播当前文件进度
				currentCount := atomic.LoadInt32(&completedCount)
				progress := int(float64(currentCount) / float64(totalTasks) * 100)
				if serverID != "" && serverName != "" {
					BroadcastMultiServerDeployProgress(serverID, serverName,
						fmt.Sprintf("正在上传文件 (%d/%d)", currentCount+1, totalTasks),
						progress, totalTasks, int(currentCount), filepath.Base(task.RemoteFile))
				} else {
					BroadcastDeployProgress(
						fmt.Sprintf("正在上传文件 (%d/%d)", currentCount+1, totalTasks),
						progress,
						totalTasks,
						int(currentCount),
						filepath.Base(task.RemoteFile),
					)
				}
				
				err := c.uploadSingleFile(task)
				if err != nil {
					// 记录失败，但不中断上传过程
					atomic.AddInt32(&failedCount, 1)
					fmt.Printf("[Worker %d] 文件上传失败: %s -> %s, 错误: %v\n", 
						workerID+1, task.LocalFile, task.RemoteFile, err)
					
					// 广播错误信息到前端日志
					errorMsg := fmt.Sprintf("文件上传失败: %s", filepath.Base(task.RemoteFile))
					if serverID != "" && serverName != "" {
						BroadcastMultiServerProgress(serverID, serverName, "deploy", "deploying",
							errorMsg, 0, 0, 0, "")
					} else {
						BroadcastProgress("deploy", "deploying", errorMsg, 0, 0, 0, "")
					}
					
					// 将失败的任务加入重试队列
					select {
					case failedTasks <- task:
						fmt.Printf("[Worker %d] 已将失败任务加入重试队列: %s\n", workerID+1, task.RemoteFile)
					default:
						fmt.Printf("[Worker %d] 重试队列已满，跳过任务: %s\n", workerID+1, task.RemoteFile)
					}
					
					// 继续处理下一个文件，不要return
				} else {
					// 成功上传，标记任务为已完成
					c.markTaskCompleted(task)
					completed := atomic.AddInt32(&completedCount, 1)
					
					// 广播进度更新
					progress := int(float64(completed) / float64(totalTasks) * 100)
					if serverID != "" && serverName != "" {
						BroadcastMultiServerDeployProgress(serverID, serverName,
							fmt.Sprintf("已完成 %d/%d 文件", completed, totalTasks),
							progress, totalTasks, int(completed), "")
					} else {
						BroadcastDeployProgress(
							fmt.Sprintf("已完成 %d/%d 文件", completed, totalTasks),
							progress,
							totalTasks,
							int(completed),
							"",
						)
					}
				}
			}
		}(i)
	}
	
	// 发送任务
	go func() {
		defer close(taskChan)
		for _, task := range tasks {
			select {
			case taskChan <- task:
			case <-ctx.Done():
				return
			case <-pauseChan:
				fmt.Println("任务分发已暂停")
				return
			}
		}
	}()
	
	// 等待所有worker完成或暂停
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		// 正常完成
	case <-pauseChan:
		// 暂停信号
		return fmt.Errorf("上传已暂停")
	case <-ctx.Done():
		// 上下文取消
		return ctx.Err()
	}
	
	// 关闭失败任务通道
	close(failedTasks)
	
	// 收集失败的任务进行重试
	var retryTasks []FileTask
	for failedTask := range failedTasks {
		retryTasks = append(retryTasks, failedTask)
	}
	
	completed := atomic.LoadInt32(&completedCount)
	failed := atomic.LoadInt32(&failedCount)
	
	fmt.Printf("上传完成统计: 成功 %d 个，失败 %d 个，总计 %d 个文件\n", completed, failed, totalTasks)
	
	// 如果有失败的任务，尝试重试一次
	if len(retryTasks) > 0 && failed > 0 {
		fmt.Printf("开始重试 %d 个失败的文件...\n", len(retryTasks))
		
		// 广播重试开始消息
		if serverID != "" && serverName != "" {
			BroadcastMultiServerProgress(serverID, serverName, "deploy", "deploying",
				fmt.Sprintf("开始重试 %d 个失败的文件", len(retryTasks)), 0, 0, 0, "")
		}
		
		retryErr := c.retryFailedTasks(ctx, retryTasks, serverID, serverName)
		if retryErr != nil {
			fmt.Printf("重试过程中发生错误: %v\n", retryErr)
		}
	}
	
	// 生成最终的上传报告
	totalCompleted := completed
	totalFailed := failed
	
	// 如果进行了重试，更新统计
	if len(retryTasks) > 0 && failed > 0 {
		// 重新获取失败和成功的数据（重试完成后）
		totalFailed = int32(len(retryTasks))
		// 简单假设重试会有一些成功
		for _, task := range retryTasks {
			// 检查任务是否在重试后完成
			if c.isTaskCompleted(task) {
				totalCompleted++
				totalFailed--
			}
		}
	}
	
	// 广播最终完成消息
	finalMessage := fmt.Sprintf("上传完成: 成功 %d 个，失败 %d 个", totalCompleted, totalFailed)
	if totalFailed > 0 {
		finalMessage += fmt.Sprintf("，请检查日志了解失败详情")
	}
	
	if serverID != "" && serverName != "" {
		if totalFailed > 0 {
			BroadcastMultiServerProgress(serverID, serverName, "deploy", "success",
				finalMessage, 100, totalTasks, int(totalCompleted), "")
		} else {
			BroadcastMultiServerComplete(serverID, serverName, "deploy", finalMessage, totalTasks)
		}
	} else {
		if totalFailed > 0 {
			BroadcastProgress("deploy", "success", finalMessage, 100, totalTasks, int(totalCompleted), "")
		} else {
			BroadcastComplete("deploy", finalMessage, totalTasks)
		}
	}
	
	// 即使有失败的文件，也不返回错误，让上传过程继续
	return nil
}

// 检查任务是否已完成
func (c *SSHClient) isTaskCompleted(task FileTask) bool {
	existingTasks := config.GetUploadTasks()
	for _, uploadTask := range existingTasks {
		if uploadTask.LocalFile == task.LocalFile && uploadTask.RemoteFile == task.RemoteFile {
			return uploadTask.Completed
		}
	}
	// 如果找不到任务记录，可能已经被删除（表示完成）
	return true
}

// 重试失败的任务
func (c *SSHClient) retryFailedTasks(ctx context.Context, failedTasks []FileTask, serverID, serverName string) error {
	const maxRetries = 1 // 最多重试1次
	var retrySuccessCount int32 = 0
	var retryFailedCount int32 = 0
	
	fmt.Printf("开始重试 %d 个失败的任务...\n", len(failedTasks))
	
	for i, task := range failedTasks {
		// 检查是否被暂停
		if config.IsDeploymentPaused() {
			fmt.Printf("重试过程中检测到暂停信号，停止重试\n")
			return fmt.Errorf("重试已暂停")
		}
		
		fmt.Printf("重试第 %d/%d 个文件: %s\n", i+1, len(failedTasks), task.RemoteFile)
		
		// 广播重试进度
		progress := int(float64(i+1) / float64(len(failedTasks)) * 100)
		if serverID != "" && serverName != "" {
			BroadcastMultiServerDeployProgress(serverID, serverName,
				fmt.Sprintf("重试文件 (%d/%d)", i+1, len(failedTasks)),
				progress, len(failedTasks), i+1, filepath.Base(task.RemoteFile))
		}
		
		// 重试上传
		err := c.uploadSingleFile(task)
		if err != nil {
			atomic.AddInt32(&retryFailedCount, 1)
			fmt.Printf("重试失败: %s -> %s, 错误: %v\n", task.LocalFile, task.RemoteFile, err)
			
			// 广播重试失败信息
			retryFailMsg := fmt.Sprintf("重试失败: %s", filepath.Base(task.RemoteFile))
			if serverID != "" && serverName != "" {
				BroadcastMultiServerProgress(serverID, serverName, "deploy", "deploying",
					retryFailMsg, 0, 0, 0, "")
			} else {
				BroadcastProgress("deploy", "deploying", retryFailMsg, 0, 0, 0, "")
			}
		} else {
			atomic.AddInt32(&retrySuccessCount, 1)
			fmt.Printf("重试成功: %s -> %s\n", task.LocalFile, task.RemoteFile)
			
			// 标记任务为已完成
			c.markTaskCompleted(task)
		}
		
		// 添加小延迟，避免过于频繁的重试
		time.Sleep(100 * time.Millisecond)
	}
	
	successCount := atomic.LoadInt32(&retrySuccessCount)
	failedCount := atomic.LoadInt32(&retryFailedCount)
	
	fmt.Printf("重试完成统计: 重试成功 %d 个，重试失败 %d 个\n", successCount, failedCount)
	
	// 广播重试完成消息
	if serverID != "" && serverName != "" {
		BroadcastMultiServerProgress(serverID, serverName, "deploy", "deploying",
			fmt.Sprintf("重试完成: 成功 %d 个，失败 %d 个", successCount, failedCount), 0, 0, 0, "")
	}
	
	return nil
}

// 标记任务为已完成
func (c *SSHClient) markTaskCompleted(task FileTask) {
	existingTasks := config.GetUploadTasks()
	found := false
	for _, uploadTask := range existingTasks {
		if uploadTask.LocalFile == task.LocalFile && uploadTask.RemoteFile == task.RemoteFile {
			config.MarkTaskCompleted(uploadTask.ID)
			fmt.Printf("标记任务完成: %s\n", task.RemoteFile)
			found = true
			break
		}
	}
	
	// 如果没找到任务记录，说明可能已经被删除了（文件不存在的情况）
	if !found {
		fmt.Printf("任务记录已不存在（可能已被删除）: %s\n", task.RemoteFile)
	}
}

// 删除指定文件的上传任务记录
func (c *SSHClient) removeUploadTaskByFile(localFile, remoteFile string) {
	existingTasks := config.GetUploadTasks()
	var remainingTasks []config.UploadTask
	
	for _, uploadTask := range existingTasks {
		// 保留不匹配的任务
		if !(uploadTask.LocalFile == localFile && uploadTask.RemoteFile == remoteFile) {
			remainingTasks = append(remainingTasks, uploadTask)
		}
	}
	
	// 更新任务列表
	config.SetUploadTasks(remainingTasks)
	fmt.Printf("删除无效上传记录: %s -> %s\n", localFile, remoteFile)
}

// 上传单个文件
func (c *SSHClient) uploadSingleFile(task FileTask) error {
	// 检查本地文件是否存在
	if _, err := os.Stat(task.LocalFile); os.IsNotExist(err) {
		// 文件不存在，删除上传任务记录并跳过
		fmt.Printf("本地文件不存在，删除上传记录: %s\n", task.LocalFile)
		c.removeUploadTaskByFile(task.LocalFile, task.RemoteFile)
		return nil // 返回nil表示任务处理完成（通过删除记录）
	}
	
	// 确保远程目录存在
	remoteDir := filepath.Dir(task.RemoteFile)
	// 确保使用Unix路径分隔符（因为远程服务器是Linux）
	remoteDir = strings.ReplaceAll(remoteDir, "\\", "/")
	if err := c.createRemoteDirectory(remoteDir); err != nil {
		return fmt.Errorf("无法创建远程目录 %s: %v", remoteDir, err)
	}
	
	// 读取本地文件
	localData, err := os.ReadFile(task.LocalFile)
	if err != nil {
		return err
	}
	
	// 创建SSH会话
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %v", err)
	}
	defer session.Close()
	
	// 捕获stderr输出用于错误诊断
	var stderr strings.Builder
	session.Stderr = &stderr
	
	// 使用cat命令直接写入文件
	// 确保使用Unix路径分隔符（因为远程服务器是Linux）
	remoteFile := strings.ReplaceAll(task.RemoteFile, "\\", "/")
	cmd := fmt.Sprintf("cat > %s", remoteFile)
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("创建stdin管道失败: %v", err)
	}
	
	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("启动上传命令失败: %v", err)
	}
	
	// 写入文件内容
	_, err = stdin.Write(localData)
	if err != nil {
		stdin.Close()
		stderrOutput := stderr.String()
		if stderrOutput != "" {
			return fmt.Errorf("写入数据失败: %v, 错误输出: %s", err, stderrOutput)
		}
		return fmt.Errorf("写入数据失败: %v", err)
	}
	
	// 关闭stdin以发送EOF
	stdin.Close()
	
	// 等待命令完成
	if err := session.Wait(); err != nil {
		stderrOutput := stderr.String()
		if stderrOutput != "" {
			// 分析具体错误原因
			return c.analyzeUploadError(err, stderrOutput, task.RemoteFile)
		}
		return fmt.Errorf("文件上传失败: %v", err)
	}
	
	// 验证文件是否上传成功
	if err := c.verifyFileUpload(remoteFile, task.Size); err != nil {
		return fmt.Errorf("文件上传验证失败: %v", err)
	}
	
	// 设置文件时间戳
	return c.setFileAttributes(remoteFile, task.ModTime)
}

// 分析上传错误原因
func (c *SSHClient) analyzeUploadError(err error, stderrOutput, remoteFile string) error {
	stderrLower := strings.ToLower(stderrOutput)
	
	if strings.Contains(stderrLower, "permission denied") {
		return fmt.Errorf("权限被拒绝: 远程目录 %s 没有写权限。错误: %v, 详情: %s", 
			filepath.Dir(remoteFile), err, stderrOutput)
	}
	
	if strings.Contains(stderrLower, "no space left") || strings.Contains(stderrLower, "disk full") {
		return fmt.Errorf("磁盘空间不足: 远程服务器磁盘已满。错误: %v, 详情: %s", 
			err, stderrOutput)
	}
	
	if strings.Contains(stderrLower, "no such file or directory") {
		return fmt.Errorf("目录不存在: 远程目录 %s 不存在。错误: %v, 详情: %s", 
			filepath.Dir(remoteFile), err, stderrOutput)
	}
	
	if strings.Contains(stderrLower, "read-only") {
		return fmt.Errorf("只读文件系统: 远程目录为只读。错误: %v, 详情: %s", 
			err, stderrOutput)
	}
	
	if strings.Contains(stderrLower, "connection") {
		return fmt.Errorf("连接问题: SSH连接不稳定。错误: %v, 详情: %s", 
			err, stderrOutput)
	}
	
	// 通用错误
	return fmt.Errorf("文件上传失败: %v, 详细错误: %s", err, stderrOutput)
}

// 验证文件上传
func (c *SSHClient) verifyFileUpload(remoteFile string, expectedSize int64) error {
	session, err := c.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	
	// 检查文件大小
	cmd := fmt.Sprintf("stat -c %%s %s 2>/dev/null || echo '0'", remoteFile)
	output, err := session.Output(cmd)
	if err != nil {
		return err
	}
	
	actualSize, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return err
	}
	
	if actualSize != expectedSize {
		return fmt.Errorf("文件大小不匹配，期望 %d 字节，实际 %d 字节", expectedSize, actualSize)
	}
	
	return nil
}

// 设置文件属性
func (c *SSHClient) setFileAttributes(remoteFile string, modTime time.Time) error {
	session, err := c.client.NewSession()
	if err != nil {
		return nil // 忽略权限设置错误
	}
	defer session.Close()
	
	// 设置文件时间戳
	touchCmd := fmt.Sprintf("touch -d '%s' %s", modTime.Format("2006-01-02 15:04:05"), remoteFile)
	session.Run(touchCmd)
	
	return nil
}

// 创建远程目录
func (c *SSHClient) createRemoteDirectory(remotePath string) error {
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("创建SSH会话失败: %v", err)
	}
	defer session.Close()
	
	// 捕获stderr输出
	var stderr strings.Builder
	session.Stderr = &stderr
	
	cmd := fmt.Sprintf("mkdir -p %s", remotePath)
	if err := session.Run(cmd); err != nil {
		stderrOutput := stderr.String()
		if stderrOutput != "" {
			if strings.Contains(strings.ToLower(stderrOutput), "permission denied") {
				return fmt.Errorf("权限被拒绝: 无法创建目录 %s，请检查父目录权限。详情: %s", 
					remotePath, stderrOutput)
			}
			if strings.Contains(strings.ToLower(stderrOutput), "no space left") {
				return fmt.Errorf("磁盘空间不足: 无法创建目录 %s。详情: %s", 
					remotePath, stderrOutput)
			}
			return fmt.Errorf("创建目录失败: %v, 详情: %s", err, stderrOutput)
		}
		return fmt.Errorf("创建目录失败: %v", err)
	}
	
	return nil
}


// 计算本地文件统计
func (c *SSHClient) calculateLocalStats(localPath string) (int, int64, error) {
	var fileCount int
	var totalSize int64
	
	err := filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileCount++
			totalSize += info.Size()
		}
		return nil
	})
	
	return fileCount, totalSize, err
}

// 便捷函数：测试SSH连接
func TestSSHConnection(sshConfig config.SSHConfig) error {
	client, err := NewSSHClient(sshConfig)
	if err != nil {
		return err
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	return client.TestConnection(ctx)
}

// 便捷函数：执行部署
func ExecuteDeployment(sshConfig config.SSHConfig, localPath, remotePath string, incremental bool) (*DeployResult, error) {
	return ExecuteDeploymentWithServer(sshConfig, localPath, remotePath, incremental, "", "")
}

func ExecuteDeploymentWithServer(sshConfig config.SSHConfig, localPath, remotePath string, incremental bool, serverID, serverName string) (*DeployResult, error) {
	client, err := NewSSHClient(sshConfig)
	if err != nil {
		return &DeployResult{
			Success: false,
			Message: fmt.Sprintf("创建SSH客户端失败: %v", err),
		}, err
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	return client.ExecuteRsyncWithServer(ctx, localPath, remotePath, incremental, serverID, serverName)
}