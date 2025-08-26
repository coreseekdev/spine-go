package storage

import (
	"fmt"
	"strings"
)

// BitmapStorageImpl implements the BitmapStorage interface
type BitmapStorageImpl struct {
	db *Database
}

// NewBitmapStorage creates a new bitmap storage instance
func NewBitmapStorage(db *Database) BitmapStorage {
	return &BitmapStorageImpl{db: db}
}

// SetBit sets the bit at offset in the string value stored at key
func (bs *BitmapStorageImpl) SetBit(key string, offset int64, value int) (int, error) {
	bs.db.mu.Lock()
	defer bs.db.mu.Unlock()

	if offset < 0 {
		return 0, fmt.Errorf("bit offset is not an integer or out of range")
	}

	if value != 0 && value != 1 {
		return 0, fmt.Errorf("bit is not an integer or out of range")
	}

	// Get existing value or create empty string
	var data string
	if val, exists := bs.db.data[key]; exists && !val.IsExpired() {
		if val.Type != TypeString {
			return 0, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
		data = val.Data.(string)
	}

	// Calculate byte and bit position
	bytePos := offset / 8
	bitPos := offset % 8

	// Extend string if necessary
	for int64(len(data)) <= bytePos {
		data += "\x00"
	}

	// Convert to byte slice for manipulation
	bytes := []byte(data)
	oldBit := (bytes[bytePos] >> (7 - bitPos)) & 1

	if value == 1 {
		bytes[bytePos] |= (1 << (7 - bitPos))
	} else {
		bytes[bytePos] &^= (1 << (7 - bitPos))
	}

	// Store back as string
	bs.db.data[key] = &Value{
		Type:      TypeString,
		Data:      string(bytes),
		ExpiresAt: nil,
	}

	return int(oldBit), nil
}

// GetBit returns the bit value at offset in the string value stored at key
func (bs *BitmapStorageImpl) GetBit(key string, offset int64) (int, error) {
	bs.db.mu.RLock()
	defer bs.db.mu.RUnlock()

	if offset < 0 {
		return 0, fmt.Errorf("bit offset is not an integer or out of range")
	}

	val, exists := bs.db.data[key]
	if !exists || val.IsExpired() {
		return 0, nil
	}

	if val.Type != TypeString {
		return 0, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	data := val.Data.(string)
	bytePos := offset / 8
	bitPos := offset % 8

	// If offset is beyond string length, return 0
	if int64(len(data)) <= bytePos {
		return 0, nil
	}

	bytes := []byte(data)
	bit := (bytes[bytePos] >> (7 - bitPos)) & 1
	return int(bit), nil
}

// BitCount returns the number of bits set to 1 in the string
func (bs *BitmapStorageImpl) BitCount(key string, start, end int64) (int64, error) {
	bs.db.mu.RLock()
	defer bs.db.mu.RUnlock()

	val, exists := bs.db.data[key]
	if !exists || val.IsExpired() {
		return 0, nil
	}

	if val.Type != TypeString {
		return 0, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	data := val.Data.(string)
	bytes := []byte(data)
	
	// Handle negative indices
	length := int64(len(bytes))
	if start < 0 {
		start = length + start
	}
	if end < 0 {
		end = length + end
	}

	// Clamp to valid range
	if start < 0 {
		start = 0
	}
	if end >= length {
		end = length - 1
	}
	if start > end {
		return 0, nil
	}

	count := int64(0)
	for i := start; i <= end; i++ {
		b := bytes[i]
		// Count bits using Brian Kernighan's algorithm
		for b != 0 {
			count++
			b &= b - 1
		}
	}

	return count, nil
}

// BitPos returns the position of the first bit set to 1 or 0
func (bs *BitmapStorageImpl) BitPos(key string, bit int, start, end int64) (int64, error) {
	bs.db.mu.RLock()
	defer bs.db.mu.RUnlock()

	if bit != 0 && bit != 1 {
		return 0, fmt.Errorf("bit is not an integer or out of range")
	}

	val, exists := bs.db.data[key]
	if !exists || val.IsExpired() {
		if bit == 0 {
			return 0, nil
		}
		return -1, nil
	}

	if val.Type != TypeString {
		return 0, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	data := val.Data.(string)
	bytes := []byte(data)
	length := int64(len(bytes))

	// Handle negative indices
	if start < 0 {
		start = length + start
	}
	if end < 0 {
		end = length + end
	}

	// Clamp to valid range
	if start < 0 {
		start = 0
	}
	if end >= length {
		end = length - 1
	}

	for byteIdx := start; byteIdx <= end; byteIdx++ {
		b := bytes[byteIdx]
		for bitIdx := int64(0); bitIdx < 8; bitIdx++ {
			currentBit := (b >> (7 - bitIdx)) & 1
			if int(currentBit) == bit {
				return byteIdx*8 + bitIdx, nil
			}
		}
	}

	return -1, nil
}

// BitOp performs bitwise operations between strings
func (bs *BitmapStorageImpl) BitOp(operation string, destkey string, keys []string) (int64, error) {
	bs.db.mu.Lock()
	defer bs.db.mu.Unlock()

	operation = strings.ToUpper(operation)
	if operation != "AND" && operation != "OR" && operation != "XOR" && operation != "NOT" {
		return 0, fmt.Errorf("syntax error")
	}

	if operation == "NOT" && len(keys) != 1 {
		return 0, fmt.Errorf("BITOP NOT must be called with a single source key")
	}

	// Get all source strings
	var sources [][]byte
	maxLen := int64(0)

	for _, key := range keys {
		var data string
		if val, exists := bs.db.data[key]; exists && !val.IsExpired() {
			if val.Type != TypeString {
				return 0, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
			}
			data = val.Data.(string)
		}
		bytes := []byte(data)
		sources = append(sources, bytes)
		if int64(len(bytes)) > maxLen {
			maxLen = int64(len(bytes))
		}
	}

	// Perform operation
	result := make([]byte, maxLen)
	for i := int64(0); i < maxLen; i++ {
		var resultByte byte
		
		switch operation {
		case "AND":
			resultByte = 0xFF // Start with all bits set for AND
			for _, src := range sources {
				var srcByte byte
				if i < int64(len(src)) {
					srcByte = src[i]
				}
				resultByte &= srcByte
			}
		case "OR":
			resultByte = 0x00 // Start with no bits set for OR
			for _, src := range sources {
				var srcByte byte
				if i < int64(len(src)) {
					srcByte = src[i]
				}
				resultByte |= srcByte
			}
		case "XOR":
			resultByte = 0x00 // Start with no bits set for XOR
			for _, src := range sources {
				var srcByte byte
				if i < int64(len(src)) {
					srcByte = src[i]
				}
				resultByte ^= srcByte
			}
		case "NOT":
			var srcByte byte
			if i < int64(len(sources[0])) {
				srcByte = sources[0][i]
			}
			resultByte = ^srcByte
		}
		
		result[i] = resultByte
	}

	// Store result
	bs.db.data[destkey] = &Value{
		Type:      TypeString,
		Data:      string(result),
		ExpiresAt: nil,
	}

	return maxLen, nil
}
