package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"spine-go/libspine/transport"
	"sync"
	"time"
)

// ChatMessage 聊天消息结构
type ChatMessage struct {
	ID        string    `json:"id"`
	User      string    `json:"user"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatRequest 聊天请求结构
type ChatRequest struct {
	Method string      `json:"method"`
	Path   string      `json:"path"`
	Data   interface{} `json:"data"`
}

// ChatResponse 聊天响应结构
type ChatResponse struct {
	Status int         `json:"status"`
	Data   interface{} `json:"data"`
	Error  string      `json:"error"`
}

// ChatHandler 聊天处理器
type ChatHandler struct {
	messages      []*ChatMessage
	mu            sync.RWMutex
	activeConns   map[string]bool // connectionID -> active
	connectionsMu sync.RWMutex
	wsTransport   interface{} // WebSocket transport for broadcasting
	staticPath    string      // 静态文件路径
}

// NewChatHandler 创建新的聊天处理器
func NewChatHandler() *ChatHandler {
	return &ChatHandler{
		messages:    make([]*ChatMessage, 0),
		activeConns: make(map[string]bool),
	}
}

// SetWebSocketTransport 设置 WebSocket 传输层
func (h *ChatHandler) SetWebSocketTransport(wsTransport interface{}) {
	h.wsTransport = wsTransport
}

// SetStaticPath 设置静态文件路径
func (h *ChatHandler) SetStaticPath(path string) {
	h.staticPath = path
}

// Handle 处理聊天请求
func (h *ChatHandler) Handle(ctx *transport.Context, req transport.Reader, res transport.Writer) error {
	// 使用 ConnInfo 中的 Reader 和 Writer
	if ctx.ConnInfo != nil {
		if ctx.ConnInfo.Reader != nil {
			req = ctx.ConnInfo.Reader
		}
		if ctx.ConnInfo.Writer != nil {
			res = ctx.ConnInfo.Writer
		}
	}

	// 持续处理消息直到连接关闭
	for {
		// 读取原始数据
		data, err := req.Read()
		if err != nil {
			// 连接关闭或读取错误，清理连接并退出
			if ctx.ConnInfo != nil {
				h.connectionsMu.Lock()
				delete(h.activeConns, ctx.ConnInfo.ID)
				h.connectionsMu.Unlock()
				log.Printf("Connection %s closed, removed from active connections", ctx.ConnInfo.ID)
			}
			return err
		}

		// 解析请求
		var chatReq ChatRequest
		log.Printf("Received request: %s", string(data))
		if err := json.Unmarshal(data, &chatReq); err != nil {
			// 发送错误响应但不关闭连接
			h.writeError(res, "Invalid request format", 400)
			continue
		}

		// 处理请求
		var handleErr error
		switch chatReq.Method {
		case "POST":
			handleErr = h.handlePostMessage(ctx, req, res, &chatReq)
		case "GET":
			handleErr = h.handleGetMessages(ctx, req, res, &chatReq)
		case "JOIN":
			handleErr = h.handleJoin(ctx, req, res, &chatReq)
		case "LEAVE":
			handleErr = h.handleLeave(ctx, req, res, &chatReq)
		case "PING":
			// 处理心跳请求
			handleErr = h.writeSuccess(res, map[string]interface{}{
				"status":  "success",
				"message": "pong",
			})
		default:
			handleErr = h.writeError(res, "Method not allowed", 405)
		}

		// 如果处理请求时出错，记录但不关闭连接
		if handleErr != nil {
			log.Printf("Error handling request: %v", handleErr)
		}
	}
}

// handlePostMessage 处理发送消息
func (h *ChatHandler) handlePostMessage(ctx *transport.Context, req transport.Reader, res transport.Writer, chatReq *ChatRequest) error {
	// 解析消息数据
	dataBytes, err := json.Marshal(chatReq.Data)
	if err != nil {
		return h.writeError(res, "Invalid message data", 400)
	}

	var msgData map[string]interface{}
	if err := json.Unmarshal(dataBytes, &msgData); err != nil {
		return h.writeError(res, "Invalid message format", 400)
	}

	user, _ := msgData["user"].(string)
	message, _ := msgData["message"].(string)

	if user == "" || message == "" {
		return h.writeError(res, "Missing required fields", 400)
	}

	msg := &ChatMessage{
		ID:        generateID(),
		User:      user,
		Message:   message,
		Timestamp: time.Now(),
	}

	h.mu.Lock()
	h.messages = append(h.messages, msg)
	h.mu.Unlock()

	// 广播消息给所有活跃连接
	h.broadcastToAll(ctx, msg)

	return h.writeSuccess(res, map[string]interface{}{
		"status":  "success",
		"message": "Message sent",
	})
}

// handleGetMessages 处理获取消息
func (h *ChatHandler) handleGetMessages(ctx *transport.Context, req transport.Reader, res transport.Writer, chatReq *ChatRequest) error {
	h.mu.RLock()
	messages := make([]*ChatMessage, len(h.messages))
	copy(messages, h.messages)
	h.mu.RUnlock()

	return h.writeSuccess(res, messages)
}

// handleJoin 处理加入聊天
func (h *ChatHandler) handleJoin(ctx *transport.Context, req transport.Reader, res transport.Writer, chatReq *ChatRequest) error {
	// 使用连接ID而不是Writer
	if ctx.ConnInfo == nil {
		return h.writeError(res, "Connection info not available", 400)
	}

	connID := ctx.ConnInfo.ID

	h.connectionsMu.Lock()
	h.activeConns[connID] = true
	h.connectionsMu.Unlock()

	return h.writeSuccess(res, map[string]interface{}{
		"status":  "success",
		"message": "Joined chat",
	})
}

// handleLeave 处理离开聊天
func (h *ChatHandler) handleLeave(ctx *transport.Context, req transport.Reader, res transport.Writer, chatReq *ChatRequest) error {
	// 使用连接ID而不是Writer
	if ctx.ConnInfo == nil {
		return h.writeError(res, "Connection info not available", 400)
	}

	connID := ctx.ConnInfo.ID

	h.connectionsMu.Lock()
	delete(h.activeConns, connID)
	h.connectionsMu.Unlock()

	return h.writeSuccess(res, map[string]interface{}{
		"status":  "success",
		"message": "Left chat",
	})
}

// broadcastToAll 使用ConnectionManager向所有活跃连接广播消息
func (h *ChatHandler) broadcastToAll(ctx *transport.Context, msg *ChatMessage) {
	if ctx == nil || ctx.ConnectionManager == nil {
		return
	}

	h.connectionsMu.RLock()
	activeConnIDs := make([]string, 0, len(h.activeConns))
	for connID := range h.activeConns {
		activeConnIDs = append(activeConnIDs, connID)
	}
	h.connectionsMu.RUnlock()

	response := &ChatResponse{
		Status: 200,
		Data:   msg,
	}

	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("broadcastToAll: Error marshaling response: %v", err)
		return
	}
	log.Printf("broadcastToAll: Broadcasting JSON message: %s", string(data))
	// 直接使用 JSON 文本而不是二进制格式

	// 向所有活跃连接广播消息
	for _, connID := range activeConnIDs {
		if connInfo, exists := ctx.ConnectionManager.GetConnection(connID); exists {
			if connInfo.Writer != nil {
				if err := connInfo.Writer.Write(data); err != nil {
					// 如果写入失败，从活跃连接中移除该连接
					h.connectionsMu.Lock()
					delete(h.activeConns, connID)
					h.connectionsMu.Unlock()
				}
			}
		}
	}

	// 向 WebSocket 客户端广播
	if h.wsTransport != nil {
		if wsTransport, ok := h.wsTransport.(interface{ Broadcast([]byte) error }); ok {
			// 直接使用 JSON 文本而不是二进制格式
			wsTransport.Broadcast(data)
		}
	}
}

// writeSuccess 写入成功响应
func (h *ChatHandler) writeSuccess(res transport.Writer, data interface{}) error {
	response := &ChatResponse{
		Status: 200,
		Data:   data,
	}

	respData, err := json.Marshal(response)
	if err != nil {
		return h.writeError(res, "Failed to marshal response", 500)
	}

	// 直接发送 JSON 文本而不是二进制格式
	log.Printf("writeSuccess: Sending JSON response: %s", string(respData))
	return res.Write(respData)
}

// writeError 写入错误响应
func (h *ChatHandler) writeError(res transport.Writer, message string, status int) error {
	response := &ChatResponse{
		Status: status,
		Error:  message,
	}

	respData, err := json.Marshal(response)
	if err != nil {
		log.Printf("writeError: Error marshaling response: %v", err)
		return res.Write([]byte(`{"error":"Internal server error"}`)) 
	}

	// 直接发送 JSON 文本而不是二进制格式
	log.Printf("writeError: Sending JSON error response: %s", string(respData))
	return res.Write(respData)
}

// createBinaryMessage 方法已删除，因为我们现在使用纯文本 JSON

// generateID 生成唯一 ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
