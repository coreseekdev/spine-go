package redis

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"spine-go/libspine/handler"
	"spine-go/libspine/transport"
)

// TestMultiplePing 测试多个PING命令
func TestMultiplePing(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "multiple-ping-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	walPath := filepath.Join(tempDir, "redis.wal")

	// 创建Redis处理器
	redisHandler, err := handler.NewRedisHandler(walPath)
	if err != nil {
		t.Fatalf("Failed to create Redis handler: %v", err)
	}
	defer redisHandler.Close()

	// 创建监听器
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	address := listener.Addr().String()
	t.Logf("Server listening on %s", address)

	// 启动服务器
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// 设置连接超时
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

		// 创建读写器
		reader := &transport.TCPReader{Conn: conn}
		writer := &transport.TCPWriter{Conn: conn}

		// 创建连接信息
		connInfo := &transport.ConnInfo{
			ID:       fmt.Sprintf("test-%d", time.Now().UnixNano()),
			Remote:   conn.RemoteAddr(),
			Protocol: "tcp",
			Metadata: make(map[string]interface{}),
			Reader:   reader,
			Writer:   writer,
		}

		// 创建服务器信息
		serverInfo := &transport.ServerInfo{
			Address: address,
			Config:  make(map[string]interface{}),
		}

		// 创建上下文
		ctx := &transport.Context{
			ServerInfo: serverInfo,
			ConnInfo:   connInfo,
		}

		// 处理连接
		if err := redisHandler.Handle(ctx, reader, writer); err != nil {
			t.Logf("Handler error: %v", err)
		}
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 创建客户端连接
	client := NewRedisTestClient(address)
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Disconnect()

	// 测试多个PING命令
	for i := 0; i < 3; i++ {
		t.Logf("Sending PING #%d", i+1)
		result, err := client.Ping()
		if err != nil {
			t.Fatalf("PING #%d failed: %v", i+1, err)
		}
		if result != "PONG" {
			t.Fatalf("PING #%d: Expected PONG, got %s", i+1, result)
		}
		t.Logf("PING #%d passed: %s", i+1, result)
		
		// 短暂延迟
		time.Sleep(10 * time.Millisecond)
	}

	// 断开连接
	client.Disconnect()
	t.Logf("Multiple PING test completed successfully")
}
