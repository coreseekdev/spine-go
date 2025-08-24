package e2e

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

// E2ETestSuite E2E 测试套件
type E2ETestSuite struct {
	serverManager     *TestServerManager
	clientFactory     *TestClientFactory
	messageValidator  *MessageValidator
	responseValidator *ResponseValidator
	connectionValidator *ConnectionValidator
	clients           map[string]TestClient
	mu                sync.RWMutex
}

// NewE2ETestSuite 创建新的 E2E 测试套件
func NewE2ETestSuite() *E2ETestSuite {
	return &E2ETestSuite{
		serverManager:       NewTestServerManager(),
		clientFactory:       NewTestClientFactory(),
		messageValidator:    NewMessageValidator(),
		responseValidator:   NewResponseValidator(),
		connectionValidator: NewConnectionValidator(),
		clients:             make(map[string]TestClient),
	}
}

// SetupTest 设置测试环境
func (suite *E2ETestSuite) SetupTest(protocols []string) error {
	// 启动测试服务器
	if err := suite.serverManager.StartServer(protocols); err != nil {
		return fmt.Errorf("failed to start test server: %v", err)
	}

	// 等待服务器完全启动
	time.Sleep(100 * time.Millisecond)
	return nil
}

// TeardownTest 清理测试环境
func (suite *E2ETestSuite) TeardownTest() error {
	// 断开所有客户端连接
	suite.mu.Lock()
	for name, client := range suite.clients {
		if client.IsConnected() {
			client.Disconnect()
		}
		delete(suite.clients, name)
	}
	suite.mu.Unlock()

	// 停止测试服务器
	if err := suite.serverManager.StopServer(); err != nil {
		return fmt.Errorf("failed to stop test server: %v", err)
	}

	// 清空验证器
	suite.messageValidator.Clear()
	return nil
}

// CreateClient 创建并连接客户端
func (suite *E2ETestSuite) CreateClient(name, protocol string) error {
	address, err := suite.serverManager.GetServerAddress(protocol)
	if err != nil {
		return fmt.Errorf("failed to get server address for %s: %v", protocol, err)
	}

	client, err := suite.clientFactory.CreateClient(protocol, address)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect client: %v", err)
	}

	suite.mu.Lock()
	suite.clients[name] = client
	suite.mu.Unlock()

	return nil
}

// GetClient 获取客户端
func (suite *E2ETestSuite) GetClient(name string) (TestClient, error) {
	suite.mu.RLock()
	defer suite.mu.RUnlock()

	client, exists := suite.clients[name]
	if !exists {
		return nil, fmt.Errorf("client %s not found", name)
	}
	return client, nil
}

// RunBasicChatTest 运行基本聊天测试
func (suite *E2ETestSuite) RunBasicChatTest(t *testing.T, protocol string) {
	// 设置测试环境
	if err := suite.SetupTest([]string{protocol}); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	defer suite.TeardownTest()

	// 创建客户端
	if err := suite.CreateClient("client1", protocol); err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client1, _ := suite.GetClient("client1")

	// 加入聊天
	if err := client1.JoinChat(); err != nil {
		t.Fatalf("Failed to join chat: %v", err)
	}

	// 发送消息
	if err := client1.SendMessage("testuser", "hello world"); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// 验证连接状态
	if err := suite.connectionValidator.ValidateConnection(client1, true); err != nil {
		t.Fatalf("Connection validation failed: %v", err)
	}

	t.Logf("Basic chat test passed for protocol: %s", protocol)
}

