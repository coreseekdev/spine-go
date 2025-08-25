package pubsub

import (
	"context"
	"path/filepath"
	"sync"
	"time"

	"spine-go/libspine/engine/resp"
	"spine-go/libspine/transport"
)

// PubSubManager 管理所有的发布订阅功能
type PubSubManager struct {
	channels    map[string]*Channel        // channel name -> Channel
	patterns    map[string]*PatternChannel // pattern -> PatternChannel
	connections map[string]*SubscriberInfo // connection ID -> subscriber info
	mu          sync.RWMutex
}

// Channel 表示一个具体的频道
type Channel struct {
	name        string
	subscribers map[string]*transport.ConnInfo // connection ID -> ConnInfo
	mu          sync.RWMutex
}

// PatternChannel 表示一个模式频道
type PatternChannel struct {
	pattern     string
	subscribers map[string]*transport.ConnInfo // connection ID -> ConnInfo
	mu          sync.RWMutex
}

// SubscriberInfo 存储订阅者的信息
type SubscriberInfo struct {
	connInfo     *transport.ConnInfo
	channels     map[string]bool // subscribed channels
	patterns     map[string]bool // subscribed patterns
	mu           sync.RWMutex
}

// NewPubSubManager 创建新的 PubSub 管理器
func NewPubSubManager() *PubSubManager {
	return &PubSubManager{
		channels:    make(map[string]*Channel),
		patterns:    make(map[string]*PatternChannel),
		connections: make(map[string]*SubscriberInfo),
	}
}

// Subscribe 订阅频道
func (psm *PubSubManager) Subscribe(connInfo *transport.ConnInfo, channelName string) error {
	psm.mu.Lock()
	defer psm.mu.Unlock()

	// 获取或创建频道
	channel, exists := psm.channels[channelName]
	if !exists {
		channel = &Channel{
			name:        channelName,
			subscribers: make(map[string]*transport.ConnInfo),
		}
		psm.channels[channelName] = channel
	}

	// 添加订阅者到频道
	channel.mu.Lock()
	channel.subscribers[connInfo.ID] = connInfo
	channel.mu.Unlock()

	// 更新连接的订阅信息
	subInfo, exists := psm.connections[connInfo.ID]
	if !exists {
		subInfo = &SubscriberInfo{
			connInfo: connInfo,
			channels: make(map[string]bool),
			patterns: make(map[string]bool),
		}
		psm.connections[connInfo.ID] = subInfo
	}

	subInfo.mu.Lock()
	subInfo.channels[channelName] = true
	subInfo.mu.Unlock()

	// 更新连接元数据
	psm.updateConnectionMetadata(connInfo)

	return nil
}

// Unsubscribe 取消订阅频道
func (psm *PubSubManager) Unsubscribe(connInfo *transport.ConnInfo, channelName string) error {
	psm.mu.Lock()
	defer psm.mu.Unlock()

	// 从频道中移除订阅者
	if channel, exists := psm.channels[channelName]; exists {
		channel.mu.Lock()
		delete(channel.subscribers, connInfo.ID)
		
		// 如果频道没有订阅者了，删除频道
		if len(channel.subscribers) == 0 {
			delete(psm.channels, channelName)
		}
		channel.mu.Unlock()
	}

	// 更新连接的订阅信息
	if subInfo, exists := psm.connections[connInfo.ID]; exists {
		subInfo.mu.Lock()
		delete(subInfo.channels, channelName)
		
		// 如果连接没有任何订阅，删除连接信息
		if len(subInfo.channels) == 0 && len(subInfo.patterns) == 0 {
			delete(psm.connections, connInfo.ID)
		}
		subInfo.mu.Unlock()
	}

	// 更新连接元数据
	psm.updateConnectionMetadata(connInfo)

	return nil
}

