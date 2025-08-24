package handler

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"spine-go/libspine/transport"
	"strings"
	"testing"
	"time"
)

// MockReader 模拟 Reader 用于测试
type MockReader struct {
	data     [][]byte
	current  int
}

func NewMockReader(data [][]byte) *MockReader {
	return &MockReader{
		data:     data,
		current:  0,
	}
}

func NewMockReaderFromRequests(requests []*transport.Request) *MockReader {
	data := make([][]byte, len(requests))
	for i, req := range requests {
		// 将 transport.Request 转换为聊天处理器期望的格式
		var requestData interface{}
		if len(req.Body) > 0 {
			// 尝试解析 req.Body 作为 JSON
			json.Unmarshal(req.Body, &requestData)
		}
		
		chatRequest := map[string]interface{}{
			"method": req.Method,
			"path":   req.Path,
			"data":   requestData,
		}
		requestBytes, _ := json.Marshal(chatRequest)
		data[i] = requestBytes
	}
	return &MockReader{
		data:     data,
		current:  0,
	}
}

func (m *MockReader) Read(p []byte) (n int, err error) {
	if m.current >= len(m.data) {
		return 0, io.EOF
	}
	data := m.data[m.current]
	m.current++
	n = copy(p, data)
	return n, nil
}

func (m *MockReader) Close() error {
	return nil
}

// MockWriter 模拟 Writer 用于测试
type MockWriter struct {
	responses [][]byte
	buffer    bytes.Buffer
}

func NewMockWriter() *MockWriter {
	return &MockWriter{
		responses: make([][]byte, 0),
	}
}

func (m *MockWriter) Write(data []byte) (n int, err error) {
	m.responses = append(m.responses, data)
	n, err = m.buffer.Write(data)
	return n, err
}

func (m *MockWriter) Close() error {
	return nil
}

func (m *MockWriter) GetResponses() [][]byte {
	return m.responses
}

func (m *MockWriter) GetLastResponse() []byte {
	if len(m.responses) == 0 {
		return nil
	}
	return m.responses[len(m.responses)-1]
}

func (m *MockWriter) GetLastResponseAsMap() map[string]interface{} {
	if len(m.responses) == 0 {
		return nil
	}
	
	data := m.responses[len(m.responses)-1]
	
	// 首先尝试直接解析 JSON 数据
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err == nil {
		return result
	}
	
	// 如果直接解析失败，尝试解析二进制格式 [4字节长度] + [数据]
	if len(data) < 4 {
		return nil
	}
	
	length := binary.BigEndian.Uint32(data[:4])
	if len(data) < int(length)+4 {
		return nil
	}
	
	jsonData := data[4 : 4+length]
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil
	}
	
	return result
}

func (m *MockWriter) Clear() {
	m.responses = make([][]byte, 0)
	m.buffer.Reset()
}

// TestHelpers 测试辅助函数
type TestHelpers struct{}

func NewTestHelpers() *TestHelpers {
	return &TestHelpers{}
}

// CreateTestRequest 创建测试请求
func (h *TestHelpers) CreateTestRequest(method, path string, body interface{}) *transport.Request {
	var bodyBytes []byte
	if body != nil {
		switch v := body.(type) {
		case []byte:
			bodyBytes = v
		case string:
			bodyBytes = []byte(v)
		default:
			bodyBytes, _ = json.Marshal(v) // 直接序列化，不要双重序列化
		}
	}
	
	return &transport.Request{
		ID:     h.GenerateID(),
		Method: method,
		Path:   path,
		Header: make(map[string]string),
		Body:   bodyBytes,
	}
}

// CreateChatRequest 创建聊天请求
func (h *TestHelpers) CreateChatRequest(method, path string, data interface{}) []byte {
	request := map[string]interface{}{
		"method": method,
		"path":   path,
		"data":   data,
	}
	
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil
	}
	
	// 创建二进制消息格式
	length := uint32(len(requestBytes))
	buffer := new(bytes.Buffer)
	binary.Write(buffer, binary.BigEndian, length)
	buffer.Write(requestBytes)
	
	return buffer.Bytes()
}

