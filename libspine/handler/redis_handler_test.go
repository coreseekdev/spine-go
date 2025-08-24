package handler

import (
	"bytes"
	"spine-go/libspine/common/resp"
	"spine-go/libspine/transport"
	"testing"
	"time"
)

// mockReader implements transport.Reader for testing
type mockReader struct {
	buf *bytes.Buffer
}

func (r *mockReader) Read(p []byte) (n int, err error) {
	return r.buf.Read(p)
}

func (r *mockReader) Close() error {
	return nil
}

// mockWriter implements transport.Writer for testing
type mockWriter struct {
	buf *bytes.Buffer
}

func (w *mockWriter) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

func (w *mockWriter) Close() error {
	return nil
}

// mockContext implements transport.Context for testing
type mockContext struct {
	connInfo *transport.ConnInfo
}

func TestRedisHandlerPing(t *testing.T) {
	// Create a new Redis handler
	handler := NewRedisHandler()

	// Create a mock reader with a PING command
	pingCmd, _ := resp.SerializeCommand("PING")
	reader := &mockReader{buf: bytes.NewBuffer(pingCmd)}

	// Create a mock writer to capture the response
	writer := &mockWriter{buf: &bytes.Buffer{}}

	// Create a mock context
	ctx := &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Parse the response
	respReader := resp.NewParser(bytes.NewReader(writer.buf.Bytes()))
	value, err := respReader.Parse()
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}

	// Verify the response is PONG
	if value.Type != resp.TypeSimpleString || value.String != "PONG" {
		t.Errorf("Expected PONG response, got %v", value)
	}
}

func TestRedisHandlerSetGet(t *testing.T) {
	// Create a new Redis handler
	handler := NewRedisHandler()

	// Create a mock reader with a SET command
	setCmd, _ := resp.SerializeCommand("SET", "mykey", "myvalue")
	reader := &mockReader{buf: bytes.NewBuffer(setCmd)}

	// Create a mock writer to capture the response
	writer := &mockWriter{buf: &bytes.Buffer{}}

	// Create a mock context
	ctx := &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the SET command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Parse the response
	respReader := resp.NewParser(bytes.NewReader(writer.buf.Bytes()))
	value, err := respReader.Parse()
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}

	// Verify the response is OK
	if value.Type != resp.TypeSimpleString || value.String != "OK" {
		t.Errorf("Expected OK response, got %v", value)
	}

	// Create a mock reader with a GET command
	getCmd, _ := resp.SerializeCommand("GET", "mykey")
	reader = &mockReader{buf: bytes.NewBuffer(getCmd)}

	// Reset the writer
	writer.buf.Reset()

	// Create a new context
	ctx = &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the GET command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Parse the response
	respReader = resp.NewParser(bytes.NewReader(writer.buf.Bytes()))
	value, err = respReader.Parse()
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}

	// Verify the response is the value we set
	if value.Type != resp.TypeBulkString || string(value.Bulk) != "myvalue" {
		t.Errorf("Expected bulk string 'myvalue', got %v", value)
	}
}

