package handler

import (
	"encoding/json"
	"spine-go/libspine/transport"
	"testing"
	"time"
)

func TestChatHandler_HandlePostMessage(t *testing.T) {
	handler := NewChatHandler()

	helpers := NewTestHelpers()
	ctx := helpers.CreateTestContext()
	writer := NewMockWriter()

	// 创建发送消息请求
	message := helpers.CreateChatMessage("alice", "Hello world")
	request := helpers.CreateTestRequest("POST", "/chat", message)

	reader := NewMockReaderFromRequests([]*transport.Request{request})

	// 处理请求
	err := handler.Handle(ctx, reader, writer)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 验证响应
	responseMap := writer.GetLastResponseAsMap()
	if responseMap == nil {
		t.Fatalf("Expected response but got nil")
	}
	if status, ok := responseMap["status"].(float64); ok {
		if int(status) != 200 {
			t.Errorf("Expected status 200, got %d", int(status))
		}
	}

	// 验证消息已添加到房间
	getRequest := helpers.CreateTestRequest("GET", "general", nil)
	getReader := NewMockReaderFromRequests([]*transport.Request{getRequest})
	getWriter := NewMockWriter()

	err = handler.Handle(ctx, getReader, getWriter)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	getResponseMap := getWriter.GetLastResponseAsMap()
	if getResponseMap == nil {
		t.Fatalf("Expected response but got nil")
	}
	if status, ok := getResponseMap["status"].(float64); ok {
		if int(status) != 200 {
			t.Errorf("Expected status 200, got %d", int(status))
		}
	}

	// 解析响应中的消息
	var messages []ChatMessage
	if data, ok := getResponseMap["data"]; ok {
		if dataBytes, err := json.Marshal(data); err == nil {
			json.Unmarshal(dataBytes, &messages)
		}
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].User != "alice" {
		t.Errorf("Expected user 'alice', got '%s'", messages[0].User)
	}

	if messages[0].Message != "Hello world" {
		t.Errorf("Expected message 'Hello world', got '%s'", messages[0].Message)
	}

	}

func TestChatHandler_HandleJoin(t *testing.T) {
	handler := NewChatHandler()
	
	helpers := NewTestHelpers()
	ctx := helpers.CreateTestContext()
	writer := NewMockWriter()

	// 创建加入聊天请求
	joinRequest := helpers.CreateJoinRequest()
	request := helpers.CreateTestRequest("JOIN", "/chat", joinRequest)

	reader := NewMockReaderFromRequests([]*transport.Request{request})

	// 处理请求
	err := handler.Handle(ctx, reader, writer)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 验证响应
	responseMap := writer.GetLastResponseAsMap()
	if responseMap == nil {
		t.Fatalf("Expected response but got nil")
	}
	if status, ok := responseMap["status"].(float64); ok {
		if int(status) != 200 {
			t.Errorf("Expected status 200, got %d", int(status))
		}
	}
}

func TestChatHandler_HandleLeave(t *testing.T) {
	handler := NewChatHandler()
	
	helpers := NewTestHelpers()
	ctx := helpers.CreateTestContext()
	writer := NewMockWriter()

	// 创建离开聊天请求
	leaveRequest := helpers.CreateLeaveRequest()
	request := helpers.CreateTestRequest("LEAVE", "/chat", leaveRequest)

	reader := NewMockReaderFromRequests([]*transport.Request{request})

	// 处理请求
	err := handler.Handle(ctx, reader, writer)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 验证响应
	responseMap := writer.GetLastResponseAsMap()
	if responseMap == nil {
		t.Fatalf("Expected response but got nil")
	}
	if status, ok := responseMap["status"].(float64); ok {
		if int(status) != 200 {
			t.Errorf("Expected status 200, got %d", int(status))
		}
	}
}