// RunMultiClientBroadcastTest 运行多客户端广播测试
func (suite *E2ETestSuite) RunMultiClientBroadcastTest(t *testing.T, protocol string) {
	// 设置测试环境
	if err := suite.SetupTest([]string{protocol}); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	defer suite.TeardownTest()

	// 创建多个客户端
	clientNames := []string{"client1", "client2", "client3"}
	for _, name := range clientNames {
		if err := suite.CreateClient(name, protocol); err != nil {
			t.Fatalf("Failed to create client %s: %v", name, err)
		}
	}

	// 所有客户端加入聊天
	for _, name := range clientNames {
		client, _ := suite.GetClient(name)
		if err := client.JoinChat(); err != nil {
			t.Fatalf("Failed to join chat for %s: %v", name, err)
		}
	}

	// 等待连接稳定
	time.Sleep(100 * time.Millisecond)

	// 验证服务器连接数
	if err := suite.connectionValidator.ValidateServerConnections(suite.serverManager, len(clientNames)); err != nil {
		t.Fatalf("Server connection validation failed: %v", err)
	}

	// 启动消息接收 goroutines，在发送消息之前开始监听
	var wg sync.WaitGroup
	messageReceived := make(chan string, len(clientNames))

	for _, name := range clientNames {
		wg.Add(1)
		go func(clientName string) {
			defer wg.Done()
			client, _ := suite.GetClient(clientName)
			
			// 持续监听消息
			for {
				response, err := client.ReceiveMessage()
				if err != nil {
					t.Logf("Client %s receive error: %v", clientName, err)
					return
				}
				
				if msg, err := suite.responseValidator.ValidateMessageResponse(response); err == nil {
					// 只记录广播消息，忽略其他响应（如JOIN的响应）
					if msg.User == "user1" && msg.Message == "broadcast test message" {
						suite.messageValidator.RecordMessage(msg.User, msg.Message, clientName, msg.Timestamp)
						messageReceived <- clientName
						return
					}
				}
			}
		}(name)
	}

	// 等待接收器启动
	time.Sleep(200 * time.Millisecond)

	// client1 发送消息
	client1, _ := suite.GetClient("client1")
	if err := client1.SendMessage("user1", "broadcast test message"); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// 等待所有消息接收完成或超时
	go func() {
		wg.Wait()
		close(messageReceived)
	}()

	receivedClients := make(map[string]bool)
	timeout := time.After(5 * time.Second)
	for {
		select {
		case clientName := <-messageReceived:
			if clientName != "" {
				receivedClients[clientName] = true
				if len(receivedClients) >= len(clientNames) {
					goto validateMessages
				}
			}
		case <-timeout:
			t.Fatalf("Timeout: only received messages from %d/%d clients: %v", len(receivedClients), len(clientNames), receivedClients)
		}
	}

validateMessages:
	// 验证广播
	if err := suite.messageValidator.ValidateBroadcast(clientNames); err != nil {
		t.Fatalf("Broadcast validation failed: %v", err)
	}

	t.Logf("Multi-client broadcast test passed for protocol: %s", protocol)
}

// RunCrossProtocolTest 运行跨协议测试
func (suite *E2ETestSuite) RunCrossProtocolTest(t *testing.T) {
	protocols := []string{"tcp", "http"}
	
	// 设置测试环境
	if err := suite.SetupTest(protocols); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	defer suite.TeardownTest()

	// 创建不同协议的客户端
	if err := suite.CreateClient("tcp_client", "tcp"); err != nil {
		t.Fatalf("Failed to create TCP client: %v", err)
	}
	
	if err := suite.CreateClient("ws_client", "http"); err != nil {
		t.Fatalf("Failed to create WebSocket client: %v", err)
	}

	// 所有客户端加入聊天
	tcpClient, _ := suite.GetClient("tcp_client")
	wsClient, _ := suite.GetClient("ws_client")

	if err := tcpClient.JoinChat(); err != nil {
		t.Fatalf("Failed to join chat for TCP client: %v", err)
	}
	
	if err := wsClient.JoinChat(); err != nil {
		t.Fatalf("Failed to join chat for WebSocket client: %v", err)
	}

	// 等待连接稳定
	time.Sleep(100 * time.Millisecond)

	// TCP 客户端发送消息
	if err := tcpClient.SendMessage("tcp_user", "cross protocol message"); err != nil {
		t.Fatalf("Failed to send message from TCP client: %v", err)
	}

	// WebSocket 客户端接收消息，可能需要跳过JOIN响应
	timeout := time.After(5 * time.Second)
	for {
		select {
		case <-timeout:
			t.Fatalf("Timeout waiting for cross-protocol message")
		default:
			if response, err := wsClient.ReceiveMessage(); err == nil {
				if msg, err := suite.responseValidator.ValidateMessageResponse(response); err == nil {
					// 跳过JOIN响应，只处理广播消息
					if msg.User == "tcp_user" && msg.Message == "cross protocol message" {
						t.Logf("Cross-protocol message received successfully: %+v", msg)
						goto testPassed
					}
					// 继续等待下一个消息
					continue
				}
			} else {
				t.Fatalf("Failed to receive cross-protocol message: %v", err)
			}
		}
	}

testPassed:

	t.Logf("Cross-protocol test passed")
}

