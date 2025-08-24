package resp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"
)

// Parser represents a RESP protocol parser
type Parser struct {
	reader *bufio.Reader
}

// NewParser creates a new RESP parser from an io.Reader
func NewParser(r io.Reader) *Parser {
	return &Parser{
		reader: bufio.NewReader(r),
	}
}

// Parse reads and parses a complete RESP value from the reader
func (p *Parser) Parse() (Value, error) {
	// Read the type byte
	typeByte, err := p.reader.ReadByte()
	if err != nil {
		return Value{}, err
	}

	switch typeByte {
	// RESP v2 types
	case TypeSimpleString:
		return p.parseSimpleString()
	case TypeError:
		return p.parseError()
	case TypeInteger:
		return p.parseInteger()
	case TypeBulkString:
		return p.parseBulkString()
	case TypeArray:
		return p.parseArray()
	
	// RESP v3 types
	case TypeNull:
		return p.parseNull()
	case TypeDouble:
		return p.parseDouble()
	case TypeBoolean:
		return p.parseBoolean()
	case TypeBlobError:
		return p.parseBlobError()
	case TypeVerbatimString:
		return p.parseVerbatimString()
	case TypeMap:
		return p.parseMap()
	case TypeSet:
		return p.parseSet()
	case TypeAttribute:
		return p.parseAttribute()
	case TypePush:
		return p.parsePush()
	case TypeBigNumber:
		return p.parseBigNumber()
	default:
		return Value{}, fmt.Errorf("%w: unexpected type byte '%c'", ErrInvalidSyntax, typeByte)
	}
}

// parseSimpleString parses a RESP simple string
func (p *Parser) parseSimpleString() (Value, error) {
	line, err := p.readLine()
	if err != nil {
		return Value{}, err
	}
	return NewSimpleString(string(line)), nil
}

// parseError parses a RESP error
func (p *Parser) parseError() (Value, error) {
	line, err := p.readLine()
	if err != nil {
		return Value{}, err
	}
	return NewError(string(line)), nil
}

// parseInteger parses a RESP integer
func (p *Parser) parseInteger() (Value, error) {
	line, err := p.readLine()
	if err != nil {
		return Value{}, err
	}
	
	n, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return Value{}, fmt.Errorf("%w: invalid integer - %v", ErrInvalidSyntax, err)
	}
	
	return NewInteger(n), nil
}

// parseBulkString parses a RESP bulk string
func (p *Parser) parseBulkString() (Value, error) {
	line, err := p.readLine()
	if err != nil {
		return Value{}, err
	}
	
	length, err := strconv.Atoi(string(line))
	if err != nil {
		return Value{}, fmt.Errorf("%w: invalid bulk length - %v", ErrInvalidBulkLength, err)
	}
	
	// Handle null bulk string
	if length == -1 {
		return NewBulkString(nil), nil
	}
	
	// Check for valid length
	if length < 0 {
		return Value{}, fmt.Errorf("%w: negative bulk length %d", ErrInvalidBulkLength, length)
	}
	
	// Read the bulk string data
	data := make([]byte, length)
	_, err = io.ReadFull(p.reader, data)
	if err != nil {
		return Value{}, fmt.Errorf("%w: %v", ErrIncompleteMessage, err)
	}
	
	// Read and discard CRLF
	cr, err := p.reader.ReadByte()
	if err != nil {
		return Value{}, fmt.Errorf("%w: missing CRLF after bulk string", ErrIncompleteMessage)
	}
	lf, err := p.reader.ReadByte()
	if err != nil {
		return Value{}, fmt.Errorf("%w: missing CRLF after bulk string", ErrIncompleteMessage)
	}
	if cr != '\r' || lf != '\n' {
		return Value{}, fmt.Errorf("%w: expected CRLF after bulk string", ErrInvalidSyntax)
	}
	
	return NewBulkString(data), nil
}

