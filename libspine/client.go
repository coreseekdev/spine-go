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
	conn      net.Conn
	reader    transport.Reader
	writer    transport.Writer
}

// NewClient 创建新的客户端
func NewClient(protocol, address string) (*Client, error) {
	var conn net.Conn
	var err error

	switch protocol {
	case "tcp":
		conn, err = net.Dial("tcp", address)
	case "unix":
		conn, err = net.Dial("unix", address)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}

	if err != nil {
		return nil, err
	}

	// 创建对应的 Reader 和 Writer
	var reader transport.Reader
	var writer transport.Writer

	switch protocol {
	case "tcp":
		reader = &transport.TCPReader{Conn: conn}
		writer = &transport.TCPWriter{Conn: conn}
	case "unix":
		reader = &transport.UnixSocketReader{Conn: conn}
		writer = &transport.UnixSocketWriter{Conn: conn}
	}

	return &Client{
		conn:   conn,
		reader: reader,
		writer: writer,
	}, nil
}

// Connect 连接到服务器（已弃用，连接在 NewClient 中建立）
func (c *Client) Connect(address string) error {
	// 连接已在 NewClient 中建立，此方法保留向后兼容
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
	if _, err := c.writer.Write(requestData); err != nil {
		return nil, err
	}

	// 读取响应
	buffer := make([]byte, 4096)
	n, err := c.reader.Read(buffer)
	if err != nil {
		return nil, err
	}
	responseData := buffer[:n]

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