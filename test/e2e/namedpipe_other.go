//go:build !windows

package e2e

import "fmt"

// connectWindows 在非 Windows 平台上返回错误
func (c *NamedPipeTestClient) connectWindows() error {
	return fmt.Errorf("named pipe is only supported on Windows")
}

// closeConnection 在非 Windows 平台上的空实现
func (c *NamedPipeTestClient) closeConnection() {
	// 非 Windows 平台无需实现
}

// writeData 在非 Windows 平台上返回错误
func (c *NamedPipeTestClient) writeData(data []byte) error {
	return fmt.Errorf("named pipe is only supported on Windows")
}

// readData 在非 Windows 平台上返回错误
func (c *NamedPipeTestClient) readData(buffer []byte) (int, error) {
	return 0, fmt.Errorf("named pipe is only supported on Windows")
}