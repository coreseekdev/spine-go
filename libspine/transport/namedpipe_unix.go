//go:build !windows

package transport

import (
	"fmt"
)

// NamedPipeTransport Unix/Linux 平台上的 Named Pipe 传输层存根
// Named Pipe 在 Unix/Linux 上不可用，返回错误
type NamedPipeTransport struct{}

// NewNamedPipeTransport 在 Unix/Linux 平台上创建 Named Pipe 传输层会返回错误
func NewNamedPipeTransport(pipeName string) (*NamedPipeTransport, error) {
	return nil, fmt.Errorf("Named Pipe transport is not supported on Unix/Linux platforms, use Unix socket instead")
}

// Start 启动传输层 - Unix/Linux 上不支持
func (t *NamedPipeTransport) Start(serverCtx *ServerContext) error {
	return fmt.Errorf("Named Pipe transport is not supported on Unix/Linux platforms, use Unix socket instead")
}

// Stop 停止传输层 - Unix/Linux 上不支持
func (t *NamedPipeTransport) Stop() error {
	return fmt.Errorf("Named Pipe transport is not supported on Unix/Linux platforms, use Unix socket instead")
}
