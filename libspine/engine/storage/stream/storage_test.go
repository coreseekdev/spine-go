package stream

import (
	"context"
	"testing"
	"time"
)

func TestNewStreamStorage(t *testing.T) {
	storage := NewStreamStorage(nil)
	if storage == nil {
		t.Fatal("NewStreamStorage returned nil")
	}
	
	if storage.streams == nil {
		t.Fatal("streams map not initialized")
	}
}

func TestXAdd(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Test basic XADD
	fields := map[string]string{"field1": "value1", "field2": "value2"}
	id, err := storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	if err != nil {
		t.Fatalf("XAdd failed: %v", err)
	}
	
	if id.Timestamp == 0 {
		t.Error("Generated ID should have non-zero timestamp")
	}
	
	// Test with specific ID (must be greater than previous)
	currentTime := uint64(time.Now().UnixMilli())
	specificID := StreamID{Timestamp: currentTime + 1000, Sequence: 1}
	id2, err := storage.XAdd("test-stream", specificID, fields, 0, false)
	if err != nil {
		t.Fatalf("XAdd with specific ID failed: %v", err)
	}
	
	if id2 != specificID {
		t.Errorf("Expected ID %v, got %v", specificID, id2)
	}
	
	// Test invalid fields
	invalidFields := map[string]string{}
	_, err = storage.XAdd("test-stream", StreamID{}, invalidFields, 0, false)
	if err == nil {
		t.Error("Expected error for empty fields")
	}
}

func TestXLen(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Test empty stream
	length, err := storage.XLen("nonexistent")
	if err != nil {
		t.Fatalf("XLen failed: %v", err)
	}
	if length != 0 {
		t.Errorf("Expected length 0, got %d", length)
	}
	
	// Add entries and test length
	fields := map[string]string{"field": "value"}
	storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	
	length, err = storage.XLen("test-stream")
	if err != nil {
		t.Fatalf("XLen failed: %v", err)
	}
	if length != 2 {
		t.Errorf("Expected length 2, got %d", length)
	}
}

func TestXRange(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Add test entries with auto-generated IDs
	fields1 := map[string]string{"field": "value1"}
	fields2 := map[string]string{"field": "value2"}
	
	id1, _ := storage.XAdd("test-stream", StreamID{}, fields1, 0, false)
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	id2, _ := storage.XAdd("test-stream", StreamID{}, fields2, 0, false)
	
	// Test range query
	entries, err := storage.XRange("test-stream", MinStreamID, MaxStreamID, 10)
	if err != nil {
		t.Fatalf("XRange failed: %v", err)
	}
	
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
	
	if len(entries) >= 2 && (entries[0].ID != id1 || entries[1].ID != id2) {
		t.Error("Entries not in correct order")
	}
	
	// Test with count limit
	entries, err = storage.XRange("test-stream", MinStreamID, MaxStreamID, 1)
	if err != nil {
		t.Fatalf("XRange with count failed: %v", err)
	}
	
	if len(entries) != 1 {
		t.Errorf("Expected 1 entry with count=1, got %d", len(entries))
	}
}

func TestXRevRange(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Add test entries with auto-generated IDs
	fields1 := map[string]string{"field": "value1"}
	fields2 := map[string]string{"field": "value2"}
	
	id1, _ := storage.XAdd("test-stream", StreamID{}, fields1, 0, false)
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	id2, _ := storage.XAdd("test-stream", StreamID{}, fields2, 0, false)
	
	// Test reverse range query
	entries, err := storage.XRevRange("test-stream", MaxStreamID, MinStreamID, 10)
	if err != nil {
		t.Fatalf("XRevRange failed: %v", err)
	}
	
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
	
	// Should be in reverse order
	if len(entries) >= 2 && (entries[0].ID != id2 || entries[1].ID != id1) {
		t.Error("Entries not in reverse order")
	}
}

