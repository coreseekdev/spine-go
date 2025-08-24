package resp

import (
	"bytes"
	"testing"
)

func TestSerializeSimpleString(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		expected []byte
		wantErr  bool
	}{
		{
			name:     "simple string",
			value:    NewSimpleString("OK"),
			expected: []byte("+OK\r\n"),
			wantErr:  false,
		},
		{
			name:     "empty simple string",
			value:    NewSimpleString(""),
			expected: []byte("+\r\n"),
			wantErr:  false,
		},
		{
			name:     "simple string with spaces",
			value:    NewSimpleString("Hello World"),
			expected: []byte("+Hello World\r\n"),
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

func TestSerializeError(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		expected []byte
		wantErr  bool
	}{
		{
			name:     "error message",
			value:    NewError("Error message"),
			expected: []byte("-Error message\r\n"),
			wantErr:  false,
		},
		{
			name:     "empty error",
			value:    NewError(""),
			expected: []byte("-\r\n"),
			wantErr:  false,
		},
		{
			name:     "error with format",
			value:    NewError("ERR unknown command 'foobar'"),
			expected: []byte("-ERR unknown command 'foobar'\r\n"),
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

func TestSerializeInteger(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		expected []byte
		wantErr  bool
	}{
		{
			name:     "positive integer",
			value:    NewInteger(1000),
			expected: []byte(":1000\r\n"),
			wantErr:  false,
		},
		{
			name:     "negative integer",
			value:    NewInteger(-1),
			expected: []byte(":-1\r\n"),
			wantErr:  false,
		},
		{
			name:     "zero",
			value:    NewInteger(0),
			expected: []byte(":0\r\n"),
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

func TestSerializeBulkString(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		expected []byte
		wantErr  bool
	}{
		{
			name:     "bulk string",
			value:    NewBulkString([]byte("hello")),
			expected: []byte("$5\r\nhello\r\n"),
			wantErr:  false,
		},
		{
			name:     "empty bulk string",
			value:    NewBulkString([]byte("")),
			expected: []byte("$0\r\n\r\n"),
			wantErr:  false,
		},
		{
			name:     "null bulk string",
			value:    NewBulkString(nil),
			expected: []byte("$-1\r\n"),
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

func TestSerializeArray(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		expected []byte
		wantErr  bool
	}{
		{
			name: "simple array",
			value: NewArray([]Value{
				NewBulkString([]byte("hello")),
				NewBulkString([]byte("world")),
			}),
			expected: []byte("*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n"),
			wantErr:  false,
		},
		{
			name:     "empty array",
			value:    NewArray([]Value{}),
			expected: []byte("*0\r\n"),
			wantErr:  false,
		},
		{
			name:     "null array",
			value:    NewArray(nil),
			expected: []byte("*-1\r\n"),
			wantErr:  false,
		},
		{
			name: "nested array",
			value: NewArray([]Value{
				NewArray([]Value{
					NewSimpleString("hello"),
					NewSimpleString("world"),
				}),
				NewBulkString([]byte("hello")),
			}),
			expected: []byte("*2\r\n*2\r\n+hello\r\n+world\r\n$5\r\nhello\r\n"),
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

func TestSerializeToBytes(t *testing.T) {
	tests := []struct {
		name     string
		value    Value
		expected []byte
		wantErr  bool
	}{
		{
			name:     "simple string",
			value:    NewSimpleString("OK"),
			expected: []byte("+OK\r\n"),
			wantErr:  false,
		},
		{
			name:     "bulk string",
			value:    NewBulkString([]byte("hello")),
			expected: []byte("$5\r\nhello\r\n"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SerializeToBytes(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("SerializeToBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !bytes.Equal(got, tt.expected) {
				t.Errorf("SerializeToBytes() got = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSerializeCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		args     []string
		expected []byte
		wantErr  bool
	}{
		{
			name:     "simple command",
			cmd:      "ECHO",
			args:     []string{"hello"},
			expected: []byte("*2\r\n$4\r\nECHO\r\n$5\r\nhello\r\n"),
			wantErr:  false,
		},
		{
			name:     "command with multiple args",
			cmd:      "SET",
			args:     []string{"key", "value"},
			expected: []byte("*3\r\n$3\r\nSET\r\n$3\r\nkey\r\n$5\r\nvalue\r\n"),
			wantErr:  false,
		},
		{
			name:     "command with no args",
			cmd:      "PING",
			args:     []string{},
			expected: []byte("*1\r\n$4\r\nPING\r\n"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SerializeCommand(tt.cmd, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("SerializeCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !bytes.Equal(got, tt.expected) {
				t.Errorf("SerializeCommand() got = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRespWriter(t *testing.T) {
	tests := []struct {
		name     string
		fn       func(w *RespWriter) error
		expected []byte
		wantErr  bool
	}{
		{
			name: "write OK",
			fn: func(w *RespWriter) error {
				return w.WriteOK()
			},
			expected: []byte("+OK\r\n"),
			wantErr:  false,
		},
		{
			name: "write PONG",
			fn: func(w *RespWriter) error {
				return w.WritePong()
			},
			expected: []byte("+PONG\r\n"),
			wantErr:  false,
		},
		{
			name: "write error string",
			fn: func(w *RespWriter) error {
				return w.WriteErrorString("ERR", "unknown command")
			},
			expected: []byte("-ERR unknown command\r\n"),
			wantErr:  false,
		},
		{
			name: "write command error",
			fn: func(w *RespWriter) error {
				return w.WriteCommandError("unknown command")
			},
			expected: []byte("-ERR unknown command\r\n"),
			wantErr:  false,
		},
		{
			name: "write syntax error",
			fn: func(w *RespWriter) error {
				return w.WriteSyntaxError("invalid syntax")
			},
			expected: []byte("-ERR syntax error invalid syntax\r\n"),
			wantErr:  false,
		},
		{
			name: "write wrong type error",
			fn: func(w *RespWriter) error {
				return w.WriteWrongTypeError()
			},
			expected: []byte("-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"),
			wantErr:  false,
		},
		{
			name: "write wrong number of arguments error",
			fn: func(w *RespWriter) error {
				return w.WriteWrongNumberOfArgumentsError("GET")
			},
			expected: []byte("-ERR wrong number of arguments for GET command\r\n"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writer := NewRespWriter(&testWriter{&buf})
			err := tt.fn(writer)
			if (err != nil) != tt.wantErr {
				t.Errorf("RespWriter function error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !bytes.Equal(buf.Bytes(), tt.expected) {
				t.Errorf("RespWriter function got = %q, want %q", buf.Bytes(), tt.expected)
			}
		})
	}
}

// testWriter implements transport.Writer for testing
type testWriter struct {
	buf *bytes.Buffer
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

func (w *testWriter) Close() error {
	return nil
}
