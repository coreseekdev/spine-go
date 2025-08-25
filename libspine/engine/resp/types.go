package resp

// RESPType represents the type of a RESP value
type RESPType int

const (
	// RESP2 types
	RESPTypeSimpleString RESPType = iota // +
	RESPTypeError                        // -
	RESPTypeInteger                      // :
	RESPTypeBulkString                   // $
	RESPTypeArray                        // *
	
	// RESP3 types
	RESPTypeNull          // _
	RESPTypeBoolean       // #
	RESPTypeDouble        // ,
	RESPTypeBigNumber     // (
	RESPTypeBulkError     // !
	RESPTypeVerbatimString // =
	RESPTypeMap           // %
	RESPTypeSet           // ~
	RESPTypeAttribute     // |
	RESPTypePush          // >
)

// String returns the string representation of the RESP type
func (t RESPType) String() string {
	switch t {
	case RESPTypeSimpleString:
		return "SimpleString"
	case RESPTypeError:
		return "Error"
	case RESPTypeInteger:
		return "Integer"
	case RESPTypeBulkString:
		return "BulkString"
	case RESPTypeArray:
		return "Array"
	case RESPTypeNull:
		return "Null"
	case RESPTypeBoolean:
		return "Boolean"
	case RESPTypeDouble:
		return "Double"
	case RESPTypeBigNumber:
		return "BigNumber"
	case RESPTypeBulkError:
		return "BulkError"
	case RESPTypeVerbatimString:
		return "VerbatimString"
	case RESPTypeMap:
		return "Map"
	case RESPTypeSet:
		return "Set"
	case RESPTypeAttribute:
		return "Attribute"
	case RESPTypePush:
		return "Push"
	default:
		return "Unknown"
	}
}

// RESPValue represents a RESP value with its type
type RESPValue struct {
	Type  RESPType
	Value interface{}
}

// NewRESPValue creates a new RESP value
func NewRESPValue(valueType RESPType, value interface{}) *RESPValue {
	return &RESPValue{
		Type:  valueType,
		Value: value,
	}
}

// AsString returns the value as a string if possible
func (v *RESPValue) AsString() (string, bool) {
	if v.Type == RESPTypeSimpleString || v.Type == RESPTypeBulkString || v.Type == RESPTypeVerbatimString {
		if str, ok := v.Value.(string); ok {
			return str, true
		}
	}
	return "", false
}

// AsInteger returns the value as an integer if possible
func (v *RESPValue) AsInteger() (int64, bool) {
	if v.Type == RESPTypeInteger {
		if i, ok := v.Value.(int64); ok {
			return i, true
		}
	}
	return 0, false
}

// AsArray returns the value as an array if possible
func (v *RESPValue) AsArray() ([]*RESPValue, bool) {
	if v.Type == RESPTypeArray {
		if arr, ok := v.Value.([]*RESPValue); ok {
			return arr, true
		}
	}
	return nil, false
}

// IsNull returns true if the value is null
func (v *RESPValue) IsNull() bool {
	return v.Type == RESPTypeNull || v.Value == nil
}