// 具体的测试函数

// TestTCPBasicChat 测试 TCP 基本聊天功能
func TestTCPBasicChat(t *testing.T) {
	suite := NewE2ETestSuite()
	suite.RunBasicChatTest(t, "tcp")
}

// TestWebSocketBasicChat 测试 WebSocket 基本聊天功能
func TestWebSocketBasicChat(t *testing.T) {
	suite := NewE2ETestSuite()
	suite.RunBasicChatTest(t, "http")
}

// TestTCPMultiClientBroadcast 测试 TCP 多客户端广播
func TestTCPMultiClientBroadcast(t *testing.T) {
	suite := NewE2ETestSuite()
	suite.RunMultiClientBroadcastTest(t, "tcp")
}

// TestWebSocketMultiClientBroadcast 测试 WebSocket 多客户端广播
func TestWebSocketMultiClientBroadcast(t *testing.T) {
	suite := NewE2ETestSuite()
	suite.RunMultiClientBroadcastTest(t, "http")
}

// TestCrossProtocolCommunication 测试跨协议通信
func TestCrossProtocolCommunication(t *testing.T) {
	suite := NewE2ETestSuite()
	suite.RunCrossProtocolTest(t)
}

// TestConnectionManagement 测试连接管理
func TestConnectionManagement(t *testing.T) {
	suite := NewE2ETestSuite()
	suite.RunConnectionManagementTest(t, "tcp")
}

// TestServerGracefulShutdown 测试服务器优雅关闭
func TestServerGracefulShutdown(t *testing.T) {
	suite := NewE2ETestSuite()
	suite.RunGracefulShutdownTest(t, "tcp")
}

// TestWebSocketServerGracefulShutdown 测试WebSocket服务器优雅关闭
func TestWebSocketServerGracefulShutdown(t *testing.T) {
	suite := NewE2ETestSuite()
	suite.RunGracefulShutdownTest(t, "http")
}

// TestUnixSocketBasicChat 测试 Unix Socket 基本聊天功能（仅在 Unix 系统上运行）
func TestUnixSocketBasicChat(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket is not supported on Windows")
	}
	suite := NewE2ETestSuite()
	suite.RunBasicChatTest(t, "unix")
}

// TestUnixSocketMultiClientBroadcast 测试 Unix Socket 多客户端广播（仅在 Unix 系统上运行）
func TestUnixSocketMultiClientBroadcast(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket is not supported on Windows")
	}
	suite := NewE2ETestSuite()
	suite.RunMultiClientBroadcastTest(t, "unix")
}

// TestNamedPipeBasicChat 测试 Named Pipe 基本聊天功能（仅在 Windows 上运行）
func TestNamedPipeBasicChat(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Named pipe is only supported on Windows")
	}
	suite := NewE2ETestSuite()
	suite.RunBasicChatTest(t, "namedpipe")
}

// TestNamedPipeMultiClientBroadcast 测试 Named Pipe 多客户端广播（仅在 Windows 上运行）
func TestNamedPipeMultiClientBroadcast(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Named pipe is only supported on Windows")
	}
	suite := NewE2ETestSuite()
	suite.RunMultiClientBroadcastTest(t, "namedpipe")
}

// TestNamedPipeConcurrentConnections 测试 Named Pipe 并发连接（仅在 Windows 上运行）
func TestNamedPipeConcurrentConnections(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Named pipe is only supported on Windows")
	}
	suite := NewE2ETestSuite()
	suite.RunNamedPipeConcurrentConnectionsTest(t)
}

