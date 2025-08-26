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

// 服务器配置结构
type ServerConfig struct {
    ID                string    `json:"id"`                   // 服务器ID
    Name              string    `json:"name"`                 // 服务器名称
    Host              string    `json:"host"`                 // 服务器地址
    Port              int       `json:"port"`                 // SSH端口
    Username          string    `json:"username,omitempty"`           // 运行时明文用户名
    EncryptedUsername string    `json:"encrypted_username,omitempty"` // 存储时加密用户名
    Password          string    `json:"password,omitempty"`           // 运行时明文密码
    EncryptedPassword string    `json:"encrypted_password,omitempty"` // 存储时加密密码
    KeyPath           string    `json:"key_path,omitempty"`           // 私钥路径
    RemotePath        string    `json:"remote_path"`                  // 远程部署路径
    Domain            string    `json:"domain"`               // 网站域名
    Enabled           bool      `json:"enabled"`              // 是否启用
    CreatedAt         time.Time `json:"created_at"`           // 创建时间
    LastDeployment    *time.Time `json:"last_deployment,omitempty"`   // 最后部署时间
}

// 服务器部署状态
type ServerDeploymentStatus struct {
    ServerID         string     `json:"server_id"`           // 服务器ID
    Status           string     `json:"status"`              // success, failed, building, deploying, paused, idle
    Message          string     `json:"message"`             // 状态消息
    Progress         int        `json:"progress"`            // 进度百分比 0-100
    FilesDeployed    int        `json:"files_deployed"`      // 已部署文件数
    BytesTransferred int64      `json:"bytes_transferred"`   // 已传输字节数
    CurrentFile      string     `json:"current_file"`        // 当前处理文件
    Speed            string     `json:"speed"`               // 传输速度
    StartTime        *time.Time `json:"start_time,omitempty"` // 开始时间
    UpdateTime       time.Time  `json:"update_time"`         // 更新时间
    CanPause         bool       `json:"can_pause"`           // 是否可暂停
    CanResume        bool       `json:"can_resume"`          // 是否可继续
    CanStop          bool       `json:"can_stop"`            // 是否可停止
}

// 多服务器部署配置
type MultiServerDeployment struct {
    Servers        []ServerConfig            `json:"servers"`          // 服务器列表
    StatusMap      map[string]ServerDeploymentStatus `json:"status_map"`       // 服务器状态映射
    GlobalSettings map[string]interface{}    `json:"global_settings"`  // 全局设置
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
    MultiDeploy     MultiServerDeployment `json:"multi_deploy"` // 多服务器部署配置
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

// 多服务器部署管理函数
func GetMultiServerDeployment() MultiServerDeployment {
    if currentConfig.MultiDeploy.StatusMap == nil {
        currentConfig.MultiDeploy.StatusMap = make(map[string]ServerDeploymentStatus)
    }
    return currentConfig.MultiDeploy
}

func GetServerConfigs() []ServerConfig {
    return currentConfig.MultiDeploy.Servers
}

func AddServerConfig(server ServerConfig) {
    if server.ID == "" {
        server.ID = generateServerID()
    }
    server.CreatedAt = time.Now()
    currentConfig.MultiDeploy.Servers = append(currentConfig.MultiDeploy.Servers, server)
    
    // 初始化服务器状态
    if currentConfig.MultiDeploy.StatusMap == nil {
        currentConfig.MultiDeploy.StatusMap = make(map[string]ServerDeploymentStatus)
    }
    currentConfig.MultiDeploy.StatusMap[server.ID] = ServerDeploymentStatus{
        ServerID:   server.ID,
        Status:     "idle",
        Message:    "等待部署",
        Progress:   0,
        UpdateTime: time.Now(),
    }
    
    SaveConfig()
}

func UpdateServerConfig(serverID string, server ServerConfig) error {
    for i, s := range currentConfig.MultiDeploy.Servers {
        if s.ID == serverID {
            server.ID = serverID
            server.CreatedAt = s.CreatedAt // 保留创建时间
            currentConfig.MultiDeploy.Servers[i] = server
            SaveConfig()
            return nil
        }
    }
    return errors.New("server not found")
}

func DeleteServerConfig(serverID string) error {
    for i, s := range currentConfig.MultiDeploy.Servers {
        if s.ID == serverID {
            // 删除服务器配置
            currentConfig.MultiDeploy.Servers = append(currentConfig.MultiDeploy.Servers[:i], currentConfig.MultiDeploy.Servers[i+1:]...)
            // 删除对应的状态
            delete(currentConfig.MultiDeploy.StatusMap, serverID)
            SaveConfig()
            return nil
        }
    }
    return errors.New("server not found")
}

func GetServerConfig(serverID string) (ServerConfig, error) {
    for _, s := range currentConfig.MultiDeploy.Servers {
        if s.ID == serverID {
            return s, nil
        }
    }
    return ServerConfig{}, errors.New("server not found")
}

func UpdateServerDeploymentStatus(serverID string, status ServerDeploymentStatus) {
    if currentConfig.MultiDeploy.StatusMap == nil {
        currentConfig.MultiDeploy.StatusMap = make(map[string]ServerDeploymentStatus)
    }
    status.ServerID = serverID
    status.UpdateTime = time.Now()
    currentConfig.MultiDeploy.StatusMap[serverID] = status
    SaveConfig()
}

func GetServerDeploymentStatus(serverID string) ServerDeploymentStatus {
    if currentConfig.MultiDeploy.StatusMap == nil {
        return ServerDeploymentStatus{
            ServerID: serverID,
            Status:   "idle",
            Message:  "等待部署",
        }
    }
    if status, exists := currentConfig.MultiDeploy.StatusMap[serverID]; exists {
        return status
    }
    return ServerDeploymentStatus{
        ServerID: serverID,
        Status:   "idle",
        Message:  "等待部署",
    }
}

func GetAllServerStatuses() map[string]ServerDeploymentStatus {
    if currentConfig.MultiDeploy.StatusMap == nil {
        currentConfig.MultiDeploy.StatusMap = make(map[string]ServerDeploymentStatus)
    }
    return currentConfig.MultiDeploy.StatusMap
}

// 生成服务器ID
func generateServerID() string {
    return "server_" + time.Now().Format("20060102150405")
}

// 加密服务器配置中的敏感信息
func SetServerConfigWithEncryption(serverID string, server ServerConfig, masterPassword string) error {
    // 加密用户名
    if server.Username != "" {
        encryptedUsername, err := encrypt(server.Username, masterPassword)
        if err != nil {
            return err
        }
        server.EncryptedUsername = encryptedUsername
        server.Username = "" // 清除明文
    }
    
    // 加密密码
    if server.Password != "" {
        encryptedPassword, err := encrypt(server.Password, masterPassword)
        if err != nil {
            return err
        }
        server.EncryptedPassword = encryptedPassword
        server.Password = "" // 清除明文
    }
    
    // 更新服务器配置
    if serverID == "" {
        AddServerConfig(server)
    } else {
        return UpdateServerConfig(serverID, server)
    }
    
    return nil
}

// 解密服务器配置
func DecryptServerConfig(server ServerConfig, masterPassword string) (ServerConfig, error) {
    var err error
    
    // 解密用户名
    if server.EncryptedUsername != "" {
        server.Username, err = decrypt(server.EncryptedUsername, masterPassword)
        if err != nil {
            return server, err
        }
    }
    
    // 解密密码
    if server.EncryptedPassword != "" {
        server.Password, err = decrypt(server.EncryptedPassword, masterPassword)
        if err != nil {
            return server, err
        }
    }
    
    return server, nil
}