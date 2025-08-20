package transport

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

// UnixSocketTransport Unix Socket 传输层实现
type UnixSocketTransport struct {
	listener  net.Listener
	path      string
	serverCtx *ServerContext
	running   bool
	mu        sync.RWMutex
	quitChan  chan struct{}
	wg        sync.WaitGroup
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
		quitChan: make(chan struct{}),
	}, nil
}

// Start 启动 Unix Socket 传输层
func (u *UnixSocketTransport) Start(serverCtx *ServerContext) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.running {
		return fmt.Errorf("Unix socket transport is already running")
	}

	u.serverCtx = serverCtx
	u.running = true

	u.wg.Add(1)
	go u.acceptConnections()

	log.Printf("Unix socket transport started on %s", u.path)
	return nil
}

// Stop 停止 Unix Socket 传输层
func (u *UnixSocketTransport) Stop() error {
	u.mu.Lock()
	defer u.mu.Unlock()

	if !u.running {
		return nil
	}

	u.running = false
	close(u.quitChan)

	if u.listener != nil {
		u.listener.Close()
	}

	// 删除 socket 文件
	os.Remove(u.path)

	u.wg.Wait()
	log.Printf("Unix socket transport stopped")
	return nil
}

// acceptConnections 接受连接
func (u *UnixSocketTransport) acceptConnections() {
	defer u.wg.Done()

	for {
		select {
		case <-u.quitChan:
			return
		default:
			conn, err := u.listener.Accept()
			if err != nil {
				if u.running {
					log.Printf("Unix socket accept error: %v", err)
				}
				return
			}

			u.wg.Add(1)
			go u.handleConnection(conn)
		}
	}
}

// handleConnection 处理连接
func (u *UnixSocketTransport) handleConnection(conn net.Conn) {
	defer u.wg.Done()
	defer conn.Close()

	reader := &UnixSocketReader{Conn: conn}
	writer := &UnixSocketWriter{Conn: conn}

	// 创建连接信息
	connInfo := &ConnInfo{
		ID:       generateID(),
		Remote:   conn.RemoteAddr(),
		Protocol: "unix",
		Metadata: make(map[string]interface{}),
		Reader:   reader,
		Writer:   writer,
	}

	// 添加到连接管理器
	u.serverCtx.Connections.AddConnection(connInfo)

	// 创建上下文
	ctx := &Context{
		ServerInfo: u.serverCtx.ServerInfo,
		ConnInfo:   connInfo,
	}

	// 连接关闭时从管理器移除
	defer u.serverCtx.Connections.RemoveConnection(connInfo.ID)

	// 持续处理连接上的数据
	for {
		// 获取处理器并处理数据
		handler := u.serverCtx.GetHandler()
		if handler != nil {
			if err := handler.Handle(ctx, reader, writer); err != nil {
				log.Printf("Unix socket handler error: %v", err)
			}
		}
	}
}

// UnixSocketReader Unix Socket 读取器
type UnixSocketReader struct {
	Conn net.Conn
}

// Read 读取原始数据
func (r *UnixSocketReader) Read() ([]byte, error) {
	buffer := make([]byte, 1024)
	n, err := r.Conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}

// Close 关闭读取器
func (r *UnixSocketReader) Close() error {
	if r.Conn != nil {
		return r.Conn.Close()
	}
	return nil
}

// UnixSocketWriter Unix Socket 写入器
type UnixSocketWriter struct {
	Conn net.Conn
}

// Write 写入原始数据
func (w *UnixSocketWriter) Write(data []byte) error {
	_, err := w.Conn.Write(data)
	return err
}

// Close 关闭写入器
func (w *UnixSocketWriter) Close() error {
	if w.Conn != nil {
		return w.Conn.Close()
	}
	return nil
}

// NewHandlers 创建 Unix Socket 读写器
func (u *UnixSocketTransport) NewHandlers(conn net.Conn) (Reader, Writer) {
	return &UnixSocketReader{Conn: conn}, &UnixSocketWriter{Conn: conn}
}

