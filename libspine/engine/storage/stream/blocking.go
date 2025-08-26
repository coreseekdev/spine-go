package stream

import (
	"context"
	"fmt"
	"time"
)

// XRead implements blocking read from multiple streams
func (s *StreamStorageImpl) XRead(ctx context.Context, clientID string, streams []string, ids []StreamID, count int64, timeout time.Duration) (*ReadResult, error) {
	// First, try non-blocking read
	result, hasData := s.tryNonBlockingRead(streams, ids, count)
	if hasData {
		return result, nil
	}

	// If no timeout specified, return immediately
	if timeout == 0 {
		return &ReadResult{Streams: []StreamReadResult{}}, nil
	}

	// Setup blocking read
	return s.setupBlockingRead(ctx, clientID, streams, ids, count, timeout)
}

// XReadGroup implements blocking read from streams for consumer groups
func (s *StreamStorageImpl) XReadGroup(ctx context.Context, clientID string, groupName, consumerName string, streams []string, ids []StreamID, count int64, timeout time.Duration, noAck bool) (*ReadResult, error) {
	// First, try to read pending entries or new messages
	result, hasData := s.tryNonBlockingGroupRead(groupName, consumerName, streams, ids, count, noAck)
	if hasData {
		return result, nil
	}

	// If no timeout specified, return immediately
	if timeout == 0 {
		return &ReadResult{Streams: []StreamReadResult{}}, nil
	}

	// Setup blocking group read
	return s.setupBlockingGroupRead(ctx, clientID, groupName, consumerName, streams, ids, count, timeout, noAck)
}

// tryNonBlockingRead attempts to read data without blocking
func (s *StreamStorageImpl) tryNonBlockingRead(streams []string, ids []StreamID, count int64) (*ReadResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var streamResults []StreamReadResult
	hasData := false

	for i, streamName := range streams {
		stream, exists := s.streams[streamName]
		if !exists {
			continue
		}

		stream.mu.RLock()
		var entries []*StreamEntry
		collected := int64(0)

		for _, entry := range stream.entries {
			if entry.ID.Compare(ids[i]) > 0 {
				entries = append(entries, entry)
				collected++
				if count > 0 && collected >= count {
					break
				}
			}
		}
		stream.mu.RUnlock()

		if len(entries) > 0 {
			streamResults = append(streamResults, StreamReadResult{
				Name:    streamName,
				Entries: entries,
			})
			hasData = true
		}
	}

	if hasData {
		return &ReadResult{Streams: streamResults}, true
	}

	return nil, false
}

// tryNonBlockingGroupRead attempts to read data for consumer groups without blocking
func (s *StreamStorageImpl) tryNonBlockingGroupRead(groupName, consumerName string, streams []string, ids []StreamID, count int64, noAck bool) (*ReadResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var streamResults []StreamReadResult
	hasData := false

	for i, streamName := range streams {
		stream, exists := s.streams[streamName]
		if !exists {
			continue
		}

		stream.mu.RLock()
		group, groupExists := stream.consumerGroups[groupName]
		if !groupExists {
			stream.mu.RUnlock()
			continue
		}

		group.mu.Lock()
		
		// Ensure consumer exists
		if _, consumerExists := group.consumers[consumerName]; !consumerExists {
			group.consumers[consumerName] = &Consumer{
				name:     consumerName,
				lastSeen: time.Now(),
			}
		}

		var entries []*StreamEntry
		collected := int64(0)

		// Check if requesting pending entries (ID != ">")
		if ids[i].Compare(MaxStreamID) != 0 {
			// Read pending entries for this consumer
			for pendingID, pending := range group.pending {
				if pending.Consumer == consumerName && pendingID.Compare(ids[i]) > 0 {
					// Find the entry in stream
					for _, entry := range stream.entries {
						if entry.ID.Compare(pendingID) == 0 {
							entries = append(entries, entry)
							collected++
							
							// Update delivery info
							pending.DeliveryTime = time.Now()
							pending.DeliveryCount++
							
							if count > 0 && collected >= count {
								break
							}
						}
					}
				}
			}
		} else {
			// Read new messages (ID == ">")
			for _, entry := range stream.entries {
				if entry.ID.Compare(group.lastID) > 0 {
					entries = append(entries, entry)
					collected++

					// Add to PEL if not NOACK
					if !noAck {
						group.pending[entry.ID] = &PendingEntry{
							ID:            entry.ID,
							Consumer:      consumerName,
							DeliveryTime:  time.Now(),
							DeliveryCount: 1,
						}
					}

					// Update group's last delivered ID
					group.lastID = entry.ID

					if count > 0 && collected >= count {
						break
					}
				}
			}
		}

		group.mu.Unlock()
		stream.mu.RUnlock()

		if len(entries) > 0 {
			streamResults = append(streamResults, StreamReadResult{
				Name:    streamName,
				Entries: entries,
			})
			hasData = true
		}
	}

	if hasData {
		return &ReadResult{Streams: streamResults}, true
	}

	return nil, false
}

