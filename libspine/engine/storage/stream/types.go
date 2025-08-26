package stream

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StreamID represents a Redis stream entry ID (timestamp-sequence)
type StreamID struct {
	Timestamp uint64 // milliseconds since epoch
	Sequence  uint64 // sequence number within the same millisecond
}

// String returns the string representation of StreamID
func (id StreamID) String() string {
	return fmt.Sprintf("%d-%d", id.Timestamp, id.Sequence)
}

// Compare compares two StreamIDs
// Returns: -1 if id < other, 0 if id == other, 1 if id > other
func (id StreamID) Compare(other StreamID) int {
	if id.Timestamp < other.Timestamp {
		return -1
	}
	if id.Timestamp > other.Timestamp {
		return 1
	}
	if id.Sequence < other.Sequence {
		return -1
	}
	if id.Sequence > other.Sequence {
		return 1
	}
	return 0
}

// StreamEntry represents a single entry in a stream
type StreamEntry struct {
	ID     StreamID
	Fields map[string]string
}

// Stream represents a Redis stream data structure
type Stream struct {
	mu              sync.RWMutex
	entries         []*StreamEntry           // ordered list of entries
	lastID          StreamID                 // last generated ID
	length          int64                    // current length
	consumerGroups  map[string]*ConsumerGroup // consumer groups
	blockedReaders  map[string]*BlockedReader // blocked XREAD clients
	maxLen          int64                     // maximum length (0 = unlimited)
	trimExact       bool                      // exact trimming vs approximate
}

// ConsumerGroup represents a consumer group
type ConsumerGroup struct {
	mu           sync.RWMutex
	name         string
	lastID       StreamID                    // last delivered ID
	consumers    map[string]*Consumer        // consumers in this group
	pending      map[StreamID]*PendingEntry  // pending entries list (PEL)
	blockedReads map[string]*BlockedGroupReader // blocked XREADGROUP clients
}

// Consumer represents a consumer within a consumer group
type Consumer struct {
	name         string
	lastSeen     time.Time
	pendingCount int64
}

// PendingEntry represents an entry in the Pending Entries List
type PendingEntry struct {
	ID            StreamID
	Consumer      string
	DeliveryTime  time.Time
	DeliveryCount int64
}

// BlockedReader represents a client blocked on XREAD
type BlockedReader struct {
	ctx        context.Context
	cancel     context.CancelFunc
	clientID   string
	streams    []string      // stream names
	ids        []StreamID    // starting IDs for each stream
	count      int64         // maximum entries to return
	timeout    time.Duration // blocking timeout
	resultChan chan *ReadResult
	createdAt  time.Time
}

// BlockedGroupReader represents a client blocked on XREADGROUP
type BlockedGroupReader struct {
	ctx          context.Context
	cancel       context.CancelFunc
	clientID     string
	groupName    string
	consumerName string
	streams      []string      // stream names
	ids          []StreamID    // starting IDs (should be ">" for new messages)
	count        int64         // maximum entries to return
	timeout      time.Duration // blocking timeout
	noAck        bool          // NOACK option
	resultChan   chan *ReadResult
	createdAt    time.Time
}

// ReadResult represents the result of a read operation
type ReadResult struct {
	Streams []StreamReadResult
	Error   error
}

// StreamReadResult represents entries read from a single stream
type StreamReadResult struct {
	Name    string
	Entries []*StreamEntry
}

// StreamRange represents a range query parameters
type StreamRange struct {
	Start StreamID
	End   StreamID
	Count int64
}

// TrimOptions represents options for stream trimming
type TrimOptions struct {
	Strategy TrimStrategy
	Threshold int64
	Exact     bool
	Limit     int64
}

// TrimStrategy represents different trimming strategies
type TrimStrategy int

const (
	TrimByLength TrimStrategy = iota
	TrimByID
)

// Special StreamID constants
var (
	MinStreamID = StreamID{Timestamp: 0, Sequence: 0}
	MaxStreamID = StreamID{Timestamp: ^uint64(0), Sequence: ^uint64(0)}
)
