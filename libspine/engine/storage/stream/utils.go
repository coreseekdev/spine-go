package stream

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseStreamID parses a string into a StreamID
func ParseStreamID(s string) (StreamID, error) {
	if s == "*" {
		// Auto-generate ID based on current time
		now := time.Now().UnixMilli()
		return StreamID{Timestamp: uint64(now), Sequence: 0}, nil
	}
	
	if s == "$" {
		// Special ID for latest entry (used in XREAD)
		return MaxStreamID, nil
	}
	
	if s == "+" {
		// Special ID for last entry (Redis 7.4+)
		return MaxStreamID, nil
	}
	
	if s == "-" {
		// Special ID for first entry
		return MinStreamID, nil
	}
	
	if s == ">" {
		// Special ID for consumer groups (new messages only)
		return MaxStreamID, nil
	}
	
	parts := strings.Split(s, "-")
	if len(parts) == 1 {
		// Incomplete ID, assume sequence is 0
		timestamp, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return StreamID{}, fmt.Errorf("invalid stream ID timestamp: %s", parts[0])
		}
		return StreamID{Timestamp: timestamp, Sequence: 0}, nil
	}
	
	if len(parts) != 2 {
		return StreamID{}, fmt.Errorf("invalid stream ID format: %s", s)
	}
	
	timestamp, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return StreamID{}, fmt.Errorf("invalid stream ID timestamp: %s", parts[0])
	}
	
	sequence, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return StreamID{}, fmt.Errorf("invalid stream ID sequence: %s", parts[1])
	}
	
	return StreamID{Timestamp: timestamp, Sequence: sequence}, nil
}

// GenerateStreamID generates a new StreamID based on the last ID and current time
func GenerateStreamID(lastID StreamID) StreamID {
	now := uint64(time.Now().UnixMilli())
	
	if now > lastID.Timestamp {
		return StreamID{Timestamp: now, Sequence: 0}
	}
	
	// Same millisecond, increment sequence
	return StreamID{Timestamp: lastID.Timestamp, Sequence: lastID.Sequence + 1}
}

// ValidateStreamID validates that an ID is valid for insertion
func ValidateStreamID(id StreamID, lastID StreamID) error {
	if id.Compare(lastID) <= 0 {
		return fmt.Errorf("The ID specified in XADD is equal or smaller than the target stream top item")
	}
	return nil
}

// IsSpecialID checks if the ID string is a special ID
func IsSpecialID(s string) bool {
	return s == "*" || s == "$" || s == "+" || s == "-" || s == ">"
}

// NextStreamID returns the next possible StreamID after the given ID
func NextStreamID(id StreamID) StreamID {
	if id.Sequence < ^uint64(0) {
		return StreamID{Timestamp: id.Timestamp, Sequence: id.Sequence + 1}
	}
	return StreamID{Timestamp: id.Timestamp + 1, Sequence: 0}
}

// PrevStreamID returns the previous possible StreamID before the given ID
func PrevStreamID(id StreamID) StreamID {
	if id.Sequence > 0 {
		return StreamID{Timestamp: id.Timestamp, Sequence: id.Sequence - 1}
	}
	if id.Timestamp > 0 {
		return StreamID{Timestamp: id.Timestamp - 1, Sequence: ^uint64(0)}
	}
	return MinStreamID
}

// MatchPattern checks if a stream name matches a pattern (for pattern-based operations)
func MatchPattern(pattern, name string) bool {
	// Simple glob pattern matching
	// For now, only support * wildcard
	if pattern == "*" {
		return true
	}
	
	if !strings.Contains(pattern, "*") {
		return pattern == name
	}
	
	// Simple pattern matching - can be enhanced later
	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		prefix := parts[0]
		suffix := parts[1]
		return strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix)
	}
	
	return false
}

// FormatStreamEntries formats stream entries for RESP output
func FormatStreamEntries(entries []*StreamEntry) []interface{} {
	result := make([]interface{}, len(entries))
	for i, entry := range entries {
		fields := make([]interface{}, 0, len(entry.Fields)*2)
		for k, v := range entry.Fields {
			fields = append(fields, k, v)
		}
		result[i] = []interface{}{entry.ID.String(), fields}
	}
	return result
}

// FormatReadResult formats read result for RESP output
func FormatReadResult(result *ReadResult) []interface{} {
	if result == nil || len(result.Streams) == 0 {
		return nil
	}
	
	output := make([]interface{}, len(result.Streams))
	for i, stream := range result.Streams {
		entries := FormatStreamEntries(stream.Entries)
		output[i] = []interface{}{stream.Name, entries}
	}
	return output
}

// CalculateApproximateLength calculates approximate length for trimming
func CalculateApproximateLength(currentLength, maxLen int64) int64 {
	// Redis uses radix tree nodes for approximation
	// For simplicity, we'll use a factor-based approach
	if currentLength <= maxLen {
		return currentLength
	}
	
	// Remove roughly 10% more than needed for efficiency
	toRemove := currentLength - maxLen
	approximateRemove := toRemove + (toRemove / 10)
	
	return currentLength - approximateRemove
}

// ValidateFieldsMap validates that fields map has even number of elements
func ValidateFieldsMap(fields map[string]string) error {
	if len(fields) == 0 {
		return fmt.Errorf("wrong number of arguments for XADD")
	}
	return nil
}

// ConvertFieldsArray converts field array to map
func ConvertFieldsArray(fields []string) (map[string]string, error) {
	if len(fields)%2 != 0 {
		return nil, fmt.Errorf("wrong number of arguments for XADD")
	}
	
	result := make(map[string]string)
	for i := 0; i < len(fields); i += 2 {
		result[fields[i]] = fields[i+1]
	}
	
	return result, nil
}