// PSubscribe 订阅模式频道
func (psm *PubSubManager) PSubscribe(connInfo *transport.ConnInfo, pattern string) error {
	psm.mu.Lock()
	defer psm.mu.Unlock()

	// 获取或创建模式频道
	patternChannel, exists := psm.patterns[pattern]
	if !exists {
		patternChannel = &PatternChannel{
			pattern:     pattern,
			subscribers: make(map[string]*transport.ConnInfo),
		}
		psm.patterns[pattern] = patternChannel
	}

	// 添加订阅者到模式频道
	patternChannel.mu.Lock()
	patternChannel.subscribers[connInfo.ID] = connInfo
	patternChannel.mu.Unlock()

	// 更新连接的订阅信息
	subInfo, exists := psm.connections[connInfo.ID]
	if !exists {
		subInfo = &SubscriberInfo{
			connInfo: connInfo,
			channels: make(map[string]bool),
			patterns: make(map[string]bool),
		}
		psm.connections[connInfo.ID] = subInfo
	}

	subInfo.mu.Lock()
	subInfo.patterns[pattern] = true
	subInfo.mu.Unlock()

	// 更新连接元数据
	psm.updateConnectionMetadata(connInfo)

	return nil
}

// PUnsubscribe 取消订阅模式频道
func (psm *PubSubManager) PUnsubscribe(connInfo *transport.ConnInfo, pattern string) error {
	psm.mu.Lock()
	defer psm.mu.Unlock()

	// 从模式频道中移除订阅者
	if patternChannel, exists := psm.patterns[pattern]; exists {
		patternChannel.mu.Lock()
		delete(patternChannel.subscribers, connInfo.ID)
		
		// 如果模式频道没有订阅者了，删除模式频道
		if len(patternChannel.subscribers) == 0 {
			delete(psm.patterns, pattern)
		}
		patternChannel.mu.Unlock()
	}

	// 更新连接的订阅信息
	if subInfo, exists := psm.connections[connInfo.ID]; exists {
		subInfo.mu.Lock()
		delete(subInfo.patterns, pattern)
		
		// 如果连接没有任何订阅，删除连接信息
		if len(subInfo.channels) == 0 && len(subInfo.patterns) == 0 {
			delete(psm.connections, connInfo.ID)
		}
		subInfo.mu.Unlock()
	}

	// 更新连接元数据
	psm.updateConnectionMetadata(connInfo)

	return nil
}

// Publish 发布消息到频道
func (psm *PubSubManager) Publish(channelName string, message string) int {
	psm.mu.RLock()
	defer psm.mu.RUnlock()

	var subscriberCount int

	// 发送给直接订阅该频道的订阅者
	if channel, exists := psm.channels[channelName]; exists {
		channel.mu.RLock()
		for _, connInfo := range channel.subscribers {
			go psm.sendMessageWithTimeout(connInfo, "message", channelName, message)
			subscriberCount++
		}
		channel.mu.RUnlock()
	}

	// 发送给匹配模式的订阅者
	for pattern, patternChannel := range psm.patterns {
		if psm.matchPattern(pattern, channelName) {
			patternChannel.mu.RLock()
			for _, connInfo := range patternChannel.subscribers {
				go psm.sendMessageWithTimeout(connInfo, "pmessage", pattern, channelName, message)
				subscriberCount++
			}
			patternChannel.mu.RUnlock()
		}
	}

	return subscriberCount
}

// RemoveConnection 移除连接的所有订阅
func (psm *PubSubManager) RemoveConnection(connID string) {
	psm.mu.Lock()
	defer psm.mu.Unlock()

	subInfo, exists := psm.connections[connID]
	if !exists {
		return
	}

	// 从所有频道中移除该连接
	for channelName := range subInfo.channels {
		if channel, exists := psm.channels[channelName]; exists {
			channel.mu.Lock()
			delete(channel.subscribers, connID)
			if len(channel.subscribers) == 0 {
				delete(psm.channels, channelName)
			}
			channel.mu.Unlock()
		}
	}

	// 从所有模式频道中移除该连接
	for pattern := range subInfo.patterns {
		if patternChannel, exists := psm.patterns[pattern]; exists {
			patternChannel.mu.Lock()
			delete(patternChannel.subscribers, connID)
			if len(patternChannel.subscribers) == 0 {
				delete(psm.patterns, pattern)
			}
			patternChannel.mu.Unlock()
		}
	}

	// 删除连接信息
	delete(psm.connections, connID)
}

// GetSubscriptionCount 获取连接的订阅数量
func (psm *PubSubManager) GetSubscriptionCount(connID string) (int, int) {
	psm.mu.RLock()
	defer psm.mu.RUnlock()

	subInfo, exists := psm.connections[connID]
	if !exists {
		return 0, 0
	}

	subInfo.mu.RLock()
	channelCount := len(subInfo.channels)
	patternCount := len(subInfo.patterns)
	subInfo.mu.RUnlock()

	return channelCount, patternCount
}

