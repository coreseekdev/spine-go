package e2e

import (
	"fmt"
	"reflect"
	"time"
)

// MessageValidator 消息验证器
type MessageValidator struct {
	expectedMessages []ExpectedMessage
	receivedMessages []ReceivedMessage
}

// ExpectedMessage 期望的消息
type ExpectedMessage struct {
	User      string
	Message   string
	Timestamp *time.Time // 可选的时间戳验证
}

// ReceivedMessage 接收到的消息
type ReceivedMessage struct {
	User      string
	Message   string
	Timestamp time.Time
	Source    string // 消息来源（客户端标识）
}

// NewMessageValidator 创建新的消息验证器
func NewMessageValidator() *MessageValidator {
	return &MessageValidator{
		expectedMessages: make([]ExpectedMessage, 0),
		receivedMessages: make([]ReceivedMessage, 0),
	}
}

// ExpectMessage 添加期望的消息
func (v *MessageValidator) ExpectMessage(user, message string) {
	v.expectedMessages = append(v.expectedMessages, ExpectedMessage{
		User:    user,
		Message: message,
	})
}

// ExpectMessageWithTime 添加带时间戳的期望消息
func (v *MessageValidator) ExpectMessageWithTime(user, message string, timestamp time.Time) {
	v.expectedMessages = append(v.expectedMessages, ExpectedMessage{
		User:      user,
		Message:   message,
		Timestamp: &timestamp,
	})
}

// RecordMessage 记录接收到的消息
func (v *MessageValidator) RecordMessage(user, message, source string, timestamp time.Time) {
	v.receivedMessages = append(v.receivedMessages, ReceivedMessage{
		User:      user,
		Message:   message,
		Timestamp: timestamp,
		Source:    source,
	})
}

// ValidateMessages 验证消息
func (v *MessageValidator) ValidateMessages() error {
	if len(v.expectedMessages) != len(v.receivedMessages) {
		return fmt.Errorf("message count mismatch: expected %d, received %d",
			len(v.expectedMessages), len(v.receivedMessages))
	}

	for i, expected := range v.expectedMessages {
		if i >= len(v.receivedMessages) {
			return fmt.Errorf("missing message at index %d: expected %+v", i, expected)
		}

		received := v.receivedMessages[i]
		if expected.User != received.User {
			return fmt.Errorf("user mismatch at index %d: expected %s, received %s",
				i, expected.User, received.User)
		}

		if expected.Message != received.Message {
			return fmt.Errorf("message mismatch at index %d: expected %s, received %s",
				i, expected.Message, received.Message)
		}

		if expected.Timestamp != nil {
			timeDiff := received.Timestamp.Sub(*expected.Timestamp)
			if timeDiff < 0 {
				timeDiff = -timeDiff
			}
			if timeDiff > time.Second {
				return fmt.Errorf("timestamp mismatch at index %d: expected %v, received %v (diff: %v)",
					i, *expected.Timestamp, received.Timestamp, timeDiff)
			}
		}
	}

	return nil
}

// ValidateBroadcast 验证消息是否正确广播到所有客户端
func (v *MessageValidator) ValidateBroadcast(expectedClients []string) error {
	messagesByContent := make(map[string][]ReceivedMessage)
	
	// 按消息内容分组
	for _, msg := range v.receivedMessages {
		key := fmt.Sprintf("%s:%s", msg.User, msg.Message)
		messagesByContent[key] = append(messagesByContent[key], msg)
	}

	// 验证每条消息是否广播到所有期望的客户端
	for content, messages := range messagesByContent {
		receivedClients := make(map[string]bool)
		for _, msg := range messages {
			receivedClients[msg.Source] = true
		}

		for _, expectedClient := range expectedClients {
			if !receivedClients[expectedClient] {
				return fmt.Errorf("message %s was not broadcast to client %s", content, expectedClient)
			}
		}

		if len(receivedClients) != len(expectedClients) {
			return fmt.Errorf("message %s was broadcast to unexpected clients", content)
		}
	}

	return nil
}

