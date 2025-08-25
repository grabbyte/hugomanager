package config

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/json"
    "errors"
    "io"
    "os"
    "path/filepath"
    "time"
)

type SSHConfig struct {
    Host              string `json:"host"`
    Port              int    `json:"port"`
    Username          string `json:"username,omitempty"`           // 运行时明文用户名
    EncryptedUsername string `json:"encrypted_username,omitempty"` // 存储时加密用户名
    Password          string `json:"password,omitempty"`           // 运行时明文密码
    EncryptedPassword string `json:"encrypted_password,omitempty"` // 存储时加密密码
    KeyPath           string `json:"key_path,omitempty"`
    RemotePath        string `json:"remote_path"`
}

type UploadTask struct {
    ID               string    `json:"id"`
    LocalFile        string    `json:"local_file"`
    RemoteFile       string    `json:"remote_file"`
    Size             int64     `json:"size"`
    Completed        bool      `json:"completed"`
    CreatedAt        time.Time `json:"created_at"`
}

type ProgressInfo struct {
    Type        string    `json:"type"`         // "build", "deploy", "complete"
    Status      string    `json:"status"`       // "building", "deploying", "success", "failed", "paused"
    Message     string    `json:"message"`      // 当前状态描述
    Progress    int       `json:"progress"`     // 进度百分比 0-100
    Total       int       `json:"total"`        // 总任务数
    Current     int       `json:"current"`      // 当前完成数
    CurrentFile string    `json:"current_file"` // 当前处理的文件
    Speed       string    `json:"speed"`        // 传输速度
    ETA         string    `json:"eta"`          // 预计剩余时间
    StartTime   *time.Time `json:"start_time,omitempty"` // 开始时间
    UpdateTime  time.Time `json:"update_time"`  // 最后更新时间
}

type DeploymentInfo struct {
    LastSyncTime     *time.Time    `json:"last_sync_time,omitempty"`
    LastSyncStatus   string        `json:"last_sync_status,omitempty"`   // "success", "failed", "building", "deploying", "paused"
    LastSyncMessage  string        `json:"last_sync_message,omitempty"`
    FilesDeployed    int           `json:"files_deployed,omitempty"`
    BytesTransferred int64         `json:"bytes_transferred,omitempty"`
    UploadTasks      []UploadTask  `json:"upload_tasks,omitempty"`       // 待上传任务队列
    IsPaused         bool          `json:"is_paused,omitempty"`          // 是否暂停
    Progress         ProgressInfo  `json:"progress,omitempty"`           // 实时进度信息
}

type Config struct {
    HugoProjectPath string         `json:"hugo_project_path"`
    SSH             SSHConfig      `json:"ssh"`
    Deployment      DeploymentInfo `json:"deployment"`
    Language        string         `json:"language,omitempty"`        // 用户主动设置的语言
    UserSetLanguage bool           `json:"user_set_language,omitempty"` // 标记是否用户主动设置
}

var currentConfig Config
var decryptionKey string // 运行时解密密钥

func init() {
    LoadConfig()
}

// 语言配置相关函数
func GetLanguage() string {
    if currentConfig.Language == "" {
        return "en-US" // 默认英文
    }
    return currentConfig.Language
}

func SetLanguage(language string) {
    currentConfig.Language = language
    currentConfig.UserSetLanguage = true // 标记为用户主动设置
    SaveConfig()
}

// 检查是否用户主动设置了语言
func IsUserSetLanguage() bool {
    return currentConfig.UserSetLanguage
}

// 获取浏览器语言（由前端调用时传入）
func SetBrowserLanguage(language string) {
    // 只有用户没有主动设置语言时，才使用浏览器语言
    if !currentConfig.UserSetLanguage {
        currentConfig.Language = language
        // 不保存到配置文件，保持为自动检测状态
    }
}

func LoadConfig() {
    configPath := "config.json"
    if _, err := os.Stat(configPath); os.IsNotExist(err) {
        // 如果配置文件不存在，使用默认配置
        currentConfig = Config{
            HugoProjectPath: "./test-hugo",
            SSH: SSHConfig{
                Port: 22,
            },
        }
        SaveConfig()
        return
    }

    data, err := os.ReadFile(configPath)
    if err != nil {
        currentConfig = Config{
            HugoProjectPath: "./test-hugo",
            SSH: SSHConfig{
                Port: 22,
            },
        }
        return
    }

    json.Unmarshal(data, &currentConfig)
}

func SaveConfig() {
    data, _ := json.MarshalIndent(currentConfig, "", "  ")
    os.WriteFile("config.json", data, 0644)
}

func GetHugoProjectPath() string {
    return currentConfig.HugoProjectPath
}

func SetHugoProjectPath(path string) {
    currentConfig.HugoProjectPath = path
    SaveConfig()
}

func GetContentDir() string {
    return filepath.Join(currentConfig.HugoProjectPath, "content")
}

func GetStaticDir() string {
    return filepath.Join(currentConfig.HugoProjectPath, "static")
}

func GetImagesDir() string {
    return filepath.Join(currentConfig.HugoProjectPath, "static", "uploads", "images")
}

