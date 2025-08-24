package handler

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"spine-go/libspine/transport"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
)

// RedisRequest Redis 请求结构
type RedisRequest struct {
	Command string      `json:"command"`
	Key     string      `json:"key"`
	Value   interface{} `json:"value"`
	TTL     int64       `json:"ttl"`
}

// RedisResponse Redis 响应结构
type RedisResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
}

// RedisHandler Redis 处理器
type RedisHandler struct {
	client *redis.Client
}

// NewRedisHandler 创建新的 Redis 处理器
func NewRedisHandler(addr string, password string, db int) *RedisHandler {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisHandler{
		client: client,
	}
}

// Handle 处理 Redis 请求
func (h *RedisHandler) Handle(ctx *transport.Context, req transport.Reader, res transport.Writer) error {
	// 读取原始数据
	buffer := make([]byte, 4096)
	n, err := req.Read(buffer)
	if err != nil {
		return h.writeError(res, "Failed to read request", 400)
	}
	data := buffer[:n]

	// 解析请求
	var redisReq RedisRequest
	if err := json.Unmarshal(data, &redisReq); err != nil {
		return h.writeError(res, "Invalid Redis request format", 400)
	}

	ctxRedis := context.Background()
	var result interface{}
	var redisErr error

	switch redisReq.Command {
	case "GET":
		result, redisErr = h.client.Get(ctxRedis, redisReq.Key).Result()
	case "SET":
		if redisReq.TTL > 0 {
			redisErr = h.client.Set(ctxRedis, redisReq.Key, redisReq.Value, time.Duration(redisReq.TTL)*time.Second).Err()
		} else {
			redisErr = h.client.Set(ctxRedis, redisReq.Key, redisReq.Value, 0).Err()
		}
	case "DELETE":
		redisErr = h.client.Del(ctxRedis, redisReq.Key).Err()
	case "EXISTS":
		result, redisErr = h.client.Exists(ctxRedis, redisReq.Key).Result()
	case "TTL":
		result, redisErr = h.client.TTL(ctxRedis, redisReq.Key).Result()
	default:
		return h.writeError(res, "Unsupported Redis command", 400)
	}

	response := &RedisResponse{
		Success: redisErr == nil,
		Data:    result,
	}

	if redisErr != nil {
		response.Error = redisErr.Error()
	}

	respData, err := json.Marshal(response)
	if err != nil {
		return h.writeError(res, "Failed to marshal response", 500)
	}

	binaryData := h.createBinaryMessage(respData)
	_, err = res.Write(binaryData)
	return err
}

// writeSuccess 写入成功响应
func (h *RedisHandler) writeSuccess(res transport.Writer, data interface{}) error {
	response := &RedisResponse{
		Success: true,
		Data:    data,
	}

	respData, err := json.Marshal(response)
	if err != nil {
		return h.writeError(res, "Failed to marshal response", 500)
	}

	binaryData := h.createBinaryMessage(respData)
	_, err = res.Write(binaryData)
	return err
}

// writeError 写入错误响应
func (h *RedisHandler) writeError(res transport.Writer, message string, status int) error {
	response := &RedisResponse{
		Success: false,
		Error:   message,
	}

	respData, err := json.Marshal(response)
	if err != nil {
		_, err := res.Write([]byte(`{"error":"Internal server error"}`))
		return err
	}

	binaryData := h.createBinaryMessage(respData)
	_, err = res.Write(binaryData)
	return err
}

// createBinaryMessage 创建二进制消息格式
func (h *RedisHandler) createBinaryMessage(data []byte) []byte {
	// 简单的协议：[4字节长度] + [数据]
	length := uint32(len(data))
	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, length)
	buffer.Write(data)
	return buffer.Bytes()
}

// generateRedisID 生成唯一 ID
func generateRedisID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// Close 关闭 Redis 连接
func (h *RedisHandler) Close() error {
	if h.client != nil {
		return h.client.Close()
	}
	return nil
}