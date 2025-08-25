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

// RedisTestSuite Redis 测试套件
type RedisTestSuite struct {
	transport *transport.TCPTransport
	address   string
	client    *RedisTestClient
}

// NewRedisTestSuite 创建新的 Redis 测试套件
func NewRedisTestSuite() *RedisTestSuite {
	return &RedisTestSuite{}
}

// SetupTest 设置测试环境
func (suite *RedisTestSuite) SetupTest() error {
	// 创建监听器
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to create listener: %v", err)
	}
	suite.address = listener.Addr().String()

	// 创建 TCP 传输层（使用现有监听器）
	suite.transport = &transport.TCPTransport{}

	// 创建临时WAL目录
	tempDir, err := os.MkdirTemp("", "redis-test-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	
	walPath := filepath.Join(tempDir, "redis.wal")

	// 创建 Redis 处理器
	redisHandler, err := handler.NewRedisHandler(walPath)
	if err != nil {
		return fmt.Errorf("failed to create Redis handler: %v", err)
	}

	// 创建服务器上下文
	serverInfo := &transport.ServerInfo{
		Address: suite.address,
		Config:  make(map[string]interface{}),
	}
	serverCtx := transport.NewServerContext(serverInfo)
	serverCtx.SetHandler(redisHandler)

	// 手动启动服务器
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go suite.handleConnection(conn, serverCtx)
		}
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 创建客户端
	suite.client = NewRedisTestClient(suite.address)
	if err := suite.client.Connect(); err != nil {
		return fmt.Errorf("failed to connect client: %v", err)
	}

	return nil
}

// handleConnection 处理连接
func (suite *RedisTestSuite) handleConnection(conn net.Conn, serverCtx *transport.ServerContext) {
	defer conn.Close()

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

	// 创建上下文
	ctx := &transport.Context{
		ServerInfo:        serverCtx.ServerInfo,
		ConnInfo:          connInfo,
		ConnectionManager: serverCtx.Connections,
	}

	// 获取处理器并处理连接
	handler := serverCtx.GetHandler()
	if handler != nil {
		handler.Handle(ctx, reader, writer)
	}
}

// TeardownTest 清理测试环境
func (suite *RedisTestSuite) TeardownTest() {
	if suite.client != nil {
		suite.client.Disconnect()
	}
}

// TestBasicCommands 测试基本 Redis 命令
func TestBasicCommands(t *testing.T) {
	suite := NewRedisTestSuite()
	if err := suite.SetupTest(); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	defer suite.TeardownTest()

	// 测试 PING
	pong, err := suite.client.Ping()
	if err != nil {
		t.Fatalf("PING failed: %v", err)
	}
	if pong != "PONG" {
		t.Fatalf("Expected PONG, got %s", pong)
	}

	// 测试 SET
	if err := suite.client.Set("test_key", "test_value"); err != nil {
		t.Fatalf("SET failed: %v", err)
	}

	// 测试 GET
	value, err := suite.client.Get("test_key")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	if value != "test_value" {
		t.Fatalf("Expected test_value, got %s", value)
	}

	// 测试 EXISTS
	exists, err := suite.client.Exists("test_key")
	if err != nil {
		t.Fatalf("EXISTS failed: %v", err)
	}
	if !exists {
		t.Fatalf("Expected key to exist")
	}

	// 测试 DEL
	count, err := suite.client.Del("test_key")
	if err != nil {
		t.Fatalf("DEL failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("Expected 1 key deleted, got %d", count)
	}

	// 验证键已删除
	exists, err = suite.client.Exists("test_key")
	if err != nil {
		t.Fatalf("EXISTS failed: %v", err)
	}
	if exists {
		t.Fatalf("Expected key to not exist")
	}

	t.Log("Basic commands test passed")
}

// TestTTLCommands 测试 TTL 相关命令
func TestTTLCommands(t *testing.T) {
	suite := NewRedisTestSuite()
	if err := suite.SetupTest(); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	defer suite.TeardownTest()

	// 测试带 TTL 的 SET
	if err := suite.client.SetEX("ttl_key", "ttl_value", 2); err != nil {
		t.Fatalf("SETEX failed: %v", err)
	}

	// 验证键存在
	value, err := suite.client.Get("ttl_key")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	if value != "ttl_value" {
		t.Fatalf("Expected ttl_value, got %s", value)
	}

	// 测试 TTL
	ttl, err := suite.client.TTL("ttl_key")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}
	if ttl <= 0 || ttl > 2 {
		t.Fatalf("Expected TTL between 0 and 2, got %d", ttl)
	}

	// 等待过期
	time.Sleep(3 * time.Second)

	// 验证键已过期
	exists, err := suite.client.Exists("ttl_key")
	if err != nil {
		t.Fatalf("EXISTS failed: %v", err)
	}
	if exists {
		t.Fatalf("Expected key to be expired")
	}

	// 测试不存在键的 TTL
	ttl, err = suite.client.TTL("nonexistent_key")
	if err != nil {
		t.Fatalf("TTL failed: %v", err)
	}
	if ttl != -2 {
		t.Fatalf("Expected TTL -2 for nonexistent key, got %d", ttl)
	}

	t.Log("TTL commands test passed")
}

