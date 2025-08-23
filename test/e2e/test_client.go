package e2e

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
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
	address   string
	conn      net.Conn
	reader    *bufio.Scanner
	mu        sync.RWMutex
	connected bool
}

// NewTCPTestClient 创建新的 TCP 测试客户端
func NewTCPTestClient(address string) *TCPTestClient {
	return &TCPTestClient{
		address: address,
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
	return nil
}

// Disconnect 断开连接
func (c *TCPTestClient) Disconnect() error {
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
func (c *TCPTestClient) SendMessage(user, message string) error {
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

// ReceiveMessage 接收消息
func (c *TCPTestClient) ReceiveMessage() (*ChatResponse, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return nil, fmt.Errorf("client is not connected")
	}

	// 设置读取超时
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	defer c.conn.SetReadDeadline(time.Time{}) // 清除超时

	// 直接从连接读取数据
	buffer := make([]byte, 4096)
	n, err := c.conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %v", err)
	}

	data := string(buffer[:n])
	
	// 处理连续的JSON对象，使用简单的括号匹配来分割
	var responses []ChatResponse
	decoder := json.NewDecoder(strings.NewReader(data))
	
	for decoder.More() {
		var response ChatResponse
		if err := decoder.Decode(&response); err == nil {
			responses = append(responses, response)
		}
	}
	
	if len(responses) == 0 {
		return nil, fmt.Errorf("no valid response found in data: %s", data)
	}
	
	// 返回第一个有效的响应
	return &responses[0], nil
}

// IsConnected 检查是否已连接
func (c *TCPTestClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// sendRequest 发送请求
func (c *TCPTestClient) sendRequest(method, path string, data interface{}) error {
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

// WebSocketTestClient WebSocket 测试客户端
type WebSocketTestClient struct {
	address   string
	conn      *websocket.Conn
	mu        sync.RWMutex
	connected bool
}

// NewWebSocketTestClient 创建新的 WebSocket 测试客户端
func NewWebSocketTestClient(address string) *WebSocketTestClient {
	return &WebSocketTestClient{
		address: address,
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
	return c.connected
}

// sendRequest 发送请求
func (c *WebSocketTestClient) sendRequest(method, path string, data interface{}) error {
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

	err = c.conn.WriteMessage(websocket.TextMessage, requestData)
	if err != nil {
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
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}
