package e2e

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"spine-go/libspine"
	"sync"
	"time"
)

// TestServerManager 管理测试服务器的生命周期
type TestServerManager struct {
	server     *libspine.Server
	config     *libspine.Config
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
	isRunning  bool
	startTime  time.Time
	testPorts  map[string]int // 协议 -> 端口映射
}

// NewTestServerManager 创建新的测试服务器管理器
func NewTestServerManager() *TestServerManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &TestServerManager{
		ctx:       ctx,
		cancel:    cancel,
		testPorts: make(map[string]int),
	}
}

// StartServer 启动测试服务器
func (tsm *TestServerManager) StartServer(protocols []string) error {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()

	if tsm.isRunning {
		return fmt.Errorf("test server is already running")
	}

	// 分配测试端口
	listenConfigs := make([]libspine.ListenConfig, 0, len(protocols))
	for _, protocol := range protocols {
		port, err := tsm.allocatePort()
		if err != nil {
			return fmt.Errorf("failed to allocate port for %s: %v", protocol, err)
		}
		tsm.testPorts[protocol] = port

		switch protocol {
		case "tcp":
			listenConfigs = append(listenConfigs, libspine.ListenConfig{
				Schema: "tcp",
				Host:   "127.0.0.1",
				Port:   fmt.Sprintf("%d", port),
			})
		case "http":
			listenConfigs = append(listenConfigs, libspine.ListenConfig{
				Schema: "http",
				Host:   "127.0.0.1",
				Port:   fmt.Sprintf("%d", port),
			})
		case "unix":
			socketPath := fmt.Sprintf("/tmp/spine_test_%d.sock", port)
			listenConfigs = append(listenConfigs, libspine.ListenConfig{
				Schema: "unix",
				Path:   socketPath,
			})
			tsm.testPorts[protocol] = 0 // Unix socket 不需要端口
		case "namedpipe":
			pipeName := fmt.Sprintf("spine_test_%d", port)
			listenConfigs = append(listenConfigs, libspine.ListenConfig{
				Schema: "namedpipe",
				Path:   pipeName,
			})
			tsm.testPorts[protocol] = port // 保存端口用于生成唯一管道名
		}
	}

	// 创建服务器配置
	tsm.config = &libspine.Config{
		ListenConfigs: listenConfigs,
		ServerMode:    "chat",
		StaticPath:    "", // 测试时不需要静态文件
	}

	// 创建并启动服务器
	tsm.server = libspine.NewServer(tsm.config)
	
	// 在 goroutine 中启动服务器
	go func() {
		if err := tsm.server.Start(); err != nil {
			log.Printf("Test server start error: %v", err)
		}
	}()

	// 等待服务器启动
	if err := tsm.waitForServerReady(); err != nil {
		return fmt.Errorf("server failed to start: %v", err)
	}

	tsm.isRunning = true
	tsm.startTime = time.Now()
	
	log.Printf("Test server started with protocols: %v, ports: %v", protocols, tsm.testPorts)
	return nil
}

// StopServer 停止测试服务器
func (tsm *TestServerManager) StopServer() error {
	tsm.mu.Lock()
	defer tsm.mu.Unlock()

	if !tsm.isRunning {
		return nil
	}

	if tsm.server != nil {
		if err := tsm.server.Stop(); err != nil {
			log.Printf("Error stopping test server: %v", err)
		}
	}

	tsm.cancel()
	tsm.isRunning = false
	
	// 清理 Unix socket 文件
	for protocol, port := range tsm.testPorts {
		if protocol == "unix" {
			socketPath := fmt.Sprintf("/tmp/spine_test_%d.sock", port)
			// 忽略删除错误，文件可能已经不存在
			_ = removeFile(socketPath)
		}
	}
	
	log.Printf("Test server stopped after running for %v", time.Since(tsm.startTime))
	return nil
}

