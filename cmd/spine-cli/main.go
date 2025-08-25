package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"spine-go/libspine/transport"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

type ChatMessage struct {
	User    string `json:"user"`
	Message string `json:"message"`
}

type RedisRequest struct {
	Command string      `json:"command"`
	Key     string      `json:"key"`
	Value   interface{} `json:"value"`
	TTL     int64       `json:"ttl"`
}

// isWindows 检测当前操作系统是否为 Windows
func isWindows() bool {
	return runtime.GOOS == "windows"
}

// getDefaultLocalPath 获取默认的本地路径
func getDefaultLocalPath() string {
	if isWindows() {
		return "/spine" // 对应 \\.\pipe\spine
	} else {
		return "/tmp/spine.sock"
	}
}

// convertLocalPath 转换本地路径
// Unix: 直接使用原路径
// Windows: 将 /abc/xyz 转换为 \\.\pipe\abc\xyz
func convertLocalPath(path string) string {
	if isWindows() {
		// Windows Named Pipe 路径转换
		if strings.HasPrefix(path, "/") {
			// 移除开头的 /，然后转换为 Windows 路径分隔符
			pipePath := strings.TrimPrefix(path, "/")
			pipePath = strings.ReplaceAll(pipePath, "/", "\\")
			return "\\\\.\\pipe\\" + pipePath
		}
		return path
	} else {
		// Unix Socket 路径直接使用
		return path
	}
}

// connectNamedPipe 连接到 Windows Named Pipe
func connectNamedPipe(pipeName string) (net.Conn, error) {
	if !isWindows() {
		return nil, fmt.Errorf("Named Pipe is only supported on Windows")
	}

	// 转换管道名称为 UTF16
	pipeName16, err := syscall.UTF16PtrFromString(pipeName)
	if err != nil {
		return nil, fmt.Errorf("failed to convert pipe name to UTF16: %v", err)
	}

	// 尝试连接，如果管道不存在则等待
	var handle windows.Handle
	for i := 0; i < 50; i++ { // 最多重试 50 次，每次等待 100ms
		// 尝试打开 named pipe，使用重叠I/O以支持超时
		handle, err = windows.CreateFile(
			pipeName16,
			windows.GENERIC_READ|windows.GENERIC_WRITE,
			0,
			nil,
			windows.OPEN_EXISTING,
			windows.FILE_FLAG_OVERLAPPED, // 使用重叠I/O以支持超时
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
		return nil, fmt.Errorf("failed to open named pipe: %v", err)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to named pipe after retries: %v", err)
	}

	return &NamedPipeConn{handle: handle}, nil
}

// NamedPipeConn Windows Named Pipe 连接包装器
type NamedPipeConn struct {
	handle windows.Handle
}

func (c *NamedPipeConn) Read(b []byte) (n int, err error) {
	var bytesRead uint32
	
	// 创建重叠结构用于异步I/O
	overlapped := &windows.Overlapped{}
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create event: %v", err)
	}
	defer windows.CloseHandle(event)
	overlapped.HEvent = event
	
	err = windows.ReadFile(c.handle, b, &bytesRead, overlapped)
	if err != nil {
		// 检查是否是管道断开
		if err == windows.ERROR_BROKEN_PIPE || err == windows.ERROR_PIPE_NOT_CONNECTED {
			return 0, io.EOF
		}
		// 检查是否是异步操作正在进行
		if err == windows.ERROR_IO_PENDING {
			// 等待操作完成，设置30秒超时
			waitResult, waitErr := windows.WaitForSingleObject(event, 30000)
			if waitErr != nil {
				return 0, fmt.Errorf("wait failed: %v", waitErr)
			}
			if waitResult == uint32(windows.WAIT_TIMEOUT) {
				return 0, fmt.Errorf("read timeout")
			}
			// 获取实际读取的字节数
			err = windows.GetOverlappedResult(c.handle, overlapped, &bytesRead, false)
			if err != nil {
				if err == windows.ERROR_BROKEN_PIPE || err == windows.ERROR_PIPE_NOT_CONNECTED {
					return 0, io.EOF
				}
				return 0, fmt.Errorf("GetOverlappedResult failed: %v", err)
			}
		} else {
			return 0, fmt.Errorf("ReadFile failed: %v", err)
		}
	}
	
	// 如果读取了0字节但没有错误，可能是管道关闭
	if bytesRead == 0 {
		return 0, io.EOF
	}
	return int(bytesRead), nil
}

