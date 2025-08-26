package stream

import (
	"sync"
	"time"
)

// StreamStorageImpl implements the StreamStorage interface
type StreamStorageImpl struct {
	mu      sync.RWMutex
	streams map[string]*Stream
}

// NewStreamStorage creates a new stream storage instance
func NewStreamStorage(db interface{}) *StreamStorageImpl {
	return &StreamStorageImpl{
		streams: make(map[string]*Stream),
	}
}

// XAdd adds a new entry to a stream
func (s *StreamStorageImpl) XAdd(key string, id StreamID, fields map[string]string, maxLen int64, exact bool) (StreamID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ValidateFieldsMap(fields); err != nil {
		return StreamID{}, err
	}

	stream, exists := s.streams[key]
	if !exists {
		stream = &Stream{
			entries:        make([]*StreamEntry, 0),
			consumerGroups: make(map[string]*ConsumerGroup),
			blockedReaders: make(map[string]*BlockedReader),
			maxLen:         maxLen,
			trimExact:      exact,
		}
		s.streams[key] = stream
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	// Generate ID if auto-generation requested
	var finalID StreamID
	if id.Timestamp == 0 && id.Sequence == 0 {
		finalID = GenerateStreamID(stream.lastID)
	} else {
		if err := ValidateStreamID(id, stream.lastID); err != nil {
			return StreamID{}, err
		}
		finalID = id
	}

	// Create new entry
	entry := &StreamEntry{
		ID:     finalID,
		Fields: fields,
	}

	// Add entry to stream
	stream.entries = append(stream.entries, entry)
	stream.lastID = finalID
	stream.length++

	// Trim if necessary
	if maxLen > 0 && stream.length > maxLen {
		s.trimStream(stream, maxLen, exact)
	}

	// Notify blocked readers
	s.notifyBlockedReaders(key, stream, entry)

	return finalID, nil
}

// XDel removes entries from a stream
func (s *StreamStorageImpl) XDel(key string, ids []StreamID) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stream, exists := s.streams[key]
	if !exists {
		return 0, nil
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	deleted := int64(0)
	newEntries := make([]*StreamEntry, 0, len(stream.entries))

	for _, entry := range stream.entries {
		shouldDelete := false
		for _, delID := range ids {
			if entry.ID.Compare(delID) == 0 {
				shouldDelete = true
				deleted++
				break
			}
		}
		if !shouldDelete {
			newEntries = append(newEntries, entry)
		}
	}

	stream.entries = newEntries
	stream.length = int64(len(newEntries))

	return deleted, nil
}

// XLen returns the length of a stream
func (s *StreamStorageImpl) XLen(key string) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, exists := s.streams[key]
	if !exists {
		return 0, nil
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	return stream.length, nil
}

// XRange returns entries in a range
func (s *StreamStorageImpl) XRange(key string, start, end StreamID, count int64) ([]*StreamEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, exists := s.streams[key]
	if !exists {
		return []*StreamEntry{}, nil
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	var result []*StreamEntry
	collected := int64(0)

	for _, entry := range stream.entries {
		// Handle special cases for min/max IDs
		startOK := start.Timestamp == 0 && start.Sequence == 0 || entry.ID.Compare(start) >= 0
		endOK := end.Timestamp == ^uint64(0) && end.Sequence == ^uint64(0) || entry.ID.Compare(end) <= 0
		
		if startOK && endOK {
			result = append(result, entry)
			collected++
			if count > 0 && collected >= count {
				break
			}
		}
	}

	return result, nil
}

// XRevRange returns entries in reverse range
func (s *StreamStorageImpl) XRevRange(key string, start, end StreamID, count int64) ([]*StreamEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, exists := s.streams[key]
	if !exists {
		return []*StreamEntry{}, nil
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()

	var result []*StreamEntry
	collected := int64(0)

	// Iterate in reverse order
	for i := len(stream.entries) - 1; i >= 0; i-- {
		entry := stream.entries[i]
		// Handle special cases for min/max IDs in reverse
		startOK := start.Timestamp == ^uint64(0) && start.Sequence == ^uint64(0) || entry.ID.Compare(start) <= 0
		endOK := end.Timestamp == 0 && end.Sequence == 0 || entry.ID.Compare(end) >= 0
		
		if startOK && endOK {
			result = append(result, entry)
			collected++
			if count > 0 && collected >= count {
				break
			}
		}
	}

	return result, nil
}

// XTrim trims a stream to a specified length
func (s *StreamStorageImpl) XTrim(key string, options TrimOptions) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stream, exists := s.streams[key]
	if !exists {
		return 0, nil
	}

	stream.mu.Lock()
	defer stream.mu.Unlock()

	originalLength := stream.length

	switch options.Strategy {
	case TrimByLength:
		if stream.length <= options.Threshold {
			return 0, nil
		}
		
		var targetLength int64
		if options.Exact {
			targetLength = options.Threshold
		} else {
			targetLength = CalculateApproximateLength(stream.length, options.Threshold)
		}
		
		toRemove := stream.length - targetLength
		if toRemove > 0 {
			if options.Limit > 0 && toRemove > options.Limit {
				toRemove = options.Limit
			}
			
			stream.entries = stream.entries[toRemove:]
			stream.length = int64(len(stream.entries))
		}
	}

	return originalLength - stream.length, nil
}

// GetStreamInfo returns stream information
func (s *StreamStorageImpl) GetStreamInfo(key string) (*Stream, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stream, exists := s.streams[key]
	return stream, exists
}

// CleanupBlockedClients removes blocked clients for a specific client ID
func (s *StreamStorageImpl) CleanupBlockedClients(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, stream := range s.streams {
		stream.mu.Lock()
		
		// Clean up blocked readers
		if reader, exists := stream.blockedReaders[clientID]; exists {
			if reader.cancel != nil {
				reader.cancel()
			}
			delete(stream.blockedReaders, clientID)
		}
		
		// Clean up blocked group readers
		for _, group := range stream.consumerGroups {
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

// trimStream trims a stream to the specified maximum length
func (s *StreamStorageImpl) trimStream(stream *Stream, maxLen int64, exact bool) {
	if stream.length <= maxLen {
		return
	}

	var targetLength int64
	if exact {
		targetLength = maxLen
	} else {
		targetLength = CalculateApproximateLength(stream.length, maxLen)
	}

	toRemove := stream.length - targetLength
	if toRemove > 0 {
		stream.entries = stream.entries[toRemove:]
		stream.length = int64(len(stream.entries))
	}
}

// notifyBlockedReaders notifies all blocked readers when new data arrives
func (s *StreamStorageImpl) notifyBlockedReaders(key string, stream *Stream, newEntry *StreamEntry) {
	// Notify XREAD blocked clients
	for clientID, reader := range stream.blockedReaders {
		go func(r *BlockedReader, cid string) {
			// Check if this reader is interested in this entry
			for i, streamName := range r.streams {
				if streamName == key && newEntry.ID.Compare(r.ids[i]) > 0 {
					// Create result
					result := &ReadResult{
						Streams: []StreamReadResult{
							{
								Name:    key,
								Entries: []*StreamEntry{newEntry},
							},
						},
					}
					
					select {
					case r.resultChan <- result:
						// Successfully sent result
					case <-r.ctx.Done():
						// Context cancelled
					default:
						// Channel full, skip
					}
					
					// Remove from blocked readers
					stream.mu.Lock()
					delete(stream.blockedReaders, cid)
					stream.mu.Unlock()
					return
				}
			}
		}(reader, clientID)
	}

	// Notify XREADGROUP blocked clients
	for _, group := range stream.consumerGroups {
		group.mu.Lock()
		for clientID, reader := range group.blockedReads {
			go func(r *BlockedGroupReader, cid string, g *ConsumerGroup) {
				// Check if this reader is interested in this entry
				for _, streamName := range r.streams {
					if streamName == key {
						// For consumer groups, check if ID is greater than last delivered
						if newEntry.ID.Compare(g.lastID) > 0 {
							// Create result and add to PEL if not NOACK
							result := &ReadResult{
								Streams: []StreamReadResult{
									{
										Name:    key,
										Entries: []*StreamEntry{newEntry},
									},
								},
							}
							
							if !r.noAck {
								// Add to pending entries list
								g.mu.Lock()
								g.pending[newEntry.ID] = &PendingEntry{
									ID:            newEntry.ID,
									Consumer:      r.consumerName,
									DeliveryTime:  time.Now(),
									DeliveryCount: 1,
								}
								g.lastID = newEntry.ID
								g.mu.Unlock()
							}
							
							select {
							case r.resultChan <- result:
								// Successfully sent result
							case <-r.ctx.Done():
								// Context cancelled
							default:
								// Channel full, skip
							}
							
							// Remove from blocked readers
							g.mu.Lock()
							delete(g.blockedReads, cid)
							g.mu.Unlock()
							return
						}
					}
				}
			}(reader, clientID, group)
		}
		group.mu.Unlock()
	}
}
