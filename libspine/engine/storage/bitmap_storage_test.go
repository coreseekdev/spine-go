package storage

import (
	"testing"
)

func TestNewBitmapStorage(t *testing.T) {
	db := NewDatabase(0)
	bitmapStorage := NewBitmapStorage(db)
	
	if bitmapStorage == nil {
		t.Error("NewBitmapStorage should not return nil")
	}
	
	// Test that it implements BitmapStorage interface
	_, ok := bitmapStorage.(BitmapStorage)
	if !ok {
		t.Error("NewBitmapStorage should return an object that implements BitmapStorage interface")
	}
}

func TestSetBit(t *testing.T) {
	db := NewDatabase(0)
	bitmapStorage := NewBitmapStorage(db)
	
	tests := []struct {
		name           string
		key            string
		offset         int64
		value          int
		expectedOldBit int
		expectError    bool
	}{
		{"Set bit 0 at offset 0", "test1", 0, 1, 0, false},
		{"Set bit 1 at offset 7", "test1", 7, 1, 0, false},
		{"Clear bit at offset 0", "test1", 0, 0, 1, false},
		{"Set bit at large offset", "test2", 1000, 1, 0, false},
		{"Invalid negative offset", "test3", -1, 1, 0, true},
		{"Invalid bit value 2", "test4", 0, 2, 0, true},
		{"Invalid bit value -1", "test5", 0, -1, 0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldBit, err := bitmapStorage.SetBit(tt.key, tt.offset, tt.value)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}
			
			if oldBit != tt.expectedOldBit {
				t.Errorf("Expected old bit %d, got %d for %s", tt.expectedOldBit, oldBit, tt.name)
			}
		})
	}
}

func TestGetBit(t *testing.T) {
	db := NewDatabase(0)
	bitmapStorage := NewBitmapStorage(db)
	
	// Set some bits first
	bitmapStorage.SetBit("test", 0, 1)
	bitmapStorage.SetBit("test", 7, 1)
	bitmapStorage.SetBit("test", 15, 1)
	
	tests := []struct {
		name        string
		key         string
		offset      int64
		expectedBit int
		expectError bool
	}{
		{"Get set bit at offset 0", "test", 0, 1, false},
		{"Get set bit at offset 7", "test", 7, 1, false},
		{"Get unset bit at offset 1", "test", 1, 0, false},
		{"Get bit from non-existent key", "nonexistent", 0, 0, false},
		{"Get bit at large offset", "test", 1000, 0, false},
		{"Invalid negative offset", "test", -1, 0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bit, err := bitmapStorage.GetBit(tt.key, tt.offset)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}
			
			if bit != tt.expectedBit {
				t.Errorf("Expected bit %d, got %d for %s", tt.expectedBit, bit, tt.name)
			}
		})
	}
}

func TestBitCount(t *testing.T) {
	db := NewDatabase(0)
	bitmapStorage := NewBitmapStorage(db)
	
	// Set up test data: 11110000 (first byte)
	bitmapStorage.SetBit("test", 0, 1)
	bitmapStorage.SetBit("test", 1, 1)
	bitmapStorage.SetBit("test", 2, 1)
	bitmapStorage.SetBit("test", 3, 1)
	// bits 4-7 remain 0
	
	// Set up second byte: 10101010
	bitmapStorage.SetBit("test", 8, 1)
	bitmapStorage.SetBit("test", 10, 1)
	bitmapStorage.SetBit("test", 12, 1)
	bitmapStorage.SetBit("test", 14, 1)
	
	tests := []struct {
		name          string
		key           string
		start         int64
		end           int64
		expectedCount int64
		expectError   bool
	}{
		{"Count all bits in first byte", "test", 0, 0, 4, false},
		{"Count all bits in second byte", "test", 1, 1, 4, false},
		{"Count bits in both bytes", "test", 0, 1, 8, false},
		{"Count with negative start", "test", -2, -1, 8, false},
		{"Count non-existent key", "nonexistent", 0, 0, 0, false},
		{"Count with start > end", "test", 5, 2, 0, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := bitmapStorage.BitCount(tt.key, tt.start, tt.end)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}
			
			if count != tt.expectedCount {
				t.Errorf("Expected count %d, got %d for %s", tt.expectedCount, count, tt.name)
			}
		})
	}
}

func TestBitPos(t *testing.T) {
	db := NewDatabase(0)
	bitmapStorage := NewBitmapStorage(db)
	
	// Set up test data: 01110000 (first byte)
	bitmapStorage.SetBit("test", 1, 1)
	bitmapStorage.SetBit("test", 2, 1)
	bitmapStorage.SetBit("test", 3, 1)
	
	tests := []struct {
		name         string
		key          string
		bit          int
		start        int64
		end          int64
		expectedPos  int64
		expectError  bool
	}{
		{"Find first 1 bit", "test", 1, 0, -1, 1, false},
		{"Find first 0 bit", "test", 0, 0, -1, 0, false},
		{"Find 1 bit in range", "test", 1, 0, 0, 1, false},
		{"Find bit in non-existent key", "nonexistent", 1, 0, -1, -1, false},
		{"Find 0 bit in non-existent key", "nonexistent", 0, 0, -1, 0, false},
		{"Invalid bit value", "test", 2, 0, -1, 0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, err := bitmapStorage.BitPos(tt.key, tt.bit, tt.start, tt.end)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}
			
			if pos != tt.expectedPos {
				t.Errorf("Expected position %d, got %d for %s", tt.expectedPos, pos, tt.name)
			}
		})
	}
}

