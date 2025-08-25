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

// TestSimpleRedisConnection 简单的Redis连接测试
func TestSimpleRedisConnection(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "simple-redis-test-*")
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
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			go func(c net.Conn) {
				defer c.Close()
				
				// 设置连接超时
				c.SetReadDeadline(time.Now().Add(5 * time.Second))
				c.SetWriteDeadline(time.Now().Add(5 * time.Second))

				// 创建读写器
				reader := &transport.TCPReader{Conn: c}
				writer := &transport.TCPWriter{Conn: c}

				// 创建连接信息
				connInfo := &transport.ConnInfo{
					ID:       fmt.Sprintf("test-%d", time.Now().UnixNano()),
					Remote:   c.RemoteAddr(),
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
			}(conn)
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

	// 测试PING命令
	t.Run("PING", func(t *testing.T) {
		result, err := client.Ping()
		if err != nil {
			t.Fatalf("PING failed: %v", err)
		}
		if result != "PONG" {
			t.Fatalf("Expected PONG, got %s", result)
		}
		t.Logf("PING test passed: %s", result)
	})

	// 测试SET/GET命令
	t.Run("SET/GET", func(t *testing.T) {
		// SET
		if err := client.Set("test_key", "test_value"); err != nil {
			t.Fatalf("SET failed: %v", err)
		}
		t.Logf("SET test_key test_value - OK")

		// GET
		value, err := client.Get("test_key")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		if value != "test_value" {
			t.Fatalf("Expected test_value, got %s", value)
		}
		t.Logf("GET test_key - %s", value)
	})

	// 关闭服务器
	listener.Close()
	
	// 等待服务器关闭
	select {
	case <-serverDone:
		t.Logf("Server closed successfully")
	case <-time.After(2 * time.Second):
		t.Logf("Server close timeout")
	}
}