func TestXDel(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Add test entries with auto-generated IDs
	fields := map[string]string{"field": "value"}
	id1, _ := storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	_, _ = storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	
	// Delete one entry
	deleted, err := storage.XDel("test-stream", []StreamID{id1})
	if err != nil {
		t.Fatalf("XDel failed: %v", err)
	}
	
	if deleted != 1 {
		t.Errorf("Expected 1 deleted entry, got %d", deleted)
	}
	
	// Verify entry was deleted
	length, _ := storage.XLen("test-stream")
	if length != 1 {
		t.Errorf("Expected length 1 after deletion, got %d", length)
	}
	
	// Try to delete non-existent entry
	deleted, err = storage.XDel("test-stream", []StreamID{{Timestamp: 9999, Sequence: 0}})
	if err != nil {
		t.Fatalf("XDel of non-existent entry failed: %v", err)
	}
	
	if deleted != 0 {
		t.Errorf("Expected 0 deleted entries for non-existent ID, got %d", deleted)
	}
}

func TestXTrim(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Add test entries
	fields := map[string]string{"field": "value"}
	for i := 0; i < 5; i++ {
		storage.XAdd("test-stream", StreamID{Timestamp: uint64(1000 + i*1000), Sequence: 0}, fields, 0, false)
	}
	
	// Trim to 3 entries
	options := TrimOptions{
		Strategy: TrimByLength,
		Threshold: 3,
		Exact:    true,
	}
	
	trimmed, err := storage.XTrim("test-stream", options)
	if err != nil {
		t.Fatalf("XTrim failed: %v", err)
	}
	
	if trimmed != 2 {
		t.Errorf("Expected 2 trimmed entries, got %d", trimmed)
	}
	
	// Verify final length
	length, _ := storage.XLen("test-stream")
	if length != 3 {
		t.Errorf("Expected length 3 after trim, got %d", length)
	}
}

func TestXRead(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Add test entry
	fields := map[string]string{"field": "value"}
	id, _ := storage.XAdd("test-stream", StreamID{Timestamp: 1000, Sequence: 0}, fields, 0, false)
	
	// Test non-blocking read
	ctx := context.Background()
	streams := []string{"test-stream"}
	ids := []StreamID{{Timestamp: 0, Sequence: 0}}
	
	result, err := storage.XRead(ctx, "client1", streams, ids, 10, 0)
	if err != nil {
		t.Fatalf("XRead failed: %v", err)
	}
	
	if len(result.Streams) != 1 {
		t.Errorf("Expected 1 stream result, got %d", len(result.Streams))
	}
	
	if len(result.Streams[0].Entries) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(result.Streams[0].Entries))
	}
	
	if result.Streams[0].Entries[0].ID != id {
		t.Error("Entry ID mismatch")
	}
}

func TestXReadBlocking(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	streams := []string{"test-stream"}
	ids := []StreamID{{Timestamp: 0, Sequence: 0}}
	
	// This should timeout since no data is available
	result, err := storage.XRead(ctx, "client1", streams, ids, 10, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("XRead blocking failed: %v", err)
	}
	
	// Should return empty result due to timeout
	if result != nil && len(result.Streams) > 0 {
		t.Error("Expected empty result due to timeout")
	}
}

func TestXGroupCreate(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Create consumer group
	err := storage.XGroupCreate("test-stream", "test-group", StreamID{Timestamp: 0, Sequence: 0}, true)
	if err != nil {
		t.Fatalf("XGroupCreate failed: %v", err)
	}
	
	// Try to create same group again (should fail)
	err = storage.XGroupCreate("test-stream", "test-group", StreamID{Timestamp: 0, Sequence: 0}, false)
	if err == nil {
		t.Error("Expected error when creating duplicate group")
	}
}

func TestXGroupCreateConsumer(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Create consumer group first
	storage.XGroupCreate("test-stream", "test-group", StreamID{Timestamp: 0, Sequence: 0}, true)
	
	// Create consumer
	err := storage.XGroupCreateConsumer("test-stream", "test-group", "consumer1")
	if err != nil {
		t.Fatalf("XGroupCreateConsumer failed: %v", err)
	}
	
	// Create same consumer again (should be idempotent)
	err = storage.XGroupCreateConsumer("test-stream", "test-group", "consumer1")
	if err != nil {
		t.Fatalf("XGroupCreateConsumer should be idempotent: %v", err)
	}
}

