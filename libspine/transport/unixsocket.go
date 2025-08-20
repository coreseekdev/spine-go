package transport

import (
	"net"
	"os"
)

// UnixSocketTransport Unix Socket 传输层实现
type UnixSocketTransport struct {
	listener net.Listener
	path     string
}

// NewUnixSocketTransport 创建新的 Unix Socket 传输层
func NewUnixSocketTransport(path string) (*UnixSocketTransport, error) {
	// 如果文件已存在，先删除
	if _, err := os.Stat(path); err == nil {
		os.Remove(path)
	}

	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	return &UnixSocketTransport{
		listener: listener,
		path:     path,
	}, nil
}

// Accept 接受连接
func (u *UnixSocketTransport) Accept() (net.Conn, error) {
	return u.listener.Accept()
}

// NewHandlers 创建 Unix Socket 的 Reader 和 Writer
func (u *UnixSocketTransport) NewHandlers(conn net.Conn) (Reader, Writer) {
	return &UnixSocketReader{conn: conn}, &UnixSocketWriter{conn: conn}
}

// Close 关闭传输层
func (u *UnixSocketTransport) Close() error {
	if u.listener != nil {
		err := u.listener.Close()
		// 删除 socket 文件
		os.Remove(u.path)
		return err
	}
	return nil
}

// UnixSocketReader Unix Socket 读取器
type UnixSocketReader struct {
	conn net.Conn
}

// Read 读取原始数据
func (r *UnixSocketReader) Read() ([]byte, error) {
	buffer := make([]byte, 1024)
	n, err := r.conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}

// Close 关闭读取器
func (r *UnixSocketReader) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// UnixSocketWriter Unix Socket 写入器
type UnixSocketWriter struct {
	conn net.Conn
}

// Write 写入原始数据
func (w *UnixSocketWriter) Write(data []byte) error {
	_, err := w.conn.Write(data)
	return err
}

// Close 关闭写入器
func (w *UnixSocketWriter) Close() error {
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}
