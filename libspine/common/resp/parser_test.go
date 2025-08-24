package resp

import (
	"bytes"
	"reflect"
	"testing"
)

func TestParseSimpleString(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected Value
		wantErr  bool
	}{
		{
			name:     "simple string",
			input:    []byte("+OK\r\n"),
			expected: NewSimpleString("OK"),
			wantErr:  false,
		},
		{
			name:     "empty simple string",
			input:    []byte("+\r\n"),
			expected: NewSimpleString(""),
			wantErr:  false,
		},
		{
			name:     "simple string with spaces",
			input:    []byte("+Hello World\r\n"),
			expected: NewSimpleString("Hello World"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(bytes.NewReader(tt.input))
			got, err := parser.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Parse() got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected Value
		wantErr  bool
	}{
		{
			name:     "error message",
			input:    []byte("-Error message\r\n"),
			expected: NewError("Error message"),
			wantErr:  false,
		},
		{
			name:     "empty error",
			input:    []byte("-\r\n"),
			expected: NewError(""),
			wantErr:  false,
		},
		{
			name:     "error with format",
			input:    []byte("-ERR unknown command 'foobar'\r\n"),
			expected: NewError("ERR unknown command 'foobar'"),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(bytes.NewReader(tt.input))
			got, err := parser.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Parse() got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseInteger(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected Value
		wantErr  bool
	}{
		{
			name:     "positive integer",
			input:    []byte(":1000\r\n"),
			expected: NewInteger(1000),
			wantErr:  false,
		},
		{
			name:     "negative integer",
			input:    []byte(":-1\r\n"),
			expected: NewInteger(-1),
			wantErr:  false,
		},
		{
			name:     "zero",
			input:    []byte(":0\r\n"),
			expected: NewInteger(0),
			wantErr:  false,
		},
		{
			name:    "invalid integer",
			input:   []byte(":abc\r\n"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(bytes.NewReader(tt.input))
			got, err := parser.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Parse() got = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseBulkString(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected Value
		wantErr  bool
	}{
		{
			name:     "bulk string",
			input:    []byte("$5\r\nhello\r\n"),
			expected: NewBulkString([]byte("hello")),
			wantErr:  false,
		},
		{
			name:     "empty bulk string",
			input:    []byte("$0\r\n\r\n"),
			expected: NewBulkString([]byte("")),
			wantErr:  false,
		},
		{
			name:     "null bulk string",
			input:    []byte("$-1\r\n"),
			expected: NewBulkString(nil),
			wantErr:  false,
		},
		{
			name:    "invalid bulk length",
			input:   []byte("$abc\r\n"),
			wantErr: true,
		},
		{
			name:    "negative bulk length (not -1)",
			input:   []byte("$-2\r\n"),
			wantErr: true,
		},
		{
			name:    "incomplete bulk string",
			input:   []byte("$5\r\nhel"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(bytes.NewReader(tt.input))
			got, err := parser.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.IsNull != tt.expected.IsNull {
					t.Errorf("Parse() got.IsNull = %v, want %v", got.IsNull, tt.expected.IsNull)
				}
				if !got.IsNull && !bytes.Equal(got.Bulk, tt.expected.Bulk) {
					t.Errorf("Parse() got.Bulk = %v, want %v", got.Bulk, tt.expected.Bulk)
				}
			}
		})
	}
}

func TestParseArray(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected Value
		wantErr  bool
	}{
		{
			name:  "simple array",
			input: []byte("*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n"),
			expected: NewArray([]Value{
				NewBulkString([]byte("hello")),
				NewBulkString([]byte("world")),
			}),
			wantErr: false,
		},
		{
			name:     "empty array",
			input:    []byte("*0\r\n"),
			expected: NewArray([]Value{}),
			wantErr:  false,
		},
		{
			name:     "null array",
			input:    []byte("*-1\r\n"),
			expected: NewArray(nil),
			wantErr:  false,
		},
		{
			name: "nested array",
			input: []byte("*2\r\n*2\r\n+hello\r\n+world\r\n$5\r\nhello\r\n"),
			expected: NewArray([]Value{
				NewArray([]Value{
					NewSimpleString("hello"),
					NewSimpleString("world"),
				}),
				NewBulkString([]byte("hello")),
			}),
			wantErr: false,
		},
		{
			name:    "invalid array length",
			input:   []byte("*abc\r\n"),
			wantErr: true,
		},
		{
			name:    "negative array length (not -1)",
			input:   []byte("*-2\r\n"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(bytes.NewReader(tt.input))
			got, err := parser.Parse()
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.IsNull != tt.expected.IsNull {
					t.Errorf("Parse() got.IsNull = %v, want %v", got.IsNull, tt.expected.IsNull)
				}
				if !got.IsNull {
					// Compare array elements
					gotArray, _ := got.ArrayValue()
					expectedArray, _ := tt.expected.ArrayValue()
					if len(gotArray) != len(expectedArray) {
						t.Errorf("Parse() got array length = %v, want %v", len(gotArray), len(expectedArray))
					}
				}
			}
		})
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []Value
		wantErr  bool
	}{
		{
			name:  "simple command",
			input: []byte("*2\r\n$4\r\nECHO\r\n$5\r\nhello\r\n"),
			expected: []Value{
				NewBulkString([]byte("ECHO")),
				NewBulkString([]byte("hello")),
			},
			wantErr: false,
		},
		{
			name:    "not an array",
			input:   []byte("+OK\r\n"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser(bytes.NewReader(tt.input))
			got, err := parser.ParseCommand()
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.expected) {
					t.Errorf("ParseCommand() got length = %v, want %v", len(got), len(tt.expected))
				}
			}
		})
	}
}

func TestParseFromBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected Value
		wantErr  bool
	}{
		{
			name:     "simple string",
			input:    []byte("+OK\r\n"),
			expected: NewSimpleString("OK"),
			wantErr:  false,
		},
		{
			name:    "invalid input",
			input:   []byte("OK\r\n"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFromBytes(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFromBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("ParseFromBytes() got = %v, want %v", got, tt.expected)
			}
		})
	}
}
