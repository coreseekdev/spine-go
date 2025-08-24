// Package resp implements the Redis Serialization Protocol (RESP)
// for use with the spine-go Redis handler.
package resp

import (
	"errors"
)

// RESP data types
const (
	// RESP v2 types
	TypeSimpleString = '+'
	TypeError        = '-'
	TypeInteger      = ':'
	TypeBulkString   = '$'
	TypeArray        = '*'
	
	// RESP v3 types
	TypeNull         = '_'
	TypeDouble       = ','
	TypeBoolean      = '#'
	TypeBlobError    = '!'
	TypeVerbatimString = '='
	TypeMap          = '%'
	TypeSet          = '~'
	TypeAttribute    = '|'
	TypePush         = '>'
	TypeBigNumber    = '('
)

// Common errors
var (
	ErrInvalidSyntax       = errors.New("resp: invalid syntax")
	ErrUnexpectedType      = errors.New("resp: unexpected type")
	ErrIncompleteMessage   = errors.New("resp: incomplete message")
	ErrInvalidBulkLength   = errors.New("resp: invalid bulk length")
	ErrInvalidArrayLength  = errors.New("resp: invalid array length")
	ErrInvalidMapLength    = errors.New("resp: invalid map length")
	ErrInvalidSetLength    = errors.New("resp: invalid set length")
	ErrInvalidFormat       = errors.New("resp: invalid format")
	ErrNil                 = errors.New("resp: nil value")
)

// DataType represents the type of a RESP value
type DataType byte

// MapItem represents a key-value pair in a RESP Map or Attribute
type MapItem struct {
	Key   Value
	Value Value
}

// Value represents a RESP protocol value
type Value struct {
	Type     DataType
	String   string    // Used for SimpleString, Error, VerbatimString
	Int      int64     // Used for Integer
	Bulk     []byte    // Used for BulkString, BlobError
	Array    []Value   // Used for Array, Set, Push
	Map      []MapItem // Used for Map, Attribute
	Double   float64   // Used for Double
	Bool     bool      // Used for Boolean
	BigNum   string    // Used for BigNumber
	IsNull   bool      // Used to indicate null values
	// For VerbatimString format (txt, mkd, etc)
	Format   string    
}

// NewSimpleString creates a new simple string value
func NewSimpleString(s string) Value {
	return Value{
		Type:   DataType(TypeSimpleString),
		String: s,
	}
}

// NewError creates a new error value
func NewError(s string) Value {
	return Value{
		Type:   DataType(TypeError),
		String: s,
	}
}

// NewInteger creates a new integer value
func NewInteger(i int64) Value {
	return Value{
		Type: DataType(TypeInteger),
		Int:  i,
	}
}

// NewBulkString creates a new bulk string value
func NewBulkString(b []byte) Value {
	if b == nil {
		return Value{
			Type:   DataType(TypeBulkString),
			IsNull: true,
		}
	}
	return Value{
		Type: DataType(TypeBulkString),
		Bulk: b,
	}
}

// NewBulkStringString creates a new bulk string value from a string
func NewBulkStringString(s string) Value {
	return NewBulkString([]byte(s))
}

// NewArray creates a new array value
func NewArray(values []Value) Value {
	if values == nil {
		return Value{
			Type:   DataType(TypeArray),
			IsNull: true,
		}
	}
	return Value{
		Type:  DataType(TypeArray),
		Array: values,
	}
}

// NewNull creates a new null value (RESP v3)
func NewNull() Value {
	return Value{
		Type:   DataType(TypeNull),
		IsNull: true,
	}
}

// NewDouble creates a new double value (RESP v3)
func NewDouble(d float64) Value {
	return Value{
		Type:   DataType(TypeDouble),
		Double: d,
	}
}

// NewBoolean creates a new boolean value (RESP v3)
func NewBoolean(b bool) Value {
	return Value{
		Type: DataType(TypeBoolean),
		Bool: b,
	}
}

// NewBlobError creates a new blob error value (RESP v3)
func NewBlobError(b []byte) Value {
	return Value{
		Type: DataType(TypeBlobError),
		Bulk: b,
	}
}

// NewBlobErrorString creates a new blob error value from a string (RESP v3)
func NewBlobErrorString(s string) Value {
	return NewBlobError([]byte(s))
}

// NewVerbatimString creates a new verbatim string value (RESP v3)
// format should be "txt" or "mkd"
func NewVerbatimString(format string, s string) Value {
	return Value{
		Type:   DataType(TypeVerbatimString),
		String: s,
		Format: format,
	}
}

