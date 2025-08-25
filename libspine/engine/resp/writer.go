package resp

import (
	"fmt"
	"strconv"

	"spine-go/libspine/transport"
)

// RESPWriter wraps a transport.Writer to provide RESP protocol writing
type RESPWriter struct {
	writer transport.Writer
}

// NewRESPWriter creates a new RESP writer
func NewRESPWriter(writer transport.Writer) *RESPWriter {
	return &RESPWriter{
		writer: writer,
	}
}

// WriteSimpleString writes a RESP simple string (+OK\r\n)
func (w *RESPWriter) WriteSimpleString(s string) error {
	response := fmt.Sprintf("+%s\r\n", s)
	_, err := w.writer.Write([]byte(response))
	return err
}

// WriteError writes a RESP error (-ERR message\r\n)
func (w *RESPWriter) WriteError(err string) error {
	response := fmt.Sprintf("-%s\r\n", err)
	_, writeErr := w.writer.Write([]byte(response))
	return writeErr
}

// WriteInteger writes a RESP integer (:123\r\n)
func (w *RESPWriter) WriteInteger(i int64) error {
	response := fmt.Sprintf(":%d\r\n", i)
	_, err := w.writer.Write([]byte(response))
	return err
}

// WriteBulkString writes a RESP bulk string ($5\r\nhello\r\n)
func (w *RESPWriter) WriteBulkString(s string) error {
	response := fmt.Sprintf("$%d\r\n%s\r\n", len(s), s)
	_, err := w.writer.Write([]byte(response))
	return err
}

// WriteNull writes a RESP null bulk string ($-1\r\n)
func (w *RESPWriter) WriteNull() error {
	response := "$-1\r\n"
	_, err := w.writer.Write([]byte(response))
	return err
}

// WriteArray writes a RESP array
func (w *RESPWriter) WriteArray(arr []interface{}) error {
	response := fmt.Sprintf("*%d\r\n", len(arr))
	_, err := w.writer.Write([]byte(response))
	if err != nil {
		return err
	}

	for _, item := range arr {
		err = w.WriteValue(item)
		if err != nil {
			return err
		}
	}

	return nil
}

// WriteStringArray writes an array of strings
func (w *RESPWriter) WriteStringArray(arr []string) error {
	response := fmt.Sprintf("*%d\r\n", len(arr))
	_, err := w.writer.Write([]byte(response))
	if err != nil {
		return err
	}

	for _, item := range arr {
		err = w.WriteBulkString(item)
		if err != nil {
			return err
		}
	}

	return nil
}

// WriteValue writes a value based on its type
func (w *RESPWriter) WriteValue(value interface{}) error {
	switch v := value.(type) {
	case nil:
		return w.WriteNull()
	case string:
		return w.WriteBulkString(v)
	case int:
		return w.WriteInteger(int64(v))
	case int32:
		return w.WriteInteger(int64(v))
	case int64:
		return w.WriteInteger(v)
	case []string:
		return w.WriteStringArray(v)
	case []interface{}:
		return w.WriteArray(v)
	case map[string]interface{}:
		return w.WriteMap(v)
	case error:
		return w.WriteError(v.Error())
	default:
		// Convert to string as fallback
		return w.WriteBulkString(fmt.Sprintf("%v", v))
	}
}

// RESP3 specific methods

// WriteMap writes a RESP3 map (%2\r\n+key1\r\n+value1\r\n+key2\r\n+value2\r\n)
func (w *RESPWriter) WriteMap(m map[string]interface{}) error {
	response := fmt.Sprintf("%%%d\r\n", len(m))
	_, err := w.writer.Write([]byte(response))
	if err != nil {
		return err
	}

	for key, value := range m {
		// Write key as bulk string
		err = w.WriteBulkString(key)
		if err != nil {
			return err
		}

		// Write value
		err = w.WriteValue(value)
		if err != nil {
			return err
		}
	}

	return nil
}

// WriteBoolean writes a RESP3 boolean (#t\r\n or #f\r\n)
func (w *RESPWriter) WriteBoolean(b bool) error {
	var response string
	if b {
		response = "#t\r\n"
	} else {
		response = "#f\r\n"
	}
	_, err := w.writer.Write([]byte(response))
	return err
}

// WriteBlobError writes a RESP3 blob error (!<length>\r\n<error>\r\n)
func (w *RESPWriter) WriteBlobError(err string) error {
	response := fmt.Sprintf("!%d\r\n%s\r\n", len(err), err)
	_, writeErr := w.writer.Write([]byte(response))
	return writeErr
}

// WriteVerbatimString writes a RESP3 verbatim string (=<length>\r\n<encoding>:<data>\r\n)
func (w *RESPWriter) WriteVerbatimString(encoding, data string) error {
	content := fmt.Sprintf("%s:%s", encoding, data)
	response := fmt.Sprintf("=%d\r\n%s\r\n", len(content), content)
	_, err := w.writer.Write([]byte(response))
	return err
}

// WriteBigNumber writes a RESP3 big number ((<number>\r\n)
func (w *RESPWriter) WriteBigNumber(number string) error {
	response := fmt.Sprintf("(%s\r\n", number)
	_, err := w.writer.Write([]byte(response))
	return err
}

// WriteDouble writes a RESP3 double (,<floating-point-number>\r\n)
func (w *RESPWriter) WriteDouble(d float64) error {
	response := fmt.Sprintf(",%s\r\n", strconv.FormatFloat(d, 'g', -1, 64))
	_, err := w.writer.Write([]byte(response))
	return err
}

// WriteSet writes a RESP3 set (~<count>\r\n<element-1>...<element-n>)
func (w *RESPWriter) WriteSet(elements []interface{}) error {
	response := fmt.Sprintf("~%d\r\n", len(elements))
	_, err := w.writer.Write([]byte(response))
	if err != nil {
		return err
	}

	for _, element := range elements {
		err = w.WriteValue(element)
		if err != nil {
			return err
		}
	}

	return nil
}

// WritePush writes a RESP3 push (><count>\r\n<element-1>...<element-n>)
func (w *RESPWriter) WritePush(elements []interface{}) error {
	response := fmt.Sprintf(">%d\r\n", len(elements))
	_, err := w.writer.Write([]byte(response))
	if err != nil {
		return err
	}

	for _, element := range elements {
		err = w.WriteValue(element)
		if err != nil {
			return err
		}
	}

	return nil
}