package main

import (
	"flag"
	"log"
	"spine-go/libspine/handler"
	"spine-go/libspine/transport"
)

func main() {
	var (
		port      = flag.String("port", "8081", "WebSocket server port")
		redisAddr = flag.String("redis-addr", "localhost:6379", "Redis server address")
		redisPass = flag.String("redis-pass", "", "Redis password")
		redisDB   = flag.Int("redis-db", 0, "Redis database number")
		mode      = flag.String("mode", "chat", "Server mode (chat/redis)")
	)
	flag.Parse()

	// 创建服务器信息和上下文
	serverInfo := &transport.ServerInfo{
		Address: "ws-server:" + *port,
		Config:  make(map[string]interface{}),
	}
	serverCtx := transport.NewServerContext(serverInfo)

	// 创建 WebSocket 传输层
	wsTransport := transport.NewWebSocketTransportWithContext(":"+*port, serverCtx)
	
	// 根据模式创建单一处理器
	var mainHandler handler.Handler
	
	switch *mode {
	case "redis":
		redisHandler := handler.NewRedisHandler(*redisAddr, *redisPass, *redisDB)
		mainHandler = redisHandler
		log.Printf("WebSocket server mode: Redis")
		
	default:
		chatHandler := handler.NewChatHandler()
		chatHandler.Start()
		defer chatHandler.Stop()
		
		// 设置 WebSocket transport 到聊天处理器
		chatHandler.SetWebSocketTransport(wsTransport)
		mainHandler = chatHandler
		log.Printf("WebSocket server mode: Chat")
	}
	
	// 直接设置处理器到服务器上下文
	serverCtx.SetHandler(mainHandler)

	log.Printf("WebSocket server starting on port %s", *port)
	log.Printf("Visit http://localhost%s for the web interface", *port)
	
	if err := wsTransport.Start(); err != nil {
		log.Fatalf("Failed to start WebSocket server: %v", err)
	}
}

