package transport

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketTransport WebSocket 传输层实现
type WebSocketTransport struct {
	server    *http.Server
	upgrader  websocket.Upgrader
	router    *gin.Engine
	serverCtx *ServerContext // 统一服务器上下文
}

// NewWebSocketTransport 创建新的 WebSocket 传输层
func NewWebSocketTransport(addr string) *WebSocketTransport {
	gin.SetMode(gin.ReleaseMode) // 设置 gin 为发布模式

	router := gin.New()
	router.Use(gin.Recovery())

	return &WebSocketTransport{
		server: &http.Server{
			Addr:    addr,
			Handler: router,
		},
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源
			},
		},
		router: router,
	}
}

// SetServerContext 设置服务器上下文
func (w *WebSocketTransport) SetServerContext(serverCtx *ServerContext) {
	w.serverCtx = serverCtx
}

// Start 启动 WebSocket 传输层
func (w *WebSocketTransport) Start(serverCtx *ServerContext) error {
	w.serverCtx = serverCtx

	// 设置路由
	w.router.GET("/ws", w.handleWebSocket)
	w.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 如果有静态文件路径，设置静态文件服务
	if staticPath, ok := serverCtx.ServerInfo.Config["static_path"].(string); ok && staticPath != "" {
		w.router.Static("/static", staticPath)
	}

	go func() {
		if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// 记录错误但不返回，因为这是在 goroutine 中
		}
	}()

	return nil
}

// handleWebSocket 处理 WebSocket 连接
func (w *WebSocketTransport) handleWebSocket(c *gin.Context) {
	conn, err := w.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 创建 Reader 和 Writer
	reader := &WebSocketReader{conn: conn}
	writer := &WebSocketWriter{conn: conn}

	// 创建连接信息
	remoteAddr := conn.RemoteAddr()
	connInfo := &ConnInfo{
		ID:       generateConnID(),
		Remote:   remoteAddr,
		Protocol: "websocket",
		Metadata: make(map[string]interface{}),
		Reader:   reader,
		Writer:   writer,
	}

	// 如果有服务器上下文，添加到统一连接管理器
	if w.serverCtx != nil {
		w.serverCtx.Connections.AddConnection(connInfo)
		defer w.serverCtx.Connections.RemoveConnection(connInfo.ID)
	}

	// 创建上下文
	var ctx *Context
	if w.serverCtx != nil {
		ctx = &Context{
			ServerInfo: w.serverCtx.ServerInfo,
			ConnInfo:   connInfo,
		}
	}

	// 处理消息直到连接关闭
	for {
		// 直接从服务器上下文获取处理器
		var handler Handler

		// 使用服务器上下文中的处理器
		if w.serverCtx != nil {
			handler = w.serverCtx.GetHandler()
		}

		if handler != nil {
			if err := handler.Handle(ctx, reader, writer); err != nil {
				// 处理错误，如果是连接关闭则退出循环
				break
			}
		} else {
			// 没有处理器，等待一段时间再尝试
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// Stop 停止 WebSocket 传输层
func (w *WebSocketTransport) Stop() error {
	// 连接关闭由统一连接管理器处理

	if w.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return w.server.Shutdown(ctx)
	}
	return nil
}

// Broadcast 广播消息到所有连接的客户端
func (w *WebSocketTransport) Broadcast(data []byte) error {
	// 广播功能现在通过统一连接管理器实现
	return nil
}

// NewHandlers 创建 WebSocket 处理器
// WebSocket 不使用传统的 NewHandlers 模式，返回 nil
func (w *WebSocketTransport) NewHandlers(conn net.Conn) (Reader, Writer) {
	return nil, nil
}

// GetConnections 获取当前连接数（通过统一连接管理器）
func (w *WebSocketTransport) GetConnections() int {
	if w.serverCtx != nil {
		stats := w.serverCtx.Connections.GetStats()
		if total, ok := stats["total"].(int); ok {
			return total
		}
	}
	return 0
}

// WebSocketReader WebSocket 读取器
type WebSocketReader struct {
	conn *websocket.Conn
}

// Read 读取原始数据
func (r *WebSocketReader) Read() ([]byte, error) {
	_, data, err := r.conn.ReadMessage()
	return data, err
}

// Close 关闭读取器
func (r *WebSocketReader) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// WebSocketWriter WebSocket 写入器
type WebSocketWriter struct {
	conn *websocket.Conn
}

// Write 写入原始数据
func (w *WebSocketWriter) Write(data []byte) error {
	return w.conn.WriteMessage(websocket.BinaryMessage, data)
}

// Close 关闭写入器
func (w *WebSocketWriter) Close() error {
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

// generateConnID 生成连接ID
func generateConnID() string {
	return "ws_" + generateID()
}
