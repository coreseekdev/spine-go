package resp

import (
	"io"
)

// RespReader adapts a transport.Reader to work with the RESP parser
type RespReader struct {
	reader io.ReadCloser
	parser *Parser
}

// NewRespReader creates a new RESP reader from a transport.Reader
func NewRespReader(r io.ReadCloser) *RespReader {
	return &RespReader{
		reader: r,
		parser: NewParser(r),
	}
}

// ReadValue reads a complete RESP value from the underlying reader
func (r *RespReader) ReadValue() (Value, error) {
	return r.parser.Parse()
}

// ReadCommand reads a RESP array as a Redis command
func (r *RespReader) ReadCommand() ([]Value, error) {
	return r.parser.ParseCommand()
}

// Close closes the underlying reader
func (r *RespReader) Close() error {
	return r.reader.Close()
}

// RespWriter adapts a transport.Writer to work with the RESP serializer
type RespWriter struct {
	writer     io.WriteCloser
	serializer *Serializer
}

// NewRespWriter creates a new RESP writer from a transport.Writer
func NewRespWriter(w io.WriteCloser) *RespWriter {
	return &RespWriter{
		writer:     w,
		serializer: NewSerializer(w),
	}
}

// WriteValue writes a RESP value to the underlying writer
func (w *RespWriter) WriteValue(v Value) error {
	if err := w.serializer.Serialize(v); err != nil {
		return err
	}
	return w.serializer.Flush()
}

// WriteSimpleString writes a simple string response
func (w *RespWriter) WriteSimpleString(s string) error {
	return w.WriteValue(NewSimpleString(s))
}

// WriteError writes an error response
func (w *RespWriter) WriteError(s string) error {
	return w.WriteValue(NewError(s))
}

// WriteInteger writes an integer response
func (w *RespWriter) WriteInteger(n int64) error {
	return w.WriteValue(NewInteger(n))
}

// WriteBulkString writes a bulk string response
func (w *RespWriter) WriteBulkString(b []byte) error {
	return w.WriteValue(NewBulkString(b))
}

// WriteBulkStringString writes a bulk string response from a string
func (w *RespWriter) WriteBulkStringString(s string) error {
	return w.WriteValue(NewBulkStringString(s))
}

// WriteArray writes an array response
func (w *RespWriter) WriteArray(values []Value) error {
	return w.WriteValue(NewArray(values))
}

// WriteNil writes a nil response
func (w *RespWriter) WriteNil() error {
	return w.WriteValue(NewBulkString(nil))
}

// Close closes the underlying writer
func (w *RespWriter) Close() error {
	return w.writer.Close()
}

// Helper functions for common Redis responses

// WriteOK writes the standard Redis OK response
func (w *RespWriter) WriteOK() error {
	return w.WriteSimpleString("OK")
}

// WritePong writes the standard Redis PONG response
func (w *RespWriter) WritePong() error {
	return w.WriteSimpleString("PONG")
}

// WriteErrorString writes a Redis error with the standard format
func (w *RespWriter) WriteErrorString(errType string, message string) error {
	return w.WriteError(errType + " " + message)
}

// WriteCommandError writes a standard Redis command error
func (w *RespWriter) WriteCommandError(message string) error {
	return w.WriteErrorString("ERR", message)
}

// WriteSyntaxError writes a standard Redis syntax error
func (w *RespWriter) WriteSyntaxError(message string) error {
	return w.WriteErrorString("ERR syntax error", message)
}

// WriteWrongTypeError writes a standard Redis wrong type error
func (w *RespWriter) WriteWrongTypeError() error {
	return w.WriteErrorString("WRONGTYPE", "Operation against a key holding the wrong kind of value")
}

// WriteWrongNumberOfArgumentsError writes a standard Redis wrong number of arguments error
func (w *RespWriter) WriteWrongNumberOfArgumentsError(cmd string) error {
	return w.WriteErrorString("ERR wrong number of arguments for", cmd+" command")
}

// RESP v3 writer methods

// WriteNull writes a RESP v3 null value
func (w *RespWriter) WriteNull() error {
	return w.WriteValue(NewNull())
}

// WriteDouble writes a RESP v3 double value
func (w *RespWriter) WriteDouble(d float64) error {
	return w.WriteValue(NewDouble(d))
}

// WriteBoolean writes a RESP v3 boolean value
func (w *RespWriter) WriteBoolean(b bool) error {
	return w.WriteValue(NewBoolean(b))
}

// WriteBlobError writes a RESP v3 blob error
func (w *RespWriter) WriteBlobError(data []byte) error {
	return w.WriteValue(NewBlobError(data))
}

// WriteVerbatimString writes a RESP v3 verbatim string
func (w *RespWriter) WriteVerbatimString(format string, content string) error {
	return w.WriteValue(NewVerbatimString(format, content))
}

// WriteMap writes a RESP v3 map
func (w *RespWriter) WriteMap(items []MapItem) error {
	return w.WriteValue(NewMap(items))
}

// WriteSet writes a RESP v3 set
func (w *RespWriter) WriteSet(values []Value) error {
	return w.WriteValue(NewSet(values))
}

// WriteAttribute writes a RESP v3 attribute
func (w *RespWriter) WriteAttribute(items []MapItem) error {
	return w.WriteValue(NewAttribute(items))
}

// WritePush writes a RESP v3 push
func (w *RespWriter) WritePush(values []Value) error {
	return w.WriteValue(NewPush(values))
}

// WriteBigNumber writes a RESP v3 big number
func (w *RespWriter) WriteBigNumber(num string) error {
	return w.WriteValue(NewBigNumber(num))
}
