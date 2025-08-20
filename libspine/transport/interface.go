package transport

import (
	"net"
	"sync"
)

// ConnectionManager 连接管理器接口，管理所有传输层的连接
type ConnectionManager interface {
	AddConnection(conn *ConnInfo)
	RemoveConnection(connID string)
	GetConnection(connID string) (*ConnInfo, bool)
	GetAllConnections() []*ConnInfo
	GetStats() map[string]interface{}
}

// connectionManager 连接管理器的具体实现，管理所有传输层的连接
type connectionManager struct {
	connections map[string]*ConnInfo
	mu          sync.RWMutex
}

// NewConnectionManager 创建新的连接管理器
func NewConnectionManager() ConnectionManager {
	return &connectionManager{
		connections: make(map[string]*ConnInfo),
	}
}

// AddConnection 添加连接
func (cm *connectionManager) AddConnection(conn *ConnInfo) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.connections[conn.ID] = conn
}

// RemoveConnection 移除连接
func (cm *connectionManager) RemoveConnection(connID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.connections, connID)
}

// GetConnection 获取连接信息
func (cm *connectionManager) GetConnection(connID string) (*ConnInfo, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, exists := cm.connections[connID]
	return conn, exists
}

// GetAllConnections 获取所有连接
func (cm *connectionManager) GetAllConnections() []*ConnInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conns := make([]*ConnInfo, 0, len(cm.connections))
	for _, conn := range cm.connections {
		conns = append(conns, conn)
	}
	return conns
}

// GetStats 获取连接统计信息
func (cm *connectionManager) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := map[string]interface{}{
		"total": len(cm.connections),
	}
	return stats
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

// Reader 用于读取请求数据
type Reader interface {
	Read() ([]byte, error)
	Close() error
}

// Writer 用于写入响应数据
type Writer interface {
	Write([]byte) error
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