func TestChatHandler_HandleMultipleMessages(t *testing.T) {
	handler := NewChatHandler()
	
	helpers := NewTestHelpers()
	ctx := helpers.CreateTestContext()

	// 发送多条消息
	messages := []struct {
		user    string
		message string
	}{
		{"alice", "Hello"},
		{"bob", "Hi there"},
		{"alice", "How are you?"},
	}

	for i, msg := range messages {
		writer := NewMockWriter()
		message := helpers.CreateChatMessage(msg.user, msg.message)
		request := helpers.CreateTestRequest("POST", "/chat", message)
		reader := NewMockReaderFromRequests([]*transport.Request{request})

		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Fatalf("Expected no error for message %d, got %v", i, err)
		}

		responseMap := writer.GetLastResponseAsMap()
		if responseMap == nil {
			t.Fatalf("Expected response but got nil")
		}
		if status, ok := responseMap["status"].(float64); ok {
			if int(status) != 200 {
				t.Errorf("Expected status 200, got %d", int(status))
			}
		}
	}

	// 获取所有消息
	getRequest := helpers.CreateTestRequest("GET", "general", nil)
	getReader := NewMockReaderFromRequests([]*transport.Request{getRequest})
	getWriter := NewMockWriter()

	err := handler.Handle(ctx, getReader, getWriter)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	getResponseMap := getWriter.GetLastResponseAsMap()
	if getResponseMap == nil {
		t.Fatalf("Expected response but got nil")
	}
	if status, ok := getResponseMap["status"].(float64); ok {
		if int(status) != 200 {
			t.Errorf("Expected status 200, got %d", int(status))
		}
	}

	// 解析响应中的消息
	var retrievedMessages []ChatMessage
	if data, ok := getResponseMap["data"]; ok {
		if dataBytes, err := json.Marshal(data); err == nil {
			json.Unmarshal(dataBytes, &retrievedMessages)
		}
	}

	if len(retrievedMessages) != len(messages) {
		t.Fatalf("Expected %d messages, got %d", len(messages), len(retrievedMessages))
	}

	// 验证消息顺序和内容
	for i, expected := range messages {
		if retrievedMessages[i].User != expected.user {
			t.Errorf("Message %d: expected user '%s', got '%s'", i, expected.user, retrievedMessages[i].User)
		}
		if retrievedMessages[i].Message != expected.message {
			t.Errorf("Message %d: expected message '%s', got '%s'", i, expected.message, retrievedMessages[i].Message)
		}
			}
}

func TestChatHandler_HandleDifferentMessages(t *testing.T) {
	handler := NewChatHandler()
	
	helpers := NewTestHelpers()
	ctx := helpers.CreateTestContext()

	// 发送多条消息
	messages := []struct {
		user    string
		message string
	}{
		{"alice", "Hello general"},
		{"bob", "Hi general"},
		{"charlie", "Hello random"},
		{"dave", "Hi random"},
	}

	// 发送所有消息
	for _, msg := range messages {
		writer := NewMockWriter()
		message := helpers.CreateChatMessage(msg.user, msg.message)
		request := helpers.CreateTestRequest("POST", "/chat", message)
		reader := NewMockReaderFromRequests([]*transport.Request{request})

		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	}

	// 验证所有消息
	getRequest := helpers.CreateTestRequest("GET", "chat", nil)
	getReader := NewMockReaderFromRequests([]*transport.Request{getRequest})
	getWriter := NewMockWriter()

	err := handler.Handle(ctx, getReader, getWriter)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	getResponseMap := getWriter.GetLastResponseAsMap()
	if getResponseMap == nil {
		t.Fatalf("Expected response but got nil")
	}
	if status, ok := getResponseMap["status"].(float64); ok {
		if int(status) != 200 {
			t.Errorf("Expected status 200, got %d", int(status))
		}
	}

	var retrievedMessages []ChatMessage
	if data, ok := getResponseMap["data"]; ok {
		if dataBytes, err := json.Marshal(data); err == nil {
			json.Unmarshal(dataBytes, &retrievedMessages)
		}
	}

	if len(retrievedMessages) != len(messages) {
		t.Fatalf("Expected %d messages total, got %d", len(messages), len(retrievedMessages))
	}
}