func GetSSHConfig() SSHConfig {
    return currentConfig.SSH
}

func SetSSHConfig(ssh SSHConfig) {
    currentConfig.SSH = ssh
    SaveConfig()
}

func GetPublicDir() string {
    return filepath.Join(currentConfig.HugoProjectPath, "public")
}

func GetDeploymentInfo() DeploymentInfo {
    return currentConfig.Deployment
}

func SetDeploymentInfo(deployment DeploymentInfo) {
    currentConfig.Deployment = deployment
    SaveConfig()
}

func UpdateDeploymentStatus(status, message string) {
    currentConfig.Deployment.LastSyncStatus = status
    currentConfig.Deployment.LastSyncMessage = message
    if status == "success" || status == "failed" {
        now := time.Now()
        currentConfig.Deployment.LastSyncTime = &now
    }
    SaveConfig()
}

func SetDeploymentStats(filesDeployed int, bytesTransferred int64) {
    currentConfig.Deployment.FilesDeployed = filesDeployed
    currentConfig.Deployment.BytesTransferred = bytesTransferred
    SaveConfig()
}

// 加密函数
func encrypt(plaintext, password string) (string, error) {
    if plaintext == "" {
        return "", nil
    }
    
    // 使用密码生成密钥
    key := sha256.Sum256([]byte(password))
    
    // 创建AES加密器
    block, err := aes.NewCipher(key[:])
    if err != nil {
        return "", err
    }
    
    // 创建GCM模式
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    // 生成随机nonce
    nonce := make([]byte, gcm.NonceSize())
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    
    // 加密
    ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
    
    // 返回base64编码的密文
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// 解密函数
func decrypt(ciphertext, password string) (string, error) {
    if ciphertext == "" {
        return "", nil
    }
    
    // 解码base64
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return "", err
    }
    
    // 使用密码生成密钥
    key := sha256.Sum256([]byte(password))
    
    // 创建AES加密器
    block, err := aes.NewCipher(key[:])
    if err != nil {
        return "", err
    }
    
    // 创建GCM模式
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    // 检查数据长度
    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return "", errors.New("ciphertext too short")
    }
    
    // 分离nonce和密文
    nonce, ciphertext_bytes := data[:nonceSize], data[nonceSize:]
    
    // 解密
    plaintext, err := gcm.Open(nil, nonce, ciphertext_bytes, nil)
    if err != nil {
        return "", err
    }
    
    return string(plaintext), nil
}

// 设置解密密钥
func SetDecryptionKey(key string) error {
    decryptionKey = key
    
    // 尝试解密SSH凭据验证密钥是否正确
    var decryptionErrors []error
    
    // 解密用户名
    if currentConfig.SSH.EncryptedUsername != "" {
        username, err := decrypt(currentConfig.SSH.EncryptedUsername, decryptionKey)
        if err != nil {
            decryptionErrors = append(decryptionErrors, err)
        } else {
            currentConfig.SSH.Username = username
        }
    }
    
    // 解密密码
    if currentConfig.SSH.EncryptedPassword != "" {
        password, err := decrypt(currentConfig.SSH.EncryptedPassword, decryptionKey)
        if err != nil {
            decryptionErrors = append(decryptionErrors, err)
        } else {
            currentConfig.SSH.Password = password
        }
    }
    
    // 如果有任何解密错误，重置密钥
    if len(decryptionErrors) > 0 {
        decryptionKey = ""
        currentConfig.SSH.Username = ""
        currentConfig.SSH.Password = ""
        return errors.New("解密密钥错误")
    }
    
    return nil
}

// 检查是否已设置解密密钥
func IsDecryptionKeySet() bool {
    return decryptionKey != ""
}

// 检查是否有加密的SSH凭据
func HasEncryptedSSHCredentials() bool {
    return currentConfig.SSH.EncryptedUsername != "" || currentConfig.SSH.EncryptedPassword != ""
}

// 检查SSH凭据是否需要解密
func NeedsDecryption() bool {
    return HasEncryptedSSHCredentials() && !IsDecryptionKeySet()
}

// 检查是否有明文的SSH凭据
func HasPlaintextSSHCredentials() bool {
    return (currentConfig.SSH.Username != "" && currentConfig.SSH.EncryptedUsername == "") ||
           (currentConfig.SSH.Password != "" && currentConfig.SSH.EncryptedPassword == "")
}

// 加密现有的明文凭据
func EncryptExistingCredentials(masterPassword string) error {
    modified := false
    ssh := currentConfig.SSH
    
    // 如果有明文用户名且没有加密版本，进行加密
    if ssh.Username != "" && ssh.EncryptedUsername == "" {
        encryptedUsername, err := encrypt(ssh.Username, masterPassword)
        if err != nil {
            return err
        }
        ssh.EncryptedUsername = encryptedUsername
        modified = true
    }
    
    // 如果有明文密码且没有加密版本，进行加密
    if ssh.Password != "" && ssh.EncryptedPassword == "" {
        encryptedPassword, err := encrypt(ssh.Password, masterPassword)
        if err != nil {
            return err
        }
        ssh.EncryptedPassword = encryptedPassword
        modified = true
    }
    
    if modified {
        currentConfig.SSH = ssh
        // 设置解密密钥以便后续使用
        decryptionKey = masterPassword
        // 保存配置（会清除明文凭据）
        return SaveConfigWithEncryption()
    }
    
    return nil
}

