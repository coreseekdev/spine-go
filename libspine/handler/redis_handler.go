package handler

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"spine-go/libspine/transport"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RedisItem 存储项结构
type RedisItem struct {
	Value     string
	ExpiresAt *time.Time
}

// RedisHandler Redis 处理器 - 使用内存数据库和 RESP 协议
type RedisHandler struct {
	store map[string]*RedisItem
	mu    sync.RWMutex
}

// NewRedisHandler 创建新的 Redis 处理器
func NewRedisHandler() *RedisHandler {
	return &RedisHandler{
		store: make(map[string]*RedisItem),
	}
}

// Handle 处理 Redis 请求 - 使用 RESP 协议
func (h *RedisHandler) Handle(ctx *transport.Context, req transport.Reader, res transport.Writer) error {
	// 使用 ConnInfo 中的 Reader 和 Writer
	if ctx.ConnInfo != nil {
		if ctx.ConnInfo.Reader != nil {
			req = ctx.ConnInfo.Reader
		}
		if ctx.ConnInfo.Writer != nil {
			res = ctx.ConnInfo.Writer
		}
	}

	// 创建 RESP 解析器
	reader := bufio.NewReader(req)

	// 持续处理消息直到连接关闭
	for {
		// 解析 RESP 命令
		command, err := h.parseRESPCommand(reader)
		if err != nil {
			// 连接关闭或读取错误
			if err == io.EOF {
				return nil
			}
			log.Printf("Error parsing RESP command: %v", err)
			h.writeRESPError(res, fmt.Sprintf("ERR %v", err))
			continue
		}

		log.Printf("Received Redis command: %v", command)

		// 处理命令
		if err := h.handleCommand(command, res); err != nil {
			log.Printf("Error handling Redis command: %v", err)
		}
	}
}

// parseRESPCommand 解析 RESP 命令
func (h *RedisHandler) parseRESPCommand(reader *bufio.Reader) ([]string, error) {
	// 读取第一行，应该是数组标识符 *
	lineBytes, isPrefix, err := reader.ReadLine()
	if err != nil {
		return nil, err
	}
	if isPrefix {
		return nil, fmt.Errorf("line too long")
	}

	lineStr := string(lineBytes)
	if len(lineStr) == 0 || lineStr[0] != '*' {
		return nil, fmt.Errorf("expected array, got: %s", lineStr)
	}

	// 解析数组长度
	arrayLen, err := strconv.Atoi(lineStr[1:])
	if err != nil {
		return nil, fmt.Errorf("invalid array length: %s", lineStr[1:])
	}

	// 读取数组元素
	command := make([]string, arrayLen)
	for i := 0; i < arrayLen; i++ {
		// 读取批量字符串长度行
		lineBytes, isPrefix, err := reader.ReadLine()
		if err != nil {
			return nil, err
		}
		if isPrefix {
			return nil, fmt.Errorf("line too long")
		}

		lengthStr := string(lineBytes)
		if len(lengthStr) == 0 || lengthStr[0] != '$' {
			return nil, fmt.Errorf("expected bulk string, got: %s", lengthStr)
		}

		// 解析字符串长度
		strLen, err := strconv.Atoi(lengthStr[1:])
		if err != nil {
			return nil, fmt.Errorf("invalid string length: %s", lengthStr[1:])
		}

		// 读取字符串内容
		content := make([]byte, strLen)
		_, err = io.ReadFull(reader, content)
		if err != nil {
			return nil, err
		}

		// 读取 CRLF
		reader.ReadLine()

		command[i] = string(content)
	}

	return command, nil
}

// handleCommand 处理 Redis 命令
func (h *RedisHandler) handleCommand(command []string, res transport.Writer) error {
	if len(command) == 0 {
		return h.writeRESPError(res, "ERR empty command")
	}

	cmd := strings.ToUpper(command[0])

	switch cmd {
	case "PING":
		return h.writeRESPSimpleString(res, "PONG")
	case "SET":
		return h.handleSET(command, res)
	case "GET":
		return h.handleGET(command, res)
	case "DEL":
		return h.handleDEL(command, res)
	case "EXISTS":
		return h.handleEXISTS(command, res)
	case "TTL":
		return h.handleTTL(command, res)
	default:
		return h.writeRESPError(res, fmt.Sprintf("ERR unknown command '%s'", cmd))
	}
}

// handleSET 处理 SET 命令
func (h *RedisHandler) handleSET(command []string, res transport.Writer) error {
	if len(command) < 3 {
		return h.writeRESPError(res, "ERR wrong number of arguments for 'set' command")
	}

	key := command[1]
	value := command[2]
	var ttl int64 = 0

	// 解析可选的 TTL 参数
	if len(command) >= 5 && strings.ToUpper(command[3]) == "EX" {
		var err error
		ttl, err = strconv.ParseInt(command[4], 10, 64)
		if err != nil {
			return h.writeRESPError(res, "ERR invalid expire time")
		}
	}

	if err := h.set(key, value, ttl); err != nil {
		return h.writeRESPError(res, fmt.Sprintf("ERR %v", err))
	}

	return h.writeRESPSimpleString(res, "OK")
}