// TestNamedPipeConnectionManagement 测试 Named Pipe 连接管理（仅在 Windows 上运行）
func TestNamedPipeConnectionManagement(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Named pipe is only supported on Windows")
	}
	suite := NewE2ETestSuite()
	suite.RunConnectionManagementTest(t, "namedpipe")
}

// RunConnectionManagementTest 运行连接管理测试
func (suite *E2ETestSuite) RunConnectionManagementTest(t *testing.T, protocol string) {
	// 设置测试环境
	if err := suite.SetupTest([]string{protocol}); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	defer suite.TeardownTest()

	// 创建客户端并连接
	if err := suite.CreateClient("client1", protocol); err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	client1, _ := suite.GetClient("client1")

	// 验证连接状态
	if err := suite.connectionValidator.ValidateConnection(client1, true); err != nil {
		t.Fatalf("Connection validation failed: %v", err)
	}

	// 加入聊天
	if err := client1.JoinChat(); err != nil {
		t.Fatalf("Failed to join chat: %v", err)
	}

	// 等待JOIN请求处理完成
	time.Sleep(50 * time.Millisecond)

	// 验证服务器连接数
	if err := suite.connectionValidator.ValidateServerConnections(suite.serverManager, 1); err != nil {
		t.Fatalf("Server connection validation failed: %v", err)
	}

	// 断开连接
	if err := client1.Disconnect(); err != nil {
		t.Fatalf("Failed to disconnect client: %v", err)
	}

	// 验证连接状态
	if err := suite.connectionValidator.ValidateConnection(client1, false); err != nil {
		t.Fatalf("Disconnection validation failed: %v", err)
	}

	// 等待服务器清理连接
	time.Sleep(200 * time.Millisecond)

	// 验证服务器连接数
	if err := suite.connectionValidator.ValidateServerConnections(suite.serverManager, 0); err != nil {
		t.Fatalf("Server connection cleanup validation failed: %v", err)
	}

	t.Logf("Connection management test passed")
}

// RunGracefulShutdownTest 运行优雅关闭测试
func (suite *E2ETestSuite) RunGracefulShutdownTest(t *testing.T, protocol string) {
	// 设置测试环境
	if err := suite.SetupTest([]string{protocol}); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}

	// 创建多个客户端并连接
	clientNames := []string{"client1", "client2", "client3"}
	for _, name := range clientNames {
		if err := suite.CreateClient(name, protocol); err != nil {
			t.Fatalf("Failed to create client %s: %v", name, err)
		}
		
		client, _ := suite.GetClient(name)
		if err := client.JoinChat(); err != nil {
			t.Fatalf("Failed to join chat for %s: %v", name, err)
		}
	}

	// 等待所有客户端连接完成
	time.Sleep(100 * time.Millisecond)
	
	// 验证所有客户端都已连接
	for _, name := range clientNames {
		client, _ := suite.GetClient(name)
		if err := suite.connectionValidator.ValidateConnection(client, true); err != nil {
			t.Fatalf("Connection validation failed for %s: %v", name, err)
		}
	}

	t.Logf("All clients connected successfully")

	// 记录关闭开始时间
	shutdownStart := time.Now()

	// 关闭服务器（不调用TeardownTest，因为我们要测试关闭行为）
	if err := suite.serverManager.Stop(); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	shutdownDuration := time.Since(shutdownStart)
	t.Logf("Server shutdown took: %v", shutdownDuration)

	// 验证服务器在合理时间内关闭（应该主动关闭客户端连接，而不是等待）
	maxShutdownTime := 2 * time.Second
	if shutdownDuration > maxShutdownTime {
		t.Fatalf("Server took too long to shutdown: %v (expected < %v)", shutdownDuration, maxShutdownTime)
	}

	// 验证服务器能够快速关闭，说明主动关闭了所有连接
	t.Logf("Graceful shutdown test passed - server closed in %v", shutdownDuration)
	
	// 验证客户端在尝试发送消息时会检测到连接已断开
	time.Sleep(100 * time.Millisecond)
	for i, name := range clientNames {
		client, _ := suite.GetClient(name)
		
		// 检查连接状态或尝试发送消息
		if client.IsConnected() {
			// 如果客户端认为还连接着，尝试发送消息应该失败
			err := client.SendMessage(fmt.Sprintf("user%d", i+1), "test after shutdown")
			if err == nil {
				t.Fatalf("Client %s should fail to send message after server shutdown", name)
			}
			t.Logf("Client %s correctly detected disconnection: %v", name, err)
		} else {
			// 客户端已经检测到断开连接
			t.Logf("Client %s correctly detected disconnection: client is not connected", name)
		}
	}
}

