package resp

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"strconv"
)

// Serializer represents a RESP protocol serializer
type Serializer struct {
	writer *bufio.Writer
}

// NewSerializer creates a new RESP serializer from an io.Writer
func NewSerializer(w io.Writer) *Serializer {
	return &Serializer{
		writer: bufio.NewWriter(w),
	}
}

// Serialize writes a RESP value to the writer
func (s *Serializer) Serialize(v Value) error {
	switch v.Type {
	// RESP v2 types
	case DataType(TypeSimpleString):
		return s.writeSimpleString(v.String)
	case DataType(TypeError):
		return s.writeError(v.String)
	case DataType(TypeInteger):
		return s.writeInteger(v.Int)
	case DataType(TypeBulkString):
		if v.IsNull {
			return s.writeNullBulkString()
		}
		return s.writeBulkString(v.Bulk)
	case DataType(TypeArray):
		if v.IsNull {
			return s.writeNullArray()
		}
		return s.writeArray(v.Array)
	
	// RESP v3 types
	case DataType(TypeNull):
		return s.writeNull()
	case DataType(TypeDouble):
		return s.writeDouble(v.Double)
	case DataType(TypeBoolean):
		return s.writeBoolean(v.Bool)
	case DataType(TypeBlobError):
		return s.writeBlobError(v.Bulk)
	case DataType(TypeVerbatimString):
		return s.writeVerbatimString(v.Format, v.String)
	case DataType(TypeMap):
		if v.IsNull {
			return s.writeNullMap()
		}
		return s.writeMap(v.Map)
	case DataType(TypeSet):
		if v.IsNull {
			return s.writeNullSet()
		}
		return s.writeSet(v.Array)
	case DataType(TypeAttribute):
		if v.IsNull {
			return s.writeNullAttribute()
		}
		return s.writeAttribute(v.Map)
	case DataType(TypePush):
		if v.IsNull {
			return s.writeNullPush()
		}
		return s.writePush(v.Array)
	case DataType(TypeBigNumber):
		return s.writeBigNumber(v.BigNum)
	default:
		return fmt.Errorf("%w: unknown type %v", ErrUnexpectedType, v.Type)
	}
}

// Flush flushes any buffered data to the underlying writer
func (s *Serializer) Flush() error {
	return s.writer.Flush()
}