func (c *NamedPipeConn) Write(b []byte) (n int, err error) {
	var bytesWritten uint32
	
	// 创建重叠结构用于异步I/O
	overlapped := &windows.Overlapped{}
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create event: %v", err)
	}
	defer windows.CloseHandle(event)
	overlapped.HEvent = event
	
	err = windows.WriteFile(c.handle, b, &bytesWritten, overlapped)
	if err != nil {
		// 检查是否是异步操作正在进行
		if err == windows.ERROR_IO_PENDING {
			// 等待操作完成，设置30秒超时
			waitResult, waitErr := windows.WaitForSingleObject(event, 30000)
			if waitErr != nil {
				return 0, fmt.Errorf("wait failed: %v", waitErr)
			}
			if waitResult == uint32(windows.WAIT_TIMEOUT) {
				return 0, fmt.Errorf("write timeout")
			}
			// 获取实际写入的字节数
			err = windows.GetOverlappedResult(c.handle, overlapped, &bytesWritten, false)
			if err != nil {
				return 0, fmt.Errorf("GetOverlappedResult failed: %v", err)
			}
		} else {
			return 0, fmt.Errorf("failed to write to named pipe: %v", err)
		}
	}
	
	if int(bytesWritten) != len(b) {
		return int(bytesWritten), fmt.Errorf("incomplete write: wrote %d bytes, expected %d", bytesWritten, len(b))
	}
	return int(bytesWritten), nil
}

func (c *NamedPipeConn) Close() error {
	return windows.CloseHandle(c.handle)
}

func (c *NamedPipeConn) LocalAddr() net.Addr {
	return &NamedPipeAddr{pipeName: "local"}
}

func (c *NamedPipeConn) RemoteAddr() net.Addr {
	return &NamedPipeAddr{pipeName: "remote"}
}

func (c *NamedPipeConn) SetDeadline(t time.Time) error {
	// Named Pipe 不支持 deadline
	return nil
}

func (c *NamedPipeConn) SetReadDeadline(t time.Time) error {
	// Named Pipe 不支持 read deadline
	return nil
}

func (c *NamedPipeConn) SetWriteDeadline(t time.Time) error {
	// Named Pipe 不支持 write deadline
	return nil
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

func main() {
	var (
		serverAddr = flag.String("server", "localhost:8080", "Server address")
		protocol   = flag.String("protocol", "tcp", "Protocol (tcp/local)")
		localPath  = flag.String("local", getDefaultLocalPath(), "Local socket/pipe path")
		mode       = flag.String("mode", "chat", "Mode (chat/redis)")
		username   = flag.String("username", "", "Username for chat mode")
	)
	flag.Parse()

	switch *mode {
	case "chat":
		runChatClient(*protocol, *serverAddr, *localPath, *username)
	case "redis":
		runRedisClient(*protocol, *serverAddr, *localPath)
	default:
		log.Fatal("Invalid mode. Use 'chat' or 'redis'")
	}
}

func runChatClient(protocol, serverAddr, localPath, username string) {
	var conn net.Conn
	var err error

	switch protocol {
	case "tcp":
		conn, err = net.Dial("tcp", serverAddr)
	case "local":
		// 根据平台转换路径并选择协议
		address := convertLocalPath(localPath)
		if isWindows() {
			conn, err = connectNamedPipe(address)
		} else {
			conn, err = net.Dial("unix", address)
		}
	default:
		log.Fatal("Unsupported protocol")
	}

	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("Connected to chat server")
	fmt.Println("Available commands:")
	fmt.Println("  /join - Join the chat")
	fmt.Println("  /leave - Leave the chat")
	fmt.Println("  /get - Get all messages")
	fmt.Println("  /quit - Quit")
	fmt.Println("  Any other message will be sent to the chat")

	// 创建一个通道来通知连接断开
	connClosed := make(chan bool, 1)
	
	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			fmt.Printf("Received: %s\n", scanner.Text())
		}
		// 连接断开时通知主线程
		if err := scanner.Err(); err != nil {
			fmt.Printf("Connection error: %v\n", err)
		}
		connClosed <- true
	}()

	scanner := bufio.NewScanner(os.Stdin)
	
	// If username wasn't provided as a command line argument, prompt for it
	if username == "" {
		fmt.Print("Enter your username: ")
		if !scanner.Scan() {
			return
		}
		username = strings.TrimSpace(scanner.Text())
	}
	
	// Join the chat automatically
	sendChatRequest(conn, "JOIN", "/chat", nil)
	fmt.Println("Joined the chat as", username)

	// 创建输入通道
	inputChan := make(chan string)
	
	// 启动输入处理 goroutine
	go func() {
		for {
			fmt.Print("> ")
			if !scanner.Scan() {
				return
			}
			input := strings.TrimSpace(scanner.Text())
			if input != "" {
				inputChan <- input
			}
		}
	}()
	
	// 主循环：处理输入和连接状态
	for {
		select {
		case input := <-inputChan:
			if input == "/quit" {
				return
			}
			
			if input == "/join" {
				sendChatRequest(conn, "JOIN", "/chat", nil)
				fmt.Println("Joined the chat")
				continue
			}
			
			if input == "/leave" {
				sendChatRequest(conn, "LEAVE", "/chat", nil)
				fmt.Println("Left the chat")
				continue
			}
			
			if input == "/get" {
				sendChatRequest(conn, "GET", "/chat", nil)
				continue
			}
			
			// 发送聊天消息
			sendChatRequest(conn, "POST", "/chat", ChatMessage{
				User:    username,
				Message: input,
			})
			
		case <-connClosed:
			fmt.Println("Connection closed. Exiting...")
			return
		}
	}
}

