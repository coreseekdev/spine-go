package transport

import (
	"net"
	"sync"
)

// Metadata keys for ConnInfo
const (
	MetadataSelectedDB    = "selected_db"    // Key for storing the selected database ID
	MetadataSubscriptions = "subscriptions"  // []string - subscribed channels
	MetadataPatternSubs   = "pattern_subs"   // []string - pattern subscriptions
	MetadataPubSubMode    = "pubsub_mode"    // bool - is in pub/sub mode
)

// ConnectionManager 连接管理器接口，管理所有传输层的连接
type ConnectionManager interface {
	AddConnection(conn *ConnInfo)
	RemoveConnection(connID string)
	GetConnection(connID string) (*ConnInfo, bool)
	GetAllConnections() []*ConnInfo
	GetStats() map[string]interface{}
	CloseAllConnections() error
}

// NewConnectionManager 创建新的连接管理器
func NewConnectionManager() ConnectionManager {
	return newConnectionManager()
}

// ServerContext 统一的服务器上下文，管理所有传输层的共享状态
type ServerContext struct {
	ServerInfo  *ServerInfo
	Connections ConnectionManager
	Handler     Handler // 单一处理器
	mu          sync.RWMutex
}

// NewServerContext 创建新的服务器上下文
func NewServerContext(serverInfo *ServerInfo) *ServerContext {
	return &ServerContext{
		ServerInfo:  serverInfo,
		Connections: NewConnectionManager(),
	}
}

// SetHandler 设置处理器
func (sc *ServerContext) SetHandler(handler Handler) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.Handler = handler
}

// GetHandler 获取处理器
func (sc *ServerContext) GetHandler() Handler {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.Handler
}

// GetStats 获取服务器统计信息
func (sc *ServerContext) GetStats() map[string]interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	handlerType := "none"
	if sc.Handler != nil {
		handlerType = "configured"
	}

	return map[string]interface{}{
		"connections": sc.Connections.GetStats(),
		"handler":     handlerType,
		"server_info": sc.ServerInfo,
	}
}

//////////////////////////////////////////////////////

// Context 包含服务器和连接信息
type Context struct {
	ServerInfo        *ServerInfo
	ConnInfo          *ConnInfo
	ConnectionManager ConnectionManager
}

// ServerInfo 服务器信息
type ServerInfo struct {
	Address string
	Config  map[string]interface{}
}

// ConnInfo 连接信息
type ConnInfo struct {
	ID       string
	Remote   net.Addr
	Protocol string
	Metadata map[string]interface{}
	Reader   Reader
	Writer   Writer
}

// Request 请求结构
type Request struct {
	ID     string
	Method string
	Path   string
	Header map[string]string
	Body   []byte
}

// Response 响应结构
type Response struct {
	ID     string
	Status int
	Header map[string]string
	Body   []byte
}

// Reader 用于读取请求数据，兼容 io.Reader 接口
type Reader interface {
	// Read 读取数据到提供的缓冲区中
	// 返回读取的字节数和可能的错误
	// 符合 io.Reader 接口规范
	Read(p []byte) (n int, err error)
	Close() error
}

// Writer 用于写入响应数据，兼容 io.Writer 接口
type Writer interface {
	// Write 将数据写入到底层数据流
	// 返回写入的字节数和可能的错误
	// 符合 io.Writer 接口规范
	Write(p []byte) (n int, err error)
	Close() error
}

// Transport 传输层接口
type Transport interface {
	// 启动传输层
	Start(serverCtx *ServerContext) error
	// 停止传输层
	Stop() error
}

// Handler 处理器接口
type Handler interface {
	Handle(ctx *Context, req Reader, res Writer) error
}
