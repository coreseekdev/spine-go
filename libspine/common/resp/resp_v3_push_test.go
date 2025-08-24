package resp

import (
	"bytes"
	"testing"
)

func TestSerializePush(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		expected []byte
		wantErr  bool
	}{
		{
			name: "simple push with array",
			value: NewPush([]Value{
				NewBulkString([]byte("message")),
				NewBulkString([]byte("Hello world")),
			}),
			expected: []byte(">2\r\n$7\r\nmessage\r\n$11\r\nHello world\r\n"),
			wantErr:  false,
		},
		{
			name:     "empty push",
			value:    NewPush([]Value{}),
			expected: []byte(">0\r\n"),
			wantErr:  false,
		},
		{
			name:     "null push",
			value:    NewPush(nil),
			expected: []byte(">?\r\n"),
			wantErr:  false,
		},
		{
			name: "complex push with mixed types",
			value: NewPush([]Value{
				NewBulkString([]byte("notification")),
				NewMap([]MapItem{
					{Key: NewBulkString([]byte("type")), Value: NewBulkString([]byte("message"))},
					{Key: NewBulkString([]byte("data")), Value: NewBulkString([]byte("New message received"))},
					{Key: NewBulkString([]byte("urgent")), Value: NewBoolean(true)},
				}),
			}),
			expected: []byte(">2\r\n$12\r\nnotification\r\n%3\r\n$4\r\ntype\r\n$7\r\nmessage\r\n$4\r\ndata\r\n$20\r\nNew message received\r\n$6\r\nurgent\r\n#t\r\n"),
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

func TestSerializeAttribute(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		expected []byte
		wantErr  bool
	}{
		{
			name: "simple attribute",
			value: NewAttribute([]MapItem{
				{Key: NewBulkString([]byte("key-1")), Value: NewBulkString([]byte("value-1"))},
				{Key: NewBulkString([]byte("key-2")), Value: NewBulkString([]byte("value-2"))},
			}),
			expected: []byte("|2\r\n$5\r\nkey-1\r\n$7\r\nvalue-1\r\n$5\r\nkey-2\r\n$7\r\nvalue-2\r\n"),
			wantErr:  false,
		},
		{
			name:     "attribute with empty annotations",
			value:    NewAttribute([]MapItem{}),
			expected: []byte("|0\r\n"),
			wantErr:  false,
		},
		{
			name:     "null attribute",
			value:    NewAttribute(nil),
			expected: []byte("|?\r\n"),
			wantErr:  false,
		},
		{
			name: "complex attribute with map value",
			value: NewAttribute([]MapItem{
				{Key: NewBulkString([]byte("server")), Value: NewBulkString([]byte("redis"))},
				{Key: NewBulkString([]byte("version")), Value: NewBulkString([]byte("7.0.0"))},
				{Key: NewBulkString([]byte("status")), Value: NewBulkString([]byte("success"))},
				{Key: NewBulkString([]byte("code")), Value: NewInteger(200)},
			}),
			expected: []byte("|4\r\n$6\r\nserver\r\n$5\r\nredis\r\n$7\r\nversion\r\n$5\r\n7.0.0\r\n$6\r\nstatus\r\n$7\r\nsuccess\r\n$4\r\ncode\r\n:200\r\n"),
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