// CreateTestContext 创建测试上下文
func (h *TestHelpers) CreateTestContext() *transport.Context {
	return &transport.Context{
		ServerInfo: &transport.ServerInfo{
			Address: "test-server:8080",
			Config:  make(map[string]interface{}),
		},
		ConnInfo: &transport.ConnInfo{
			ID:       h.GenerateID(),
			Protocol: "test",
			Metadata: make(map[string]interface{}),
		},
		ConnectionManager: transport.NewConnectionManager(),
	}
}

// GenerateID 生成测试 ID
func (h *TestHelpers) GenerateID() string {
	return fmt.Sprintf("test-%d", time.Now().UnixNano())
}

// AssertResponse 断言响应
func (h *TestHelpers) AssertResponse(t *testing.T, writer *MockWriter, expectedStatus int, expectedBodyContains string) {
	t.Helper()
	
	responseMap := writer.GetLastResponseAsMap()
	if responseMap == nil {
		t.Fatalf("Expected response but got nil")
	}
	
	// 检查状态字段
	if status, ok := responseMap["status"].(float64); ok {
		if int(status) != expectedStatus {
			t.Errorf("Expected status %d, got %d", expectedStatus, int(status))
		}
	}
	
	// 检查错误字段
	if expectedBodyContains != "" {
		if errorStr, ok := responseMap["error"].(string); ok {
			if !strings.Contains(errorStr, expectedBodyContains) {
				t.Errorf("Expected response error to contain '%s', got '%s'", expectedBodyContains, errorStr)
			}
		} else if dataStr, ok := responseMap["data"].(string); ok {
			if !strings.Contains(dataStr, expectedBodyContains) {
				t.Errorf("Expected response data to contain '%s', got '%s'", expectedBodyContains, dataStr)
			}
		}
	}
}

// AssertJSONResponse 断言 JSON 响应
func (h *TestHelpers) AssertJSONResponse(t *testing.T, writer *MockWriter, expectedStatus int, expectedJSON map[string]interface{}) {
	t.Helper()
	
	responseMap := writer.GetLastResponseAsMap()
	if responseMap == nil {
		t.Fatalf("Expected response but got nil")
	}
	
	// 检查状态字段
	if status, ok := responseMap["status"].(float64); ok {
		if int(status) != expectedStatus {
			t.Errorf("Expected status %d, got %d", expectedStatus, int(status))
		}
	}
	
	// 检查数据字段
	if data, ok := responseMap["data"]; ok {
		if dataMap, ok := data.(map[string]interface{}); ok {
			if !h.JSONEqual(expectedJSON, dataMap) {
				t.Errorf("Expected JSON %v, got %v", expectedJSON, data)
			}
		} else {
			t.Errorf("Expected data to be map[string]interface{}, got %T", data)
		}
	}
}

// JSONEqual 比较 JSON 对象是否相等
func (h *TestHelpers) JSONEqual(a, b map[string]interface{}) bool {
	aBytes, _ := json.Marshal(a)
	bBytes, _ := json.Marshal(b)
	return bytes.Equal(aBytes, bBytes)
}

// Wait 等待指定时间
func (h *TestHelpers) Wait(duration time.Duration) {
	time.Sleep(duration)
}

// CreateChatMessage 创建聊天消息
func (h *TestHelpers) CreateChatMessage(user, message string) map[string]interface{} {
	return map[string]interface{}{
		"user":    user,
		"message": message,
	}
}

// CreateJoinRequest 创建加入聊天请求
func (h *TestHelpers) CreateJoinRequest() map[string]interface{} {
	return map[string]interface{}{}
}

// CreateLeaveRequest 创建离开聊天请求
func (h *TestHelpers) CreateLeaveRequest() map[string]interface{} {
	return map[string]interface{}{}
}