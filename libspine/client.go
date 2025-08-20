package libspine

import (
	"encoding/json"
	"fmt"
	"net"
	"spine-go/libspine/transport"
	"time"
)

// Client 客户端结构
type Client struct {
	transport transport.Transport
	conn      net.Conn
	reader    transport.Reader
	writer    transport.Writer
}

// NewClient 创建新的客户端
func NewClient(protocol, address string) (*Client, error) {
	var t transport.Transport
	var err error

	switch protocol {
	case "tcp":
		t, err = transport.NewTCPTransport(address)
	case "unix":
		t, err = transport.NewUnixSocketTransport(address)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}

	if err != nil {
		return nil, err
	}

	return &Client{
		transport: t,
	}, nil
}

// Connect 连接到服务器
func (c *Client) Connect(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}

	c.conn = conn
	c.reader, c.writer = c.transport.NewHandlers(conn)
	return nil
}

// SendRequest 发送请求
func (c *Client) SendRequest(method, path string, body []byte) (*transport.Response, error) {
	if c.writer == nil {
		return nil, fmt.Errorf("client not connected")
	}

	// 创建请求对象
	request := map[string]interface{}{
		"id":     generateClientID(),
		"method": method,
		"path":   path,
		"body":   string(body),
	}

	// 序列化为 JSON
	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// 发送原始数据
	if err := c.writer.Write(requestData); err != nil {
		return nil, err
	}

	// 读取响应
	responseData, err := c.reader.Read()
	if err != nil {
		return nil, err
	}

	// 解析响应
	var response transport.Response
	if err := json.Unmarshal(responseData, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// SendJSON 发送 JSON 请求
func (c *Client) SendJSON(method, path string, data interface{}) (*transport.Response, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return c.SendRequest(method, path, body)
}

// Close 关闭客户端
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// generateClientID 生成唯一 ID
func generateClientID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}