// TestMultipleKeys 测试多键操作
func TestMultipleKeys(t *testing.T) {
	suite := NewRedisTestSuite()
	if err := suite.SetupTest(); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	defer suite.TeardownTest()

	// 设置多个键
	keys := []string{"key1", "key2", "key3"}
	values := []string{"value1", "value2", "value3"}

	for i, key := range keys {
		if err := suite.client.Set(key, values[i]); err != nil {
			t.Fatalf("SET %s failed: %v", key, err)
		}
	}

	// 验证所有键都存在
	for _, key := range keys {
		exists, err := suite.client.Exists(key)
		if err != nil {
			t.Fatalf("EXISTS %s failed: %v", key, err)
		}
		if !exists {
			t.Fatalf("Expected key %s to exist", key)
		}
	}

	// 删除所有键
	for _, key := range keys {
		count, err := suite.client.Del(key)
		if err != nil {
			t.Fatalf("DEL %s failed: %v", key, err)
		}
		if count != 1 {
			t.Fatalf("Expected 1 key deleted for %s, got %d", key, count)
		}
	}

	// 验证所有键都已删除
	for _, key := range keys {
		exists, err := suite.client.Exists(key)
		if err != nil {
			t.Fatalf("EXISTS %s failed: %v", key, err)
		}
		if exists {
			t.Fatalf("Expected key %s to not exist", key)
		}
	}

	t.Log("Multiple keys test passed")
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	suite := NewRedisTestSuite()
	if err := suite.SetupTest(); err != nil {
		t.Fatalf("Failed to setup test: %v", err)
	}
	defer suite.TeardownTest()

	// 测试获取不存在的键
	_, err := suite.client.Get("nonexistent_key")
	if err == nil {
		t.Fatalf("Expected error for nonexistent key")
	}

	// 测试删除不存在的键
	count, err := suite.client.Del("nonexistent_key")
	if err != nil {
		t.Fatalf("DEL nonexistent key failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("Expected 0 keys deleted for nonexistent key, got %d", count)
	}

	// 测试无效命令（通过 SendCommand 发送）
	_, err = suite.client.SendCommand("INVALID_COMMAND")
	if err == nil {
		t.Fatalf("Expected error for invalid command")
	}

	t.Log("Error handling test passed")
}

// BenchmarkSetGet 性能测试
func BenchmarkSetGet(b *testing.B) {
	suite := NewRedisTestSuite()
	if err := suite.SetupTest(); err != nil {
		b.Fatalf("Failed to setup test: %v", err)
	}
	defer suite.TeardownTest()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench_key_%d", i)
		value := fmt.Sprintf("bench_value_%d", i)

		if err := suite.client.Set(key, value); err != nil {
			b.Fatalf("SET failed: %v", err)
		}

		if _, err := suite.client.Get(key); err != nil {
			b.Fatalf("GET failed: %v", err)
		}
	}
}
