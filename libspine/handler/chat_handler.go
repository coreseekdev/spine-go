package handler

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
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
	Room      string    `json:"room"`
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
	rooms        map[string][]*ChatMessage
	mu           sync.RWMutex
	broadcast    chan *ChatMessage
	clients      map[string][]transport.Writer
	clientsMu    sync.RWMutex
	wsTransport  interface{} // WebSocket transport for broadcasting
	staticPath   string      // 静态文件路径
}

// NewChatHandler 创建新的聊天处理器
func NewChatHandler() *ChatHandler {
	return &ChatHandler{
		rooms:     make(map[string][]*ChatMessage),
		broadcast: make(chan *ChatMessage, 100),
		clients:   make(map[string][]transport.Writer),
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

// Start 启动聊天处理器
func (h *ChatHandler) Start() {
	go h.broadcastMessages()
}

// Stop 停止聊天处理器
func (h *ChatHandler) Stop() {
	close(h.broadcast)
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

	// 读取原始数据
	data, err := req.Read()
	if err != nil {
		return h.writeError(res, "Failed to read request", 400)
	}

	// 解析请求
	var chatReq ChatRequest
	if err := json.Unmarshal(data, &chatReq); err != nil {
		return h.writeError(res, "Invalid request format", 400)
	}

	switch chatReq.Method {
	case "POST":
		return h.handlePostMessage(ctx, req, res, &chatReq)
	case "GET":
		return h.handleGetMessages(ctx, req, res, &chatReq)
	case "JOIN":
		return h.handleJoinRoom(ctx, req, res, &chatReq)
	case "LEAVE":
		return h.handleLeaveRoom(ctx, req, res, &chatReq)
	default:
		return h.writeError(res, "Method not allowed", 405)
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
	room, _ := msgData["room"].(string)

	if user == "" || message == "" || room == "" {
		return h.writeError(res, "Missing required fields", 400)
	}

	msg := &ChatMessage{
		ID:        generateID(),
		User:      user,
		Message:   message,
		Timestamp: time.Now(),
		Room:      room,
	}

	h.mu.Lock()
	h.rooms[room] = append(h.rooms[room], msg)
	h.mu.Unlock()

	// 广播消息
	h.broadcast <- msg

	return h.writeSuccess(res, map[string]interface{}{
		"status":  "success",
		"message": "Message sent",
	})
}

// handleGetMessages 处理获取消息
func (h *ChatHandler) handleGetMessages(ctx *transport.Context, req transport.Reader, res transport.Writer, chatReq *ChatRequest) error {
	room := chatReq.Path
	if room == "" {
		return h.writeError(res, "Room not specified", 400)
	}

	h.mu.RLock()
	messages := h.rooms[room]
	h.mu.RUnlock()

	return h.writeSuccess(res, messages)
}

// handleJoinRoom 处理加入房间
func (h *ChatHandler) handleJoinRoom(ctx *transport.Context, req transport.Reader, res transport.Writer, chatReq *ChatRequest) error {
	var joinData map[string]interface{}
	dataBytes, _ := json.Marshal(chatReq.Data)
	json.Unmarshal(dataBytes, &joinData)

	room, _ := joinData["room"].(string)
	if room == "" {
		return h.writeError(res, "Room not specified", 400)
	}

	// 使用 ConnInfo 中的 Writer
	writer := res
	if ctx.ConnInfo != nil && ctx.ConnInfo.Writer != nil {
		writer = ctx.ConnInfo.Writer
	}

	h.clientsMu.Lock()
	h.clients[room] = append(h.clients[room], writer)
	h.clientsMu.Unlock()

	return h.writeSuccess(res, map[string]interface{}{
		"status":  "success",
		"message": "Joined room",
	})
}

// handleLeaveRoom 处理离开房间
func (h *ChatHandler) handleLeaveRoom(ctx *transport.Context, req transport.Reader, res transport.Writer, chatReq *ChatRequest) error {
	var leaveData map[string]interface{}
	dataBytes, _ := json.Marshal(chatReq.Data)
	json.Unmarshal(dataBytes, &leaveData)

	room, _ := leaveData["room"].(string)
	if room == "" {
		return h.writeError(res, "Room not specified", 400)
	}

	// 使用 ConnInfo 中的 Writer
	writer := res
	if ctx.ConnInfo != nil && ctx.ConnInfo.Writer != nil {
		writer = ctx.ConnInfo.Writer
	}

	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	if clients, exists := h.clients[room]; exists {
		for i, client := range clients {
			if client == writer {
				h.clients[room] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
	}

	return h.writeSuccess(res, map[string]interface{}{
		"status":  "success",
		"message": "Left room",
	})
}

// broadcastMessages 广播消息
func (h *ChatHandler) broadcastMessages() {
	for msg := range h.broadcast {
		h.clientsMu.RLock()
		clients := h.clients[msg.Room]
		h.clientsMu.RUnlock()

		response := &ChatResponse{
			Status: 200,
			Data:    msg,
		}

		data, _ := json.Marshal(response)
		responseData := h.createBinaryMessage(data)

		// 向传统客户端广播
		for _, client := range clients {
			if err := client.Write(responseData); err != nil {
				// 移除失败的客户端
				h.clientsMu.Lock()
				for i, c := range h.clients[msg.Room] {
					if c == client {
						h.clients[msg.Room] = append(h.clients[msg.Room][:i], h.clients[msg.Room][i+1:]...)
						break
					}
				}
				h.clientsMu.Unlock()
			}
		}

		// 向 WebSocket 客户端广播
		if h.wsTransport != nil {
			if wsTransport, ok := h.wsTransport.(interface{ Broadcast([]byte) error }); ok {
				wsTransport.Broadcast(responseData)
			}
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

	binaryData := h.createBinaryMessage(respData)
	return res.Write(binaryData)
}

// writeError 写入错误响应
func (h *ChatHandler) writeError(res transport.Writer, message string, status int) error {
	response := &ChatResponse{
		Status: status,
		Error:  message,
	}

	respData, err := json.Marshal(response)
	if err != nil {
		return res.Write([]byte(`{"error":"Internal server error"}`))
	}

	binaryData := h.createBinaryMessage(respData)
	return res.Write(binaryData)
}

// createBinaryMessage 创建二进制消息格式
func (h *ChatHandler) createBinaryMessage(data []byte) []byte {
	// 简单的协议：[4字节长度] + [数据]
	length := uint32(len(data))
	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, length)
	buffer.Write(data)
	return buffer.Bytes()
}

// generateID 生成唯一 ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}