// parseArray parses a RESP array
func (p *Parser) parseArray() (Value, error) {
	line, err := p.readLine()
	if err != nil {
		return Value{}, err
	}
	
	length, err := strconv.Atoi(string(line))
	if err != nil {
		return Value{}, fmt.Errorf("%w: invalid array length - %v", ErrInvalidArrayLength, err)
	}
	
	// Handle null array
	if length == -1 {
		return NewArray(nil), nil
	}
	
	// Check for valid length
	if length < 0 {
		return Value{}, fmt.Errorf("%w: negative array length %d", ErrInvalidArrayLength, length)
	}
	
	// Parse array elements
	elements := make([]Value, length)
	for i := 0; i < length; i++ {
		val, err := p.Parse()
		if err != nil {
			return Value{}, err
		}
		elements[i] = val
	}
	
	return NewArray(elements), nil
}

// readLine reads a line ending with CRLF and returns the line without the CRLF
func (p *Parser) readLine() ([]byte, error) {
	var line []byte
	for {
		b, err := p.reader.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == '\r' {
			b, err = p.reader.ReadByte()
			if err != nil {
				return nil, err
			}
			if b != '\n' {
				return nil, fmt.Errorf("%w: expected LF after CR", ErrInvalidSyntax)
			}
			return line, nil
		}
		line = append(line, b)
	}
}

// ParseFromBytes parses a RESP value from a byte slice
func ParseFromBytes(data []byte) (Value, error) {
	return NewParser(bytes.NewReader(data)).Parse()
}

// parseNull parses a RESP v3 null value
func (p *Parser) parseNull() (Value, error) {
	// Read and discard CRLF
	cr, err := p.reader.ReadByte()
	if err != nil {
		return Value{}, fmt.Errorf("%w: missing CRLF after null", ErrIncompleteMessage)
	}
	lf, err := p.reader.ReadByte()
	if err != nil {
		return Value{}, fmt.Errorf("%w: missing CRLF after null", ErrIncompleteMessage)
	}
	if cr != '\r' || lf != '\n' {
		return Value{}, fmt.Errorf("%w: expected CRLF after null", ErrInvalidSyntax)
	}
	
	return NewNull(), nil
}

// parseDouble parses a RESP v3 double value
func (p *Parser) parseDouble() (Value, error) {
	line, err := p.readLine()
	if err != nil {
		return Value{}, err
	}
	
	// Handle special values
	switch string(line) {
	case "inf":
		return NewDouble(math.Inf(1)), nil
	case "-inf":
		return NewDouble(math.Inf(-1)), nil
	case "nan", "-nan", "NaN", "-NaN", "NAN", "-NAN":
		return NewDouble(math.NaN()), nil
	}
	
	// Parse regular float
	d, err := strconv.ParseFloat(string(line), 64)
	if err != nil {
		return Value{}, fmt.Errorf("%w: invalid double - %v", ErrInvalidSyntax, err)
	}
	
	return NewDouble(d), nil
}

// parseBoolean parses a RESP v3 boolean value
func (p *Parser) parseBoolean() (Value, error) {
	b, err := p.reader.ReadByte()
	if err != nil {
		return Value{}, err
	}
	
	// Read and discard CRLF
	cr, err := p.reader.ReadByte()
	if err != nil {
		return Value{}, fmt.Errorf("%w: missing CRLF after boolean", ErrIncompleteMessage)
	}
	lf, err := p.reader.ReadByte()
	if err != nil {
		return Value{}, fmt.Errorf("%w: missing CRLF after boolean", ErrIncompleteMessage)
	}
	if cr != '\r' || lf != '\n' {
		return Value{}, fmt.Errorf("%w: expected CRLF after boolean", ErrInvalidSyntax)
	}
	
	switch b {
	case 't':
		return NewBoolean(true), nil
	case 'f':
		return NewBoolean(false), nil
	default:
		return Value{}, fmt.Errorf("%w: invalid boolean value '%c'", ErrInvalidSyntax, b)
	}
}