func TestRedisHandlerSetWithExpiry(t *testing.T) {
	// Create a new Redis handler
	handler := NewRedisHandler()

	// Create a mock reader with a SET command with expiry
	setCmd, _ := resp.SerializeCommand("SET", "expkey", "expvalue", "EX", "1")
	reader := &mockReader{buf: bytes.NewBuffer(setCmd)}

	// Create a mock writer to capture the response
	writer := &mockWriter{buf: &bytes.Buffer{}}

	// Create a mock context
	ctx := &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the SET command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Parse the response
	respReader := resp.NewParser(bytes.NewReader(writer.buf.Bytes()))
	value, err := respReader.Parse()
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}

	// Verify the response is OK
	if value.Type != resp.TypeSimpleString || value.String != "OK" {
		t.Errorf("Expected OK response, got %v", value)
	}

	// Create a mock reader with a GET command
	getCmd, _ := resp.SerializeCommand("GET", "expkey")
	reader = &mockReader{buf: bytes.NewBuffer(getCmd)}

	// Reset the writer
	writer.buf.Reset()

	// Create a new context
	ctx = &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the GET command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Parse the response
	respReader = resp.NewParser(bytes.NewReader(writer.buf.Bytes()))
	value, err = respReader.Parse()
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}

	// Verify the response is the value we set
	if value.Type != resp.TypeBulkString || string(value.Bulk) != "expvalue" {
		t.Errorf("Expected bulk string 'expvalue', got %v", value)
	}

	// Wait for the key to expire
	time.Sleep(1100 * time.Millisecond)

	// Create a mock reader with another GET command
	getCmd, _ = resp.SerializeCommand("GET", "expkey")
	reader = &mockReader{buf: bytes.NewBuffer(getCmd)}

	// Reset the writer
	writer.buf.Reset()

	// Create a new context
	ctx = &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the GET command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Parse the response
	respReader = resp.NewParser(bytes.NewReader(writer.buf.Bytes()))
	value, err = respReader.Parse()
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}

	// Verify the response is nil (key expired)
	if value.Type != resp.TypeBulkString || value.Bulk != nil {
		t.Errorf("Expected nil bulk string, got %v", value)
	}
}

func TestRedisHandlerDel(t *testing.T) {
	// Create a new Redis handler
	handler := NewRedisHandler()

	// Create a mock reader with a SET command
	setCmd, _ := resp.SerializeCommand("SET", "delkey", "delvalue")
	reader := &mockReader{buf: bytes.NewBuffer(setCmd)}

	// Create a mock writer to capture the response
	writer := &mockWriter{buf: &bytes.Buffer{}}

	// Create a mock context
	ctx := &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the SET command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Create a mock reader with a DEL command
	delCmd, _ := resp.SerializeCommand("DEL", "delkey")
	reader = &mockReader{buf: bytes.NewBuffer(delCmd)}

	// Reset the writer
	writer.buf.Reset()

	// Create a new context
	ctx = &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the DEL command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Parse the response
	respReader := resp.NewParser(bytes.NewReader(writer.buf.Bytes()))
	value, err := respReader.Parse()
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}

	// Verify the response is 1 (1 key deleted)
	if value.Type != resp.TypeInteger || value.Int != 1 {
		t.Errorf("Expected integer 1, got %v", value)
	}

	// Create a mock reader with a GET command
	getCmd, _ := resp.SerializeCommand("GET", "delkey")
	reader = &mockReader{buf: bytes.NewBuffer(getCmd)}

	// Reset the writer
	writer.buf.Reset()

	// Create a new context
	ctx = &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the GET command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Parse the response
	respReader = resp.NewParser(bytes.NewReader(writer.buf.Bytes()))
	value, err = respReader.Parse()
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}

	// Verify the response is nil (key deleted)
	if value.Type != resp.TypeBulkString || value.Bulk != nil {
		t.Errorf("Expected nil bulk string, got %v", value)
	}
}

func TestRedisHandlerExists(t *testing.T) {
	// Create a new Redis handler
	handler := NewRedisHandler()

	// Create a mock reader with a SET command
	setCmd, _ := resp.SerializeCommand("SET", "existskey", "existsvalue")
	reader := &mockReader{buf: bytes.NewBuffer(setCmd)}

	// Create a mock writer to capture the response
	writer := &mockWriter{buf: &bytes.Buffer{}}

	// Create a mock context
	ctx := &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the SET command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Create a mock reader with an EXISTS command
	existsCmd, _ := resp.SerializeCommand("EXISTS", "existskey")
	reader = &mockReader{buf: bytes.NewBuffer(existsCmd)}

	// Reset the writer
	writer.buf.Reset()

	// Create a new context
	ctx = &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the EXISTS command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Parse the response
	respReader := resp.NewParser(bytes.NewReader(writer.buf.Bytes()))
	value, err := respReader.Parse()
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}

	// Verify the response is 1 (key exists)
	if value.Type != resp.TypeInteger || value.Int != 1 {
		t.Errorf("Expected integer 1, got %v", value)
	}

	// Create a mock reader with an EXISTS command for a non-existent key
	existsCmd, _ = resp.SerializeCommand("EXISTS", "nonexistentkey")
	reader = &mockReader{buf: bytes.NewBuffer(existsCmd)}

	// Reset the writer
	writer.buf.Reset()

	// Create a new context
	ctx = &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the EXISTS command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Parse the response
	respReader = resp.NewParser(bytes.NewReader(writer.buf.Bytes()))
	value, err = respReader.Parse()
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}

	// Verify the response is 0 (key does not exist)
	if value.Type != resp.TypeInteger || value.Int != 0 {
		t.Errorf("Expected integer 0, got %v", value)
	}
}

