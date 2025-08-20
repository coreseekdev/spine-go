package libspine

import (
	"encoding/json"
	"fmt"
	"spine-go/libspine/handler"
	"spine-go/libspine/transport"
	"sync"
	"testing"
)

func TestChatHandler_Integration(t *testing.T) {
	// 创建聊天处理器
	chatHandler := handler.NewChatHandler()
	
	helpers := handler.NewTestHelpers()
	ctx := helpers.CreateTestContext()

	// 测试完整的聊天流程
	scenarios := []struct {
		name     string
		request  *handler.MockReader
		validate func(t *testing.T, writer *handler.MockWriter)
	}{
		{
			name: "Send message to general room",
			request: handler.NewMockReaderFromRequests([]*transport.Request{
				helpers.CreateTestRequest("POST", "/chat", map[string]interface{}{
					"user":    "alice",
					"message": "Hello everyone!",
					"room":    "general",
				}),
			}),
			validate: func(t *testing.T, writer *handler.MockWriter) {
				responseMap := writer.GetLastResponseAsMap()
				if responseMap == nil {
					t.Fatalf("Expected response but got nil")
				}
				if status, ok := responseMap["status"].(float64); ok {
					if int(status) != 200 {
						t.Errorf("Expected status 200, got %d", int(status))
					}
				}
			},
		},
		{
			name: "Send another message to general room",
			request: handler.NewMockReaderFromRequests([]*transport.Request{
				helpers.CreateTestRequest("POST", "/chat", map[string]interface{}{
					"user":    "bob",
					"message": "Hi alice!",
					"room":    "general",
				}),
			}),
			validate: func(t *testing.T, writer *handler.MockWriter) {
				responseMap := writer.GetLastResponseAsMap()
				if responseMap == nil {
					t.Fatalf("Expected response but got nil")
				}
				if status, ok := responseMap["status"].(float64); ok {
					if int(status) != 200 {
						t.Errorf("Expected status 200, got %d", int(status))
					}
				}
			},
		},
		{
			name: "Get messages from general room",
			request: handler.NewMockReaderFromRequests([]*transport.Request{
				helpers.CreateTestRequest("GET", "general", nil),
			}),
			validate: func(t *testing.T, writer *handler.MockWriter) {
				responseMap := writer.GetLastResponseAsMap()
				if responseMap == nil {
					t.Fatalf("Expected response but got nil")
				}
				if status, ok := responseMap["status"].(float64); ok {
					if int(status) != 200 {
						t.Errorf("Expected status 200, got %d", int(status))
					}
				}
				
				var messages []handler.ChatMessage
				if data, ok := responseMap["data"]; ok {
					if dataBytes, err := json.Marshal(data); err == nil {
						json.Unmarshal(dataBytes, &messages)
					}
				}
				
				if len(messages) < 2 {
					t.Errorf("Expected at least 2 messages, got %d", len(messages))
				}
				
				// 验证消息顺序
				if messages[0].User != "alice" {
					t.Errorf("Expected first message from 'alice', got '%s'", messages[0].User)
				}
				if messages[1].User != "bob" {
					t.Errorf("Expected second message from 'bob', got '%s'", messages[1].User)
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			writer := handler.NewMockWriter()
			
			// 处理请求
			err := chatHandler.Handle(ctx, scenario.request, writer)
			if err != nil {
				t.Fatalf("Failed to handle request: %v", err)
			}
			
			// 验证响应
			scenario.validate(t, writer)
		})
	}
}

func TestChatHandler_MultipleMessagesIntegration(t *testing.T) {
	// 创建聊天处理器
	chatHandler := handler.NewChatHandler()
	
	helpers := handler.NewTestHelpers()
	ctx := helpers.CreateTestContext()

	// 测试多个消息的发送
	messages := []struct {
		user    string
		message string
	}{
		{"user_general", "Message from general"},
		{"user_random", "Message from random"},
		{"user_help", "Message from help"},
	}
	
	// 发送所有消息
	for _, msg := range messages {
		request := handler.NewMockReaderFromRequests([]*transport.Request{
			helpers.CreateTestRequest("POST", "/chat", map[string]interface{}{
				"user":    msg.user,
				"message": msg.message,
			}),
		})
		
		writer := handler.NewMockWriter()
		
		err := chatHandler.Handle(ctx, request, writer)
		if err != nil {
			t.Fatalf("Failed to send message from %s: %v", msg.user, err)
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

	// 验证所有消息都在同一个空间中
	request := handler.NewMockReaderFromRequests([]*transport.Request{
		helpers.CreateTestRequest("GET", "chat", nil),
	})
	writer := handler.NewMockWriter()
	
	err := chatHandler.Handle(ctx, request, writer)
	if err != nil {
		t.Fatalf("Failed to get all messages: %v", err)
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
	
	var allMessages []handler.ChatMessage
	if data, ok := responseMap["data"]; ok {
		if dataBytes, err := json.Marshal(data); err == nil {
			json.Unmarshal(dataBytes, &allMessages)
		}
	}
	
	if len(allMessages) != len(messages) {
		t.Errorf("Expected %d messages total, got %d", len(messages), len(allMessages))
	}
	
	// 验证所有消息都存在
	for i, expectedMsg := range messages {
		found := false
		for _, actualMsg := range allMessages {
			if actualMsg.User == expectedMsg.user && actualMsg.Message == expectedMsg.message {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Message %d: expected message from %s with content '%s' not found", i, expectedMsg.user, expectedMsg.message)
		}
	}
}

func TestChatHandler_ConcurrentAccess(t *testing.T) {
	// 创建聊天处理器
	chatHandler := handler.NewChatHandler()
	
	helpers := handler.NewTestHelpers()
	ctx := helpers.CreateTestContext()

	// 并发测试
	var wg sync.WaitGroup
	numGoroutines := 10
	messagesPerGoroutine := 5
	room := "concurrent"

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			for j := 0; j < messagesPerGoroutine; j++ {
				request := handler.NewMockReaderFromRequests([]*transport.Request{
					helpers.CreateTestRequest("POST", "/chat", map[string]interface{}{
						"user":    fmt.Sprintf("user_%d", id),
						"message": fmt.Sprintf("Message %d from user %d", j, id),
						"room":    room,
					}),
				})
				
				writer := handler.NewMockWriter()
				
				err := chatHandler.Handle(ctx, request, writer)
				if err != nil {
					t.Errorf("Goroutine %d: failed to send message %d: %v", id, j, err)
				}
				
				responseMap := writer.GetLastResponseAsMap()
				if responseMap == nil {
					t.Errorf("Goroutine %d: expected response for message %d", id, j)
					continue
				}
				if status, ok := responseMap["status"].(float64); ok {
					if int(status) != 200 {
						t.Errorf("Goroutine %d: expected success response for message %d", id, j)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// 验证所有消息都已发送
	request := handler.NewMockReaderFromRequests([]*transport.Request{
		helpers.CreateTestRequest("GET", room, nil),
	})
	writer := handler.NewMockWriter()
	
	err := chatHandler.Handle(ctx, request, writer)
	if err != nil {
		t.Fatalf("Failed to get concurrent messages: %v", err)
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
	
	var messages []handler.ChatMessage
	if data, ok := responseMap["data"]; ok {
		if dataBytes, err := json.Marshal(data); err == nil {
			json.Unmarshal(dataBytes, &messages)
		}
	}
	
	expectedMessages := numGoroutines * messagesPerGoroutine
	if len(messages) != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, len(messages))
	}

	// 验证消息总数正确
	// 注意：在并发测试中，用户ID可能重复，这是正常的
}

func TestChatHandler_RoomOperations(t *testing.T) {
	// 创建聊天处理器
	chatHandler := handler.NewChatHandler()
	
	helpers := handler.NewTestHelpers()
	ctx := helpers.CreateTestContext()

	// 测试房间操作
	operations := []struct {
		name     string
		request  *handler.MockReader
		validate func(t *testing.T, writer *handler.MockWriter)
	}{
		{
			name: "Join room",
			request: handler.NewMockReaderFromRequests([]*transport.Request{
				helpers.CreateTestRequest("JOIN", "/chat", map[string]interface{}{
					"room": "test_room",
				}),
			}),
			validate: func(t *testing.T, writer *handler.MockWriter) {
				responseMap := writer.GetLastResponseAsMap()
				if responseMap == nil {
					t.Fatalf("Expected response but got nil")
				}
				if status, ok := responseMap["status"].(float64); ok {
					if int(status) != 200 {
						t.Errorf("Expected status 200, got %d", int(status))
					}
				}
			},
		},
		{
			name: "Send message to room",
			request: handler.NewMockReaderFromRequests([]*transport.Request{
				helpers.CreateTestRequest("POST", "/chat", map[string]interface{}{
					"user":    "test_user",
					"message": "Test message",
					"room":    "test_room",
				}),
			}),
			validate: func(t *testing.T, writer *handler.MockWriter) {
				responseMap := writer.GetLastResponseAsMap()
				if responseMap == nil {
					t.Fatalf("Expected response but got nil")
				}
				if status, ok := responseMap["status"].(float64); ok {
					if int(status) != 200 {
						t.Errorf("Expected status 200, got %d", int(status))
					}
				}
			},
		},
		{
			name: "Leave room",
			request: handler.NewMockReaderFromRequests([]*transport.Request{
				helpers.CreateTestRequest("LEAVE", "/chat", map[string]interface{}{
					"room": "test_room",
				}),
			}),
			validate: func(t *testing.T, writer *handler.MockWriter) {
				responseMap := writer.GetLastResponseAsMap()
				if responseMap == nil {
					t.Fatalf("Expected response but got nil")
				}
				if status, ok := responseMap["status"].(float64); ok {
					if int(status) != 200 {
						t.Errorf("Expected status 200, got %d", int(status))
					}
				}
			},
		},
		{
			name: "Verify message persists after leaving",
			request: handler.NewMockReaderFromRequests([]*transport.Request{
				helpers.CreateTestRequest("GET", "test_room", nil),
			}),
			validate: func(t *testing.T, writer *handler.MockWriter) {
				responseMap := writer.GetLastResponseAsMap()
				if responseMap == nil {
					t.Fatalf("Expected response but got nil")
				}
				if status, ok := responseMap["status"].(float64); ok {
					if int(status) != 200 {
						t.Errorf("Expected status 200, got %d", int(status))
					}
				}
				
				var messages []handler.ChatMessage
				if data, ok := responseMap["data"]; ok {
					if dataBytes, err := json.Marshal(data); err == nil {
						json.Unmarshal(dataBytes, &messages)
					}
				}
				
				if len(messages) != 1 {
					t.Errorf("Expected 1 message, got %d", len(messages))
				}
				
				if len(messages) > 0 {
					if messages[0].User != "test_user" {
						t.Errorf("Expected user 'test_user', got '%s'", messages[0].User)
					}
					if messages[0].Message != "Test message" {
						t.Errorf("Expected message 'Test message', got '%s'", messages[0].Message)
					}
				}
			},
		},
	}

	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			writer := handler.NewMockWriter()
			
			// 处理请求
			err := chatHandler.Handle(ctx, operation.request, writer)
			if err != nil {
				t.Fatalf("Failed to handle request: %v", err)
			}
			
			// 验证响应
			operation.validate(t, writer)
		})
	}
}