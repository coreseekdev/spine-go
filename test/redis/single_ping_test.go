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

// TestSinglePing 测试单个PING命令
func TestSinglePing(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "single-ping-test-*")
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

	// 启动服务器 - 只处理一个连接
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// 设置连接超时
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

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

	// 测试单个PING命令
	result, err := client.Ping()
	if err != nil {
		t.Fatalf("PING failed: %v", err)
	}
	if result != "PONG" {
		t.Fatalf("Expected PONG, got %s", result)
	}
	t.Logf("Single PING test passed: %s", result)

	// 立即断开连接
	client.Disconnect()
	t.Logf("Test completed successfully")
}