// writeSimpleString writes a RESP simple string
func (s *Serializer) writeSimpleString(str string) error {
	if _, err := s.writer.Write([]byte{TypeSimpleString}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString(str); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writeError writes a RESP error
func (s *Serializer) writeError(str string) error {
	if _, err := s.writer.Write([]byte{TypeError}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString(str); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writeInteger writes a RESP integer
func (s *Serializer) writeInteger(n int64) error {
	if _, err := s.writer.Write([]byte{TypeInteger}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString(strconv.FormatInt(n, 10)); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writeNullBulkString writes a RESP null bulk string
func (s *Serializer) writeNullBulkString() error {
	if _, err := s.writer.Write([]byte{TypeBulkString}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString("-1"); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writeBulkString writes a RESP bulk string
func (s *Serializer) writeBulkString(data []byte) error {
	if _, err := s.writer.Write([]byte{TypeBulkString}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString(strconv.Itoa(len(data))); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	if _, err := s.writer.Write(data); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writeNullArray writes a RESP null array
func (s *Serializer) writeNullArray() error {
	if _, err := s.writer.Write([]byte{TypeArray}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString("-1"); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writeArray writes a RESP array
func (s *Serializer) writeArray(array []Value) error {
	if _, err := s.writer.Write([]byte{TypeArray}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString(strconv.Itoa(len(array))); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	
	for _, v := range array {
		if err := s.Serialize(v); err != nil {
			return err
		}
	}
	
	return nil
}

// SerializeToBytes serializes a RESP value to a byte slice
func SerializeToBytes(v Value) ([]byte, error) {
	var buf io.Writer = &bytesWriter{bytes: make([]byte, 0, 64)}
	s := NewSerializer(buf)
	if err := s.Serialize(v); err != nil {
		return nil, err
	}
	if err := s.Flush(); err != nil {
		return nil, err
	}
	return buf.(*bytesWriter).bytes, nil
}

// bytesWriter is a simple io.Writer that writes to a byte slice
type bytesWriter struct {
	bytes []byte
}

func (w *bytesWriter) Write(p []byte) (int, error) {
	w.bytes = append(w.bytes, p...)
	return len(p), nil
}

// Helper functions for common serialization tasks

// RESP v3 serialization methods

// writeNull writes a RESP v3 null value
func (s *Serializer) writeNull() error {
	if _, err := s.writer.Write([]byte{TypeNull, '\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writeDouble writes a RESP v3 double value
func (s *Serializer) writeDouble(d float64) error {
	if _, err := s.writer.Write([]byte{TypeDouble}); err != nil {
		return err
	}
	
	// Handle special values
	var str string
	switch {
	case math.IsInf(d, 1):
		str = "inf"
	case math.IsInf(d, -1):
		str = "-inf"
	case math.IsNaN(d):
		str = "nan"
	default:
		str = strconv.FormatFloat(d, 'f', -1, 64)
	}
	
	if _, err := s.writer.WriteString(str); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writeBoolean writes a RESP v3 boolean value
func (s *Serializer) writeBoolean(b bool) error {
	if _, err := s.writer.Write([]byte{TypeBoolean}); err != nil {
		return err
	}
	
	var val byte
	if b {
		val = 't'
	} else {
		val = 'f'
	}
	
	if _, err := s.writer.Write([]byte{val, '\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writeBlobError writes a RESP v3 blob error
func (s *Serializer) writeBlobError(data []byte) error {
	if _, err := s.writer.Write([]byte{TypeBlobError}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString(strconv.Itoa(len(data))); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	if _, err := s.writer.Write(data); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writeVerbatimString writes a RESP v3 verbatim string
func (s *Serializer) writeVerbatimString(format string, content string) error {
	if _, err := s.writer.Write([]byte{TypeVerbatimString}); err != nil {
		return err
	}
	
	// Format the verbatim string as format:content
	data := format + ":" + content
	if _, err := s.writer.WriteString(strconv.Itoa(len(data))); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString(data); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writeNullMap writes a RESP v3 null map
func (s *Serializer) writeNullMap() error {
	if _, err := s.writer.Write([]byte{TypeMap}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString("?"); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writeMap writes a RESP v3 map
func (s *Serializer) writeMap(items []MapItem) error {
	if _, err := s.writer.Write([]byte{TypeMap}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString(strconv.Itoa(len(items))); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	
	for _, item := range items {
		if err := s.Serialize(item.Key); err != nil {
			return err
		}
		if err := s.Serialize(item.Value); err != nil {
			return err
		}
	}
	
	return nil
}

// writeNullSet writes a RESP v3 null set
func (s *Serializer) writeNullSet() error {
	if _, err := s.writer.Write([]byte{TypeSet}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString("?"); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writeSet writes a RESP v3 set
func (s *Serializer) writeSet(array []Value) error {
	if _, err := s.writer.Write([]byte{TypeSet}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString(strconv.Itoa(len(array))); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	
	for _, v := range array {
		if err := s.Serialize(v); err != nil {
			return err
		}
	}
	
	return nil
}

// writeNullAttribute writes a RESP v3 null attribute
func (s *Serializer) writeNullAttribute() error {
	if _, err := s.writer.Write([]byte{TypeAttribute}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString("?"); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writeAttribute writes a RESP v3 attribute
func (s *Serializer) writeAttribute(items []MapItem) error {
	if _, err := s.writer.Write([]byte{TypeAttribute}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString(strconv.Itoa(len(items))); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	
	for _, item := range items {
		if err := s.Serialize(item.Key); err != nil {
			return err
		}
		if err := s.Serialize(item.Value); err != nil {
			return err
		}
	}
	
	return nil
}

// writeNullPush writes a RESP v3 null push
func (s *Serializer) writeNullPush() error {
	if _, err := s.writer.Write([]byte{TypePush}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString("?"); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// writePush writes a RESP v3 push
func (s *Serializer) writePush(array []Value) error {
	if _, err := s.writer.Write([]byte{TypePush}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString(strconv.Itoa(len(array))); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	
	for _, v := range array {
		if err := s.Serialize(v); err != nil {
			return err
		}
	}
	
	return nil
}

// writeBigNumber writes a RESP v3 big number
func (s *Serializer) writeBigNumber(num string) error {
	if _, err := s.writer.Write([]byte{TypeBigNumber}); err != nil {
		return err
	}
	if _, err := s.writer.WriteString(num); err != nil {
		return err
	}
	if _, err := s.writer.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

// SerializeCommand serializes a Redis command as a RESP array
func SerializeCommand(cmd string, args ...string) ([]byte, error) {
	// Create array with command and args
	values := make([]Value, 1+len(args))
	values[0] = NewBulkStringString(cmd)
	
	for i, arg := range args {
		values[i+1] = NewBulkStringString(arg)
	}
	
	// Serialize as array
	return SerializeToBytes(NewArray(values))
}
