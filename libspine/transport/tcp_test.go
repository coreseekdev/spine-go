package transport

import (
	"fmt"
	"net"
	"testing"
	"time"
)

func TestTCPTransport_NewAndStart(t *testing.T) {
	// 创建一个临时端口进行测试
	transport, err := NewTCPTransport("localhost:0")
	if err != nil {
		t.Fatalf("Failed to create TCP transport: %v", err)
	}
	defer transport.Stop()

	// 创建服务器上下文
	serverInfo := &ServerInfo{
		Address: "localhost:0",
		Config:  make(map[string]interface{}),
	}
	serverCtx := NewServerContext(serverInfo)

	// 启动传输层
	err = transport.Start(serverCtx)
	if err != nil {
		t.Fatalf("Failed to start TCP transport: %v", err)
	}

	// 给一点时间让服务器启动
	time.Sleep(10 * time.Millisecond)
}

func TestUnixSocketTransport_NewAndStart(t *testing.T) {
	// 创建一个临时 socket 路径
	socketPath := "/tmp/test_spine_" + time.Now().Format("20060102150405") + ".sock"

	transport, err := NewUnixSocketTransport(socketPath)
	if err != nil {
		t.Fatalf("Failed to create Unix socket transport: %v", err)
	}
	defer transport.Stop()

	// 创建服务器上下文
	serverInfo := &ServerInfo{
		Address: socketPath,
		Config:  make(map[string]interface{}),
	}
	serverCtx := NewServerContext(serverInfo)

	// 启动传输层
	err = transport.Start(serverCtx)
	if err != nil {
		t.Fatalf("Failed to start Unix socket transport: %v", err)
	}

	// 给一点时间让服务器启动
	time.Sleep(10 * time.Millisecond)
}

func TestTransportInterface_Implementation(t *testing.T) {
	// 测试 TCP 传输实现接口
	var transportInterface Transport

	// 创建模拟 TCP 传输来测试接口
	tcpTransport := &TCPTransport{}
	transportInterface = tcpTransport

	// 验证接口方法存在
	if transportInterface == nil {
		t.Fatal("TCP transport does not implement Transport interface")
	}

	// 测试 Unix Socket 传输实现接口
	unixTransport := &UnixSocketTransport{}
	transportInterface = unixTransport

	if transportInterface == nil {
		t.Fatal("Unix socket transport does not implement Transport interface")
	}
}

func TestTransportHandlerCreation(t *testing.T) {
	// 创建一个模拟连接
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	// 测试 TCP 处理器创建
	tcpTransport := &TCPTransport{}
	reader, writer := tcpTransport.NewHandlers(server)

	if reader == nil {
		t.Fatal("TCP reader should not be nil")
	}
	if writer == nil {
		t.Fatal("TCP writer should not be nil")
	}

	// 测试 WebSocket 处理器创建
	wsTransport := &WebSocketTransport{}
	reader, writer = wsTransport.NewHandlers(server)

	// WebSocket 使用不同的处理方式，可能返回 nil
	_ = reader
	_ = writer

	// 测试 Unix Socket 处理器创建
	unixTransport := &UnixSocketTransport{}
	reader, writer = unixTransport.NewHandlers(server)

	if reader == nil {
		t.Fatal("Unix socket reader should not be nil")
	}
	if writer == nil {
		t.Fatal("Unix socket writer should not be nil")
	}
}

func TestTransportRoundTrip(t *testing.T) {
	// 创建一个模拟连接
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	done := make(chan bool)
	errChan := make(chan error, 1)

	// 客户端 goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("panic in client goroutine: %v", r)
				return
			}
		}()

		// 发送测试数据
		testData := []byte("Hello, World!")
		_, err := client.Write(testData)
		if err != nil {
			errChan <- fmt.Errorf("failed to write test data: %v", err)
			return
		}

		// 等待响应
		response := make([]byte, 1024)
		n, err := client.Read(response)
		if err != nil {
			errChan <- fmt.Errorf("failed to read response: %v", err)
			return
		}

		// 验证响应
		if string(response[:n]) != `{"status":"success"}` {
			errChan <- fmt.Errorf("expected response 'success', got %s", string(response[:n]))
			return
		}

		done <- true
	}()

	// 服务器端处理
	reader := &TCPReader{Conn: server}
	writer := &TCPWriter{Conn: server}

	// 读取客户端数据
	data, err := reader.Read()
	if err != nil {
		t.Fatalf("Failed to read data: %v", err)
	}

	// 验证数据
	if string(data) != "Hello, World!" {
		t.Errorf("Expected 'Hello, World!', got %s", string(data))
	}

	// 发送响应
	response := []byte(`{"status":"success"}`)
	err = writer.Write(response)
	if err != nil {
		t.Fatalf("Failed to write response: %v", err)
	}

	// 等待客户端完成，带超时
	select {
	case <-done:
		// 测试成功完成
	case err := <-errChan:
		t.Fatalf("Client goroutine error: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out waiting for client to complete")
	}
}
