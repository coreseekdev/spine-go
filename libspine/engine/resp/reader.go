package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"

	"spine-go/libspine/transport"
)

// RESPReader wraps a transport.Reader to provide RESP protocol parsing
type RESPReader struct {
	reader *bufio.Reader
}

// NewRESPReader creates a new RESP reader
func NewRESPReader(reader transport.Reader) *RESPReader {
	return &RESPReader{
		reader: bufio.NewReader(reader),
	}
}

// ReadBulkString reads a bulk string directly (helper method)
func (r *RESPReader) ReadBulkString() (string, error) {
	line, err := r.readLine()
	if err != nil {
		return "", err
	}

	if len(line) == 0 || line[0] != '$' {
		return "", fmt.Errorf("expected bulk string, got %s", line)
	}

	val, err := r.readBulkString(line[1:])
	if err != nil {
		return "", err
	}

	if val.IsNull() {
		return "", nil
	}

	str, ok := val.AsString()
	if !ok {
		return "", fmt.Errorf("expected string, got %T", val.Value)
	}

	return str, nil
}

// ReadValue reads a single RESP value and returns both type and value
func (r *RESPReader) ReadValue() (*RESPValue, error) {
	return r.readValue()
}

// ReadValueLegacy reads a single RESP value (legacy interface for backward compatibility)
func (r *RESPReader) ReadValueLegacy() (interface{}, error) {
	value, err := r.readValue()
	if err != nil {
		return nil, err
	}
	return value.Value, nil
}

// readValue is the internal implementation
func (r *RESPReader) readValue() (*RESPValue, error) {
	line, err := r.readLine()
	if err != nil {
		return nil, err
	}

	if len(line) == 0 {
		return nil, fmt.Errorf("empty line")
	}

	switch line[0] {
	case '+':
		// Simple string
		return NewRESPValue(RESPTypeSimpleString, line[1:]), nil
	case '-':
		// Error
		return NewRESPValue(RESPTypeError, fmt.Errorf("redis error: %s", line[1:])), nil
	case ':':
		// Integer
		val, err := strconv.ParseInt(line[1:], 10, 64)
		if err != nil {
			return nil, err
		}
		return NewRESPValue(RESPTypeInteger, val), nil
	case '$':
		// Bulk string
		return r.readBulkString(line[1:])
	case '*':
		// Array
		return r.readArray(line[1:])
	case '%':
		// Map (RESP3)
		return r.readMap(line[1:])
	case '#':
		// Boolean (RESP3)
		return r.readBoolean(line[1:])
	case '!':
		// Blob error (RESP3)
		return r.readBlobError(line[1:])
	case '=':
		// Verbatim string (RESP3)
		return r.readVerbatimString(line[1:])
	case '(':
		// Big number (RESP3)
		return r.readBigNumber(line[1:])
	default:
		return nil, fmt.Errorf("unknown RESP type: %c", line[0])
	}
}

// readLine reads a line ending with \r\n
func (r *RESPReader) readLine() (string, error) {
	line, err := r.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Remove \r\n
	if len(line) >= 2 && line[len(line)-2] == '\r' {
		return line[:len(line)-2], nil
	}
	return line[:len(line)-1], nil
}

// readBulkString reads a bulk string
func (r *RESPReader) readBulkString(lengthStr string) (*RESPValue, error) {
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid bulk string length: %w", err)
	}

	if length == -1 {
		return NewRESPValue(RESPTypeNull, nil), nil // Null bulk string
	}

	if length == 0 {
		// Empty string, still need to read \r\n
		_, err := r.readLine()
		if err != nil {
			return nil, err
		}
		return NewRESPValue(RESPTypeBulkString, ""), nil
	}

	// Read the string data
	buf := make([]byte, length)
	_, err = io.ReadFull(r.reader, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read bulk string data: %w", err)
	}

	// Read trailing \r\n
	_, err = r.readLine()
	if err != nil {
		return nil, fmt.Errorf("failed to read bulk string trailer: %w", err)
	}

	return NewRESPValue(RESPTypeBulkString, string(buf)), nil
}

// readArray reads an array
func (r *RESPReader) readArray(lengthStr string) (*RESPValue, error) {
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid array length: %w", err)
	}

	if length == -1 {
		return NewRESPValue(RESPTypeNull, nil), nil // Null array
	}

	arr := make([]*RESPValue, length)
	for i := 0; i < length; i++ {
		value, err := r.ReadValue()
		if err != nil {
			return nil, fmt.Errorf("failed to read array element %d: %w", i, err)
		}
		arr[i] = value
	}

	return NewRESPValue(RESPTypeArray, arr), nil
}

// readMap reads a map (RESP3)
func (r *RESPReader) readMap(lengthStr string) (*RESPValue, error) {
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid map length: %w", err)
	}

	if length == -1 {
		return NewRESPValue(RESPTypeNull, nil), nil // Null map
	}

	m := make(map[string]*RESPValue)
	for i := 0; i < length; i++ {
		// Read key
		key, err := r.ReadValue()
		if err != nil {
			return nil, fmt.Errorf("failed to read map key %d: %w", i, err)
		}
		keyStr, ok := key.AsString()
		if !ok {
			return nil, fmt.Errorf("map key must be string, got %T", key.Value)
		}

		// Read value
		value, err := r.ReadValue()
		if err != nil {
			return nil, fmt.Errorf("failed to read map value %d: %w", i, err)
		}

		m[keyStr] = value
	}

	return NewRESPValue(RESPTypeMap, m), nil
}

// readBoolean reads a boolean (RESP3)
func (r *RESPReader) readBoolean(valueStr string) (*RESPValue, error) {
	switch valueStr {
	case "t":
		return NewRESPValue(RESPTypeBoolean, true), nil
	case "f":
		return NewRESPValue(RESPTypeBoolean, false), nil
	default:
		return nil, fmt.Errorf("invalid boolean value: %s", valueStr)
	}
}

// readBlobError reads a blob error (RESP3)
func (r *RESPReader) readBlobError(lengthStr string) (*RESPValue, error) {
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid blob error length: %w", err)
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(r.reader, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob error data: %w", err)
	}

	// Read trailing \r\n
	_, err = r.readLine()
	if err != nil {
		return nil, fmt.Errorf("failed to read blob error trailer: %w", err)
	}

	return NewRESPValue(RESPTypeBulkError, fmt.Errorf("redis blob error: %s", string(buf))), nil
}

// readVerbatimString reads a verbatim string (RESP3)
func (r *RESPReader) readVerbatimString(lengthStr string) (*RESPValue, error) {
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid verbatim string length: %w", err)
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(r.reader, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read verbatim string data: %w", err)
	}

	// Read trailing \r\n
	_, err = r.readLine()
	if err != nil {
		return nil, fmt.Errorf("failed to read verbatim string trailer: %w", err)
	}

	// Verbatim strings have format: <encoding>:<data>
	data := string(buf)
	if len(data) >= 4 && data[3] == ':' {
		return NewRESPValue(RESPTypeVerbatimString, data[4:]), nil // Return data without encoding prefix
	}
	return NewRESPValue(RESPTypeVerbatimString, data), nil
}

// readBigNumber reads a big number (RESP3)
func (r *RESPReader) readBigNumber(valueStr string) (*RESPValue, error) {
	// For simplicity, return as string
	// In production, you might want to use big.Int
	return NewRESPValue(RESPTypeBigNumber, valueStr), nil
}