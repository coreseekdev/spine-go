package redis

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// RedisTestClient Redis 测试客户端
type RedisTestClient struct {
	address string
	conn    net.Conn
	reader  *bufio.Reader
}

// NewRedisTestClient 创建新的 Redis 测试客户端
func NewRedisTestClient(address string) *RedisTestClient {
	return &RedisTestClient{
		address: address,
	}
}

// Connect 连接到 Redis 服务器
func (c *RedisTestClient) Connect() error {
	conn, err := net.DialTimeout("tcp", c.address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", c.address, err)
	}

	c.conn = conn
	c.reader = bufio.NewReader(conn)
	return nil
}

// Disconnect 断开连接
func (c *RedisTestClient) Disconnect() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SendCommand 发送 Redis 命令
func (c *RedisTestClient) SendCommand(args ...string) (interface{}, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	// 构建 RESP 命令
	cmd := fmt.Sprintf("*%d\r\n", len(args))
	for _, arg := range args {
		cmd += fmt.Sprintf("$%d\r\n%s\r\n", len(arg), arg)
	}

	// 发送命令
	_, err := c.conn.Write([]byte(cmd))
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	// 读取响应
	return c.readResponse()
}

// readResponse 读取 RESP 响应
func (c *RedisTestClient) readResponse() (interface{}, error) {
	line, err := c.readLine()
	if err != nil {
		return nil, err
	}

	if len(line) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	switch line[0] {
	case '+': // Simple String
		return line[1:], nil
	case '-': // Error
		return nil, fmt.Errorf("redis error: %s", line[1:])
	case ':': // Integer
		return strconv.ParseInt(line[1:], 10, 64)
	case '$': // Bulk String
		return c.readBulkString(line[1:])
	case '*': // Array
		return c.readArray(line[1:])
	default:
		return nil, fmt.Errorf("unknown response type: %c", line[0])
	}
}

// readLine 读取一行
func (c *RedisTestClient) readLine() (string, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(line, "\r\n"), nil
}

// readBulkString 读取批量字符串
func (c *RedisTestClient) readBulkString(lengthStr string) (interface{}, error) {
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid bulk string length: %s", lengthStr)
	}

	if length == -1 {
		return nil, nil // NULL
	}

	if length == 0 {
		// 读取 CRLF
		c.readLine()
		return "", nil
	}

	// 读取字符串内容
	content := make([]byte, length)
	_, err = io.ReadFull(c.reader, content)
	if err != nil {
		return nil, err
	}

	// 读取 CRLF
	c.readLine()

	return string(content), nil
}

// readArray 读取数组
func (c *RedisTestClient) readArray(lengthStr string) (interface{}, error) {
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid array length: %s", lengthStr)
	}

	if length == -1 {
		return nil, nil // NULL array
	}

	array := make([]interface{}, length)
	for i := 0; i < length; i++ {
		element, err := c.readResponse()
		if err != nil {
			return nil, err
		}
		array[i] = element
	}

	return array, nil
}

// Ping 发送 PING 命令
func (c *RedisTestClient) Ping() (string, error) {
	result, err := c.SendCommand("PING")
	if err != nil {
		return "", err
	}
	if str, ok := result.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("unexpected response type for PING")
}

// Set 设置键值
func (c *RedisTestClient) Set(key, value string) error {
	_, err := c.SendCommand("SET", key, value)
	return err
}

// SetEX 设置带过期时间的键值
func (c *RedisTestClient) SetEX(key, value string, seconds int64) error {
	_, err := c.SendCommand("SET", key, value, "EX", fmt.Sprintf("%d", seconds))
	return err
}

// Get 获取键值
func (c *RedisTestClient) Get(key string) (string, error) {
	result, err := c.SendCommand("GET", key)
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", fmt.Errorf("key not found")
	}
	if str, ok := result.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("unexpected response type for GET")
}

// Del 删除键
func (c *RedisTestClient) Del(key string) (int64, error) {
	result, err := c.SendCommand("DEL", key)
	if err != nil {
		return 0, err
	}
	if count, ok := result.(int64); ok {
		return count, nil
	}
	return 0, fmt.Errorf("unexpected response type for DEL")
}

// Exists 检查键是否存在
func (c *RedisTestClient) Exists(key string) (bool, error) {
	result, err := c.SendCommand("EXISTS", key)
	if err != nil {
		return false, err
	}
	if count, ok := result.(int64); ok {
		return count > 0, nil
	}
	return false, fmt.Errorf("unexpected response type for EXISTS")
}

// TTL 获取键的过期时间
func (c *RedisTestClient) TTL(key string) (int64, error) {
	result, err := c.SendCommand("TTL", key)
	if err != nil {
		return 0, err
	}
	if ttl, ok := result.(int64); ok {
		return ttl, nil
	}
	return 0, fmt.Errorf("unexpected response type for TTL")
}
