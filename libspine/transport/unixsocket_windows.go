//go:build windows

package transport

import (
	"fmt"
)

// UnixSocketTransport Windows 平台上的 Unix Socket 传输层存根
// Unix Socket 在 Windows 上不可用，返回错误
type UnixSocketTransport struct{}

// NewUnixSocketTransport 在 Windows 平台上创建 Unix Socket 传输层会返回错误
func NewUnixSocketTransport(socketPath string) (*UnixSocketTransport, error) {
	return nil, fmt.Errorf("Unix socket transport is not supported on Windows platform")
}

// Start 启动传输层 - Windows 上不支持
func (t *UnixSocketTransport) Start(serverCtx *ServerContext) error {
	return fmt.Errorf("Unix socket transport is not supported on Windows platform")
}

// Stop 停止传输层 - Windows 上不支持
func (t *UnixSocketTransport) Stop() error {
	return fmt.Errorf("Unix socket transport is not supported on Windows platform")
}
