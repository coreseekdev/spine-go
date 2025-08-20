package transport

import (
	"net"
	"sync"
	"time"
)

// ServerContext 统一的服务器上下文，管理所有传输层的共享状态
type ServerContext struct {
	ServerInfo  *ServerInfo
	Connections *ConnectionManager
	Handler     Handler // 单一处理器
	mu          sync.RWMutex
	startTime   time.Time
}

// NewServerContext 创建新的服务器上下文
func NewServerContext(serverInfo *ServerInfo) *ServerContext {
	return &ServerContext{
		ServerInfo:  serverInfo,
		Connections: NewConnectionManager(),
		startTime:   time.Now(),
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

// GetUptime 获取服务器运行时间
func (sc *ServerContext) GetUptime() time.Duration {
	return time.Since(sc.startTime)
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
		"uptime":       sc.GetUptime().String(),
		"connections":  sc.Connections.GetStats(),
		"handler":      handlerType,
		"server_info":  sc.ServerInfo,
	}
}

// ConnectionManager 连接管理器，管理所有传输层的连接
type ConnectionManager struct {
	connections map[string]*ConnInfo
	byProtocol map[string]map[string]*ConnInfo
	mu          sync.RWMutex
}

// NewConnectionManager 创建新的连接管理器
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*ConnInfo),
		byProtocol:  make(map[string]map[string]*ConnInfo),
	}
}

// AddConnection 添加连接
func (cm *ConnectionManager) AddConnection(conn *ConnInfo) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	cm.connections[conn.ID] = conn
	
	if _, exists := cm.byProtocol[conn.Protocol]; !exists {
		cm.byProtocol[conn.Protocol] = make(map[string]*ConnInfo)
	}
	cm.byProtocol[conn.Protocol][conn.ID] = conn
}

// RemoveConnection 移除连接
func (cm *ConnectionManager) RemoveConnection(connID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	
	if conn, exists := cm.connections[connID]; exists {
		delete(cm.connections, connID)
		if protocolConns, exists := cm.byProtocol[conn.Protocol]; exists {
			delete(protocolConns, connID)
			if len(protocolConns) == 0 {
				delete(cm.byProtocol, conn.Protocol)
			}
		}
	}
}

// GetConnection 获取连接信息
func (cm *ConnectionManager) GetConnection(connID string) (*ConnInfo, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	conn, exists := cm.connections[connID]
	return conn, exists
}

// GetAllConnections 获取所有连接
func (cm *ConnectionManager) GetAllConnections() []*ConnInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	conns := make([]*ConnInfo, 0, len(cm.connections))
	for _, conn := range cm.connections {
		conns = append(conns, conn)
	}
	return conns
}

// GetConnectionsByProtocol 获取指定协议的连接
func (cm *ConnectionManager) GetConnectionsByProtocol(protocol string) []*ConnInfo {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	protocolConns := make([]*ConnInfo, 0)
	if conns, exists := cm.byProtocol[protocol]; exists {
		for _, conn := range conns {
			protocolConns = append(protocolConns, conn)
		}
	}
	return protocolConns
}

// GetStats 获取连接统计信息
func (cm *ConnectionManager) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total": len(cm.connections),
	}
	
	for protocol, conns := range cm.byProtocol {
		stats[protocol] = len(conns)
	}
	
	return stats
}


// Context 包含服务器和连接信息
type Context struct {
	ServerInfo *ServerInfo
	ConnInfo   *ConnInfo
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
	// 接受连接
	Accept() (net.Conn, error)
	// 创建 Reader 和 Writer
	NewHandlers(conn net.Conn) (Reader, Writer)
	// 关闭传输层
	Close() error
}

// Handler 处理器接口
type Handler interface {
	Handle(ctx *Context, req Reader, res Writer) error
}
