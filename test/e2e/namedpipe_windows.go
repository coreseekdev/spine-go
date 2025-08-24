//go:build windows

package e2e

import (
	"fmt"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// connectWindows 在 Windows 平台上连接到 named pipe
func (c *NamedPipeTestClient) connectWindows() error {
	// 转换管道名称为 UTF16
	pipeName16, err := syscall.UTF16PtrFromString(c.pipeName)
	if err != nil {
		return fmt.Errorf("failed to convert pipe name to UTF16: %v", err)
	}

	// 尝试连接，如果管道不存在则等待
	var handle windows.Handle
	for i := 0; i < 50; i++ { // 最多重试 50 次，每次等待 100ms
		// 尝试打开 named pipe
		handle, err = windows.CreateFile(
			pipeName16,
			windows.GENERIC_READ|windows.GENERIC_WRITE,
			0,
			nil,
			windows.OPEN_EXISTING,
			0,
			0,
		)
		if err == nil {
			break // 连接成功
		}

		// 如果是文件不存在错误，等待后重试
		if err == windows.ERROR_FILE_NOT_FOUND {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// 其他错误直接返回
		return fmt.Errorf("failed to open named pipe: %v", err)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to named pipe after retries: %v", err)
	}

	c.conn = handle
	return nil
}

// closeConnection 关闭 Windows named pipe 连接
func (c *NamedPipeTestClient) closeConnection() {
	if handle, ok := c.conn.(windows.Handle); ok {
		windows.CloseHandle(handle)
	}
}

// writeData 向 Windows named pipe 写入数据
func (c *NamedPipeTestClient) writeData(data []byte) error {
	handle, ok := c.conn.(windows.Handle)
	if !ok {
		return fmt.Errorf("invalid connection handle")
	}

	var bytesWritten uint32
	err := windows.WriteFile(handle, data, &bytesWritten, nil)
	if err != nil {
		return fmt.Errorf("failed to write to named pipe: %v", err)
	}

	if int(bytesWritten) != len(data) {
		return fmt.Errorf("incomplete write: wrote %d bytes, expected %d", bytesWritten, len(data))
	}

	return nil
}

// readData 从 Windows named pipe 读取数据
func (c *NamedPipeTestClient) readData(buffer []byte) (int, error) {
	handle, ok := c.conn.(windows.Handle)
	if !ok {
		return 0, fmt.Errorf("invalid connection handle")
	}

	var bytesRead uint32
	err := windows.ReadFile(handle, buffer, &bytesRead, nil)
	if err != nil {
		// 检查是否是管道断开
		if err == windows.ERROR_BROKEN_PIPE || err == windows.ERROR_PIPE_NOT_CONNECTED {
			return 0, fmt.Errorf("pipe disconnected")
		}
		return 0, fmt.Errorf("failed to read from named pipe: %v", err)
	}

	return int(bytesRead), nil
}