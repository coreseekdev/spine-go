package libspine

import (
	"fmt"
	"log"
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
	Path   string // 路径， ws / unix socket / named pipe 可用
}

// Config 服务器配置
type Config struct {
	ListenConfigs []ListenConfig // 监听配置数组
	ServerMode    string         // "chat" 或 "redis"
	StaticPath    string         // 静态文件路径，用于 chat webui
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
			if err := s.startTransport(config, s.config.ServerMode, s.config.StaticPath); err != nil {
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
func (s *Server) startTransport(config ListenConfig, _ string, staticPath string) error {
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

		s.mu.Lock()
		s.transports = append(s.transports, transportInstance)
		s.mu.Unlock()

		log.Printf("TCP transport starting on %s", address)
		return transportInstance.Start(s.serverCtx)

	case "unix":
		address = config.Path
		transportInstance, err = transport.NewUnixSocketTransport(address)
		if err != nil {
			return err
		}

		s.mu.Lock()
		s.transports = append(s.transports, transportInstance)
		s.mu.Unlock()

		log.Printf("Unix socket transport starting on %s", address)
		return transportInstance.Start(s.serverCtx)

	case "namedpipe":
		address = config.Path
		transportInstance, err = transport.NewNamedPipeTransport(address)
		if err != nil {
			return err
		}

		s.mu.Lock()
		s.transports = append(s.transports, transportInstance)
		s.mu.Unlock()

		log.Printf("Named pipe transport starting on %s", address)
		return transportInstance.Start(s.serverCtx)

	case "http":
		address := config.Host + ":" + config.Port
		if config.Path != "" {
			address += "/" + config.Path
		}
		transportInstance = transport.NewWebSocketTransport(address)

		s.mu.Lock()
		s.transports = append(s.transports, transportInstance)
		s.mu.Unlock()

		// 设置静态文件路径到服务器上下文
		if staticPath != "" {
			s.serverCtx.ServerInfo.Config["static_path"] = staticPath
		}

		log.Printf("WebSocket transport starting on %s", address)
		if staticPath != "" {
			log.Printf("WebSocket static files path: %s", staticPath)
		}
		return transportInstance.Start(s.serverCtx)

	default:
		return fmt.Errorf("unsupported schema: %s", config.Schema)
	}
}

// Stop 停止服务器
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 首先主动关闭所有客户端连接
	if s.serverCtx != nil && s.serverCtx.Connections != nil {
		log.Printf("Closing all active connections before server shutdown")
		if err := s.serverCtx.Connections.CloseAllConnections(); err != nil {
			log.Printf("Error closing connections: %v", err)
		}
	}

	var errs []error
	for i, transport := range s.transports {
		if err := transport.Stop(); err != nil {
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
	// var mainHandler handler.Handler
	/*
		// 根据服务器模式选择处理器
		switch s.config.ServerMode {
		case "chat":
			chatHandler := handler.NewChatHandler()
			if s.config.StaticPath != "" {
				chatHandler.SetStaticPath(s.config.StaticPath)
			}
			mainHandler = chatHandler
			log.Printf("Server mode: Chat")
			if s.config.StaticPath != "" {
				log.Printf("Static files path: %s", s.config.StaticPath)
			}

		//case "redis":
		//	redisHandler := handler.NewRedisHandler(s.config.RedisAddr, s.config.RedisPass, s.config.RedisDB)
		//	mainHandler = redisHandler
		//	log.Printf("Server mode: Redis")

		default:
			// 默认使用聊天模式
			chatHandler := handler.NewChatHandler()
			if s.config.StaticPath != "" {
				chatHandler.SetStaticPath(s.config.StaticPath)
			}
			mainHandler = chatHandler
			log.Printf("Server mode: Chat (default)")
			if s.config.StaticPath != "" {
				log.Printf("Static files path: %s", s.config.StaticPath)
			}
		}
	*/
	if s.config.ServerMode == "chat" {
		chatHandler := handler.NewChatHandler()
		if s.config.StaticPath != "" {
			chatHandler.SetStaticPath(s.config.StaticPath)
		}
		// 直接设置处理器到服务器上下文
		s.serverCtx.SetHandler(chatHandler)
	} else if s.config.ServerMode == "redis" {
		redisHandler := handler.NewRedisHandler()
		s.serverCtx.SetHandler(redisHandler)
	}

	log.Printf("Registered handler for server mode: %s", s.config.ServerMode)
}