// RunNamedPipeConcurrentConnectionsTest 运行 Named Pipe 并发连接测试
func (suite *E2ETestSuite) RunNamedPipeConcurrentConnectionsTest(t *testing.T) {
	// 设置测试环境
	if err := suite.SetupTest([]string{"namedpipe"}); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	defer suite.TeardownTest()

	// 测试大量并发连接
	clientCount := 10
	clientNames := make([]string, clientCount)
	for i := 0; i < clientCount; i++ {
		clientNames[i] = fmt.Sprintf("client%d", i+1)
	}

	// 并发创建和连接客户端
	var wg sync.WaitGroup
	errorChan := make(chan error, clientCount)

	for _, name := range clientNames {
		wg.Add(1)
		go func(clientName string) {
			defer wg.Done()
			if err := suite.CreateClient(clientName, "namedpipe"); err != nil {
				errorChan <- fmt.Errorf("failed to create client %s: %v", clientName, err)
				return
			}
			
			client, _ := suite.GetClient(clientName)
			if err := client.JoinChat(); err != nil {
				errorChan <- fmt.Errorf("failed to join chat for %s: %v", clientName, err)
				return
			}
		}(name)
	}

	// 等待所有客户端连接完成
	wg.Wait()
	close(errorChan)

	// 检查是否有错误
	for err := range errorChan {
		if err != nil {
			t.Fatalf("Concurrent connection error: %v", err)
		}
	}

	// 等待连接稳定
	time.Sleep(200 * time.Millisecond)

	// 验证所有客户端都已连接
	for _, name := range clientNames {
		client, _ := suite.GetClient(name)
		if err := suite.connectionValidator.ValidateConnection(client, true); err != nil {
			t.Fatalf("Connection validation failed for %s: %v", name, err)
		}
	}

	// 验证服务器连接数
	if err := suite.connectionValidator.ValidateServerConnections(suite.serverManager, clientCount); err != nil {
			t.Fatalf("Server connection validation failed: %v", err)
		}

	// 测试并发消息发送
	messageCount := 5
	var messageWg sync.WaitGroup
	messageErrorChan := make(chan error, clientCount*messageCount)

	for i, name := range clientNames {
		messageWg.Add(1)
		go func(clientIndex int, clientName string) {
			defer messageWg.Done()
			client, _ := suite.GetClient(clientName)
			
			for j := 0; j < messageCount; j++ {
				message := fmt.Sprintf("concurrent message %d from %s", j+1, clientName)
				if err := client.SendMessage(fmt.Sprintf("user%d", clientIndex+1), message); err != nil {
					messageErrorChan <- fmt.Errorf("failed to send message from %s: %v", clientName, err)
					return
				}
				// 短暂延迟避免过快发送
				time.Sleep(10 * time.Millisecond)
			}
		}(i, name)
	}

	// 等待所有消息发送完成
	messageWg.Wait()
	close(messageErrorChan)

	// 检查消息发送是否有错误
	for err := range messageErrorChan {
		if err != nil {
			t.Fatalf("Concurrent message sending error: %v", err)
		}
	}

	// 验证所有客户端仍然连接
	for _, name := range clientNames {
		client, _ := suite.GetClient(name)
		if err := suite.connectionValidator.ValidateConnection(client, true); err != nil {
			t.Fatalf("Connection validation after messaging failed for %s: %v", name, err)
		}
	}

	t.Logf("Named Pipe concurrent connections test passed with %d clients", clientCount)
}
