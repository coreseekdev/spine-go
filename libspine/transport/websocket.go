package transport

import (
	"context"
	"io"
	"log"
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

	// 获取静态文件路径
	staticPath := ""
	if serverCtx != nil && serverCtx.ServerInfo != nil && serverCtx.ServerInfo.Config != nil {
		if pathValue, ok := serverCtx.ServerInfo.Config["static_path"].(string); ok && pathValue != "" {
			staticPath = pathValue
		}
	}

	// 设置静态文件服务
	if staticPath != "" {
		// 使用配置的静态文件路径
		log.Printf("Using configured static path: %s", staticPath)
		w.router.StaticFile("/", staticPath+"/index.html")
		w.router.StaticFile("/index.html", staticPath+"/index.html")
		w.router.StaticFile("/style.css", staticPath+"/style.css")
		w.router.StaticFile("/chat.js", staticPath+"/chat.js")
		w.router.Static("/static", staticPath)
	} else {
		// 使用默认的静态文件路径
		log.Printf("Using default static path: web/")
		w.router.StaticFile("/", "web/index.html")
		w.router.StaticFile("/index.html", "web/index.html")
		w.router.StaticFile("/style.css", "web/style.css")
		w.router.StaticFile("/chat.js", "web/chat.js")
		w.router.Static("/static", "./web")
	}

	go func() {
		if err := w.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("WebSocket server error: %v", err)
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
			ServerInfo:        w.serverCtx.ServerInfo,
			ConnInfo:          connInfo,
			ConnectionManager: w.serverCtx.Connections,
		}
	}

	// 获取处理器
	var handler Handler
	if w.serverCtx != nil {
		handler = w.serverCtx.GetHandler()
	}

	// 如果有处理器，调用一次 Handle 方法
	// Handle 方法内部会处理消息直到连接关闭
	if handler != nil {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			// 处理网络相关的常见错误，避免过多日志
			if isNetworkError(err) {
				log.Printf("WebSocket connection closed: %s", connInfo.ID)
			} else {
				log.Printf("WebSocket handler error: %v", err)
			}
		}
	} else {
		log.Printf("No handler available for WebSocket connection: %s", connInfo.ID)
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
	conn       *websocket.Conn
	reader     io.Reader // 当前消息的 reader
	messageType int      // 当前消息类型
}

// Read 读取数据到提供的缓冲区中，符合 io.Reader 接口
func (r *WebSocketReader) Read(p []byte) (n int, err error) {
	// 如果没有当前 reader，获取下一个消息的 reader
	if r.reader == nil {
		r.messageType, r.reader, err = r.conn.NextReader()
		if err != nil {
			return 0, err
		}
	}
	
	// 从当前 reader 读取数据
	n, err = r.reader.Read(p)
	
	// 如果遇到 EOF，说明当前消息读取完毕，清空 reader 准备读取下一个消息
	if err == io.EOF {
		r.reader = nil
		// 对于 WebSocket，消息结束的 EOF 不应该传播给上层
		// 上层应该继续调用 Read 来获取下一个消息
		if n == 0 {
			// 如果没有读取到数据且遇到 EOF，递归调用读取下一个消息
			return r.Read(p)
		}
		// 如果读取到了数据，返回数据但不返回 EOF 错误
		err = nil
	}
	
	return n, err
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

// Write 写入数据，符合 io.Writer 接口
func (w *WebSocketWriter) Write(p []byte) (n int, err error) {
	log.Printf("WebSocketWriter.Write: Sending message type: %d, data: %s", websocket.TextMessage, string(p))
	err = w.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
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

// NewHandlers 创建 WebSocket 读写器（为了兼容测试接口）
func (w *WebSocketTransport) NewHandlers(conn net.Conn) (Reader, Writer) {
	// WebSocket 不能直接从 net.Conn 创建，返回 nil
	// 实际的 WebSocket 连接通过 HTTP 升级创建
	return nil, nil
}

// isNetworkError 判断是否为网络相关的常见错误
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return errMsg == "EOF" ||
		errMsg == "write: broken pipe" ||
		errMsg == "connection reset by peer" ||
		errMsg == "use of closed network connection" ||
		websocket.IsCloseError(err) ||
		websocket.IsUnexpectedCloseError(err)
}
