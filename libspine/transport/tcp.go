package transport

import (
	"fmt"
	"net"
	"time"
)

// TCPTransport TCP 传输层实现
type TCPTransport struct {
	listener net.Listener
}

// NewTCPTransport 创建新的 TCP 传输层
func NewTCPTransport(addr string) (*TCPTransport, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &TCPTransport{listener: listener}, nil
}

// Accept 接受连接
func (t *TCPTransport) Accept() (net.Conn, error) {
	return t.listener.Accept()
}

// NewHandlers 创建 TCP 的 Reader 和 Writer
func (t *TCPTransport) NewHandlers(conn net.Conn) (Reader, Writer) {
	return &TCPReader{conn: conn}, &TCPWriter{conn: conn}
}

// Close 关闭传输层
func (t *TCPTransport) Close() error {
	if t.listener != nil {
		return t.listener.Close()
	}
	return nil
}

// TCPReader TCP 读取器
type TCPReader struct {
	conn net.Conn
}

// Read 读取原始数据
func (r *TCPReader) Read() ([]byte, error) {
	buffer := make([]byte, 1024)
	n, err := r.conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}

// Close 关闭读取器
func (r *TCPReader) Close() error {
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

// TCPWriter TCP 写入器
type TCPWriter struct {
	conn net.Conn
}

// Write 写入原始数据
func (w *TCPWriter) Write(data []byte) error {
	_, err := w.conn.Write(data)
	return err
}

// Close 关闭写入器
func (w *TCPWriter) Close() error {
	if w.conn != nil {
		return w.conn.Close()
	}
	return nil
}

// generateID 生成唯一 ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
