package libspine

import (
	"fmt"
	"log"
	"net"
	"spine-go/libspine/handler"
	"spine-go/libspine/transport"
	"sync"
	"time"
)

// Server 服务器结构
type Server struct {
	transports map[string]transport.Transport
	config     *Config
	serverCtx  *transport.ServerContext
	mu         sync.RWMutex
}

// Config 服务器配置
type Config struct {
	TCPPort      string
	UnixSocket   string
	WSPort       string
	RedisAddr    string
	RedisPass    string
	RedisDB      int
	EnableTCP    bool
	EnableUnix   bool
	EnableWS     bool
	ServerMode   string // "chat" 或 "redis"
}

// NewServer 创建新的服务器
func NewServer(config *Config) *Server {
	serverInfo := &transport.ServerInfo{
		Address: "spine-server",
		Config:  make(map[string]interface{}),
	}
	
	return &Server{
		transports: make(map[string]transport.Transport),
		serverCtx:  transport.NewServerContext(serverInfo),
		config:     config,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	// 注册默认处理器
	s.registerHandlers()

	// 启动各种传输层
	var errs []error
	var wg sync.WaitGroup

	if s.config.EnableTCP {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.startTCP(); err != nil {
				errs = append(errs, fmt.Errorf("TCP error: %v", err))
			}
		}()
	}

	if s.config.EnableUnix {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.startUnix(); err != nil {
				errs = append(errs, fmt.Errorf("Unix socket error: %v", err))
			}
		}()
	}

	if s.config.EnableWS {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.startWebSocket(); err != nil {
				errs = append(errs, fmt.Errorf("WebSocket error: %v", err))
			}
		}()
	}

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("server errors: %v", errs)
	}

	return nil
}

// startTCP 启动 TCP 服务器
func (s *Server) startTCP() error {
	tcpTransport, err := transport.NewTCPTransport(":" + s.config.TCPPort)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.transports["tcp"] = tcpTransport
	s.mu.Unlock()

	log.Printf("TCP server started on port %s", s.config.TCPPort)
	return s.serveTransport(tcpTransport, "tcp")
}

// startUnix 启动 Unix Socket 服务器
func (s *Server) startUnix() error {
	if s.config.UnixSocket == "" {
		return fmt.Errorf("unix socket path not specified")
	}

	unixTransport, err := transport.NewUnixSocketTransport(s.config.UnixSocket)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.transports["unix"] = unixTransport
	s.mu.Unlock()

	log.Printf("Unix socket server started on %s", s.config.UnixSocket)
	return s.serveTransport(unixTransport, "unix")
}

// startWebSocket 启动 WebSocket 服务器
func (s *Server) startWebSocket() error {
	wsTransport := transport.NewWebSocketTransportWithContext(":"+s.config.WSPort, s.serverCtx)
	
	s.mu.Lock()
	s.transports["ws"] = wsTransport
	s.mu.Unlock()

	log.Printf("WebSocket server started on port %s", s.config.WSPort)
	return wsTransport.Start()
}

// serveTransport 服务传输层
func (s *Server) serveTransport(t transport.Transport, protocol string) error {
	for {
		conn, err := t.Accept()
		if err != nil {
			return err
		}

		go s.handleConnection(conn, protocol, t)
	}
}

// handleConnection 处理连接
func (s *Server) handleConnection(conn net.Conn, protocol string, t transport.Transport) {
	defer conn.Close()

	reader, writer := t.NewHandlers(conn)
	
	// 创建连接信息
	connInfo := &transport.ConnInfo{
		ID:       generateServerID(),
		Remote:   conn.RemoteAddr(),
		Protocol: protocol,
		Metadata: make(map[string]interface{}),
	}
	
	// 添加到统一连接管理器
	s.serverCtx.Connections.AddConnection(connInfo)
	
	// 创建上下文
	ctx := &transport.Context{
		ServerInfo: s.serverCtx.ServerInfo,
		ConnInfo:   connInfo,
	}
	
	// 连接关闭时从管理器移除
	defer s.serverCtx.Connections.RemoveConnection(connInfo.ID)

	// 持续处理连接上的数据
	for {
		data, err := reader.Read()
		if err != nil {
			break
		}

		// 直接从服务器上下文获取处理器
		handler := s.serverCtx.GetHandler()
		if handler != nil {
			// 创建一个新的 reader 只包含当前数据
			singleReader := &singleDataReader{data: data}
			if err := handler.Handle(ctx, singleReader, writer); err != nil {
				// 处理错误，可以记录日志
				log.Printf("Handler error: %v", err)
			}
		}
	}
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

// registerHandlers 注册处理器
func (s *Server) registerHandlers() {
	var mainHandler handler.Handler
	
	// 根据服务器模式选择处理器
	switch s.config.ServerMode {
	case "chat":
		chatHandler := handler.NewChatHandler()
		chatHandler.Start()
		mainHandler = chatHandler
		log.Printf("Server mode: Chat")
		
	case "redis":
		redisHandler := handler.NewRedisHandler(s.config.RedisAddr, s.config.RedisPass, s.config.RedisDB)
		mainHandler = redisHandler
		log.Printf("Server mode: Redis")
		
	default:
		// 默认使用聊天模式
		chatHandler := handler.NewChatHandler()
		chatHandler.Start()
		mainHandler = chatHandler
		log.Printf("Server mode: Chat (default)")
	}
	
	// 直接设置处理器到服务器上下文
	s.serverCtx.SetHandler(mainHandler)
	
	log.Printf("Registered handler for server mode: %s", s.config.ServerMode)
}


// Stop 停止服务器
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error
	for name, transport := range s.transports {
		if err := transport.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s transport error: %v", name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("stop errors: %v", errs)
	}

	return nil
}

// GetServerContext 获取服务器上下文
func (s *Server) GetServerContext() *transport.ServerContext {
	return s.serverCtx
}

// GetStats 获取服务器统计信息
func (s *Server) GetStats() map[string]interface{} {
	return s.serverCtx.GetStats()
}

// GetConnections 获取所有连接
func (s *Server) GetConnections() []*transport.ConnInfo {
	return s.serverCtx.Connections.GetAllConnections()
}

// GetConnectionsByProtocol 获取指定协议的连接
func (s *Server) GetConnectionsByProtocol(protocol string) []*transport.ConnInfo {
	return s.serverCtx.Connections.GetConnectionsByProtocol(protocol)
}

// generateServerID 生成唯一 ID
func generateServerID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}