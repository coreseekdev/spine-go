package stream

import (
	"testing"
)

// Test ParseStreamID
func TestParseStreamID(t *testing.T) {
	tests := []struct {
		input    string
		expected StreamID
		hasError bool
	}{
		{"1234-5", StreamID{Timestamp: 1234, Sequence: 5}, false},
		{"0-0", StreamID{Timestamp: 0, Sequence: 0}, false},
		{"9999999999999-9999999999999", StreamID{Timestamp: 9999999999999, Sequence: 9999999999999}, false},
		{"invalid", StreamID{}, true},
		{"1234", StreamID{Timestamp: 1234, Sequence: 0}, false}, // Single number is valid
		{"1234-", StreamID{}, true},
		{"-5", StreamID{}, true},
		{"abc-5", StreamID{}, true},
		{"1234-abc", StreamID{}, true},
	}

	for _, test := range tests {
		result, err := ParseStreamID(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %s, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("For input %s, expected %v, got %v", test.input, test.expected, result)
			}
		}
	}
}

// Test IsSpecialID
func TestIsSpecialID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"$", true},
		{">", true},
		{"*", true},
		{"+", true},
		{"-", true},
		{"1234-5", false},
		{"0-0", false},
		{"normal", false},
		{"", false},
	}

	for _, test := range tests {
		result := IsSpecialID(test.input)
		if result != test.expected {
			t.Errorf("For input %s, expected %v, got %v", test.input, test.expected, result)
		}
	}
}

// Test NextStreamID
func TestNextStreamID(t *testing.T) {
	tests := []struct {
		input    StreamID
		expected StreamID
	}{
		{StreamID{Timestamp: 1000, Sequence: 5}, StreamID{Timestamp: 1000, Sequence: 6}},
		{StreamID{Timestamp: 1000, Sequence: 5}, StreamID{Timestamp: 1000, Sequence: 6}},
		{StreamID{Timestamp: 0, Sequence: 0}, StreamID{Timestamp: 0, Sequence: 1}},
	}

	for _, test := range tests {
		result := NextStreamID(test.input)
		if result != test.expected {
			t.Errorf("For input %v, expected %v, got %v", test.input, test.expected, result)
		}
	}
}

// Test PrevStreamID
func TestPrevStreamID(t *testing.T) {
	tests := []struct {
		input    StreamID
		expected StreamID
	}{
		{StreamID{Timestamp: 1000, Sequence: 5}, StreamID{Timestamp: 1000, Sequence: 4}},
		{StreamID{Timestamp: 1000, Sequence: 5}, StreamID{Timestamp: 1000, Sequence: 4}},
		{StreamID{Timestamp: 0, Sequence: 1}, StreamID{Timestamp: 0, Sequence: 0}},
		{StreamID{Timestamp: 0, Sequence: 0}, StreamID{Timestamp: 0, Sequence: 0}}, // Edge case
	}

	for _, test := range tests {
		result := PrevStreamID(test.input)
		if result != test.expected {
			t.Errorf("For input %v, expected %v, got %v", test.input, test.expected, result)
		}
	}
}

// Test MatchPattern
func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern  string
		text     string
		expected bool
	}{
		{"*", "anything", true},
		{"test*", "test123", true},
		{"test*", "testing", true},
		{"test*", "tes", false},
		{"*test", "mytest", true},
		{"*test", "test", true},
		{"*test", "testing", false},
		{"test", "test", true},
		{"test", "Test", false},
		{"", "", true},
		{"", "text", false},
	}

	for _, test := range tests {
		result := MatchPattern(test.pattern, test.text)
		if result != test.expected {
			t.Errorf("For pattern %s and text %s, expected %v, got %v", test.pattern, test.text, test.expected, result)
		}
	}
}

// Test FormatStreamEntries
func TestFormatStreamEntries(t *testing.T) {
	entries := []*StreamEntry{
		{
			ID:     StreamID{Timestamp: 1000, Sequence: 1},
			Fields: map[string]string{"field1": "value1", "field2": "value2"},
		},
		{
			ID:     StreamID{Timestamp: 1000, Sequence: 2},
			Fields: map[string]string{"field3": "value3"},
		},
	}

	result := FormatStreamEntries(entries)
	if len(result) != 2 { // 2 entries
		t.Errorf("Expected 2 elements, got %d", len(result))
	}

	// Check that result contains entry data
	if result == nil {
		t.Error("Expected non-nil result")
	}
}

// Test FormatReadResult
func TestFormatReadResult(t *testing.T) {
	readResult := &ReadResult{
		Streams: []StreamReadResult{
			{
				Name: "stream1",
				Entries: []*StreamEntry{
					{
						ID:     StreamID{Timestamp: 1000, Sequence: 1},
						Fields: map[string]string{"field1": "value1"},
					},
				},
			},
			{
				Name: "stream2",
				Entries: []*StreamEntry{
					{
						ID:     StreamID{Timestamp: 1000, Sequence: 2},
						Fields: map[string]string{"field2": "value2"},
					},
				},
			},
		},
	}

	result := FormatReadResult(readResult)
	if len(result) != 2 { // 2 streams
		t.Errorf("Expected 2 elements, got %d", len(result))
	}
}

// Test CalculateApproximateLength
func TestCalculateApproximateLength(t *testing.T) {
	// Test with sample data
	result := CalculateApproximateLength(10, 100)
	if result <= 0 {
		t.Errorf("Expected positive length, got %d", result)
	}
	
	// Test with zero values
	result = CalculateApproximateLength(0, 0)
	if result != 0 {
		t.Errorf("Expected 0 for zero inputs, got %d", result)
	}
}

// Test ConvertFieldsArray
func TestConvertFieldsArray(t *testing.T) {
	tests := []struct {
		input    []string
		expected map[string]string
		hasError bool
	}{
		{[]string{"key1", "value1", "key2", "value2"}, map[string]string{"key1": "value1", "key2": "value2"}, false},
		{[]string{"key1", "value1"}, map[string]string{"key1": "value1"}, false},
		{[]string{}, map[string]string{}, false},
		{[]string{"key1"}, nil, true}, // Odd number of elements
		{[]string{"key1", "value1", "key2"}, nil, true}, // Odd number of elements
	}

	for _, test := range tests {
		result, err := ConvertFieldsArray(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input %v, but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for input %v: %v", test.input, err)
			}
			if len(result) != len(test.expected) {
				t.Errorf("For input %v, expected %v, got %v", test.input, test.expected, result)
			}
			for k, v := range test.expected {
				if result[k] != v {
					t.Errorf("For input %v, expected %v, got %v", test.input, test.expected, result)
					break
				}
			}
		}
	}
}
