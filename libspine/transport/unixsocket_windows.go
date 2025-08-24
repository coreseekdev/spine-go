//go:build windows

package transport

import (
	"fmt"
	"net"
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

// UnixSocketReader Windows 平台上的 Unix Socket 读取器存根
type UnixSocketReader struct {
	Conn net.Conn
}

// Read 读取数据 - Windows 上不支持
func (r *UnixSocketReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("Unix socket is not supported on Windows platform")
}

// Close 关闭读取器 - Windows 上不支持
func (r *UnixSocketReader) Close() error {
	return fmt.Errorf("Unix socket is not supported on Windows platform")
}

// UnixSocketWriter Windows 平台上的 Unix Socket 写入器存根
type UnixSocketWriter struct {
	Conn net.Conn
}

// Write 写入数据 - Windows 上不支持
func (w *UnixSocketWriter) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("Unix socket is not supported on Windows platform")
}

// Close 关闭写入器 - Windows 上不支持
func (w *UnixSocketWriter) Close() error {
	return fmt.Errorf("Unix socket is not supported on Windows platform")
}