// sendMessage 发送消息给订阅者（使用 RESP3 Push）
func (psm *PubSubManager) sendMessage(connInfo *transport.ConnInfo, msgType string, args ...string) {
	if connInfo.Writer == nil {
		return
	}

	// 构建 RESP3 Push 消息
	var elements []interface{}
	elements = append(elements, msgType)
	for _, arg := range args {
		elements = append(elements, arg)
	}

	// 使用 RESP3 WritePush 发送消息
	respWriter := resp.NewRESPWriter(connInfo.Writer)
	respWriter.WritePush(elements)
}

// sendMessageWithTimeout 带超时的异步消息发送
func (psm *PubSubManager) sendMessageWithTimeout(connInfo *transport.ConnInfo, msgType string, args ...string) {
	if connInfo.Writer == nil {
		return
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 构建 RESP3 Push 消息
	var elements []interface{}
	elements = append(elements, msgType)
	for _, arg := range args {
		elements = append(elements, arg)
	}

	// 使用 channel 来处理写入操作的超时
	done := make(chan error, 1)
	go func() {
		respWriter := resp.NewRESPWriter(connInfo.Writer)
		err := respWriter.WritePush(elements)
		done <- err
	}()

	// 等待写入完成或超时
	select {
	case err := <-done:
		if err != nil {
			// 写入失败，可能连接已断开，从订阅中移除该连接
			psm.RemoveConnection(connInfo.ID)
		}
	case <-ctx.Done():
		// 超时，连接可能很慢或已死，从订阅中移除
		psm.RemoveConnection(connInfo.ID)
	}
}

// updateConnectionMetadata 更新连接的元数据
func (psm *PubSubManager) updateConnectionMetadata(connInfo *transport.ConnInfo) {
	if connInfo.Metadata == nil {
		connInfo.Metadata = make(map[string]interface{})
	}

	subInfo, exists := psm.connections[connInfo.ID]
	if !exists {
		connInfo.Metadata[transport.MetadataPubSubMode] = false
		connInfo.Metadata[transport.MetadataSubscriptions] = []string{}
		connInfo.Metadata[transport.MetadataPatternSubs] = []string{}
		return
	}

	subInfo.mu.RLock()
	defer subInfo.mu.RUnlock()

	// 更新 pub/sub 模式状态
	inPubSubMode := len(subInfo.channels) > 0 || len(subInfo.patterns) > 0
	connInfo.Metadata[transport.MetadataPubSubMode] = inPubSubMode

	// 更新订阅的频道列表
	channels := make([]string, 0, len(subInfo.channels))
	for channel := range subInfo.channels {
		channels = append(channels, channel)
	}
	connInfo.Metadata[transport.MetadataSubscriptions] = channels

	// 更新订阅的模式列表
	patterns := make([]string, 0, len(subInfo.patterns))
	for pattern := range subInfo.patterns {
		patterns = append(patterns, pattern)
	}
	connInfo.Metadata[transport.MetadataPatternSubs] = patterns
}

// matchPattern 检查频道名是否匹配模式
func (psm *PubSubManager) matchPattern(pattern, channel string) bool {
	// 使用 filepath.Match 进行简单的模式匹配
	// 支持 * 和 ? 通配符
	matched, err := filepath.Match(pattern, channel)
	if err != nil {
		return false
	}
	return matched
}

// GetChannelSubscribers 获取频道的订阅者数量
func (psm *PubSubManager) GetChannelSubscribers(channelName string) int {
	psm.mu.RLock()
	defer psm.mu.RUnlock()

	if channel, exists := psm.channels[channelName]; exists {
		channel.mu.RLock()
		count := len(channel.subscribers)
		channel.mu.RUnlock()
		return count
	}
	return 0
}

// GetAllChannels 获取所有活跃的频道
func (psm *PubSubManager) GetAllChannels() []string {
	psm.mu.RLock()
	defer psm.mu.RUnlock()

	channels := make([]string, 0, len(psm.channels))
	for channelName := range psm.channels {
		channels = append(channels, channelName)
	}
	return channels
}

// GetAllPatterns 获取所有活跃的模式
func (psm *PubSubManager) GetAllPatterns() []string {
	psm.mu.RLock()
	defer psm.mu.RUnlock()

	patterns := make([]string, 0, len(psm.patterns))
	for pattern := range psm.patterns {
		patterns = append(patterns, pattern)
	}
	return patterns
}
