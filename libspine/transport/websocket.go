package transport

import (
	"net"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketTransport WebSocket 传输层实现
type WebSocketTransport struct {
	server      *http.Server
	upgrader    websocket.Upgrader
	connections map[*websocket.Conn]bool
	mu          sync.RWMutex
	router      *gin.Engine
	serverCtx   *ServerContext // 统一服务器上下文
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
		connections: make(map[*websocket.Conn]bool),
		router:      router,
	}
}

// NewWebSocketTransportWithContext 创建新的 WebSocket 传输层并传入服务器上下文
func NewWebSocketTransportWithContext(addr string, serverCtx *ServerContext) *WebSocketTransport {
	gin.SetMode(gin.ReleaseMode) // 设置 gin 为发布模式

	router := gin.New()
	router.Use(gin.Recovery())

	ws := &WebSocketTransport{
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
		connections: make(map[*websocket.Conn]bool),
		router:      router,
		serverCtx:   serverCtx,
	}

	// 设置路由
	router.GET("/ws", ws.handleWebSocket)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	router.Static("/static", "./static")

	return ws
}

// SetServerContext 设置服务器上下文
func (w *WebSocketTransport) SetServerContext(serverCtx *ServerContext) {
	w.serverCtx = serverCtx
}

// Start 启动 WebSocket 服务器
func (w *WebSocketTransport) Start() error {
	return w.server.ListenAndServe()
}

// handleWebSocket 处理 WebSocket 连接
func (w *WebSocketTransport) handleWebSocket(c *gin.Context) {
	conn, err := w.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	w.mu.Lock()
	w.connections[conn] = true
	w.mu.Unlock()

	// 创建连接信息
	remoteAddr := conn.RemoteAddr()
	connInfo := &ConnInfo{
		ID:       generateConnID(),
		Remote:   remoteAddr,
		Protocol: "websocket",
		Metadata: make(map[string]interface{}),
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
	} else {
		// 兼容旧版本，创建本地上下文
		serverInfo := &ServerInfo{
			Address: w.server.Addr,
			Config:  make(map[string]interface{}),
		}
		ctx = &Context{
			ServerInfo: serverInfo,
			ConnInfo:   connInfo,
		}
	}

	// 创建 Reader 和 Writer
	reader := &WebSocketReader{conn: conn}
	writer := &WebSocketWriter{conn: conn}

	// 启动一个 goroutine 来处理消息
	go func() {
		defer func() {
			w.mu.Lock()
			delete(w.connections, conn)
			w.mu.Unlock()
		}()

		for {
			data, err := reader.Read()
			if err != nil {
				break
			}

			// 直接从服务器上下文获取处理器
			var handler Handler
			
			// 使用服务器上下文中的处理器
			if w.serverCtx != nil {
				handler = w.serverCtx.GetHandler()
			}
			
			if handler != nil {
				// 创建一个新的 reader 只包含当前数据
				singleReader := &singleDataReader{data: data}
				if err := handler.Handle(ctx, singleReader, writer); err != nil {
					// 处理错误，可以记录日志但不中断连接
					continue
				}
			}
		}
	}()

	// 保持连接直到客户端断开
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}

	w.mu.Lock()
	delete(w.connections, conn)
	w.mu.Unlock()
}

// Accept 实现 Transport 接口
// WebSocket 不使用传统的 Accept 模式
func (w *WebSocketTransport) Accept() (net.Conn, error) {
	return nil, net.ErrClosed
}

// NewHandlers 实现 Transport 接口
// WebSocket 使用自己的处理方式，这里返回 nil
func (w *WebSocketTransport) NewHandlers(conn net.Conn) (Reader, Writer) {
	return nil, nil
}

// Close 关闭传输层
func (w *WebSocketTransport) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for conn := range w.connections {
		conn.Close()
	}

	if w.server != nil {
		return w.server.Close()
	}
	return nil
}

// Broadcast 广播消息到所有连接的客户端
func (w *WebSocketTransport) Broadcast(data []byte) error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var lastErr error
	for conn := range w.connections {
		if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			lastErr = err
			// 移除失败的连接
			w.mu.Lock()
			delete(w.connections, conn)
			w.mu.Unlock()
		}
	}
	return lastErr
}

// GetConnections 获取当前连接数
func (w *WebSocketTransport) GetConnections() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.connections)
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

// singleDataReader 单次数据读取器
type singleDataReader struct {
	data []byte
	read bool
}

func (r *singleDataReader) Read() ([]byte, error) {
	if r.read {
		return nil, net.ErrClosed
	}
	r.read = true
	return r.data, nil
}

func (r *singleDataReader) Close() error {
	return nil
}

// generateConnID 生成连接ID
func generateConnID() string {
	return "ws_" + generateID()
}