func runRedisClient(protocol, serverAddr, localPath string) {
	var conn net.Conn
	var err error

	switch protocol {
	case "tcp":
		conn, err = net.Dial("tcp", serverAddr)
	case "local":
		// 根据平台转换路径并选择协议
		address := convertLocalPath(localPath)
		if isWindows() {
			conn, err = connectNamedPipe(address)
		} else {
			conn, err = net.Dial("unix", address)
		}
	default:
		log.Fatal("Unsupported protocol")
	}

	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	fmt.Println("Connected to Redis server")
	fmt.Println("Available commands:")
	fmt.Println("  SET <key> <value> [ttl] - Set key value")
	fmt.Println("  GET <key> - Get key value")
	fmt.Println("  DELETE <key> - Delete key")
	fmt.Println("  EXISTS <key> - Check if key exists")
	fmt.Println("  TTL <key> - Get key TTL")
	fmt.Println("  /quit - Quit")

	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			fmt.Printf("Response: %s\n", scanner.Text())
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("redis> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if input == "/quit" {
			break
		}

		parts := strings.Fields(input)
		if len(parts) == 0 {
			continue
		}

		command := strings.ToUpper(parts[0])
		var request RedisRequest

		switch command {
		case "SET":
			if len(parts) < 3 {
				fmt.Println("Usage: SET <key> <value> [ttl]")
				continue
			}
			request = RedisRequest{
				Command: command,
				Key:     parts[1],
				Value:   parts[2],
			}
			if len(parts) > 3 {
				request.TTL = 0 // 这里可以解析 TTL
			}

		case "GET", "DELETE", "EXISTS", "TTL":
			if len(parts) < 2 {
				fmt.Printf("Usage: %s <key>\n", command)
				continue
			}
			request = RedisRequest{
				Command: command,
				Key:     parts[1],
			}

		default:
			fmt.Printf("Unknown command: %s\n", command)
			continue
		}

		sendRedisRequest(conn, request)
	}
}

func sendChatRequest(conn net.Conn, method, path string, data interface{}) {
	request := transport.Request{
		ID:     generateID(),
		Method: method,
		Path:   path,
	}

	if data != nil {
		body, err := json.Marshal(data)
		if err != nil {
			log.Printf("Failed to marshal data: %v", err)
			return
		}
		request.Body = body
	}

	sendRequest(conn, request)
}

func sendRedisRequest(conn net.Conn, request RedisRequest) {
	body, err := json.Marshal(request)
	if err != nil {
		log.Printf("Failed to marshal request: %v", err)
		return
	}

	req := transport.Request{
		ID:     generateID(),
		Method: "POST",
		Path:   "/redis",
		Body:   body,
	}

	sendRequest(conn, req)
}

func sendRequest(conn net.Conn, request transport.Request) {
	// 将请求对象序列化为 JSON
	chatReq := struct {
		Method string          `json:"method"`
		Path   string          `json:"path"`
		Data   json.RawMessage `json:"data"`
	}{
		Method: request.Method,
		Path:   request.Path,
		Data:   request.Body,
	}

	// 序列化为 JSON
	jsonData, err := json.Marshal(chatReq)
	if err != nil {
		log.Printf("Failed to marshal request to JSON: %v", err)
		return
	}

	// 添加换行符以支持 JSONL 协议
	jsonData = append(jsonData, '\n')

	// 发送 JSON 数据
	_, err = conn.Write(jsonData)
	if err != nil {
		log.Printf("Failed to send request: %v", err)
	}
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
