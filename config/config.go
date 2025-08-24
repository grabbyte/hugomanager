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
    Username          string `json:"username"`
    Password          string `json:"password,omitempty"`          // 运行时明文密码
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

type DeploymentInfo struct {
    LastSyncTime     *time.Time    `json:"last_sync_time,omitempty"`
    LastSyncStatus   string        `json:"last_sync_status,omitempty"`   // "success", "failed", "building", "deploying", "paused"
    LastSyncMessage  string        `json:"last_sync_message,omitempty"`
    FilesDeployed    int           `json:"files_deployed,omitempty"`
    BytesTransferred int64         `json:"bytes_transferred,omitempty"`
    UploadTasks      []UploadTask  `json:"upload_tasks,omitempty"`       // 待上传任务队列
    IsPaused         bool          `json:"is_paused,omitempty"`          // 是否暂停
}

type Config struct {
    HugoProjectPath string         `json:"hugo_project_path"`
    SSH             SSHConfig      `json:"ssh"`
    Deployment      DeploymentInfo `json:"deployment"`
}

var currentConfig Config
var decryptionKey string // 运行时解密密钥

func init() {
    LoadConfig()
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
    return filepath.Join(currentConfig.HugoProjectPath, "static", "images")
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
    
    // 尝试解密SSH密码验证密钥是否正确
    if currentConfig.SSH.EncryptedPassword != "" {
        _, err := decrypt(currentConfig.SSH.EncryptedPassword, decryptionKey)
        if err != nil {
            decryptionKey = ""
            return errors.New("解密密钥错误")
        }
        // 解密成功，更新运行时密码
        currentConfig.SSH.Password, _ = decrypt(currentConfig.SSH.EncryptedPassword, decryptionKey)
    }
    
    return nil
}

// 检查是否已设置解密密钥
func IsDecryptionKeySet() bool {
    return decryptionKey != ""
}

// 加密并保存SSH配置
func SetSSHConfigWithEncryption(ssh SSHConfig, masterPassword string) error {
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
    
    // 清除运行时明文密码
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