func TestBitOp(t *testing.T) {
	db := NewDatabase(0)
	bitmapStorage := NewBitmapStorage(db)
	
	// Set up test data
	// key1: 11110000
	bitmapStorage.SetBit("key1", 0, 1)
	bitmapStorage.SetBit("key1", 1, 1)
	bitmapStorage.SetBit("key1", 2, 1)
	bitmapStorage.SetBit("key1", 3, 1)
	
	// key2: 10101010
	bitmapStorage.SetBit("key2", 0, 1)
	bitmapStorage.SetBit("key2", 2, 1)
	bitmapStorage.SetBit("key2", 4, 1)
	bitmapStorage.SetBit("key2", 6, 1)
	
	tests := []struct {
		name           string
		operation      string
		destkey        string
		keys           []string
		expectedLength int64
		expectError    bool
	}{
		{"AND operation", "AND", "result1", []string{"key1", "key2"}, 1, false},
		{"OR operation", "OR", "result2", []string{"key1", "key2"}, 1, false},
		{"XOR operation", "XOR", "result3", []string{"key1", "key2"}, 1, false},
		{"NOT operation", "NOT", "result4", []string{"key1"}, 1, false},
		{"Invalid operation", "INVALID", "result5", []string{"key1"}, 0, true},
		{"NOT with multiple keys", "NOT", "result6", []string{"key1", "key2"}, 0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			length, err := bitmapStorage.BitOp(tt.operation, tt.destkey, tt.keys)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got none", tt.name)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.name, err)
				return
			}
			
			if length != tt.expectedLength {
				t.Errorf("Expected length %d, got %d for %s", tt.expectedLength, length, tt.name)
			}
			
			// Verify the result exists
			if !tt.expectError {
				bit, err := bitmapStorage.GetBit(tt.destkey, 0)
				if err != nil {
					t.Errorf("Failed to get result bit for %s: %v", tt.name, err)
				}
				
				// Basic sanity check - result should be valid bit value
				if bit != 0 && bit != 1 {
					t.Errorf("Invalid result bit value %d for %s", bit, tt.name)
				}
			}
		})
	}
}

func TestBitOpOperations(t *testing.T) {
	db := NewDatabase(0)
	bitmapStorage := NewBitmapStorage(db)
	
	// Set up specific test data for operation verification
	// key1: 11000000 (first byte)
	bitmapStorage.SetBit("op1", 0, 1)
	bitmapStorage.SetBit("op1", 1, 1)
	
	// key2: 10100000 (first byte)
	bitmapStorage.SetBit("op2", 0, 1)
	bitmapStorage.SetBit("op2", 2, 1)
	
	// Test AND: 11000000 AND 10100000 = 10000000
	bitmapStorage.BitOp("AND", "and_result", []string{"op1", "op2"})
	bit0, _ := bitmapStorage.GetBit("and_result", 0)
	bit1, _ := bitmapStorage.GetBit("and_result", 1)
	bit2, _ := bitmapStorage.GetBit("and_result", 2)
	
	if bit0 != 1 || bit1 != 0 || bit2 != 0 {
		t.Errorf("AND operation failed: expected bits [1,0,0], got [%d,%d,%d]", bit0, bit1, bit2)
	}
	
	// Test OR: 11000000 OR 10100000 = 11100000
	bitmapStorage.BitOp("OR", "or_result", []string{"op1", "op2"})
	bit0, _ = bitmapStorage.GetBit("or_result", 0)
	bit1, _ = bitmapStorage.GetBit("or_result", 1)
	bit2, _ = bitmapStorage.GetBit("or_result", 2)
	
	if bit0 != 1 || bit1 != 1 || bit2 != 1 {
		t.Errorf("OR operation failed: expected bits [1,1,1], got [%d,%d,%d]", bit0, bit1, bit2)
	}
	
	// Test XOR: 11000000 XOR 10100000 = 01100000
	bitmapStorage.BitOp("XOR", "xor_result", []string{"op1", "op2"})
	bit0, _ = bitmapStorage.GetBit("xor_result", 0)
	bit1, _ = bitmapStorage.GetBit("xor_result", 1)
	bit2, _ = bitmapStorage.GetBit("xor_result", 2)
	
	if bit0 != 0 || bit1 != 1 || bit2 != 1 {
		t.Errorf("XOR operation failed: expected bits [0,1,1], got [%d,%d,%d]", bit0, bit1, bit2)
	}
	
	// Test NOT: NOT 11000000 = 00111111
	bitmapStorage.BitOp("NOT", "not_result", []string{"op1"})
	bit0, _ = bitmapStorage.GetBit("not_result", 0)
	bit1, _ = bitmapStorage.GetBit("not_result", 1)
	bit2, _ = bitmapStorage.GetBit("not_result", 2)
	
	if bit0 != 0 || bit1 != 0 || bit2 != 1 {
		t.Errorf("NOT operation failed: expected bits [0,0,1], got [%d,%d,%d]", bit0, bit1, bit2)
	}
}