func TestXAck(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Setup: create group and add entry
	storage.XGroupCreate("test-stream", "test-group", StreamID{Timestamp: 0, Sequence: 0}, true)
	fields := map[string]string{"field": "value"}
	id, _ := storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	
	// Read entry (adds to PEL) - use ">" to read new messages
	ctx := context.Background()
	streams := []string{"test-stream"}
	ids := []StreamID{MaxStreamID} // Use ">" equivalent to read new messages
	
	storage.XReadGroup(ctx, "client1", "test-group", "consumer1", streams, ids, 1, 0, false)
	
	// Acknowledge the entry
	acked, err := storage.XAck("test-stream", "test-group", []StreamID{id})
	if err != nil {
		t.Fatalf("XAck failed: %v", err)
	}
	
	if acked != 1 {
		t.Errorf("Expected 1 acknowledged entry, got %d", acked)
	}
}

func TestGetStreamInfo(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Test non-existent stream
	stream, exists := storage.GetStreamInfo("nonexistent")
	if exists {
		t.Error("Expected false for non-existent stream")
	}
	if stream != nil {
		t.Error("Expected nil stream for non-existent stream")
	}
	
	// Add entry and test
	fields := map[string]string{"field": "value"}
	storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	
	stream, exists = storage.GetStreamInfo("test-stream")
	if !exists {
		t.Error("Expected true for existing stream")
	}
	if stream == nil {
		t.Error("Expected non-nil stream")
	}
}

func TestCleanupBlockedClients(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// This should not panic even with no blocked clients
	storage.CleanupBlockedClients("client1")
	
	// Add a stream to ensure the method works with existing streams
	fields := map[string]string{"field": "value"}
	storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	
	storage.CleanupBlockedClients("client1")
}

// Test XGroupDelConsumer
func TestXGroupDelConsumer(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Add stream and create consumer group
	fields := map[string]string{"field": "value"}
	streamID, _ := storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	storage.XGroupCreate("test-stream", "group1", streamID, false)
	storage.XGroupCreateConsumer("test-stream", "group1", "consumer1")
	
	// Test deleting existing consumer
	deleted, err := storage.XGroupDelConsumer("test-stream", "group1", "consumer1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if deleted != 0 {
		t.Errorf("Expected 0 pending entries, got %d", deleted)
	}
	
	// Test deleting non-existent consumer
	deleted, err = storage.XGroupDelConsumer("test-stream", "group1", "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent consumer")
	}
	
	// Test with non-existent group
	deleted, err = storage.XGroupDelConsumer("test-stream", "nonexistent", "consumer1")
	if err == nil {
		t.Error("Expected error for non-existent group")
	}
}

// Test XGroupDestroy
func TestXGroupDestroy(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Add stream and create consumer group
	fields := map[string]string{"field": "value"}
	streamID, _ := storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	storage.XGroupCreate("test-stream", "group1", streamID, false)
	
	// Test destroying existing group
	err := storage.XGroupDestroy("test-stream", "group1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Test destroying non-existent group
	err = storage.XGroupDestroy("test-stream", "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent group")
	}
	
	// Test with non-existent stream
	err = storage.XGroupDestroy("nonexistent", "group1")
	if err == nil {
		t.Error("Expected error for non-existent stream")
	}
}

// Test XGroupSetID
func TestXGroupSetID(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Add stream and create consumer group
	fields := map[string]string{"field": "value"}
	streamID, _ := storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	storage.XGroupCreate("test-stream", "group1", streamID, false)
	
	// Test setting ID for existing group
	newID := StreamID{Timestamp: 2000, Sequence: 0}
	err := storage.XGroupSetID("test-stream", "group1", newID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Test with non-existent group
	err = storage.XGroupSetID("test-stream", "nonexistent", newID)
	if err == nil {
		t.Error("Expected error for non-existent group")
	}
	
	// Test with non-existent stream
	err = storage.XGroupSetID("nonexistent", "group1", newID)
	if err == nil {
		t.Error("Expected error for non-existent stream")
	}
}

// Test XInfoStream
func TestXInfoStream(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Add entries to stream
	fields1 := map[string]string{"field1": "value1"}
	fields2 := map[string]string{"field2": "value2"}
	_, _ = storage.XAdd("test-stream", StreamID{}, fields1, 0, false)
	_, _ = storage.XAdd("test-stream", StreamID{}, fields2, 0, false)
	
	// Test getting stream info
	info, err := storage.XInfoStream("test-stream")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if info.Length != int64(2) {
		t.Errorf("Expected length 2, got %v", info.Length)
	}
	if info.FirstEntry == nil {
		t.Error("Expected first-entry to be set")
	}
	if info.LastEntry == nil {
		t.Error("Expected last-entry to be set")
	}
	
	// Test with non-existent stream
	info, err = storage.XInfoStream("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent stream")
	}
}

