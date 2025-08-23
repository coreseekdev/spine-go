package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"spine-go/libspine"
	"strings"
	"syscall"
)

func main() {
	// 解析命令行参数
	var (
		listenArgs []string
		staticPath = flag.String("static", "", "Static files path for chat webui")
		serverMode = flag.String("mode", "chat", "Server mode (chat/redis)")
	)

	// 自定义 flag 函数来收集多个 --listen 参数
	flag.Func("listen", "Listen address (format: schema://host:port, e.g., tcp://:8080, http://:8000, unix:///tmp/spine.sock). Can be specified multiple times.", func(value string) error {
		listenArgs = append(listenArgs, value)
		return nil
	})

	flag.Parse()

	// 解析监听地址
	var listenConfigs []libspine.ListenConfig
	for _, addr := range listenArgs {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}

		// 解析 schema://host:port 格式
		parts := strings.SplitN(addr, "://", 2)
		if len(parts) != 2 {
			log.Printf("Invalid listen address format: %s (expected schema://host:port)", addr)
			continue
		}

		schema := parts[0]
		hostPort := parts[1]

		// 对于 unix schema，hostPort 就是路径
		if schema == "unix" {
			listenConfigs = append(listenConfigs, libspine.ListenConfig{
				Schema: schema,
				Host:   "",
				Port:   "",
				Path:   hostPort,
			})
		} else {
			// 对于 tcp 和 ws，分割 host 和 port
			host, port := "", hostPort
			if strings.Contains(hostPort, ":") {
				if strings.HasPrefix(hostPort, ":") {
					// :8080 格式
					port = hostPort[1:]
				} else {
					// host:port 格式
					lastColon := strings.LastIndex(hostPort, ":")
					host = hostPort[:lastColon]
					port = hostPort[lastColon+1:]
				}
			}

			listenConfigs = append(listenConfigs, libspine.ListenConfig{
				Schema: schema,
				Host:   host,
				Port:   port,
				Path:   "",
			})
		}
	}

	// 如果没有指定监听地址，使用默认配置
	if len(listenConfigs) == 0 {
		listenConfigs = []libspine.ListenConfig{
			{Schema: "tcp", Host: "", Port: "8080", Path: ""},
			{Schema: "http", Host: "", Port: "8000", Path: ""},
		}
	}

	// 创建服务器配置
	config := &libspine.Config{
		ListenConfigs: listenConfigs,
		ServerMode:    *serverMode,
		StaticPath:    *staticPath,
	}

	// 创建服务器
	server := libspine.NewServer(config)

	// 如果有静态文件路径，设置到服务器上下文中
	if *staticPath != "" {
		serverCtx := server.GetServerContext()
		serverCtx.ServerInfo.Config["static_path"] = *staticPath
	}

	// 启动服务器
	go func() {
		log.Println("Starting Spine server...")
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := server.Stop(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}

	log.Println("Server stopped")
}
