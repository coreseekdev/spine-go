package transport

import (
	"fmt"
	"sync"
)

// connectionManager 连接管理器的具体实现，管理所有传输层的连接
type connectionManager struct {
	connections map[string]*ConnInfo
	mu          sync.RWMutex
}

// newConnectionManager 创建新的连接管理器实例
func newConnectionManager() *connectionManager {
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

// CloseAllConnections 关闭所有连接
func (cm *connectionManager) CloseAllConnections() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var errs []error
	for connID, conn := range cm.connections {
		// 关闭Reader和Writer，忽略已关闭连接的错误
		if conn.Reader != nil {
			if err := conn.Reader.Close(); err != nil && !isConnectionClosedError(err) {
				errs = append(errs, fmt.Errorf("failed to close reader for connection %s: %v", connID, err))
			}
		}
		if conn.Writer != nil {
			if err := conn.Writer.Close(); err != nil && !isConnectionClosedError(err) {
				errs = append(errs, fmt.Errorf("failed to close writer for connection %s: %v", connID, err))
			}
		}
	}

	// 清空连接映射
	cm.connections = make(map[string]*ConnInfo)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}
	return nil
}

// isConnectionClosedError 检查是否为连接已关闭的错误
func isConnectionClosedError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	// 匹配各种连接关闭相关的错误
	return errMsg == "use of closed network connection" ||
		errMsg == "EOF" ||
		errMsg == "write: broken pipe" ||
		errMsg == "connection reset by peer" ||
		// 匹配包含这些关键词的错误消息
		containsAny(errMsg, []string{
			"use of closed network connection",
			"close tcp",
			"broken pipe",
			"connection reset",
		})
}

// containsAny 检查字符串是否包含任何一个子字符串
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}