func TestRedisHandlerTTL(t *testing.T) {
	// Create a new Redis handler
	handler := NewRedisHandler()

	// Create a mock reader with a SET command with expiry
	setCmd, _ := resp.SerializeCommand("SET", "ttlkey", "ttlvalue", "EX", "5")
	reader := &mockReader{buf: bytes.NewBuffer(setCmd)}

	// Create a mock writer to capture the response
	writer := &mockWriter{buf: &bytes.Buffer{}}

	// Create a mock context
	ctx := &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the SET command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Create a mock reader with a TTL command
	ttlCmd, _ := resp.SerializeCommand("TTL", "ttlkey")
	reader = &mockReader{buf: bytes.NewBuffer(ttlCmd)}

	// Reset the writer
	writer.buf.Reset()

	// Create a new context
	ctx = &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the TTL command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Parse the response
	respReader := resp.NewParser(bytes.NewReader(writer.buf.Bytes()))
	value, err := respReader.Parse()
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}

	// Verify the response is a positive integer (TTL in seconds)
	if value.Type != resp.TypeInteger || value.Int <= 0 {
		t.Errorf("Expected positive integer TTL, got %v", value)
	}

	// Create a mock reader with a TTL command for a non-existent key
	ttlCmd, _ = resp.SerializeCommand("TTL", "nonexistentkey")
	reader = &mockReader{buf: bytes.NewBuffer(ttlCmd)}

	// Reset the writer
	writer.buf.Reset()

	// Create a new context
	ctx = &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the TTL command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Parse the response
	respReader = resp.NewParser(bytes.NewReader(writer.buf.Bytes()))
	value, err = respReader.Parse()
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}

	// Verify the response is -2 (key does not exist)
	if value.Type != resp.TypeInteger || value.Int != -2 {
		t.Errorf("Expected integer -2, got %v", value)
	}
}

func TestRedisHandlerUnknownCommand(t *testing.T) {
	// Create a new Redis handler
	handler := NewRedisHandler()

	// Create a mock reader with an unknown command
	unknownCmd, _ := resp.SerializeCommand("UNKNOWN", "arg1", "arg2")
	reader := &mockReader{buf: bytes.NewBuffer(unknownCmd)}

	// Create a mock writer to capture the response
	writer := &mockWriter{buf: &bytes.Buffer{}}

	// Create a mock context
	ctx := &transport.Context{
		ConnInfo: &transport.ConnInfo{
			Reader: reader,
			Writer: writer,
		},
	}

	// Process the command
	go func() {
		err := handler.Handle(ctx, reader, writer)
		if err != nil {
			t.Errorf("Handle() error = %v", err)
		}
	}()

	// Wait for the command to be processed
	time.Sleep(100 * time.Millisecond)

	// Parse the response
	respReader := resp.NewParser(bytes.NewReader(writer.buf.Bytes()))
	value, err := respReader.Parse()
	if err != nil {
		t.Errorf("Parse() error = %v", err)
	}

	// Verify the response is an error
	if value.Type != resp.TypeError {
		t.Errorf("Expected error response, got %v", value)
	}
}
