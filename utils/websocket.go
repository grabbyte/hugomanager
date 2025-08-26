package utils

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// 进度消息类型
type ProgressMessage struct {
	Type        string    `json:"type"`         // "build", "deploy", "complete", "error"
	Status      string    `json:"status"`       // "building", "deploying", "success", "failed", "paused"
	Message     string    `json:"message"`      // 状态描述
	Progress    int       `json:"progress"`     // 进度百分比 0-100
	Total       int       `json:"total"`        // 总任务数
	Current     int       `json:"current"`      // 当前完成数
	CurrentFile string    `json:"current_file"` // 当前处理的文件
	Speed       string    `json:"speed"`        // 传输速度
	ETA         string    `json:"eta"`          // 预计剩余时间
	Timestamp   time.Time `json:"timestamp"`    // 时间戳
	ServerID    string    `json:"server_id"`    // 服务器ID，用于多服务器部署
	ServerName  string    `json:"server_name"`  // 服务器名称
}

// 连接管理器
type ConnectionManager struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan ProgressMessage
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mutex      sync.RWMutex
}

var Manager = &ConnectionManager{
	clients:    make(map[*websocket.Conn]bool),
	broadcast:  make(chan ProgressMessage, 256),
	register:   make(chan *websocket.Conn),
	unregister: make(chan *websocket.Conn),
}

// 启动连接管理器
func (manager *ConnectionManager) Start() {
	go func() {
		for {
			select {
			case conn := <-manager.register:
				manager.mutex.Lock()
				manager.clients[conn] = true
				manager.mutex.Unlock()
				log.Printf("WebSocket 客户端连接: %v", conn.RemoteAddr())

			case conn := <-manager.unregister:
				manager.mutex.Lock()
				if _, ok := manager.clients[conn]; ok {
					delete(manager.clients, conn)
					conn.Close()
				}
				manager.mutex.Unlock()
				log.Printf("WebSocket 客户端断开: %v", conn.RemoteAddr())

			case message := <-manager.broadcast:
				manager.mutex.RLock()
				messageData, err := json.Marshal(message)
				if err != nil {
					log.Printf("序列化进度消息失败: %v", err)
					manager.mutex.RUnlock()
					continue
				}

				for conn := range manager.clients {
					select {
					case <-time.After(10 * time.Second):
						// 写入超时，关闭连接
						delete(manager.clients, conn)
						conn.Close()
					default:
						if err := conn.WriteMessage(websocket.TextMessage, messageData); err != nil {
							log.Printf("发送消息失败: %v", err)
							delete(manager.clients, conn)
							conn.Close()
						}
					}
				}
				manager.mutex.RUnlock()
			}
		}
	}()
}

// WebSocket连接处理器
func HandleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败: %v", err)
		return
	}

	// 注册连接
	Manager.register <- conn

	// 发送当前状态
	SendCurrentStatus(conn)

	// 处理连接
	go func() {
		defer func() {
			Manager.unregister <- conn
		}()

		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		// 心跳检测
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
					if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
						return
					}
				}
			}
		}()

		for {
			// 读取消息（主要是为了检测连接状态）
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket意外关闭: %v", err)
				}
				break
			}
		}
	}()
}

// 发送当前状态给新连接的客户端
func SendCurrentStatus(conn *websocket.Conn) {
	message := ProgressMessage{
		Type:      "status",
		Status:    "ready",
		Message:   "WebSocket连接已建立",
		Progress:  0,
		Timestamp: time.Now(),
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		log.Printf("序列化初始状态失败: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, messageData); err != nil {
		log.Printf("发送初始状态失败: %v", err)
	}
}

// 广播进度消息
func BroadcastProgress(msgType, status, message string, progress, total, current int, currentFile string) {
	progressMsg := ProgressMessage{
		Type:        msgType,
		Status:      status,
		Message:     message,
		Progress:    progress,
		Total:       total,
		Current:     current,
		CurrentFile: currentFile,
		Timestamp:   time.Now(),
	}

	select {
	case Manager.broadcast <- progressMsg:
		// 消息已发送到广播通道
	default:
		// 通道已满，跳过这条消息
		log.Printf("进度广播通道已满，跳过消息: %s", message)
	}
}

// 广播多服务器进度消息
func BroadcastMultiServerProgress(serverID, serverName, msgType, status, message string, progress, total, current int, currentFile string) {
	progressMsg := ProgressMessage{
		Type:        msgType,
		Status:      status,
		Message:     message,
		Progress:    progress,
		Total:       total,
		Current:     current,
		CurrentFile: currentFile,
		Timestamp:   time.Now(),
		ServerID:    serverID,
		ServerName:  serverName,
	}

	select {
	case Manager.broadcast <- progressMsg:
		// 消息已发送到广播通道
	default:
		// 通道已满，跳过这条消息
		log.Printf("多服务器进度广播通道已满，跳过消息: %s", message)
	}
}

// 广播构建进度
func BroadcastBuildProgress(message string, progress int) {
	BroadcastProgress("build", "building", message, progress, 100, progress, "")
}

// 广播部署进度
func BroadcastDeployProgress(message string, progress, total, current int, currentFile string) {
	BroadcastProgress("deploy", "deploying", message, progress, total, current, currentFile)
}

// 广播完成消息
func BroadcastComplete(msgType, message string, total int) {
	BroadcastProgress(msgType, "success", message, 100, total, total, "")
}

// 广播错误消息
func BroadcastError(msgType, message string) {
	BroadcastProgress(msgType, "failed", message, 0, 0, 0, "")
}

// 广播暂停消息
func BroadcastPause(message string, progress, total, current int) {
	BroadcastProgress("deploy", "paused", message, progress, total, current, "")
}

// 多服务器专用广播函数
func BroadcastMultiServerBuildProgress(serverID, serverName, message string, progress int) {
	BroadcastMultiServerProgress(serverID, serverName, "build", "building", message, progress, 100, progress, "")
}

func BroadcastMultiServerDeployProgress(serverID, serverName, message string, progress, total, current int, currentFile string) {
	BroadcastMultiServerProgress(serverID, serverName, "deploy", "deploying", message, progress, total, current, currentFile)
}

func BroadcastMultiServerComplete(serverID, serverName, msgType, message string, total int) {
	BroadcastMultiServerProgress(serverID, serverName, msgType, "success", message, 100, total, total, "")
}

func BroadcastMultiServerError(serverID, serverName, msgType, message string) {
	BroadcastMultiServerProgress(serverID, serverName, msgType, "failed", message, 0, 0, 0, "")
}

func BroadcastMultiServerPause(serverID, serverName, message string, progress, total, current int) {
	BroadcastMultiServerProgress(serverID, serverName, "deploy", "paused", message, progress, total, current, "")
}

// 获取连接数
func GetConnectionCount() int {
	Manager.mutex.RLock()
	defer Manager.mutex.RUnlock()
	return len(Manager.clients)
}