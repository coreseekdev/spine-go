package stream

import (
	"fmt"
	"time"
)

// XGroupCreate creates a new consumer group
func (s *StreamStorageImpl) XGroupCreate(key, groupName string, id StreamID, mkStream bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stream, exists := s.streams[key]
	if !exists {
		if !mkStream {
			return fmt.Errorf("NOKEY No such key '%s'", key)
		}
		// Create new stream
		stream = &Stream{
			entries:        make([]*StreamEntry, 0),
			consumerGroups: make(map[string]*ConsumerGroup),
			blockedReaders: make(map[string]*BlockedReader),
		}
		s.streams[key] = stream
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	// Check if group already exists
	if _, exists := stream.consumerGroups[groupName]; exists {
		return fmt.Errorf("BUSYGROUP Consumer Group name already exists")
	}

	// Validate ID
	var startID StreamID
	if id.Compare(MaxStreamID) == 0 {
		// Special case: $ means start from the end
		if len(stream.entries) > 0 {
			startID = stream.entries[len(stream.entries)-1].ID
		} else {
			startID = MinStreamID
		}
	} else {
		startID = id
	}

	// Create consumer group
	group := &ConsumerGroup{
		name:         groupName,
		lastID:       startID,
		consumers:    make(map[string]*Consumer),
		pending:      make(map[StreamID]*PendingEntry),
		blockedReads: make(map[string]*BlockedGroupReader),
	}

	stream.consumerGroups[groupName] = group
	return nil
}

// XGroupCreateConsumer creates a consumer in a group
func (s *StreamStorageImpl) XGroupCreateConsumer(key, groupName, consumerName string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, exists := s.streams[key]
	if !exists {
		return fmt.Errorf("NOKEY No such key '%s'", key)
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	group, exists := stream.consumerGroups[groupName]
	if !exists {
		return fmt.Errorf("NOGROUP No such consumer group '%s' for key '%s'", groupName, key)
	}

	group.mu.Lock()
	defer group.mu.Unlock()

	// Check if consumer already exists (idempotent operation)
	if _, exists := group.consumers[consumerName]; exists {
		return nil // Already exists, return success
	}

	// Create consumer
	group.consumers[consumerName] = &Consumer{
		name:     consumerName,
		lastSeen: time.Now(),
	}

	return nil
}

// XGroupDelConsumer deletes a consumer from a group
func (s *StreamStorageImpl) XGroupDelConsumer(key, groupName, consumerName string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stream, exists := s.streams[key]
	if !exists {
		return 0, fmt.Errorf("NOKEY No such key '%s'", key)
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	group, exists := stream.consumerGroups[groupName]
	if !exists {
		return 0, fmt.Errorf("NOGROUP No such consumer group '%s' for key '%s'", groupName, key)
	}

	group.mu.Lock()
	defer group.mu.Unlock()

	// Check if consumer exists
	if _, exists := group.consumers[consumerName]; !exists {
		return 0, fmt.Errorf("NOGROUP No such consumer '%s' in group '%s'", consumerName, groupName)
	}

	// Count and remove pending entries for this consumer
	pendingCount := int64(0)
	for id, pending := range group.pending {
		if pending.Consumer == consumerName {
			delete(group.pending, id)
			pendingCount++
		}
	}

	// Remove consumer
	delete(group.consumers, consumerName)

	return pendingCount, nil
}

// XGroupDestroy destroys a consumer group
func (s *StreamStorageImpl) XGroupDestroy(key, groupName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stream, exists := s.streams[key]
	if !exists {
		return fmt.Errorf("NOKEY No such key '%s'", key)
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	// Check if group exists
	if _, exists := stream.consumerGroups[groupName]; !exists {
		return fmt.Errorf("NOGROUP No such consumer group '%s' for key '%s'", groupName, key)
	}

	// Remove group
	delete(stream.consumerGroups, groupName)
	return nil
}

// XGroupSetID sets the last delivered ID for a consumer group
func (s *StreamStorageImpl) XGroupSetID(key, groupName string, id StreamID) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, exists := s.streams[key]
	if !exists {
		return fmt.Errorf("NOKEY No such key '%s'", key)
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	group, exists := stream.consumerGroups[groupName]
	if !exists {
		return fmt.Errorf("NOGROUP No such consumer group '%s' for key '%s'", groupName, key)
	}

	group.mu.Lock()
	defer group.mu.Unlock()

	// Set the last delivered ID
	group.lastID = id
	return nil
}

// XAck acknowledges processed messages
func (s *StreamStorageImpl) XAck(key, groupName string, ids []StreamID) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, exists := s.streams[key]
	if !exists {
		return 0, nil // Redis returns 0 for non-existent streams
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	group, exists := stream.consumerGroups[groupName]
	if !exists {
		return 0, nil // Redis returns 0 for non-existent groups
	}

	group.mu.Lock()
	defer group.mu.Unlock()

	acknowledged := int64(0)
	for _, id := range ids {
		if _, exists := group.pending[id]; exists {
			delete(group.pending, id)
			acknowledged++
		}
	}

	return acknowledged, nil
}

// XInfoStream returns information about a stream
func (s *StreamStorageImpl) XInfoStream(key string) (*StreamInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, exists := s.streams[key]
	if !exists {
		return nil, fmt.Errorf("NOKEY No such key '%s'", key)
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	info := &StreamInfo{
		Length:          stream.length,
		RadixTreeKeys:   stream.length, // Simplified
		RadixTreeNodes:  stream.length, // Simplified
		Groups:          int64(len(stream.consumerGroups)),
		LastGeneratedID: stream.lastID,
	}

	if len(stream.entries) > 0 {
		info.FirstEntry = stream.entries[0]
		info.LastEntry = stream.entries[len(stream.entries)-1]
	}

	return info, nil
}

// XInfoGroups returns information about consumer groups
func (s *StreamStorageImpl) XInfoGroups(key string) ([]*GroupInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, exists := s.streams[key]
	if !exists {
		return nil, fmt.Errorf("NOKEY No such key '%s'", key)
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	var groups []*GroupInfo
	for _, group := range stream.consumerGroups {
		group.mu.RLock()
		info := &GroupInfo{
			Name:            group.name,
			Consumers:       int64(len(group.consumers)),
			Pending:         int64(len(group.pending)),
			LastDeliveredID: group.lastID,
		}
		groups = append(groups, info)
		group.mu.RUnlock()
	}

	return groups, nil
}

// XInfoConsumers returns information about consumers in a group
func (s *StreamStorageImpl) XInfoConsumers(key, groupName string) ([]*ConsumerInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, exists := s.streams[key]
	if !exists {
		return nil, fmt.Errorf("NOKEY No such key '%s'", key)
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	group, exists := stream.consumerGroups[groupName]
	if !exists {
		return nil, fmt.Errorf("NOGROUP No such consumer group '%s' for key '%s'", groupName, key)
	}

	group.mu.RLock()
	defer group.mu.RUnlock()

	var consumers []*ConsumerInfo
	now := time.Now()

	for _, consumer := range group.consumers {
		// Count pending entries for this consumer
		pendingCount := int64(0)
		for _, pending := range group.pending {
			if pending.Consumer == consumer.name {
				pendingCount++
			}
		}

		info := &ConsumerInfo{
			Name:     consumer.name,
			Pending:  pendingCount,
			Idle:     now.Sub(consumer.lastSeen),
			Inactive: now.Sub(consumer.lastSeen), // Same as idle for simplicity
		}
		consumers = append(consumers, info)
	}

	return consumers, nil
}

// XPending returns pending entries information
func (s *StreamStorageImpl) XPending(key, groupName string, start, end StreamID, count int64, consumerName string) (*PendingInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, exists := s.streams[key]
	if !exists {
		return nil, fmt.Errorf("NOKEY No such key '%s'", key)
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	group, exists := stream.consumerGroups[groupName]
	if !exists {
		return nil, fmt.Errorf("NOGROUP No such consumer group '%s' for key '%s'", groupName, key)
	}

	group.mu.RLock()
	defer group.mu.RUnlock()

	info := &PendingInfo{
		Consumers: make(map[string]int64),
		Entries:   make([]*PendingEntryInfo, 0),
	}

	// Count entries and build consumer map
	now := time.Now()
	var minID, maxID StreamID
	first := true

	for id, pending := range group.pending {
		// Filter by consumer if specified
		if consumerName != "" && pending.Consumer != consumerName {
			continue
		}

		// Filter by ID range
		if id.Compare(start) >= 0 && id.Compare(end) <= 0 {
			info.Count++
			info.Consumers[pending.Consumer]++

			if first {
				minID = id
				maxID = id
				first = false
			} else {
				if id.Compare(minID) < 0 {
					minID = id
				}
				if id.Compare(maxID) > 0 {
					maxID = id
				}
			}

			// Add detailed entry info if count allows
			if count > 0 && int64(len(info.Entries)) < count {
				entryInfo := &PendingEntryInfo{
					ID:            id,
					Consumer:      pending.Consumer,
					ElapsedTime:   now.Sub(pending.DeliveryTime),
					DeliveryCount: pending.DeliveryCount,
				}
				info.Entries = append(info.Entries, entryInfo)
			}
		}
	}

	if !first {
		info.StartID = minID
		info.EndID = maxID
	}

	return info, nil
}