// Test XInfoGroups
func TestXInfoGroups(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Add stream and create consumer groups
	fields := map[string]string{"field": "value"}
	streamID, _ := storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	storage.XGroupCreate("test-stream", "group1", streamID, false)
	storage.XGroupCreate("test-stream", "group2", streamID, false)
	
	// Test getting groups info
	groups, err := storage.XInfoGroups("test-stream")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}
	
	// Test with non-existent stream
	groups, err = storage.XInfoGroups("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent stream")
	}
}

// Test XInfoConsumers
func TestXInfoConsumers(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Add stream and create consumer group with consumers
	fields := map[string]string{"field": "value"}
	streamID, _ := storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	storage.XGroupCreate("test-stream", "group1", streamID, false)
	storage.XGroupCreateConsumer("test-stream", "group1", "consumer1")
	storage.XGroupCreateConsumer("test-stream", "group1", "consumer2")
	
	// Test getting consumers info
	consumers, err := storage.XInfoConsumers("test-stream", "group1")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(consumers) != 2 {
		t.Errorf("Expected 2 consumers, got %d", len(consumers))
	}
	
	// Test with non-existent group
	consumers, err = storage.XInfoConsumers("test-stream", "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent group")
	}
}

// Test XPending
func TestXPending(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Add stream and create consumer group
	fields := map[string]string{"field": "value"}
	streamID, _ := storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	storage.XGroupCreate("test-stream", "group1", streamID, false)
	storage.XGroupCreateConsumer("test-stream", "group1", "consumer1")
	
	// Test getting pending info
	pending, err := storage.XPending("test-stream", "group1", StreamID{}, StreamID{Timestamp: 9999999999999, Sequence: 9999999999999}, 10, "")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if pending == nil {
		t.Error("Expected pending result to be non-nil")
	}
	
	// Test with non-existent group
	pending, err = storage.XPending("test-stream", "nonexistent", StreamID{}, StreamID{Timestamp: 9999999999999, Sequence: 9999999999999}, 10, "")
	if err == nil {
		t.Error("Expected error for non-existent group")
	}
}

// Test XReadGroup
func TestXReadGroup(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Add stream and create consumer group
	fields := map[string]string{"field": "value"}
	_, _ = storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	storage.XGroupCreate("test-stream", "group1", StreamID{Timestamp: 0, Sequence: 0}, false)
	storage.XGroupCreateConsumer("test-stream", "group1", "consumer1")
	
	// Test non-blocking read
	ctx := context.Background()
	streams := []string{"test-stream"}
	ids := []StreamID{{Timestamp: 0, Sequence: 0}}
	result, err := storage.XReadGroup(ctx, "client1", "group1", "consumer1", streams, ids, 10, 0, false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Error("Expected result to be non-nil")
	}
	
	// Test with non-existent group - should return empty result, not error
	result, err = storage.XReadGroup(ctx, "client1", "nonexistent", "consumer1", streams, ids, 10, 0, false)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil || len(result.Streams) != 0 {
		t.Error("Expected empty result for non-existent group")
	}
}

// Test blocking group read functions
func TestBlockingGroupRead(t *testing.T) {
	storage := NewStreamStorage(nil)
	
	// Add stream and create consumer group
	fields := map[string]string{"field": "value"}
	_, _ = storage.XAdd("test-stream", StreamID{}, fields, 0, false)
	storage.XGroupCreate("test-stream", "group1", StreamID{Timestamp: 0, Sequence: 0}, false)
	storage.XGroupCreateConsumer("test-stream", "group1", "consumer1")
	
	// Test blocking read with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	
	streams := []string{"test-stream"}
	ids := []StreamID{{Timestamp: 9999999999999, Sequence: 9999999999999}}
	result, err := storage.XReadGroup(ctx, "client1", "group1", "consumer1", streams, ids, 10, 100*time.Millisecond, false)
	
	// Should return empty result on timeout
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Error("Expected non-nil result")
	}
	if len(result.Streams) != 0 {
		t.Error("Expected empty streams on timeout")
	}
	
	// Test cleanup
	storage.CleanupBlockedClients("client1")
}
