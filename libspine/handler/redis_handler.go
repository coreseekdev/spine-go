package handler

import (
	"fmt"
	"io"
	"log"
	"spine-go/libspine/common/resp"
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
	// Protocol version (2 or 3)
	protocolVersion int
}

// NewRedisHandler 创建新的 Redis 处理器
func NewRedisHandler() *RedisHandler {
	return &RedisHandler{
		store: make(map[string]*RedisItem),
		protocolVersion: 2, // Default to RESP v2
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

	// 创建 RESP 解析器和序列化器
	respReader := resp.NewRespReader(req)
	respWriter := resp.NewRespWriter(res)

	// 持续处理消息直到连接关闭
	for {
		// 解析 RESP 命令
		value, err := respReader.ReadValue()
		if err != nil {
			// 连接关闭或读取错误
			if err == io.EOF {
				return nil
			}
			log.Printf("Error parsing RESP command: %v", err)
			respWriter.WriteErrorString("ERR", err.Error())
			continue
		}

		// 确保命令是数组类型
		if value.Type != resp.TypeArray {
			respWriter.WriteSyntaxError("expected array command")
			continue
		}

		// 提取命令参数
		command := make([]string, 0, len(value.Array))
		for _, item := range value.Array {
			if item.Type == resp.TypeBulkString {
				command = append(command, string(item.Bulk))
			} else {
				respWriter.WriteSyntaxError("expected bulk string command arguments")
				continue
			}
		}

		if len(command) == 0 {
			respWriter.WriteErrorString("ERR", "empty command")
			continue
		}

		log.Printf("Received Redis command: %v", command)

		// 处理命令
		if err := h.handleCommand(command, respWriter); err != nil {
			log.Printf("Error handling Redis command: %v", err)
		}
	}
}

// 不再需要 parseRESPCommand 方法，使用 resp.Parser 代替

// handleCommand 处理 Redis 命令
func (h *RedisHandler) handleCommand(command []string, writer *resp.RespWriter) error {
	if len(command) == 0 {
		return writer.WriteErrorString("ERR", "empty command")
	}

	cmd := strings.ToUpper(command[0])

	switch cmd {
	case "PING":
		return writer.WritePong()
	case "HELLO":
		return h.handleHELLO(command, writer)
	case "SET":
		return h.handleSET(command, writer)
	case "GET":
		return h.handleGET(command, writer)
	case "DEL":
		return h.handleDEL(command, writer)
	case "EXISTS":
		return h.handleEXISTS(command, writer)
	case "TTL":
		return h.handleTTL(command, writer)
	default:
		return writer.WriteCommandError(fmt.Sprintf("unknown command '%s'", cmd))
	}
}

// handleSET 处理 SET 命令
func (h *RedisHandler) handleSET(command []string, writer *resp.RespWriter) error {
	if len(command) < 3 {
		return writer.WriteWrongNumberOfArgumentsError("SET")
	}

	key := command[1]
	value := command[2]
	var ttl int64 = 0

	// 解析可选的 TTL 参数
	if len(command) >= 5 && strings.ToUpper(command[3]) == "EX" {
		var err error
		ttl, err = strconv.ParseInt(command[4], 10, 64)
		if err != nil {
			return writer.WriteErrorString("ERR", "invalid expire time")
		}
	}

	if err := h.set(key, value, ttl); err != nil {
		return writer.WriteErrorString("ERR", err.Error())
	}

	return writer.WriteOK()
}

// handleGET 处理 GET 命令
func (h *RedisHandler) handleGET(command []string, writer *resp.RespWriter) error {
	if len(command) != 2 {
		return writer.WriteWrongNumberOfArgumentsError("GET")
	}

	key := command[1]
	value, err := h.get(key)
	if err != nil {
		return writer.WriteNil()
	}

	return writer.WriteBulkString([]byte(value))
}

// handleDEL 处理 DEL 命令
func (h *RedisHandler) handleDEL(command []string, writer *resp.RespWriter) error {
	if len(command) < 2 {
		return writer.WriteWrongNumberOfArgumentsError("DEL")
	}

	deleted := 0
	for i := 1; i < len(command); i++ {
		if count, _ := h.delete(command[i]); count > 0 {
			deleted++
		}
	}

	return writer.WriteInteger(int64(deleted))
}

// handleEXISTS 处理 EXISTS 命令
func (h *RedisHandler) handleEXISTS(command []string, writer *resp.RespWriter) error {
	if len(command) < 2 {
		return writer.WriteWrongNumberOfArgumentsError("EXISTS")
	}

	exists := 0
	for i := 1; i < len(command); i++ {
		if count, _ := h.exists(command[i]); count > 0 {
			exists++
		}
	}

	return writer.WriteInteger(int64(exists))
}

// handleTTL 处理 TTL 命令
func (h *RedisHandler) handleTTL(command []string, writer *resp.RespWriter) error {
	if len(command) != 2 {
		return writer.WriteWrongNumberOfArgumentsError("TTL")
	}

	key := command[1]
	ttl, _ := h.ttl(key)
	return writer.WriteInteger(ttl)
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

// handleHELLO handles the HELLO command for protocol version negotiation
// HELLO [protover [AUTH username password] [SETNAME clientname]]
func (h *RedisHandler) handleHELLO(command []string, writer *resp.RespWriter) error {
	// Default to current protocol version if not specified
	protocolVersion := h.protocolVersion
	
	// Parse protocol version if provided
	if len(command) >= 2 {
		ver, err := strconv.Atoi(command[1])
		if err != nil {
			return writer.WriteErrorString("ERR", "Protocol version is not an integer or out of range")
		}
		
		// Only support versions 2 and 3
		if ver != 2 && ver != 3 {
			return writer.WriteErrorString("ERR", "HELLO only supports RESP protocol versions 2 and 3")
		}
		
		protocolVersion = ver
	}
	
	// Update handler's protocol version
	h.protocolVersion = protocolVersion
	
	// Create response map
	responseMap := make(map[string]interface{})
	responseMap["server"] = "spine-go"
	responseMap["version"] = "1.0.0"
	responseMap["proto"] = protocolVersion
	responseMap["id"] = 0 // Server ID
	responseMap["mode"] = "standalone"
	responseMap["role"] = "master"
	responseMap["modules"] = []interface{}{}
	
	// If using RESP v3, return as a map
	if protocolVersion == 3 {
		// Convert to RESP v3 map
		mapItems := make([]resp.MapItem, 0, len(responseMap))
		
		for k, v := range responseMap {
			var value resp.Value
			switch val := v.(type) {
			case string:
				value = resp.NewBulkStringString(val)
			case int:
				value = resp.NewInteger(int64(val))
			case []interface{}:
				// Convert array
				arrayValues := make([]resp.Value, len(val))
				for i, item := range val {
					switch arrItem := item.(type) {
					case string:
						arrayValues[i] = resp.NewBulkStringString(arrItem)
					case int:
						arrayValues[i] = resp.NewInteger(int64(arrItem))
					default:
						arrayValues[i] = resp.NewNull()
					}
				}
				value = resp.NewArray(arrayValues)
			default:
				value = resp.NewNull()
			}
			
			mapItems = append(mapItems, resp.MapItem{
				Key:   resp.NewBulkStringString(k),
				Value: value,
			})
		}
		
		return writer.WriteValue(resp.NewMap(mapItems))
	}
	
	// For RESP v2, return as an array of bulk strings
	responseArray := make([]resp.Value, 0, len(responseMap)*2)
	
	// Add each key-value pair as consecutive elements
	for k, v := range responseMap {
		responseArray = append(responseArray, resp.NewBulkStringString(k))
		
		switch val := v.(type) {
		case string:
			responseArray = append(responseArray, resp.NewBulkStringString(val))
		case int:
			responseArray = append(responseArray, resp.NewBulkStringString(strconv.Itoa(val)))
		case []interface{}:
			// For empty array, add empty bulk string
			responseArray = append(responseArray, resp.NewBulkStringString(""))
		default:
			responseArray = append(responseArray, resp.NewBulkStringString(""))
		}
	}
	
	return writer.WriteValue(resp.NewArray(responseArray))
}

// 不再需要 RESP 协议写入方法，使用 resp.RespWriter 代替

// Close 关闭内存数据库连接
func (h *RedisHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 清空内存存储
	h.store = make(map[string]*RedisItem)
	return nil
}
