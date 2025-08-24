//go:build windows

package e2e

import (
	"encoding/json"
	"testing"
	"time"
)

// TestNamedPipeDebug 调试 Named Pipe 连接问题
func TestNamedPipeDebug(t *testing.T) {
	// 创建测试套件
	suite := NewE2ETestSuite()
	
	// 设置测试环境
	if err := suite.SetupTest([]string{"namedpipe"}); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	defer suite.TeardownTest()

	// 等待服务器完全启动
	time.Sleep(500 * time.Millisecond)
	
	// 获取服务器地址
	address, err := suite.serverManager.GetServerAddress("namedpipe")
	if err != nil {
		t.Fatalf("Failed to get server address: %v", err)
	}
	
	t.Logf("Connecting to named pipe: %s", address)
	
	// 创建客户端
	client := NewNamedPipeTestClient(address)
	
	// 连接到服务器
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()
	
	t.Logf("Connected successfully")
	
	// 等待连接稳定
	time.Sleep(100 * time.Millisecond)
	
	// 验证连接状态
	if !client.IsConnected() {
		t.Fatalf("Client should be connected")
	}
	
	t.Logf("Connection verified")
	
	// 尝试发送一个简单的请求
	request := ChatRequest{
		Method: "JOIN",
		Path:   "/chat",
		Data:   nil,
	}
	
	requestData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}
	
	t.Logf("Sending request: %s", string(requestData))
	
	// 使用 goroutine 发送数据以避免阻塞
	done := make(chan error, 1)
	go func() {
		err := client.writeData(append(requestData, '\n'))
		done <- err
	}()
	
	// 等待发送完成或超时
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		t.Logf("Request sent successfully")
	case <-time.After(5 * time.Second):
		t.Fatalf("Timeout sending request")
	}
	
	// 等待响应
	time.Sleep(1 * time.Second)
	
	// 尝试读取响应
	buffer := make([]byte, 1024)
	n, err := client.readData(buffer)
	if err != nil {
		t.Logf("Read error (expected): %v", err)
	} else {
		t.Logf("Received response: %s", string(buffer[:n]))
	}
	
	t.Logf("Debug test completed")
}