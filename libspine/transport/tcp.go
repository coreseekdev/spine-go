package transport

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// TCPTransport TCP 传输层实现
type TCPTransport struct {
	listener  net.Listener
	serverCtx *ServerContext
	running   bool
	mu        sync.RWMutex
	quitChan  chan struct{}
	wg        sync.WaitGroup
}

// NewTCPTransport 创建新的 TCP 传输层
func NewTCPTransport(addr string) (*TCPTransport, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &TCPTransport{
		listener: listener,
		quitChan: make(chan struct{}),
	}, nil
}

// Start 启动 TCP 传输层
func (t *TCPTransport) Start(serverCtx *ServerContext) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return fmt.Errorf("TCP transport is already running")
	}

	t.serverCtx = serverCtx
	t.running = true

	t.wg.Add(1)
	go t.acceptConnections()

	log.Printf("TCP transport started on %s", t.listener.Addr())
	return nil
}

// Stop 停止 TCP 传输层
func (t *TCPTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	t.running = false
	close(t.quitChan)

	if t.listener != nil {
		t.listener.Close()
	}

	t.wg.Wait()
	log.Printf("TCP transport stopped")
	return nil
}

// acceptConnections 接受连接
func (t *TCPTransport) acceptConnections() {
	defer t.wg.Done()

	for {
		select {
		case <-t.quitChan:
			return
		default:
			conn, err := t.listener.Accept()
			if err != nil {
				if t.running {
					log.Printf("TCP accept error: %v", err)
				}
				return
			}

			t.wg.Add(1)
			go t.handleConnection(conn)
		}
	}
}

// handleConnection 处理连接
func (t *TCPTransport) handleConnection(conn net.Conn) {
	defer t.wg.Done()
	defer conn.Close()

	reader := &TCPReader{Conn: conn}
	writer := &TCPWriter{Conn: conn}

	// 创建连接信息
	connInfo := &ConnInfo{
		ID:       generateID(),
		Remote:   conn.RemoteAddr(),
		Protocol: "tcp",
		Metadata: make(map[string]interface{}),
		Reader:   reader,
		Writer:   writer,
	}

	// 添加到连接管理器
	t.serverCtx.Connections.AddConnection(connInfo)

	// 创建上下文
	ctx := &Context{
		ServerInfo:        t.serverCtx.ServerInfo,
		ConnInfo:          connInfo,
		ConnectionManager: t.serverCtx.Connections,
	}

	// 连接关闭时从管理器移除
	defer t.serverCtx.Connections.RemoveConnection(connInfo.ID)

	// 持续处理连接上的数据
	for {
		// 获取处理器并处理数据
		handler := t.serverCtx.GetHandler()
		if handler != nil {
			if err := handler.Handle(ctx, reader, writer); err != nil {
				log.Printf("TCP handler error: %v", err)
			}
		}
	}
}

// TCPReader TCP 读取器
type TCPReader struct {
	Conn net.Conn
}

// Read 读取原始数据
func (r *TCPReader) Read() ([]byte, error) {
	buffer := make([]byte, 1024)
	n, err := r.Conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}

// Close 关闭读取器
func (r *TCPReader) Close() error {
	if r.Conn != nil {
		return r.Conn.Close()
	}
	return nil
}

// TCPWriter TCP 写入器
type TCPWriter struct {
	Conn net.Conn
}

// Write 写入原始数据
func (w *TCPWriter) Write(data []byte) error {
	_, err := w.Conn.Write(data)
	return err
}

// Close 关闭写入器
func (w *TCPWriter) Close() error {
	if w.Conn != nil {
		return w.Conn.Close()
	}
	return nil
}

// NewHandlers 创建 TCP 读写器
func (t *TCPTransport) NewHandlers(conn net.Conn) (Reader, Writer) {
	return &TCPReader{Conn: conn}, &TCPWriter{Conn: conn}
}

// generateID 生成唯一 ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