// NewMap creates a new map value (RESP v3)
func NewMap(items []MapItem) Value {
	if items == nil {
		return Value{
			Type:   DataType(TypeMap),
			IsNull: true,
		}
	}
	return Value{
		Type: DataType(TypeMap),
		Map:  items,
	}
}

// NewSet creates a new set value (RESP v3)
func NewSet(values []Value) Value {
	if values == nil {
		return Value{
			Type:   DataType(TypeSet),
			IsNull: true,
		}
	}
	return Value{
		Type:  DataType(TypeSet),
		Array: values,
	}
}

// NewAttribute creates a new attribute value (RESP v3)
func NewAttribute(items []MapItem) Value {
	if items == nil {
		return Value{
			Type:   DataType(TypeAttribute),
			IsNull: true,
		}
	}
	return Value{
		Type: DataType(TypeAttribute),
		Map:  items,
	}
}

// NewPush creates a new push value (RESP v3)
func NewPush(values []Value) Value {
	if values == nil {
		return Value{
			Type:   DataType(TypePush),
			IsNull: true,
		}
	}
	return Value{
		Type:  DataType(TypePush),
		Array: values,
	}
}

// NewBigNumber creates a new big number value (RESP v3)
func NewBigNumber(s string) Value {
	return Value{
		Type:   DataType(TypeBigNumber),
		BigNum: s,
	}
}

// IsNil returns true if the value is nil
func (v Value) IsNil() bool {
	return v.IsNull
}

// StringValue returns the string value
func (v Value) StringValue() (string, error) {
	switch v.Type {
	case DataType(TypeSimpleString), DataType(TypeError):
		return v.String, nil
	case DataType(TypeBulkString):
		if v.IsNull {
			return "", ErrNil
		}
		return string(v.Bulk), nil
	case DataType(TypeVerbatimString):
		return v.String, nil
	case DataType(TypeBlobError):
		return string(v.Bulk), nil
	default:
		return "", ErrUnexpectedType
	}
}

// IntValue returns the integer value
func (v Value) IntValue() (int64, error) {
	if v.Type == DataType(TypeInteger) {
		return v.Int, nil
	}
	return 0, ErrUnexpectedType
}

// DoubleValue returns the double value (RESP v3)
func (v Value) DoubleValue() (float64, error) {
	if v.Type == DataType(TypeDouble) {
		return v.Double, nil
	}
	return 0, ErrUnexpectedType
}

// BoolValue returns the boolean value (RESP v3)
func (v Value) BoolValue() (bool, error) {
	if v.Type == DataType(TypeBoolean) {
		return v.Bool, nil
	}
	return false, ErrUnexpectedType
}

// MapValue returns the map value (RESP v3)
func (v Value) MapValue() ([]MapItem, error) {
	if v.Type == DataType(TypeMap) || v.Type == DataType(TypeAttribute) {
		if v.IsNull {
			return nil, ErrNil
		}
		return v.Map, nil
	}
	return nil, ErrUnexpectedType
}

// SetValue returns the set value (RESP v3)
func (v Value) SetValue() ([]Value, error) {
	if v.Type == DataType(TypeSet) {
		if v.IsNull {
			return nil, ErrNil
		}
		return v.Array, nil
	}
	return nil, ErrUnexpectedType
}

// PushValue returns the push value (RESP v3)
func (v Value) PushValue() ([]Value, error) {
	if v.Type == DataType(TypePush) {
		if v.IsNull {
			return nil, ErrNil
		}
		return v.Array, nil
	}
	return nil, ErrUnexpectedType
}

// BigNumberValue returns the big number value (RESP v3)
func (v Value) BigNumberValue() (string, error) {
	if v.Type == DataType(TypeBigNumber) {
		return v.BigNum, nil
	}
	return "", ErrUnexpectedType
}

// VerbatimStringValue returns the verbatim string value and format (RESP v3)
func (v Value) VerbatimStringValue() (format string, content string, err error) {
	if v.Type == DataType(TypeVerbatimString) {
		return v.Format, v.String, nil
	}
	return "", "", ErrUnexpectedType
}

// ArrayValue returns the array value
func (v Value) ArrayValue() ([]Value, error) {
	if v.Type == DataType(TypeArray) {
		if v.IsNull {
			return nil, ErrNil
		}
		return v.Array, nil
	}
	return nil, ErrUnexpectedType
}

// BulkValue returns the bulk string value
func (v Value) BulkValue() ([]byte, error) {
	if v.Type == DataType(TypeBulkString) {
		if v.IsNull {
			return nil, ErrNil
		}
		return v.Bulk, nil
	}
	return nil, ErrUnexpectedType
}