// setupBlockingRead sets up a blocking read operation
func (s *StreamStorageImpl) setupBlockingRead(ctx context.Context, clientID string, streams []string, ids []StreamID, count int64, timeout time.Duration) (*ReadResult, error) {
	// Create context with timeout
	blockCtx, cancel := context.WithTimeout(ctx, timeout)
	
	// Create result channel
	resultChan := make(chan *ReadResult, 1)
	
	// Create blocked reader
	reader := &BlockedReader{
		ctx:        blockCtx,
		cancel:     cancel,
		clientID:   clientID,
		streams:    streams,
		ids:        ids,
		count:      count,
		timeout:    timeout,
		resultChan: resultChan,
		createdAt:  time.Now(),
	}

	// Register reader with all relevant streams
	s.mu.Lock()
	for _, streamName := range streams {
		stream, exists := s.streams[streamName]
		if !exists {
			// Create empty stream for future notifications
			stream = &Stream{
				entries:        make([]*StreamEntry, 0),
				consumerGroups: make(map[string]*ConsumerGroup),
				blockedReaders: make(map[string]*BlockedReader),
			}
			s.streams[streamName] = stream
		}
		
		stream.mu.Lock()
		stream.blockedReaders[clientID] = reader
		stream.mu.Unlock()
	}
	s.mu.Unlock()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result, nil
	case <-blockCtx.Done():
		// Timeout or cancellation - cleanup and return empty result
		s.cleanupBlockedReader(clientID, streams)
		return &ReadResult{Streams: []StreamReadResult{}}, nil
	}
}

// setupBlockingGroupRead sets up a blocking read operation for consumer groups
func (s *StreamStorageImpl) setupBlockingGroupRead(ctx context.Context, clientID string, groupName, consumerName string, streams []string, ids []StreamID, count int64, timeout time.Duration, noAck bool) (*ReadResult, error) {
	// Create context with timeout
	blockCtx, cancel := context.WithTimeout(ctx, timeout)
	
	// Create result channel
	resultChan := make(chan *ReadResult, 1)
	
	// Create blocked group reader
	reader := &BlockedGroupReader{
		ctx:          blockCtx,
		cancel:       cancel,
		clientID:     clientID,
		groupName:    groupName,
		consumerName: consumerName,
		streams:      streams,
		ids:          ids,
		count:        count,
		timeout:      timeout,
		noAck:        noAck,
		resultChan:   resultChan,
		createdAt:    time.Now(),
	}

	// Register reader with all relevant consumer groups
	s.mu.Lock()
	for _, streamName := range streams {
		stream, exists := s.streams[streamName]
		if !exists {
			// Create empty stream
			stream = &Stream{
				entries:        make([]*StreamEntry, 0),
				consumerGroups: make(map[string]*ConsumerGroup),
				blockedReaders: make(map[string]*BlockedReader),
			}
			s.streams[streamName] = stream
		}
		
		stream.mu.Lock()
		group, groupExists := stream.consumerGroups[groupName]
		if !groupExists {
			stream.mu.Unlock()
			s.mu.Unlock()
			return nil, fmt.Errorf("NOGROUP No such key '%s' or consumer group '%s'", streamName, groupName)
		}
		
		group.mu.Lock()
		if group.blockedReads == nil {
			group.blockedReads = make(map[string]*BlockedGroupReader)
		}
		group.blockedReads[clientID] = reader
		group.mu.Unlock()
		stream.mu.Unlock()
	}
	s.mu.Unlock()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result, nil
	case <-blockCtx.Done():
		// Timeout or cancellation - cleanup and return empty result
		s.cleanupBlockedGroupReader(clientID, groupName, streams)
		return &ReadResult{Streams: []StreamReadResult{}}, nil
	}
}

// cleanupBlockedReader removes a blocked reader from all streams
func (s *StreamStorageImpl) cleanupBlockedReader(clientID string, streams []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, streamName := range streams {
		if stream, exists := s.streams[streamName]; exists {
			stream.mu.Lock()
			if reader, exists := stream.blockedReaders[clientID]; exists {
				if reader.cancel != nil {
					reader.cancel()
				}
				delete(stream.blockedReaders, clientID)
			}
			stream.mu.Unlock()
		}
	}
}

// cleanupBlockedGroupReader removes a blocked group reader from all consumer groups
func (s *StreamStorageImpl) cleanupBlockedGroupReader(clientID string, groupName string, streams []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, streamName := range streams {
		if stream, exists := s.streams[streamName]; exists {
			stream.mu.Lock()
			if group, exists := stream.consumerGroups[groupName]; exists {
				group.mu.Lock()
				if reader, exists := group.blockedReads[clientID]; exists {
					if reader.cancel != nil {
						reader.cancel()
					}
					delete(group.blockedReads, clientID)
				}
				group.mu.Unlock()
			}
			stream.mu.Unlock()
		}
	}
}
