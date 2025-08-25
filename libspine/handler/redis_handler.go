package handler

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"

	"spine-go/libspine/engine"
	"spine-go/libspine/engine/commands"
	"spine-go/libspine/engine/resp"
	"spine-go/libspine/transport"
)

// RedisHandler Redis 处理器 - 使用新的 engine 架构
type RedisHandler struct {
	engine *engine.Engine
}

// NewRedisHandler 创建新的 Redis 处理器
func NewRedisHandler(walPath string) (*RedisHandler, error) {
	if walPath == "" {
		// 默认WAL路径
		walPath = filepath.Join(".", "redis.wal")
	}

	// 创建engine实例
	engineInstance, err := engine.NewEngine(walPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	// 注册所有命令
	if err := commands.RegisterAllCommands(engineInstance.GetCommandRegistry()); err != nil {
		return nil, fmt.Errorf("failed to register commands: %w", err)
	}

	return &RedisHandler{
		engine: engineInstance,
	}, nil
}

// Handle 处理 Redis 请求 - 使用新的 engine 架构
func (h *RedisHandler) Handle(ctx *transport.Context, req transport.Reader, res transport.Writer) error {
	// 使用 ConnInfo 中的 Reader 和 Writer
	if ctx.ConnInfo != nil {
		if ctx.ConnInfo.Reader != nil {
			req = ctx.ConnInfo.Reader
		}
		if ctx.ConnInfo.Writer != nil {
			res = ctx.ConnInfo.Writer
		}
	}

	// 创建 RESP 读取器
	respReader := resp.NewRESPReader(req)

	// 持续处理消息直到连接关闭
	for {
		// 解析 RESP 命令
		command, args, err := respReader.ReadCommand()
		if err != nil {
			// 连接关闭或读取错误
			if err == io.EOF {
				return nil
			}
			log.Printf("Error parsing RESP command: %v", err)
			// 写入错误响应
			respWriter := resp.NewRESPWriter(res)
			respWriter.WriteError(fmt.Sprintf("ERR %v", err))
			continue
		}

		log.Printf("Received Redis command: %s %v", command, args)

		// 使用engine处理命令
		ctxWithCancel := context.Background()
		if err := h.engine.ExecuteCommand(ctxWithCancel, command, args, req, res); err != nil {
			log.Printf("Error executing command: %v", err)
			// 写入错误响应
			respWriter := resp.NewRESPWriter(res)
			respWriter.WriteError(fmt.Sprintf("ERR %v", err))
		}
	}
}

// Close 关闭Redis处理器
func (h *RedisHandler) Close() error {
	if h.engine != nil {
		return h.engine.Close()
	}
	return nil
}
