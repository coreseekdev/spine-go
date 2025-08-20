package transport

import (
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"
)

// TransportTestHelpers transport 测试辅助函数
type TransportTestHelpers struct{}

func NewTransportTestHelpers() *TransportTestHelpers {
	return &TransportTestHelpers{}
}

// GenerateID 生成测试 ID
func (h *TransportTestHelpers) GenerateID() string {
	return fmt.Sprintf("test-%d", time.Now().UnixNano())
}

// CreateTestConnection 创建测试连接
func (h *TransportTestHelpers) CreateTestConnection() (net.Conn, net.Conn) {
	server, client := net.Pipe()
	return server, client
}

// CreateTestTCPRequest 创建 TCP 格式的测试请求
func (h *TransportTestHelpers) CreateTestTCPRequest(method, path string, body interface{}) string {
	var bodyBytes []byte
	if body != nil {
		switch v := body.(type) {
		case []byte:
			bodyBytes = v
		case string:
			bodyBytes = []byte(v)
		default:
			bodyBytes, _ = json.Marshal(body)
		}
	}
	
	return fmt.Sprintf("%s %s HTTP/1.1\r\nContent-Length: %d\r\n\r\n%s", 
		method, path, len(bodyBytes), string(bodyBytes))
}

// CreateTestJSONRequest 创建 JSON 格式的测试请求
func (h *TransportTestHelpers) CreateTestJSONRequest(method, path string, body interface{}) *Request {
	var bodyBytes []byte
	if body != nil {
		switch v := body.(type) {
		case []byte:
			bodyBytes = v
		case string:
			bodyBytes = []byte(v)
		default:
			bodyBytes, _ = json.Marshal(body)
		}
	}
	
	return &Request{
		ID:     h.GenerateID(),
		Method: method,
		Path:   path,
		Header: make(map[string]string),
		Body:   bodyBytes,
	}
}

// AssertTCPRequest 断言 TCP 请求
func (h *TransportTestHelpers) AssertTCPRequest(t *testing.T, req *Request, expectedMethod, expectedPath string, expectedBodyContains string) {
	t.Helper()
	
	if req == nil {
		t.Fatalf("Expected request but got nil")
	}
	
	if req.Method != expectedMethod {
		t.Errorf("Expected method %s, got %s", expectedMethod, req.Method)
	}
	
	if req.Path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, req.Path)
	}
	
	if expectedBodyContains != "" && !h.Contains(string(req.Body), expectedBodyContains) {
		t.Errorf("Expected request body to contain '%s', got '%s'", expectedBodyContains, string(req.Body))
	}
}

// Contains 检查字符串是否包含子字符串
func (h *TransportTestHelpers) Contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(s)] == substr[:len(substr)]
}

// Wait 等待指定时间
func (h *TransportTestHelpers) Wait(duration time.Duration) {
	time.Sleep(duration)
}