// handleGET 处理 GET 命令
func (h *RedisHandler) handleGET(command []string, res transport.Writer) error {
	if len(command) != 2 {
		return h.writeRESPError(res, "ERR wrong number of arguments for 'get' command")
	}

	key := command[1]
	value, err := h.get(key)
	if err != nil {
		return h.writeRESPNull(res)
	}

	return h.writeRESPBulkString(res, value)
}

// handleDEL 处理 DEL 命令
func (h *RedisHandler) handleDEL(command []string, res transport.Writer) error {
	if len(command) < 2 {
		return h.writeRESPError(res, "ERR wrong number of arguments for 'del' command")
	}

	deleted := 0
	for i := 1; i < len(command); i++ {
		if count, _ := h.delete(command[i]); count > 0 {
			deleted++
		}
	}

	return h.writeRESPInteger(res, int64(deleted))
}

// handleEXISTS 处理 EXISTS 命令
func (h *RedisHandler) handleEXISTS(command []string, res transport.Writer) error {
	if len(command) < 2 {
		return h.writeRESPError(res, "ERR wrong number of arguments for 'exists' command")
	}

	exists := 0
	for i := 1; i < len(command); i++ {
		if count, _ := h.exists(command[i]); count > 0 {
			exists++
		}
	}

	return h.writeRESPInteger(res, int64(exists))
}

// handleTTL 处理 TTL 命令
func (h *RedisHandler) handleTTL(command []string, res transport.Writer) error {
	if len(command) != 2 {
		return h.writeRESPError(res, "ERR wrong number of arguments for 'ttl' command")
	}

	key := command[1]
	ttl, _ := h.ttl(key)
	return h.writeRESPInteger(res, ttl)
}

// get 获取键值
func (h *RedisHandler) get(key string) (string, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	item, exists := h.store[key]
	if !exists {
		return "", fmt.Errorf("key not found")
	}

	// 检查是否过期
	if item.ExpiresAt != nil && time.Now().After(*item.ExpiresAt) {
		delete(h.store, key)
		return "", fmt.Errorf("key not found")
	}

	return item.Value, nil
}

// set 设置键值
func (h *RedisHandler) set(key string, value string, ttl int64) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	item := &RedisItem{
		Value: value,
	}

	if ttl > 0 {
		expiresAt := time.Now().Add(time.Duration(ttl) * time.Second)
		item.ExpiresAt = &expiresAt
	}

	h.store[key] = item
	return nil
}

// delete 删除键
func (h *RedisHandler) delete(key string) (int64, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	_, exists := h.store[key]
	if exists {
		delete(h.store, key)
		return 1, nil
	}
	return 0, nil
}

// exists 检查键是否存在
func (h *RedisHandler) exists(key string) (int64, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	item, exists := h.store[key]
	if !exists {
		return 0, nil
	}

	// 检查是否过期
	if item.ExpiresAt != nil && time.Now().After(*item.ExpiresAt) {
		delete(h.store, key)
		return 0, nil
	}

	return 1, nil
}

// ttl 获取键的过期时间
func (h *RedisHandler) ttl(key string) (int64, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	item, exists := h.store[key]
	if !exists {
		return -2, nil // key does not exist
	}

	if item.ExpiresAt == nil {
		return -1, nil // key exists but has no expiration
	}

	ttl := time.Until(*item.ExpiresAt).Seconds()
	if ttl <= 0 {
		delete(h.store, key)
		return -2, nil
	}

	return int64(ttl), nil
}

// RESP 协议写入方法

// writeRESPError 写入 RESP 错误响应
func (h *RedisHandler) writeRESPError(res transport.Writer, message string) error {
	response := fmt.Sprintf("-%s\r\n", message)
	_, err := res.Write([]byte(response))
	return err
}

// writeRESPSimpleString 写入 RESP 简单字符串
func (h *RedisHandler) writeRESPSimpleString(res transport.Writer, message string) error {
	response := fmt.Sprintf("+%s\r\n", message)
	_, err := res.Write([]byte(response))
	return err
}

// writeRESPBulkString 写入 RESP 批量字符串
func (h *RedisHandler) writeRESPBulkString(res transport.Writer, message string) error {
	response := fmt.Sprintf("$%d\r\n%s\r\n", len(message), message)
	_, err := res.Write([]byte(response))
	return err
}

// writeRESPNull 写入 RESP 空值
func (h *RedisHandler) writeRESPNull(res transport.Writer) error {
	response := "$-1\r\n"
	_, err := res.Write([]byte(response))
	return err
}

// writeRESPInteger 写入 RESP 整数
func (h *RedisHandler) writeRESPInteger(res transport.Writer, value int64) error {
	response := fmt.Sprintf(":%d\r\n", value)
	_, err := res.Write([]byte(response))
	return err
}

// Close 关闭内存数据库连接
func (h *RedisHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 清空内存存储
	h.store = make(map[string]*RedisItem)
	return nil
}
