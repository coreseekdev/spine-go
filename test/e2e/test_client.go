package e2e

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// TestClient 测试客户端接口
type TestClient interface {
	Connect() error
	Disconnect() error
	SendMessage(user, message string) error
	JoinChat() error
	LeaveChat() error
	GetMessages() error
	ReceiveMessage() (*ChatResponse, error)
	IsConnected() bool
}

// ChatRequest 聊天请求结构
type ChatRequest struct {
	Method string      `json:"method"`
	Path   string      `json:"path"`
	Data   interface{} `json:"data"`
}

// ChatResponse 聊天响应结构
type ChatResponse struct {
	Status int         `json:"status"`
	Data   interface{} `json:"data"`
	Error  string      `json:"error"`
}

// ChatMessage 聊天消息结构
type ChatMessage struct {
	ID        string    `json:"id"`
	User      string    `json:"user"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// TCPTestClient TCP 测试客户端
type TCPTestClient struct {
	address     string
	conn        net.Conn
	reader      *bufio.Scanner
	mu          sync.RWMutex
	connected   bool
	messageChan chan *ChatResponse // 用于接收消息的通道
	stopChan    chan struct{}      // 用于停止后台读取的通道
}

// NewTCPTestClient 创建新的 TCP 测试客户端
func NewTCPTestClient(address string) *TCPTestClient {
	return &TCPTestClient{
		address:     address,
		messageChan: make(chan *ChatResponse, 100), // 缓冲通道
		stopChan:    make(chan struct{}),
	}
}

// Connect 连接到服务器
func (c *TCPTestClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("client is already connected")
	}

	conn, err := net.DialTimeout("tcp", c.address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", c.address, err)
	}

	c.conn = conn
	c.reader = bufio.NewScanner(conn)
	c.connected = true
	
	// 启动后台消息读取 goroutine
	go c.readMessages()
	
	return nil
}

// Disconnect 断开连接
func (c *TCPTestClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	// 停止后台读取
	close(c.stopChan)

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	c.connected = false
	c.reader = nil
	
	// 重新创建通道以便重连
	c.messageChan = make(chan *ChatResponse, 100)
	c.stopChan = make(chan struct{})
	
	return nil
}

// readMessages 后台读取消息的方法
func (c *TCPTestClient) readMessages() {
	for {
		select {
		case <-c.stopChan:
			return
		default:
			c.mu.RLock()
			if !c.connected || c.conn == nil {
				c.mu.RUnlock()
				return
			}
			
			// 设置读取超时
			c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			
			buffer := make([]byte, 4096)
			n, err := c.conn.Read(buffer)
			c.mu.RUnlock()
			
			if err != nil {
				// 如果是超时错误，继续循环
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				// 其他错误表示连接断开，更新连接状态
				c.mu.Lock()
				c.connected = false
				c.mu.Unlock()
				return
			}
			
			if n > 0 {
				data := string(buffer[:n])
				// 处理可能的多个JSON对象
				decoder := json.NewDecoder(strings.NewReader(data))
				for decoder.More() {
					var response ChatResponse
					if err := decoder.Decode(&response); err == nil {
						select {
						case c.messageChan <- &response:
						case <-c.stopChan:
							return
						default:
							// 通道满了，丢弃旧消息
						}
					}
				}
			}
		}
	}
}

// checkConnectionStatus 检查并更新连接状态
func (c *TCPTestClient) checkConnectionStatus() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return
	}

	// 尝试写入空数据检测连接状态
	_, err := c.conn.Write([]byte{})
	if err != nil {
		c.connected = false
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
	}
}

// SendMessage 发送聊天消息
func (c *TCPTestClient) SendMessage(user, message string) error {
	// 检查连接状态
	if !c.IsConnected() {
		return fmt.Errorf("client is not connected")
	}
	return c.sendRequest("POST", "/chat", map[string]interface{}{
		"user":    user,
		"message": message,
	})
}

// JoinChat 加入聊天
func (c *TCPTestClient) JoinChat() error {
	return c.sendRequest("JOIN", "/chat", nil)
}

// LeaveChat 离开聊天
func (c *TCPTestClient) LeaveChat() error {
	return c.sendRequest("LEAVE", "/chat", nil)
}

// GetMessages 获取消息
func (c *TCPTestClient) GetMessages() error {
	return c.sendRequest("GET", "/chat", nil)
}

// ReceiveMessage 接收消息 - 从消息通道读取
func (c *TCPTestClient) ReceiveMessage() (*ChatResponse, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client is not connected")
	}

	select {
	case response := <-c.messageChan:
		return response, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for message")
	}
}

// IsConnected 检查连接状态
func (c *TCPTestClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 简单检查连接状态，不进行实际网络操作
	return c.connected && c.conn != nil
}

// sendRequest 发送请求（异步，不等待响应）
func (c *TCPTestClient) sendRequest(method, path string, data interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return fmt.Errorf("client is not connected")
	}

	request := ChatRequest{
		Method: method,
		Path:   path,
		Data:   data,
	}

	requestData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	// 发送请求（不等待响应）
	_, err = c.conn.Write(append(requestData, '\n'))
	if err != nil {
		// 写入失败时更新连接状态
		c.mu.RUnlock()
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		c.mu.RLock()
		return fmt.Errorf("failed to send request: %v", err)
	}

	return nil
}


// WebSocketTestClient WebSocket 测试客户端
type WebSocketTestClient struct {
	address   string
	conn      *websocket.Conn
	connected bool
	mu        sync.RWMutex
}

// NewWebSocketTestClient 创建新的 WebSocket 测试客户端
func NewWebSocketTestClient(address string) *WebSocketTestClient {
	return &WebSocketTestClient{
		address:   address,
		connected: false,
	}
}

// Connect 连接到服务器
func (c *WebSocketTestClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("client is already connected")
	}

	u := url.URL{Scheme: "ws", Host: c.address, Path: "/ws"}
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", u.String(), err)
	}

	c.conn = conn
	c.connected = true
	return nil
}

// Disconnect 断开连接
func (c *WebSocketTestClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	if c.conn != nil {
		c.conn.Close()
	}
	c.connected = false
	return nil
}

// SendMessage 发送聊天消息
func (c *WebSocketTestClient) SendMessage(user, message string) error {
	// 检查连接状态
	if !c.IsConnected() {
		return fmt.Errorf("client is not connected")
	}
	return c.sendRequest("POST", "/chat", map[string]interface{}{
		"user":    user,
		"message": message,
	})
}

// JoinChat 加入聊天
func (c *WebSocketTestClient) JoinChat() error {
	return c.sendRequest("JOIN", "/chat", nil)
}

// LeaveChat 离开聊天
func (c *WebSocketTestClient) LeaveChat() error {
	return c.sendRequest("LEAVE", "/chat", nil)
}

// GetMessages 获取消息
func (c *WebSocketTestClient) GetMessages() error {
	return c.sendRequest("GET", "/chat", nil)
}

// ReceiveMessage 接收消息
func (c *WebSocketTestClient) ReceiveMessage() (*ChatResponse, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return nil, fmt.Errorf("client is not connected")
	}

	_, data, err := c.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %v", err)
	}

	var response ChatResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &response, nil
}

// IsConnected 检查是否已连接
func (c *WebSocketTestClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return false
	}

	// 尝试写入一个ping消息来检测连接状态
	err := c.conn.WriteMessage(websocket.PingMessage, []byte{})
	if err != nil {
		// 写入失败表示连接断开
		c.mu.RUnlock()
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		c.mu.RLock()
		return false
	}

	return true
}

// sendRequest 发送请求
func (c *WebSocketTestClient) sendRequest(method, path string, data interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return fmt.Errorf("client is not connected")
	}

	request := ChatRequest{
		Method: method,
		Path:   path,
		Data:   data,
	}

	requestData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	err = c.conn.WriteMessage(websocket.TextMessage, requestData)
	if err != nil {
		// 更新连接状态
		c.mu.RUnlock()
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		c.mu.RLock()
		return fmt.Errorf("failed to send request: %v", err)
	}

	return nil
}

// UnixSocketTestClient Unix Socket 测试客户端
type UnixSocketTestClient struct {
	socketPath string
	conn       net.Conn
	reader     *bufio.Scanner
	mu         sync.RWMutex
	connected  bool
}

// NewUnixSocketTestClient 创建新的 Unix Socket 测试客户端
func NewUnixSocketTestClient(socketPath string) *UnixSocketTestClient {
	return &UnixSocketTestClient{
		socketPath: socketPath,
	}
}

// Connect 连接到服务器
func (c *UnixSocketTestClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("client is already connected")
	}

	conn, err := net.DialTimeout("unix", c.socketPath, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", c.socketPath, err)
	}

	c.conn = conn
	c.reader = bufio.NewScanner(conn)
	c.connected = true
	return nil
}

// Disconnect 断开连接
func (c *UnixSocketTestClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	if c.conn != nil {
		c.conn.Close()
	}
	c.connected = false
	return nil
}

// SendMessage 发送聊天消息
func (c *UnixSocketTestClient) SendMessage(user, message string) error {
	return c.sendRequest("POST", "/chat", map[string]interface{}{
		"user":    user,
		"message": message,
	})
}

// JoinChat 加入聊天
func (c *UnixSocketTestClient) JoinChat() error {
	return c.sendRequest("JOIN", "/chat", nil)
}

// LeaveChat 离开聊天
func (c *UnixSocketTestClient) LeaveChat() error {
	return c.sendRequest("LEAVE", "/chat", nil)
}

// GetMessages 获取消息
func (c *UnixSocketTestClient) GetMessages() error {
	return c.sendRequest("GET", "/chat", nil)
}

// ReceiveMessage 接收消息
func (c *UnixSocketTestClient) ReceiveMessage() (*ChatResponse, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.reader == nil {
		return nil, fmt.Errorf("client is not connected")
	}

	if !c.reader.Scan() {
		return nil, fmt.Errorf("failed to read message: %v", c.reader.Err())
	}

	data := c.reader.Text()
	var response ChatResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &response, nil
}

// IsConnected 检查是否已连接
func (c *UnixSocketTestClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// sendRequest 发送请求
func (c *UnixSocketTestClient) sendRequest(method, path string, data interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("client is not connected")
	}

	request := ChatRequest{
		Method: method,
		Path:   path,
		Data:   data,
	}

	requestData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	// 添加换行符作为消息分隔符
	requestData = append(requestData, '\n')

	_, err = c.conn.Write(requestData)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	return nil
}

// TestClientFactory 测试客户端工厂
type TestClientFactory struct{}

// NewTestClientFactory 创建新的测试客户端工厂
func NewTestClientFactory() *TestClientFactory {
	return &TestClientFactory{}
}

// CreateClient 根据协议创建测试客户端
func (f *TestClientFactory) CreateClient(protocol, address string) (TestClient, error) {
	switch protocol {
	case "tcp":
		return NewTCPTestClient(address), nil
	case "http":
		return NewWebSocketTestClient(address), nil
	case "unix":
		return NewUnixSocketTestClient(address), nil
	case "namedpipe":
		return NewNamedPipeTestClient(address), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// NamedPipeTestClient Named Pipe 测试客户端
type NamedPipeTestClient struct {
	pipeName    string
	conn        interface{} // 在 Windows 上是 windows.Handle，其他平台为 nil
	reader      *bufio.Scanner
	mu          sync.RWMutex
	connected   bool
	messageChan chan *ChatResponse
	stopChan    chan struct{}
}

// NewNamedPipeTestClient 创建新的 Named Pipe 测试客户端
func NewNamedPipeTestClient(pipeName string) *NamedPipeTestClient {
	// 确保管道名称格式正确
	if runtime.GOOS == "windows" && len(pipeName) > 0 && pipeName[0] != '\\' {
		pipeName = `\\.\pipe\` + pipeName
	}
	
	return &NamedPipeTestClient{
		pipeName:    pipeName,
		messageChan: make(chan *ChatResponse, 100),
		stopChan:    make(chan struct{}),
	}
}

// Connect 连接到服务器
func (c *NamedPipeTestClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return fmt.Errorf("client is already connected")
	}

	if runtime.GOOS != "windows" {
		return fmt.Errorf("named pipe is only supported on Windows")
	}

	// 在 Windows 上连接到 named pipe
	err := c.connectWindows()
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", c.pipeName, err)
	}

	c.connected = true
	
	// 启动后台消息读取 goroutine
	go c.readMessages()
	
	return nil
}

// Disconnect 断开连接
func (c *NamedPipeTestClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	// 停止后台读取
	close(c.stopChan)

	if c.conn != nil {
		c.closeConnection()
		c.conn = nil
	}

	c.connected = false
	c.reader = nil
	
	// 重新创建通道以便重连
	c.messageChan = make(chan *ChatResponse, 100)
	c.stopChan = make(chan struct{})
	
	return nil
}

// SendMessage 发送聊天消息
func (c *NamedPipeTestClient) SendMessage(user, message string) error {
	if !c.IsConnected() {
		return fmt.Errorf("client is not connected")
	}
	return c.sendRequest("POST", "/chat", map[string]interface{}{
		"user":    user,
		"message": message,
	})
}

// JoinChat 加入聊天
func (c *NamedPipeTestClient) JoinChat() error {
	return c.sendRequest("JOIN", "/chat", nil)
}

// LeaveChat 离开聊天
func (c *NamedPipeTestClient) LeaveChat() error {
	return c.sendRequest("LEAVE", "/chat", nil)
}

// GetMessages 获取消息
func (c *NamedPipeTestClient) GetMessages() error {
	return c.sendRequest("GET", "/chat", nil)
}

// ReceiveMessage 接收消息
func (c *NamedPipeTestClient) ReceiveMessage() (*ChatResponse, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client is not connected")
	}

	select {
	case response := <-c.messageChan:
		return response, nil
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout waiting for message")
	}
}

// IsConnected 检查连接状态
func (c *NamedPipeTestClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected && c.conn != nil
}

// sendRequest 发送请求
func (c *NamedPipeTestClient) sendRequest(method, path string, data interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return fmt.Errorf("client is not connected")
	}

	request := ChatRequest{
		Method: method,
		Path:   path,
		Data:   data,
	}

	requestData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	// 发送请求
	err = c.writeData(append(requestData, '\n'))
	if err != nil {
		// 写入失败时更新连接状态
		c.mu.RUnlock()
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
		c.mu.RLock()
		return fmt.Errorf("failed to send request: %v", err)
	}

	return nil
}

// readMessages 后台读取消息的方法
func (c *NamedPipeTestClient) readMessages() {
	for {
		select {
		case <-c.stopChan:
			return
		default:
			c.mu.RLock()
			if !c.connected || c.conn == nil {
				c.mu.RUnlock()
				return
			}
			
			buffer := make([]byte, 4096)
			n, err := c.readData(buffer)
			c.mu.RUnlock()
			
			if err != nil {
				// 读取错误表示连接断开
				c.mu.Lock()
				c.connected = false
				c.mu.Unlock()
				return
			}
			
			if n > 0 {
				data := string(buffer[:n])
				// 处理可能的多个JSON对象
				decoder := json.NewDecoder(strings.NewReader(data))
				for decoder.More() {
					var response ChatResponse
					if err := decoder.Decode(&response); err == nil {
						select {
						case c.messageChan <- &response:
						case <-c.stopChan:
							return
						default:
							// 通道满了，丢弃旧消息
						}
					}
				}
			}
			
			// 短暂休眠避免过度占用CPU
			time.Sleep(10 * time.Millisecond)
		}
	}
}
