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

	// 主动关闭所有活跃连接，而不是等待它们自然结束
	if t.serverCtx != nil && t.serverCtx.Connections != nil {
		t.serverCtx.Connections.CloseAllConnections()
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

	reader := &TCPReader{Conn: conn, quitChan: t.quitChan}
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

	// 监听quit信号，如果收到则立即关闭连接
	go func() {
		select {
		case <-t.quitChan:
			conn.Close()
		}
	}()

	// 获取处理器
	handler := t.serverCtx.GetHandler()
	if handler != nil {
		// 只调用一次 Handle，让 Handle 方法负责持续处理连接
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			// 只记录非网络错误，避免大量 broken pipe 日志
			if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
				if err.Error() != "EOF" && err.Error() != "write: broken pipe" && 
				   err.Error() != "use of closed network connection" {
					log.Printf("TCP handler error: %v", err)
				}
			}
		}
	}
}

// TCPReader TCP 读取器
type TCPReader struct {
	Conn     net.Conn
	quitChan <-chan struct{}
}

// Read 读取数据到提供的缓冲区中，符合 io.Reader 接口
func (r *TCPReader) Read(p []byte) (n int, err error) {
	return r.Conn.Read(p)
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

// Write 写入数据，符合 io.Writer 接口
func (w *TCPWriter) Write(p []byte) (n int, err error) {
	// 确保数据以换行符结尾，这样客户端可以正确读取
	data := p
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	n, err = w.Conn.Write(data)
	if err != nil {
		return n, err
	}
	
	// 立即刷新数据，确保广播消息能及时发送
	if tcpConn, ok := w.Conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)
	}
	
	return n, nil
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