// parseBlobError parses a RESP v3 blob error
func (p *Parser) parseBlobError() (Value, error) {
	// Blob errors are like bulk strings
	line, err := p.readLine()
	if err != nil {
		return Value{}, err
	}
	
	length, err := strconv.Atoi(string(line))
	if err != nil {
		return Value{}, fmt.Errorf("%w: invalid blob error length - %v", ErrInvalidBulkLength, err)
	}
	
	// Check for valid length
	if length < 0 {
		return Value{}, fmt.Errorf("%w: negative blob error length %d", ErrInvalidBulkLength, length)
	}
	
	// Read the blob error data
	data := make([]byte, length)
	_, err = io.ReadFull(p.reader, data)
	if err != nil {
		return Value{}, fmt.Errorf("%w: %v", ErrIncompleteMessage, err)
	}
	
	// Read and discard CRLF
	cr, err := p.reader.ReadByte()
	if err != nil {
		return Value{}, fmt.Errorf("%w: missing CRLF after blob error", ErrIncompleteMessage)
	}
	lf, err := p.reader.ReadByte()
	if err != nil {
		return Value{}, fmt.Errorf("%w: missing CRLF after blob error", ErrIncompleteMessage)
	}
	if cr != '\r' || lf != '\n' {
		return Value{}, fmt.Errorf("%w: expected CRLF after blob error", ErrInvalidSyntax)
	}
	
	return NewBlobError(data), nil
}

// parseVerbatimString parses a RESP v3 verbatim string
func (p *Parser) parseVerbatimString() (Value, error) {
	// Verbatim strings are like bulk strings
	line, err := p.readLine()
	if err != nil {
		return Value{}, err
	}
	
	length, err := strconv.Atoi(string(line))
	if err != nil {
		return Value{}, fmt.Errorf("%w: invalid verbatim string length - %v", ErrInvalidBulkLength, err)
	}
	
	// Check for valid length
	if length < 4 { // At least 4 bytes for format (3) + colon (1)
		return Value{}, fmt.Errorf("%w: verbatim string length too short %d", ErrInvalidBulkLength, length)
	}
	
	// Read the verbatim string data
	data := make([]byte, length)
	_, err = io.ReadFull(p.reader, data)
	if err != nil {
		return Value{}, fmt.Errorf("%w: %v", ErrIncompleteMessage, err)
	}
	
	// Read and discard CRLF
	cr, err := p.reader.ReadByte()
	if err != nil {
		return Value{}, fmt.Errorf("%w: missing CRLF after verbatim string", ErrIncompleteMessage)
	}
	lf, err := p.reader.ReadByte()
	if err != nil {
		return Value{}, fmt.Errorf("%w: missing CRLF after verbatim string", ErrIncompleteMessage)
	}
	if cr != '\r' || lf != '\n' {
		return Value{}, fmt.Errorf("%w: expected CRLF after verbatim string", ErrInvalidSyntax)
	}
	
	// Extract format and content
	if len(data) < 4 || data[3] != ':' {
		return Value{}, fmt.Errorf("%w: invalid verbatim string format", ErrInvalidFormat)
	}
	
	format := string(data[0:3])
	content := string(data[4:])
	
	return NewVerbatimString(format, content), nil
}

// parseMap parses a RESP v3 map
func (p *Parser) parseMap() (Value, error) {
	line, err := p.readLine()
	if err != nil {
		return Value{}, err
	}
	
	length, err := strconv.Atoi(string(line))
	if err != nil {
		return Value{}, fmt.Errorf("%w: invalid map length - %v", ErrInvalidMapLength, err)
	}
	
	// Handle null map
	if length == -1 {
		return NewMap(nil), nil
	}
	
	// Check for valid length
	if length < 0 {
		return Value{}, fmt.Errorf("%w: negative map length %d", ErrInvalidMapLength, length)
	}
	
	// Parse map elements (key-value pairs)
	items := make([]MapItem, length)
	for i := 0; i < length; i++ {
		// Parse key
		key, err := p.Parse()
		if err != nil {
			return Value{}, err
		}
		
		// Parse value
		val, err := p.Parse()
		if err != nil {
			return Value{}, err
		}
		
		items[i] = MapItem{Key: key, Value: val}
	}
	
	return NewMap(items), nil
}