// 加密并保存SSH配置
func SetSSHConfigWithEncryption(ssh SSHConfig, masterPassword string) error {
    // 加密用户名
    if ssh.Username != "" {
        encryptedUsername, err := encrypt(ssh.Username, masterPassword)
        if err != nil {
            return err
        }
        ssh.EncryptedUsername = encryptedUsername
        ssh.Username = "" // 清除明文用户名，不保存到文件
    }
    
    // 加密密码
    if ssh.Password != "" {
        encryptedPassword, err := encrypt(ssh.Password, masterPassword)
        if err != nil {
            return err
        }
        ssh.EncryptedPassword = encryptedPassword
        ssh.Password = "" // 清除明文密码，不保存到文件
    }
    
    currentConfig.SSH = ssh
    return SaveConfigWithEncryption()
}

// 保存配置时加密敏感信息
func SaveConfigWithEncryption() error {
    // 创建配置副本用于保存
    configToSave := currentConfig
    
    // 清除运行时明文凭据，只保存加密后的版本
    configToSave.SSH.Username = ""
    configToSave.SSH.Password = ""
    
    data, err := json.MarshalIndent(configToSave, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile("config.json", data, 0644)
}

// 上传任务管理函数
func SetUploadTasks(tasks []UploadTask) {
    currentConfig.Deployment.UploadTasks = tasks
    SaveConfig()
}

func GetUploadTasks() []UploadTask {
    return currentConfig.Deployment.UploadTasks
}

func AddUploadTask(task UploadTask) {
    currentConfig.Deployment.UploadTasks = append(currentConfig.Deployment.UploadTasks, task)
    SaveConfig()
}

func MarkTaskCompleted(taskID string) {
    for i := range currentConfig.Deployment.UploadTasks {
        if currentConfig.Deployment.UploadTasks[i].ID == taskID {
            currentConfig.Deployment.UploadTasks[i].Completed = true
            break
        }
    }
    SaveConfig()
}

func RemoveCompletedTasks() {
    var pendingTasks []UploadTask
    for _, task := range currentConfig.Deployment.UploadTasks {
        if !task.Completed {
            pendingTasks = append(pendingTasks, task)
        }
    }
    currentConfig.Deployment.UploadTasks = pendingTasks
    SaveConfig()
}

func SetDeploymentPaused(paused bool) {
    currentConfig.Deployment.IsPaused = paused
    if paused {
        currentConfig.Deployment.LastSyncStatus = "paused"
        currentConfig.Deployment.LastSyncMessage = "上传已暂停"
    }
    SaveConfig()
}

func IsDeploymentPaused() bool {
    return currentConfig.Deployment.IsPaused
}

func GetPendingTasksCount() int {
    count := 0
    for _, task := range currentConfig.Deployment.UploadTasks {
        if !task.Completed {
            count++
        }
    }
    return count
}

// 进度管理函数
func UpdateProgress(progressType, status, message string, progress, total, current int, currentFile, speed, eta string) {
    now := time.Now()
    
    // 如果是新任务，设置开始时间
    if currentConfig.Deployment.Progress.StartTime == nil && (status == "building" || status == "deploying") {
        currentConfig.Deployment.Progress.StartTime = &now
    }
    
    currentConfig.Deployment.Progress = ProgressInfo{
        Type:        progressType,
        Status:      status,
        Message:     message,
        Progress:    progress,
        Total:       total,
        Current:     current,
        CurrentFile: currentFile,
        Speed:       speed,
        ETA:         eta,
        StartTime:   currentConfig.Deployment.Progress.StartTime,
        UpdateTime:  now,
    }
    
    // 同时更新部署状态
    currentConfig.Deployment.LastSyncStatus = status
    currentConfig.Deployment.LastSyncMessage = message
    
    if status == "success" || status == "failed" {
        currentConfig.Deployment.LastSyncTime = &now
        // 任务完成时清除开始时间
        currentConfig.Deployment.Progress.StartTime = nil
    }
    
    SaveConfig()
}

// 获取当前进度信息
func GetCurrentProgress() ProgressInfo {
    return currentConfig.Deployment.Progress
}

// 清除进度信息
func ClearProgress() {
    currentConfig.Deployment.Progress = ProgressInfo{}
    SaveConfig()
}

// 检查是否有正在进行的任务
func HasActiveTask() bool {
    status := currentConfig.Deployment.Progress.Status
    return status == "building" || status == "deploying"
}

// 计算任务持续时间
func GetTaskDuration() time.Duration {
    if currentConfig.Deployment.Progress.StartTime == nil {
        return 0
    }
    return time.Since(*currentConfig.Deployment.Progress.StartTime)
}