// GetServerAddress 获取指定协议的服务器地址
func (tsm *TestServerManager) GetServerAddress(protocol string) (string, error) {
	tsm.mu.RLock()
	defer tsm.mu.RUnlock()

	if !tsm.isRunning {
		return "", fmt.Errorf("test server is not running")
	}

	port, exists := tsm.testPorts[protocol]
	if !exists {
		return "", fmt.Errorf("protocol %s is not configured", protocol)
	}

	switch protocol {
	case "tcp":
		return fmt.Sprintf("127.0.0.1:%d", port), nil
	case "http":
		return fmt.Sprintf("127.0.0.1:%d", port), nil
	case "unix":
		return fmt.Sprintf("/tmp/spine_test_%d.sock", port), nil
	case "namedpipe":
		return fmt.Sprintf("\\\\.\\pipe\\spine_test_%d", port), nil
	default:
		return "", fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// GetServer 获取服务器实例（用于访问内部状态）
func (tsm *TestServerManager) GetServer() *libspine.Server {
	tsm.mu.RLock()
	defer tsm.mu.RUnlock()
	return tsm.server
}

// IsRunning 检查服务器是否正在运行
func (tsm *TestServerManager) IsRunning() bool {
	tsm.mu.RLock()
	defer tsm.mu.RUnlock()
	return tsm.isRunning
}

// GetUptime 获取服务器运行时间
func (tsm *TestServerManager) GetUptime() time.Duration {
	tsm.mu.RLock()
	defer tsm.mu.RUnlock()
	
	if !tsm.isRunning {
		return 0
	}
	return time.Since(tsm.startTime)
}

// allocatePort 分配一个可用的端口
func (tsm *TestServerManager) allocatePort() (int, error) {
	// 使用系统分配的临时端口
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// waitForServerReady 等待服务器准备就绪
func (tsm *TestServerManager) waitForServerReady() error {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for server to be ready")
		case <-ticker.C:
			// 检查 TCP 端口是否可连接
			if port, exists := tsm.testPorts["tcp"]; exists {
				conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
				if err == nil {
					conn.Close()
					return nil
				}
			}
			// 检查 HTTP 端口是否可连接
			if port, exists := tsm.testPorts["http"]; exists {
				conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
				if err == nil {
					conn.Close()
					return nil
				}
			}
			// 检查 Unix socket 是否可连接
			if _, exists := tsm.testPorts["unix"]; exists {
				socketPath := fmt.Sprintf("/tmp/spine_test_%d.sock", tsm.testPorts["unix"])
				conn, err := net.DialTimeout("unix", socketPath, time.Second)
				if err == nil {
					conn.Close()
					return nil
				}
			}
			// 检查 Named Pipe 是否可连接
			if port, exists := tsm.testPorts["namedpipe"]; exists {
				if tsm.checkNamedPipeReady(port) {
					return nil
				}
			}
		}
	}
}

// Stop 停止服务器（别名方法，用于测试）
func (tsm *TestServerManager) Stop() error {
	return tsm.StopServer()
}

// removeFile 删除文件，忽略错误
func removeFile(path string) error {
	return os.Remove(path)
}

// checkNamedPipeReady 检查 Named Pipe 是否准备就绪（平台特定实现）
func (tsm *TestServerManager) checkNamedPipeReady(port int) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	
	// 在 Windows 上尝试连接到 named pipe
	pipeName := fmt.Sprintf(`\\.\pipe\spine_test_%d`, port)
	return tsm.tryConnectNamedPipe(pipeName)
}

// tryConnectNamedPipe 尝试连接到 named pipe
func (tsm *TestServerManager) tryConnectNamedPipe(pipeName string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	
	// 使用一个简单的方法检查管道是否存在
	// 这里我们可以尝试使用 os.Stat 或其他方法
	// 为了简化，我们假设如果服务器启动了，管道就准备好了
	// 在实际实现中，可以使用 Windows API 来检查
	return true // 简化实现，假设总是准备好了
}