// GetReceivedMessages 获取接收到的消息
func (v *MessageValidator) GetReceivedMessages() []ReceivedMessage {
	return v.receivedMessages
}

// GetExpectedMessages 获取期望的消息
func (v *MessageValidator) GetExpectedMessages() []ExpectedMessage {
	return v.expectedMessages
}

// Clear 清空验证器
func (v *MessageValidator) Clear() {
	v.expectedMessages = v.expectedMessages[:0]
	v.receivedMessages = v.receivedMessages[:0]
}

// ResponseValidator 响应验证器
type ResponseValidator struct{}

// NewResponseValidator 创建新的响应验证器
func NewResponseValidator() *ResponseValidator {
	return &ResponseValidator{}
}

// ValidateSuccessResponse 验证成功响应
func (v *ResponseValidator) ValidateSuccessResponse(response *ChatResponse, expectedData interface{}) error {
	if response == nil {
		return fmt.Errorf("response is nil")
	}

	if response.Status != 200 {
		return fmt.Errorf("expected status 200, got %d", response.Status)
	}

	if response.Error != "" {
		return fmt.Errorf("unexpected error in response: %s", response.Error)
	}

	if expectedData != nil && !reflect.DeepEqual(response.Data, expectedData) {
		return fmt.Errorf("data mismatch: expected %+v, got %+v", expectedData, response.Data)
	}

	return nil
}

// ValidateErrorResponse 验证错误响应
func (v *ResponseValidator) ValidateErrorResponse(response *ChatResponse, expectedStatus int, expectedError string) error {
	if response == nil {
		return fmt.Errorf("response is nil")
	}

	if response.Status != expectedStatus {
		return fmt.Errorf("expected status %d, got %d", expectedStatus, response.Status)
	}

	if expectedError != "" && response.Error != expectedError {
		return fmt.Errorf("expected error '%s', got '%s'", expectedError, response.Error)
	}

	return nil
}

// ValidateMessageResponse 验证消息响应
func (v *ResponseValidator) ValidateMessageResponse(response *ChatResponse) (*ChatMessage, error) {
	if err := v.ValidateSuccessResponse(response, nil); err != nil {
		return nil, err
	}

	// 尝试将 Data 转换为 ChatMessage
	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("response data is not a map")
	}

	message := &ChatMessage{}
	
	if id, exists := dataMap["id"]; exists {
		if idStr, ok := id.(string); ok {
			message.ID = idStr
		}
	}
	
	if user, exists := dataMap["user"]; exists {
		if userStr, ok := user.(string); ok {
			message.User = userStr
		}
	}
	
	if msg, exists := dataMap["message"]; exists {
		if msgStr, ok := msg.(string); ok {
			message.Message = msgStr
		}
	}
	
	if timestamp, exists := dataMap["timestamp"]; exists {
		if timestampStr, ok := timestamp.(string); ok {
			if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
				message.Timestamp = t
			}
		}
	}

	return message, nil
}

// ConnectionValidator 连接验证器
type ConnectionValidator struct{}

// NewConnectionValidator 创建新的连接验证器
func NewConnectionValidator() *ConnectionValidator {
	return &ConnectionValidator{}
}

// ValidateConnection 验证连接状态
func (v *ConnectionValidator) ValidateConnection(client TestClient, shouldBeConnected bool) error {
	isConnected := client.IsConnected()
	if isConnected != shouldBeConnected {
		if shouldBeConnected {
			return fmt.Errorf("client should be connected but is not")
		} else {
			return fmt.Errorf("client should not be connected but is")
		}
	}
	return nil
}

// ValidateServerConnections 验证服务器连接数
func (v *ConnectionValidator) ValidateServerConnections(server *TestServerManager, expectedCount int) error {
	if !server.IsRunning() {
		return fmt.Errorf("server is not running")
	}

	serverInstance := server.GetServer()
	if serverInstance == nil {
		return fmt.Errorf("server instance is nil")
	}

	connections := serverInstance.GetConnections()
	actualCount := len(connections)
	
	if actualCount != expectedCount {
		return fmt.Errorf("expected %d connections, got %d", expectedCount, actualCount)
	}

	return nil
}
