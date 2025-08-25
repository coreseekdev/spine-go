//go:build windows

package transport

import (
	"fmt"
	"log"
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
// Named Pipe 的特性：每个管道实例只能服务一个客户端
// 为了支持多客户端，我们需要为每个连接创建新的管道实例
func (t *NamedPipeTransport) acceptConnections() {
	defer t.wg.Done()

	for {
		select {
		case <-t.quitChan:
			return
		default:
			// 为每个潜在的客户端连接创建新的管道实例
			// 使用相同的管道名称但不同的实例
			pipeHandle, err := t.createNamedPipeInstance()
			if err != nil {
				if t.running {
					log.Printf("Named Pipe create error: %v", err)
				}
				// 如果创建失败，稍等后重试
				select {
				case <-t.quitChan:
					return
				case <-time.After(100 * time.Millisecond):
					continue
				}
			}

			// 等待客户端连接到这个管道实例
			err = t.waitForClientConnection(pipeHandle)
			if err != nil {
				windows.CloseHandle(pipeHandle)
				if t.running {
					log.Printf("Named Pipe connect error: %v", err)
				}
				continue
			}

			// 成功连接后，启动处理协程
			// 同时继续循环创建新的管道实例等待下一个客户端
			t.wg.Add(1)
			go t.handleConnection(pipeHandle)
		}
	}
}

// createNamedPipeInstance 创建命名管道实例
// 每次调用都会创建一个新的管道实例，支持多客户端连接
func (t *NamedPipeTransport) createNamedPipeInstance() (windows.Handle, error) {
	pipeName, err := windows.UTF16PtrFromString(t.pipeName)
	if err != nil {
		return windows.InvalidHandle, err
	}

	handle, err := windows.CreateNamedPipe(
		pipeName,
		windows.PIPE_ACCESS_DUPLEX, // 移除 FILE_FLAG_OVERLAPPED
		windows.PIPE_TYPE_BYTE|windows.PIPE_READMODE_BYTE|windows.PIPE_WAIT,
		windows.PIPE_UNLIMITED_INSTANCES,
		4096, // 输出缓冲区大小
		4096, // 输入缓冲区大小
		1000, // 1秒超时
		nil,  // 默认安全属性
	)

	if err != nil {
		return windows.InvalidHandle, err
	}

	return handle, nil
}

// waitForClientConnection 等待客户端连接到管道实例
// 使用 goroutine 和 channel 来实现可中断的连接等待
func (t *NamedPipeTransport) waitForClientConnection(handle windows.Handle) error {
	// 使用 channel 来处理连接结果
	connectChan := make(chan error, 1)
	
	// 在 goroutine 中执行连接等待
	go func() {
		err := windows.ConnectNamedPipe(handle, nil)
		if err != nil {
			if err == windows.ERROR_PIPE_CONNECTED {
				// 客户端已经连接
				connectChan <- nil
				return
			}
			// 其他错误
			connectChan <- fmt.Errorf("ConnectNamedPipe failed: %v", err)
			return
		}
		connectChan <- nil
	}()

	// 等待连接完成或服务器停止
	select {
	case err := <-connectChan:
		return err
	case <-t.quitChan:
		// 服务器正在停止，关闭管道句柄来中断连接等待
		windows.CloseHandle(handle)
		return fmt.Errorf("server stopping")
	}
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
	closed bool
	mu     sync.Mutex
}

// Close 关闭连接，确保 handle 只被关闭一次
func (c *NamedPipeConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return nil // 已经关闭，直接返回
	}
	
	c.closed = true
	return windows.CloseHandle(c.handle)
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
	// 检查连接是否已关闭
	select {
	case <-r.quitChan:
		return 0, fmt.Errorf("connection closed")
	default:
	}

	// 使用同步读取，但设置较短的超时
	var bytesRead uint32
	err = windows.ReadFile(r.conn.handle, p, &bytesRead, nil)
	if err != nil {
		// 检查是否是管道断开
		if err == windows.ERROR_BROKEN_PIPE || err == windows.ERROR_PIPE_NOT_CONNECTED {
			return 0, fmt.Errorf("pipe disconnected")
		}
		return 0, fmt.Errorf("ReadFile failed: %v", err)
	}

	return int(bytesRead), nil
}

func (r *NamedPipeReader) Close() error {
	return r.conn.Close()
}

// NamedPipeWriter Named Pipe 写入器
type NamedPipeWriter struct {
	conn *NamedPipeConn
}

func (w *NamedPipeWriter) Write(p []byte) (n int, err error) {
	// 创建重叠结构用于异步I/O
	overlapped := &windows.Overlapped{}
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create event: %v", err)
	}
	defer windows.CloseHandle(event)
	overlapped.HEvent = event

	// 启动异步写入
	var bytesWritten uint32
	err = windows.WriteFile(w.conn.handle, p, &bytesWritten, overlapped)
	if err != nil && err != windows.ERROR_IO_PENDING {
		return 0, fmt.Errorf("WriteFile failed: %v", err)
	}

	// 如果立即完成，直接返回
	if err == nil {
		return int(bytesWritten), nil
	}

	// 等待异步操作完成
	wait, err := windows.WaitForSingleObject(event, 5000) // 5秒超时
	if err != nil {
		return 0, fmt.Errorf("WaitForSingleObject failed: %v", err)
	}
	
	if wait == 0x00000102 { // WAIT_TIMEOUT
		windows.CancelIo(w.conn.handle)
		return 0, fmt.Errorf("write timeout")
	}
	
	if wait == windows.WAIT_OBJECT_0 {
		// 操作完成，获取结果
		err = windows.GetOverlappedResult(w.conn.handle, overlapped, &bytesWritten, false)
		if err != nil {
			return 0, fmt.Errorf("GetOverlappedResult failed: %v", err)
		}
		return int(bytesWritten), nil
	}
	
	return 0, fmt.Errorf("unexpected wait result: %d", wait)
}

func (w *NamedPipeWriter) Close() error {
	return w.conn.Close()
}
