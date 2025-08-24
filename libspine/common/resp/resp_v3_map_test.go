package resp

import (
	"bytes"
	"testing"
)

func TestSerializeMap(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		expected []byte
		wantErr  bool
	}{
		{
			name: "simple map",
			value: NewMap([]MapItem{
				{Key: NewBulkString([]byte("name")), Value: NewBulkString([]byte("John"))},
				{Key: NewBulkString([]byte("age")), Value: NewInteger(30)},
			}),
			expected: []byte("%2\r\n$4\r\nname\r\n$4\r\nJohn\r\n$3\r\nage\r\n:30\r\n"),
			wantErr:  false,
		},
		{
			name:     "empty map",
			value:    NewMap([]MapItem{}),
			expected: []byte("%0\r\n"),
			wantErr:  false,
		},
		{
			name:     "null map",
			value:    NewMap(nil),
			expected: []byte("%?\r\n"),
			wantErr:  false,
		},
		{
			name: "nested map",
			value: NewMap([]MapItem{
				{
					Key: NewBulkString([]byte("user")),
					Value: NewMap([]MapItem{
						{Key: NewBulkString([]byte("name")), Value: NewBulkString([]byte("John"))},
						{Key: NewBulkString([]byte("active")), Value: NewBoolean(true)},
					}),
				},
				{Key: NewBulkString([]byte("version")), Value: NewInteger(3)},
			}),
			expected: []byte("%2\r\n$4\r\nuser\r\n%2\r\n$4\r\nname\r\n$4\r\nJohn\r\n$6\r\nactive\r\n#t\r\n$7\r\nversion\r\n:3\r\n"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			serializer := NewSerializer(&buf)
			err := serializer.Serialize(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Serialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err := serializer.Flush(); err != nil {
				t.Errorf("Flush() error = %v", err)
				return
			}
			if !bytes.Equal(buf.Bytes(), tt.expected) {
				t.Errorf("Serialize() got = %q, want %q", buf.Bytes(), tt.expected)
			}
		})
	}
}

func TestSerializeSet(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		expected []byte
		wantErr  bool
	}{
		{
			name: "simple set",
			value: NewSet([]Value{
				NewBulkString([]byte("apple")),
				NewBulkString([]byte("banana")),
				NewBulkString([]byte("orange")),
			}),
			expected: []byte("~3\r\n$5\r\napple\r\n$6\r\nbanana\r\n$6\r\norange\r\n"),
			wantErr:  false,
		},
		{
			name:     "empty set",
			value:    NewSet([]Value{}),
			expected: []byte("~0\r\n"),
			wantErr:  false,
		},
		{
			name:     "null set",
			value:    NewSet(nil),
			expected: []byte("~?\r\n"),
			wantErr:  false,
		},
		{
			name: "mixed type set",
			value: NewSet([]Value{
				NewBulkString([]byte("string")),
				NewInteger(42),
				NewBoolean(true),
			}),
			expected: []byte("~3\r\n$6\r\nstring\r\n:42\r\n#t\r\n"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			serializer := NewSerializer(&buf)
			err := serializer.Serialize(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Serialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err := serializer.Flush(); err != nil {
				t.Errorf("Flush() error = %v", err)
				return
			}
			if !bytes.Equal(buf.Bytes(), tt.expected) {
				t.Errorf("Serialize() got = %q, want %q", buf.Bytes(), tt.expected)
			}
		})
	}
}

func TestSerializeBigNumber(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		expected []byte
		wantErr  bool
	}{
		{
			name:     "positive big number",
			value:    NewBigNumber("12345678901234567890"),
			expected: []byte("(12345678901234567890\r\n"),
			wantErr:  false,
		},
		{
			name:     "negative big number",
			value:    NewBigNumber("-12345678901234567890"),
			expected: []byte("(-12345678901234567890\r\n"),
			wantErr:  false,
		},
		{
			name:     "zero big number",
			value:    NewBigNumber("0"),
			expected: []byte("(0\r\n"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			serializer := NewSerializer(&buf)
			err := serializer.Serialize(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Serialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err := serializer.Flush(); err != nil {
				t.Errorf("Flush() error = %v", err)
				return
			}
			if !bytes.Equal(buf.Bytes(), tt.expected) {
				t.Errorf("Serialize() got = %q, want %q", buf.Bytes(), tt.expected)
			}
		})
	}
}
