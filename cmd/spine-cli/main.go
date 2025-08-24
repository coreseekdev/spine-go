package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"spine-go/libspine/transport"
	"strings"
	"time"
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

func main() {
	var (
		serverAddr = flag.String("server", "localhost:8080", "Server address")
		protocol   = flag.String("protocol", "tcp", "Protocol (tcp/unix)")
		socketPath = flag.String("socket", "/tmp/spine.sock", "Unix socket path")
		mode       = flag.String("mode", "chat", "Mode (chat/redis)")
		username   = flag.String("username", "", "Username for chat mode")
	)
	flag.Parse()

	switch *mode {
	case "chat":
		runChatClient(*protocol, *serverAddr, *socketPath, *username)
	case "redis":
		runRedisClient(*protocol, *serverAddr, *socketPath)
	default:
		log.Fatal("Invalid mode. Use 'chat' or 'redis'")
	}
}

func runChatClient(protocol, serverAddr, socketPath, username string) {
	var conn net.Conn
	var err error

	switch protocol {
	case "tcp":
		conn, err = net.Dial("tcp", serverAddr)
	case "unix":
		conn, err = net.Dial("unix", socketPath)
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

	go func() {
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			fmt.Printf("Received: %s\n", scanner.Text())
		}
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

	for {
		fmt.Print("> ")
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
	}
}

func runRedisClient(protocol, serverAddr, socketPath string) {
	var conn net.Conn
	var err error

	switch protocol {
	case "tcp":
		conn, err = net.Dial("tcp", serverAddr)
	case "unix":
		conn, err = net.Dial("unix", socketPath)
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