// parseSet parses a RESP v3 set
func (p *Parser) parseSet() (Value, error) {
	line, err := p.readLine()
	if err != nil {
		return Value{}, err
	}
	
	length, err := strconv.Atoi(string(line))
	if err != nil {
		return Value{}, fmt.Errorf("%w: invalid set length - %v", ErrInvalidSetLength, err)
	}
	
	// Handle null set
	if length == -1 {
		return NewSet(nil), nil
	}
	
	// Check for valid length
	if length < 0 {
		return Value{}, fmt.Errorf("%w: negative set length %d", ErrInvalidSetLength, length)
	}
	
	// Parse set elements
	elements := make([]Value, length)
	for i := 0; i < length; i++ {
		val, err := p.Parse()
		if err != nil {
			return Value{}, err
		}
		elements[i] = val
	}
	
	return NewSet(elements), nil
}

// parseAttribute parses a RESP v3 attribute
func (p *Parser) parseAttribute() (Value, error) {
	line, err := p.readLine()
	if err != nil {
		return Value{}, err
	}
	
	length, err := strconv.Atoi(string(line))
	if err != nil {
		return Value{}, fmt.Errorf("%w: invalid attribute length - %v", ErrInvalidMapLength, err)
	}
	
	// Handle null attribute
	if length == -1 {
		return NewAttribute(nil), nil
	}
	
	// Check for valid length
	if length < 0 {
		return Value{}, fmt.Errorf("%w: negative attribute length %d", ErrInvalidMapLength, length)
	}
	
	// Parse attribute elements (key-value pairs)
	items := make([]MapItem, length)
	for i := 0; i < length; i++ {
		// Parse key
		key, err := p.Parse()
		if err != nil {
			return Value{}, err
		}
		
		// Parse value
		val, err := p.Parse()
		if err != nil {
			return Value{}, err
		}
		
		items[i] = MapItem{Key: key, Value: val}
	}
	
	return NewAttribute(items), nil
}

// parsePush parses a RESP v3 push
func (p *Parser) parsePush() (Value, error) {
	line, err := p.readLine()
	if err != nil {
		return Value{}, err
	}
	
	length, err := strconv.Atoi(string(line))
	if err != nil {
		return Value{}, fmt.Errorf("%w: invalid push length - %v", ErrInvalidArrayLength, err)
	}
	
	// Handle null push
	if length == -1 {
		return NewPush(nil), nil
	}
	
	// Check for valid length
	if length < 0 {
		return Value{}, fmt.Errorf("%w: negative push length %d", ErrInvalidArrayLength, length)
	}
	
	// Parse push elements
	elements := make([]Value, length)
	for i := 0; i < length; i++ {
		val, err := p.Parse()
		if err != nil {
			return Value{}, err
		}
		elements[i] = val
	}
	
	return NewPush(elements), nil
}

// parseBigNumber parses a RESP v3 big number
func (p *Parser) parseBigNumber() (Value, error) {
	line, err := p.readLine()
	if err != nil {
		return Value{}, err
	}
	
	return NewBigNumber(string(line)), nil
}

// ParseCommand parses a RESP array as a Redis command
func (p *Parser) ParseCommand() ([]Value, error) {
	val, err := p.Parse()
	if err != nil {
		return nil, err
	}
	
	// Commands must be arrays
	if val.Type != DataType(TypeArray) {
		return nil, fmt.Errorf("%w: expected array for command", ErrUnexpectedType)
	}
	
	// Get the array elements
	elements, err := val.ArrayValue()
	if err != nil {
		return nil, err
	}
	
	return elements, nil
}
