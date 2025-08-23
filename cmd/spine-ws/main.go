package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 消息结构
type Message struct {
	ID     string      `json:"id,omitempty"`
	Method string      `json:"method,omitempty"`
	Path   string      `json:"path,omitempty"`
	Data   interface{} `json:"data,omitempty"`
	Status int         `json:"status,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// 客户端配置
type Config struct {
	Host     string
	Port     int
	Username string
	Secure   bool
}

func main() {
	// 解析命令行参数
	var config Config
	flag.StringVar(&config.Host, "host", "localhost", "服务器主机名")
	flag.IntVar(&config.Port, "port", 8000, "服务器端口")
	flag.StringVar(&config.Username, "username", "WebUser", "聊天用户名")
	flag.BoolVar(&config.Secure, "secure", false, "使用安全连接 (wss)")
	flag.Parse()

	// 设置日志格式
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// 构建 WebSocket URL
	scheme := "ws"
	if config.Secure {
		scheme = "wss"
	}
	u := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", config.Host, config.Port),
		Path:   "/ws",
	}
	log.Printf("连接到 %s", u.String())

	// 创建客户端状态
	var (
		conn      *websocket.Conn
		messageID int
		mutex     sync.Mutex
		done      = make(chan struct{})
		interrupt = make(chan os.Signal, 1)
	)
	signal.Notify(interrupt, os.Interrupt)

	// 连接函数
	connect := func() bool {
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Printf("连接失败: %v, 将在 5 秒后重试", err)
			return false
		}
		conn = c
		log.Println("连接成功!")
		return true
	}

	// 首次连接
	if !connect() {
		log.Fatal("无法连接到服务器")
	}
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	// 心跳检测
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				mutex.Lock()
				if conn != nil {
					// 发送 PING 消息
					messageID++
					pingMsg := Message{
						ID:     fmt.Sprintf("%d", messageID),
						Method: "PING",
						Path:   "/",
					}
					pingData, _ := json.Marshal(pingMsg)
					if err := conn.WriteMessage(websocket.TextMessage, pingData); err != nil {
						log.Printf("心跳发送失败: %v, 尝试重连", err)
						conn.Close()
						if connect() {
							// 重新加入聊天
							joinChat(conn, &messageID, config.Username)
						}
					}
				}
				mutex.Unlock()
			}
		}
	}()

	// 处理接收到的消息
	go func() {
		for {
			if conn == nil {
				time.Sleep(time.Second)
				continue
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("读取错误: %v, 尝试重连", err)
				mutex.Lock()
				if conn != nil {
					conn.Close()
				}
				if connect() {
					// 重新加入聊天
					joinChat(conn, &messageID, config.Username)
				} else {
					conn = nil
					time.Sleep(5 * time.Second) // 等待一段时间再尝试
				}
				mutex.Unlock()
				continue
			}

			// 解析消息
			var msg Message
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("JSON解析错误: %v, 原始消息: %s", err, string(message))
				continue
			}

			// 根据消息类型处理
			if msg.Data != nil {
				// 处理聊天消息
				if data, ok := msg.Data.(map[string]interface{}); ok {
					if user, hasUser := data["user"].(string); hasUser {
						if message, hasMessage := data["message"].(string); hasMessage {
							timestamp := time.Now().Format("15:04:05")
							fmt.Printf("[%s] %s: %s\n", timestamp, user, message)
							continue
						}
					}
				}
			}

			// 处理系统消息
			if msg.Status == 200 {
				if msg.Data != nil {
					if data, ok := msg.Data.(map[string]interface{}); ok {
						if message, hasMessage := data["message"].(string); hasMessage {
							fmt.Printf("系统: %s\n", message)
							continue
						}
					}
				}
			}

			// 处理错误消息
			if msg.Error != "" {
				fmt.Printf("错误: %s\n", msg.Error)
				continue
			}

			// 其他消息类型
			log.Printf("收到消息: %s", string(message))
		}
	}()

	// 自动加入聊天
	joinChat(conn, &messageID, config.Username)

	// 处理用户输入
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Printf("已以用户名 '%s' 加入聊天。输入消息发送，输入 /quit 退出。\n", config.Username)
		
		for scanner.Scan() {
			text := scanner.Text()
			
			// 处理命令
			if strings.HasPrefix(text, "/") {
				cmd := strings.TrimSpace(strings.TrimPrefix(text, "/"))
				
				switch cmd {
				case "quit", "exit":
					close(done)
					return
				case "help":
					fmt.Println("可用命令:")
					fmt.Println("  /quit, /exit - 退出程序")
					fmt.Println("  /help - 显示帮助信息")
					continue
				default:
					fmt.Printf("未知命令: %s\n", cmd)
					continue
				}
			}
			
			// 发送聊天消息
			if text != "" {
				mutex.Lock()
				if conn != nil {
					sendChatMessage(conn, &messageID, config.Username, text)
				} else {
					fmt.Println("未连接到服务器，无法发送消息")
				}
				mutex.Unlock()
			}
		}
	}()

	// 等待中断信号或完成信号
	select {
		case <-done:
			log.Println("程序正常退出")
		case <-interrupt:
			log.Println("收到中断信号，关闭连接...")
			mutex.Lock()
			if conn != nil {
				// 发送关闭消息
				err := conn.WriteMessage(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				)
				if err != nil {
					log.Println("写入关闭消息错误:", err)
				}
			}
			mutex.Unlock()
			
			// 等待一小段时间
			time.Sleep(time.Second)
	}
}

// 加入聊天
func joinChat(conn *websocket.Conn, messageID *int, username string) {
	*messageID++
	joinRequest := Message{
		ID:     fmt.Sprintf("%d", *messageID),
		Method: "JOIN",
		Path:   "/chat",
		Data:   map[string]interface{}{},
	}
	
	requestData, err := json.Marshal(joinRequest)
	if err != nil {
		log.Println("JSON编码错误:", err)
		return
	}
	
	if err := conn.WriteMessage(websocket.TextMessage, requestData); err != nil {
		log.Println("发送JOIN请求错误:", err)
	}
}

// 发送聊天消息
func sendChatMessage(conn *websocket.Conn, messageID *int, username, text string) {
	*messageID++
	chatRequest := Message{
		ID:     fmt.Sprintf("%d", *messageID),
		Method: "POST",
		Path:   "/chat",
		Data: map[string]interface{}{
			"user":    username,
			"message": text,
		},
	}
	
	requestData, err := json.Marshal(chatRequest)
	if err != nil {
		log.Println("JSON编码错误:", err)
		return
	}
	
	if err := conn.WriteMessage(websocket.TextMessage, requestData); err != nil {
		log.Println("发送消息错误:", err)
	}
}
