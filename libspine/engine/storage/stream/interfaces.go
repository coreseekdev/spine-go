package stream

import (
	"context"
	"time"
)

// StreamStorage defines the interface for stream operations
type StreamStorage interface {
	// Basic stream operations
	XAdd(key string, id StreamID, fields map[string]string, maxLen int64, exact bool) (StreamID, error)
	XDel(key string, ids []StreamID) (int64, error)
	XLen(key string) (int64, error)
	XRange(key string, start, end StreamID, count int64) ([]*StreamEntry, error)
	XRevRange(key string, start, end StreamID, count int64) ([]*StreamEntry, error)
	XTrim(key string, options TrimOptions) (int64, error)
	
	// Blocking read operations
	XRead(ctx context.Context, clientID string, streams []string, ids []StreamID, count int64, timeout time.Duration) (*ReadResult, error)
	XReadGroup(ctx context.Context, clientID string, groupName, consumerName string, streams []string, ids []StreamID, count int64, timeout time.Duration, noAck bool) (*ReadResult, error)
	
	// Consumer group management
	XGroupCreate(key, groupName string, id StreamID, mkStream bool) error
	XGroupCreateConsumer(key, groupName, consumerName string) error
	XGroupDelConsumer(key, groupName, consumerName string) (int64, error)
	XGroupDestroy(key, groupName string) error
	XGroupSetID(key, groupName string, id StreamID) error
	
	// Information and monitoring
	XInfoStream(key string) (*StreamInfo, error)
	XInfoGroups(key string) ([]*GroupInfo, error)
	XInfoConsumers(key, groupName string) ([]*ConsumerInfo, error)
	XPending(key, groupName string, start, end StreamID, count int64, consumerName string) (*PendingInfo, error)
	
	// Acknowledgment
	XAck(key, groupName string, ids []StreamID) (int64, error)
	
	// Cleanup and management
	CleanupBlockedClients(clientID string)
	GetStreamInfo(key string) (*Stream, bool)
}

// StreamInfo represents information about a stream
type StreamInfo struct {
	Length          int64
	RadixTreeKeys   int64
	RadixTreeNodes  int64
	Groups          int64
	LastGeneratedID StreamID
	FirstEntry      *StreamEntry
	LastEntry       *StreamEntry
}

// GroupInfo represents information about a consumer group
type GroupInfo struct {
	Name            string
	Consumers       int64
	Pending         int64
	LastDeliveredID StreamID
}

// ConsumerInfo represents information about a consumer
type ConsumerInfo struct {
	Name     string
	Pending  int64
	Idle     time.Duration
	Inactive time.Duration
}

// PendingInfo represents pending entries information
type PendingInfo struct {
	Count     int64
	StartID   StreamID
	EndID     StreamID
	Consumers map[string]int64
	Entries   []*PendingEntryInfo
}

// PendingEntryInfo represents detailed pending entry information
type PendingEntryInfo struct {
	ID            StreamID
	Consumer      string
	ElapsedTime   time.Duration
	DeliveryCount int64
}
