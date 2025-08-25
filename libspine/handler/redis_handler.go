package handler

import (
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"path/filepath"
	"strconv"

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

	// 初始化Metadata如果不存在
	if ctx.ConnInfo.Metadata == nil {
		ctx.ConnInfo.Metadata = make(map[string]interface{})
	}

	// 在连接的Metadata中存储选择的数据库编号
	var defaultDB int = 0 // 默认使用数据库0
	ctx.ConnInfo.Metadata[transport.MetadataSelectedDB] = defaultDB

	// 持续处理消息直到连接关闭
	for {
		// 为每个命令创建新的 RESP 请求读取器
		reqReader := resp.NewReqReader(req)
		
		// 获取命令名称和参数数量（一次性解析）
		command, nargs, err := reqReader.ParseCommand()
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

		log.Printf("Received Redis command: %s (args: %d)", command, nargs)

		// 处理特殊命令，如 SELECT
		if command == "SELECT" && nargs == 1 {
			// 获取参数解析器
			valueReader, err := reqReader.NextReader()
			if err != nil {
				respWriter := resp.NewRESPWriter(res)
				respWriter.WriteError(fmt.Sprintf("ERR %v", err))
				continue
			}

			// 读取数据库编号
			dbStr, err := valueReader.ReadBulkString()
			if err != nil {
				respWriter := resp.NewRESPWriter(res)
				respWriter.WriteError(fmt.Sprintf("ERR %v", err))
				continue
			}

			// 解析数据库编号
			dbNum, err := strconv.Atoi(dbStr)
			if err != nil {
				respWriter := resp.NewRESPWriter(res)
				respWriter.WriteError(fmt.Sprintf("ERR invalid DB index: %s", dbStr))
				continue
			}

			// 验证数据库编号范围（Redis默认支持0-15）
			if dbNum < 0 || dbNum > 15 {
				respWriter := resp.NewRESPWriter(res)
				respWriter.WriteError(fmt.Sprintf("ERR invalid DB index: %d", dbNum))
				continue
			}

			// 初始化Metadata如果不存在
			if ctx.ConnInfo.Metadata == nil {
				ctx.ConnInfo.Metadata = make(map[string]interface{})
			}

			// 在连接的Metadata中存储选择的数据库编号
			ctx.ConnInfo.Metadata[transport.MetadataSelectedDB] = dbNum

			// 返回成功响应
			respWriter := resp.NewRESPWriter(res)
			respWriter.WriteSimpleString("OK")
			continue
		}

		// 计算命令哈希值
		cmdHash := hashString(command)

		// 创建RESP响应写入器
		respWriter := resp.NewRESPWriter(res)

		// 直接传递 transport.Context、命令哈希、命令名称、reqReader 和 respWriter 到 ExecuteCommand
		if err := h.engine.ExecuteCommand(ctx, cmdHash, command, reqReader, respWriter); err != nil {
			log.Printf("Error executing command: %v", err)

			// 检查是否为未知命令错误
			if engine.IsUnknownCommandError(err) {
				// 对于未知命令错误，返回特定的错误消息
				respWriter.WriteError(fmt.Sprintf("ERR unknown command '%s'", command))
			} else {
				// 其他错误类型
				respWriter.WriteError(fmt.Sprintf("ERR %v", err))
			}
		}
	}
}

// hashString 计算字符串的32位FNV-1a哈希值
func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// Close 关闭Redis处理器
func (h *RedisHandler) Close() error {
	if h.engine != nil {
		return h.engine.Close()
	}
	return nil
}