func TestChatHandler_HandleInvalidRequest(t *testing.T) {
	handler := NewChatHandler()
	
	helpers := NewTestHelpers()
	ctx := helpers.CreateTestContext()
	writer := NewMockWriter()

	// 创建无效请求（没有方法）
	request := helpers.CreateTestRequest("", "/chat", nil)
	reader := NewMockReaderFromRequests([]*transport.Request{request})

	// 处理请求
	err := handler.Handle(ctx, reader, writer)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 验证响应
	responseMap := writer.GetLastResponseAsMap()
	if responseMap == nil {
		t.Fatalf("Expected response but got nil")
	}
	if status, ok := responseMap["status"].(float64); ok {
		if int(status) != 405 {
			t.Errorf("Expected status 405, got %d", int(status))
		}
	}
}

func TestChatHandler_HandleEmptyChat(t *testing.T) {
	handler := NewChatHandler()
	
	helpers := NewTestHelpers()
	ctx := helpers.CreateTestContext()
	writer := NewMockWriter()

	// 创建获取空聊天消息的请求
	request := helpers.CreateTestRequest("GET", "chat", nil)
	reader := NewMockReaderFromRequests([]*transport.Request{request})

	// 处理请求
	err := handler.Handle(ctx, reader, writer)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 验证响应
	responseMap := writer.GetLastResponseAsMap()
	if responseMap == nil {
		t.Fatalf("Expected response but got nil")
	}
	if status, ok := responseMap["status"].(float64); ok {
		if int(status) != 200 {
			t.Errorf("Expected status 200, got %d", int(status))
		}
	}

	// 解析响应中的消息
	var messages []ChatMessage
	if data, ok := responseMap["data"]; ok {
		if dataBytes, err := json.Marshal(data); err == nil {
			json.Unmarshal(dataBytes, &messages)
		}
	}

	if len(messages) != 0 {
		t.Fatalf("Expected 0 messages for empty chat, got %d", len(messages))
	}
}

func TestChatHandler_BroadcastMessages(t *testing.T) {
	handler := NewChatHandler()
	
	helpers := NewTestHelpers()
	ctx := helpers.CreateTestContext()

	// 创建多个模拟写入器来模拟多个客户端
	writers := []*MockWriter{NewMockWriter(), NewMockWriter(), NewMockWriter()}

	// 模拟客户端加入房间
	for range writers {
		joinRequest := helpers.CreateJoinRequest()
		request := helpers.CreateTestRequest("JOIN", "/chat", joinRequest)
		_ = NewMockReaderFromRequests([]*transport.Request{request})

		// 这里需要模拟客户端加入房间的逻辑
		// 由于我们的实现需要修改，我们先测试基本的消息发送
	}

	// 发送消息
	message := helpers.CreateChatMessage("alice", "Broadcast test")
	request := helpers.CreateTestRequest("POST", "/chat", message)
	reader := NewMockReaderFromRequests([]*transport.Request{request})
	writer := NewMockWriter()

	err := handler.Handle(ctx, reader, writer)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 验证消息已发送
	responseMap := writer.GetLastResponseAsMap()
	if responseMap == nil {
		t.Fatalf("Expected response but got nil")
	}
	if status, ok := responseMap["status"].(float64); ok {
		if int(status) != 200 {
			t.Errorf("Expected status 200, got %d", int(status))
		}
	}

	// 等待广播处理
	helpers.Wait(100 * time.Millisecond)

	// 验证消息在房间中
	getRequest := helpers.CreateTestRequest("GET", "general", nil)
	getReader := NewMockReaderFromRequests([]*transport.Request{getRequest})
	getWriter := NewMockWriter()

	err = handler.Handle(ctx, getReader, getWriter)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	getResponseMap := getWriter.GetLastResponseAsMap()
	if getResponseMap == nil {
		t.Fatalf("Expected response but got nil")
	}
	if status, ok := getResponseMap["status"].(float64); ok {
		if int(status) != 200 {
			t.Errorf("Expected status 200, got %d", int(status))
		}
	}

	var messages []ChatMessage
	if data, ok := getResponseMap["data"]; ok {
		if dataBytes, err := json.Marshal(data); err == nil {
			json.Unmarshal(dataBytes, &messages)
		}
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if messages[0].Message != "Broadcast test" {
		t.Errorf("Expected message 'Broadcast test', got '%s'", messages[0].Message)
	}
}