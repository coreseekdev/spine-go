//go:build windows

package transport

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"golang.org/x/sys/windows"
)

// NamedPipeTransport Windows Named Pipe 传输层实现
type NamedPipeTransport struct {
	pipeName  string
	serverCtx *ServerContext
	running   bool
	mu        sync.RWMutex
	quitChan  chan struct{}
	wg        sync.WaitGroup
}

// NewNamedPipeTransport 创建新的 Named Pipe 传输层
func NewNamedPipeTransport(pipeName string) (*NamedPipeTransport, error) {
	// 确保管道名称格式正确
	if pipeName[0] != '\\' {
		pipeName = `\\.\pipe\` + pipeName
	}
	
	return &NamedPipeTransport{
		pipeName: pipeName,
		quitChan: make(chan struct{}),
	}, nil
}

// Start 启动 Named Pipe 传输层
func (t *NamedPipeTransport) Start(serverCtx *ServerContext) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.running {
		return fmt.Errorf("Named Pipe transport is already running")
	}

	t.serverCtx = serverCtx
	t.running = true

	t.wg.Add(1)
	go t.acceptConnections()

	log.Printf("Named Pipe transport started on %s", t.pipeName)
	return nil
}

// Stop 停止 Named Pipe 传输层
func (t *NamedPipeTransport) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.running {
		return nil
	}

	t.running = false
	close(t.quitChan)

	// 主动关闭所有活跃连接
	if t.serverCtx != nil && t.serverCtx.Connections != nil {
		t.serverCtx.Connections.CloseAllConnections()
	}

	t.wg.Wait()
	log.Printf("Named Pipe transport stopped")
	return nil
}

// acceptConnections 接受连接
func (t *NamedPipeTransport) acceptConnections() {
	defer t.wg.Done()

	for {
		select {
		case <-t.quitChan:
			return
		default:
			// 创建命名管道
			pipeHandle, err := t.createNamedPipe()
			if err != nil {
				if t.running {
					log.Printf("Named Pipe create error: %v", err)
				}
				return
			}

			// 等待客户端连接
			err = t.connectNamedPipe(pipeHandle)
			if err != nil {
				windows.CloseHandle(pipeHandle)
				if t.running {
					log.Printf("Named Pipe connect error: %v", err)
				}
				continue
			}

			t.wg.Add(1)
			go t.handleConnection(pipeHandle)
		}
	}
}

// createNamedPipe 创建命名管道
func (t *NamedPipeTransport) createNamedPipe() (windows.Handle, error) {
	pipeName, err := windows.UTF16PtrFromString(t.pipeName)
	if err != nil {
		return windows.InvalidHandle, err
	}

	handle, err := windows.CreateNamedPipe(
		pipeName,
		windows.PIPE_ACCESS_DUPLEX,
		windows.PIPE_TYPE_BYTE|windows.PIPE_READMODE_BYTE|windows.PIPE_WAIT,
		windows.PIPE_UNLIMITED_INSTANCES,
		4096, // 输出缓冲区大小
		4096, // 输入缓冲区大小
		0,    // 默认超时
		nil,  // 默认安全属性
	)

	if err != nil {
		return windows.InvalidHandle, err
	}

	return handle, nil
}

// connectNamedPipe 等待客户端连接
func (t *NamedPipeTransport) connectNamedPipe(handle windows.Handle) error {
	return windows.ConnectNamedPipe(handle, nil)
}

// handleConnection 处理连接
func (t *NamedPipeTransport) handleConnection(handle windows.Handle) {
	defer t.wg.Done()
	defer windows.CloseHandle(handle)

	conn := &NamedPipeConn{handle: handle}
	reader := &NamedPipeReader{conn: conn, quitChan: t.quitChan}
	writer := &NamedPipeWriter{conn: conn}

	// 创建连接信息
	connInfo := &ConnInfo{
		ID:       generateID(),
		Remote:   &NamedPipeAddr{pipeName: t.pipeName},
		Protocol: "namedpipe",
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
			windows.CloseHandle(handle)
		}
	}()

	// 获取处理器
	handler := t.serverCtx.GetHandler()
	if handler != nil {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			log.Printf("Named Pipe handler error: %v", err)
		}
	}
}

// NamedPipeConn Named Pipe 连接包装器
type NamedPipeConn struct {
	handle windows.Handle
}

// NamedPipeAddr Named Pipe 地址实现
type NamedPipeAddr struct {
	pipeName string
}

func (a *NamedPipeAddr) Network() string {
	return "namedpipe"
}

func (a *NamedPipeAddr) String() string {
	return a.pipeName
}

// NamedPipeReader Named Pipe 读取器
type NamedPipeReader struct {
	conn     *NamedPipeConn
	quitChan chan struct{}
}

func (r *NamedPipeReader) Read(p []byte) (n int, err error) {
	select {
	case <-r.quitChan:
		return 0, fmt.Errorf("connection closed")
	default:
	}

	var bytesRead uint32
	err = windows.ReadFile(r.conn.handle, p, &bytesRead, nil)
	if err != nil {
		return 0, err
	}
	return int(bytesRead), nil
}

func (r *NamedPipeReader) Close() error {
	return windows.CloseHandle(r.conn.handle)
}

// NamedPipeWriter Named Pipe 写入器
type NamedPipeWriter struct {
	conn *NamedPipeConn
}

func (w *NamedPipeWriter) Write(p []byte) (n int, err error) {
	var bytesWritten uint32
	err = windows.WriteFile(w.conn.handle, p, &bytesWritten, nil)
	if err != nil {
		return 0, err
	}
	return int(bytesWritten), nil
}

func (w *NamedPipeWriter) Close() error {
	return windows.CloseHandle(w.conn.handle)
}
