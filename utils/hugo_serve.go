package utils

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"hugo-manager-go/config"
)

// HugoServeManager manages Hugo serve process
type HugoServeManager struct {
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	ctx     context.Context
	running bool
	port    int
	mutex   sync.RWMutex
}

var hugoServeManager = &HugoServeManager{
	port: 1313, // Hugo default port
}

// GetHugoServeManager returns the singleton instance
func GetHugoServeManager() *HugoServeManager {
	return hugoServeManager
}

// IsRunning checks if Hugo serve is currently running
func (h *HugoServeManager) IsRunning() bool {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.running
}

// GetPort returns the current serve port
func (h *HugoServeManager) GetPort() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.port
}

// Start starts Hugo serve process
func (h *HugoServeManager) Start(port int) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.running {
		return fmt.Errorf("Hugo serve已经在运行中")
	}

	// Get Hugo project path
	projectPath := config.GetHugoProjectPath()
	if projectPath == "" {
		return fmt.Errorf("未配置Hugo项目路径")
	}

	// Check if Hugo is available
	if _, err := exec.LookPath("hugo"); err != nil {
		return fmt.Errorf("未找到Hugo命令，请确保Hugo已安装并在PATH中")
	}

	// Check if port is available
	if !h.isPortAvailable(port) {
		// Try to find an available port starting from the requested port
		for i := port; i < port+10; i++ {
			if h.isPortAvailable(i) {
				port = i
				break
			}
		}
		if !h.isPortAvailable(port) {
			return fmt.Errorf("端口 %d 及其后续端口都被占用", port)
		}
	}

	// Create context for cancellation
	h.ctx, h.cancel = context.WithCancel(context.Background())

	// Prepare Hugo serve command
	args := []string{
		"serve",
		"--port", strconv.Itoa(port),
		"--bind", "0.0.0.0",
		"--buildDrafts",
		"--buildFuture",
		"--disableFastRender",
	}

	h.cmd = exec.CommandContext(h.ctx, "hugo", args...)
	h.cmd.Dir = projectPath

	// Start the command
	if err := h.cmd.Start(); err != nil {
		h.cancel()
		return fmt.Errorf("启动Hugo serve失败: %v", err)
	}

	h.running = true
	h.port = port

	// Monitor the process
	go h.monitor()

	return nil
}

// Stop stops Hugo serve process
func (h *HugoServeManager) Stop() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if !h.running {
		return fmt.Errorf("Hugo serve未运行")
	}

	// Cancel context to stop the command
	if h.cancel != nil {
		h.cancel()
	}

	// Wait for process to finish (with timeout)
	if h.cmd != nil && h.cmd.Process != nil {
		done := make(chan error, 1)
		go func() {
			done <- h.cmd.Wait()
		}()

		select {
		case <-done:
			// Process finished normally
		case <-time.After(5 * time.Second):
			// Force kill if not finished within 5 seconds
			if err := h.cmd.Process.Kill(); err != nil {
				return fmt.Errorf("强制终止Hugo serve失败: %v", err)
			}
		}
	}

	h.running = false
	h.cmd = nil
	h.cancel = nil

	return nil
}

// Restart restarts Hugo serve with the same port
func (h *HugoServeManager) Restart() error {
	currentPort := h.GetPort()
	if err := h.Stop(); err != nil && !h.IsRunning() {
		// If stop failed but not running, that's ok
	}
	
	// Wait a moment for port to be released
	time.Sleep(500 * time.Millisecond)
	
	return h.Start(currentPort)
}

// GetStatus returns current status information
func (h *HugoServeManager) GetStatus() map[string]interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	status := map[string]interface{}{
		"running": h.running,
		"port":    h.port,
	}

	if h.running {
		status["url"] = fmt.Sprintf("http://localhost:%d", h.port)
		status["pid"] = h.cmd.Process.Pid
	}

	return status
}

// monitor watches the Hugo serve process
func (h *HugoServeManager) monitor() {
	if h.cmd == nil {
		return
	}

	// Wait for process to finish
	err := h.cmd.Wait()

	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.running = false
	h.cmd = nil
	h.cancel = nil

	if err != nil && h.ctx.Err() == nil {
		// Process died unexpectedly (not due to cancellation)
		fmt.Printf("Hugo serve进程异常退出: %v\n", err)
	}
}

// isPortAvailable checks if a port is available
func (h *HugoServeManager) isPortAvailable(port int) bool {
	conn, err := http.Get(fmt.Sprintf("http://localhost:%d", port))
	if err != nil {
		return true // Port is available if we can't connect
	}
	conn.Body.Close()
	return false // Port is occupied if we can connect
}

// GetPreviewURL returns the preview URL if Hugo serve is running
func (h *HugoServeManager) GetPreviewURL() string {
	if h.IsRunning() {
		return fmt.Sprintf("http://localhost:%d", h.GetPort())
	}
	return ""
}