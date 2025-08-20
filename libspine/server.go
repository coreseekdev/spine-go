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
	transports []transport.Transport
	config     *Config
	serverCtx  *transport.ServerContext
	mu         sync.RWMutex
	startTime  time.Time
}

// ListenConfig 监听配置
type ListenConfig struct {
	Schema string // "tcp", "unix", "ws"
	Host   string // 监听主机
	Port   string // 监听端口
	URL    string // 仅 WebSocket 时可用，用于 webui
}

// Config 服务器配置
type Config struct {
	ListenConfigs []ListenConfig // 监听配置数组
	ServerMode    string         // "chat" 或 "redis"
	StaticPath    string         // 静态文件路径，用于 chat webui
	RedisAddr     string
	RedisPass     string
	RedisDB       int
}

// NewServer 创建新的服务器
func NewServer(config *Config) *Server {
	serverInfo := &transport.ServerInfo{
		Address: "spine-server",
		Config:  make(map[string]interface{}),
	}

	return &Server{
		transports: make([]transport.Transport, 0),
		serverCtx:  transport.NewServerContext(serverInfo),
		config:     config,
		startTime:  time.Now(),
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	// 注册处理器
	s.registerHandlers()

	// 启动各种传输层
	var errs []error
	var wg sync.WaitGroup

	for _, listenConfig := range s.config.ListenConfigs {
		wg.Add(1)
		go func(config ListenConfig) {
			defer wg.Done()
			if err := s.startTransport(config); err != nil {
				errs = append(errs, fmt.Errorf("%s error: %v", config.Schema, err))
			}
		}(listenConfig)
	}

	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("server errors: %v", errs)
	}

	return nil
}

// startTransport 根据配置启动传输层
func (s *Server) startTransport(config ListenConfig) error {
	var transportInstance transport.Transport
	var err error
	var address string

	switch config.Schema {
	case "tcp":
		address = config.Host + ":" + config.Port
		transportInstance, err = transport.NewTCPTransport(address)
		if err != nil {
			return err
		}
		log.Printf("TCP server started on %s", address)
		return s.serveTransport(transportInstance, "tcp")

	case "unix":
		address = config.Host + config.Port
		transportInstance, err = transport.NewUnixSocketTransport(address)
		if err != nil {
			return err
		}
		log.Printf("Unix socket server started on %s", address)
		return s.serveTransport(transportInstance, "unix")

	case "ws":
		address := config.Host + ":" + config.Port
		wsTransport := transport.NewWebSocketTransportWithContext(address, s.serverCtx)
		
		s.mu.Lock()
		s.transports = append(s.transports, wsTransport)
		s.mu.Unlock()

		log.Printf("WebSocket server started on %s", address)
		// WebSocket transport has its own Start method
		return wsTransport.Start()

	default:
		return fmt.Errorf("unsupported schema: %s", config.Schema)
	}
}

// serveTransport 服务传输层
func (s *Server) serveTransport(t transport.Transport, protocol string) error {
	s.mu.Lock()
	s.transports = append(s.transports, t)
	s.mu.Unlock()

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
		// data, err := reader.Read()
		// if err != nil {
		// 	break
		// }

		// 直接从服务器上下文获取处理器
		handler := s.serverCtx.GetHandler()
		if handler != nil {
			// 创建一个新的 reader 只包含当前数据
			// singleReader := &singleDataReader{data: data}
			if err := handler.Handle(ctx, reader, writer); err != nil {
				// 处理错误，可以记录日志
				log.Printf("Handler error: %v", err)
			}
		}
	}
}

// Stop 停止服务器
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error
	for i, transport := range s.transports {
		if err := transport.Close(); err != nil {
			errs = append(errs, fmt.Errorf("transport %d error: %v", i, err))
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

// GetUptime 获取服务器运行时间
func (s *Server) GetUptime() time.Duration {
	return time.Since(s.startTime)
}

// GetStats 获取服务器统计信息
func (s *Server) GetStats() map[string]interface{} {
	return s.serverCtx.GetStats()
}

// GetConnections 获取所有连接
func (s *Server) GetConnections() []*transport.ConnInfo {
	return s.serverCtx.Connections.GetAllConnections()
}

// registerHandlers 注册处理器
func (s *Server) registerHandlers() {
	var mainHandler handler.Handler
	
	// 根据服务器模式选择处理器
	switch s.config.ServerMode {
	case "chat":
		chatHandler := handler.NewChatHandler()
		if s.config.StaticPath != "" {
			chatHandler.SetStaticPath(s.config.StaticPath)
		}
		chatHandler.Start()
		mainHandler = chatHandler
		log.Printf("Server mode: Chat")
		if s.config.StaticPath != "" {
			log.Printf("Static files path: %s", s.config.StaticPath)
		}
		
	case "redis":
		redisHandler := handler.NewRedisHandler(s.config.RedisAddr, s.config.RedisPass, s.config.RedisDB)
		mainHandler = redisHandler
		log.Printf("Server mode: Redis")
		
	default:
		// 默认使用聊天模式
		chatHandler := handler.NewChatHandler()
		if s.config.StaticPath != "" {
			chatHandler.SetStaticPath(s.config.StaticPath)
		}
		chatHandler.Start()
		mainHandler = chatHandler
		log.Printf("Server mode: Chat (default)")
		if s.config.StaticPath != "" {
			log.Printf("Static files path: %s", s.config.StaticPath)
		}
	}
	
	// 直接设置处理器到服务器上下文
	s.serverCtx.SetHandler(mainHandler)
	
	log.Printf("Registered handler for server mode: %s", s.config.ServerMode)
}

// generateServerID 生成唯一 ID
func generateServerID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
