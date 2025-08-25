package test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"spine-go/libspine/engine"
	"spine-go/libspine/engine/commands"
	"spine-go/libspine/engine/resp"
	"spine-go/libspine/transport"
)

// mockWriter implements transport.Writer for testing
type mockWriter struct {
	buffer *bytes.Buffer
}

func newMockWriter() *mockWriter {
	return &mockWriter{
		buffer: &bytes.Buffer{},
	}
}

func (w *mockWriter) Write(p []byte) (n int, err error) {
	return w.buffer.Write(p)
}

func (w *mockWriter) Close() error {
	return nil
}

func (w *mockWriter) String() string {
	return w.buffer.String()
}

// mockReader implements transport.Reader for testing
type mockReader struct {
	data   *bytes.Reader
	closed bool
}

func newMockReader(data string) *mockReader {
	return &mockReader{
		data: bytes.NewReader([]byte(data)),
	}
}

func (r *mockReader) Read(p []byte) (n int, err error) {
	if r.closed {
		return 0, nil
	}
	return r.data.Read(p)
}

func (r *mockReader) Close() error {
	r.closed = true
	return nil
}

func TestPubSubBasicFlow(t *testing.T) {
	// Create engine
	engine, err := engine.NewEngine(":memory:")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Register all commands (this is normally done in NewRedisHandler)
	if err := commands.RegisterAllCommands(engine.GetCommandRegistry()); err != nil {
		t.Fatalf("Failed to register commands: %v", err)
	}

	// Create mock connections
	publisher := &transport.ConnInfo{
		ID:       "publisher",
		Metadata: make(map[string]interface{}),
		Writer:   newMockWriter(),
	}

	subscriber := &transport.ConnInfo{
		ID:       "subscriber",
		Metadata: make(map[string]interface{}),
		Writer:   newMockWriter(),
	}

	// Create transport contexts
	pubCtx := &transport.Context{ConnInfo: publisher}
	subCtx := &transport.Context{ConnInfo: subscriber}

	// Test SUBSCRIBE command
	subscribeCmd := "*2\r\n$9\r\nSUBSCRIBE\r\n$7\r\nchannel\r\n"
	subReader := resp.NewReqReader(newMockReader(subscribeCmd))
	subWriter := resp.NewRESPWriter(subscriber.Writer)

	// Parse and execute SUBSCRIBE
	command, _, err := subReader.ParseCommand()
	if err != nil {
		t.Fatalf("Failed to parse SUBSCRIBE command: %v", err)
	}

	cmdHash := hashString(command)
	err = engine.ExecuteCommand(subCtx, cmdHash, command, subReader, subWriter)
	if err != nil {
		t.Fatalf("Failed to execute SUBSCRIBE command: %v", err)
	}

	// Check subscription confirmation
	output := subscriber.Writer.(*mockWriter).String()
	if !strings.Contains(output, ">3\r\n$9\r\nsubscribe\r\n$7\r\nchannel\r\n:1\r\n") {
		t.Errorf("Expected subscription confirmation, got: %s", output)
	}

	// Test PUBLISH command
	publishCmd := "*3\r\n$7\r\nPUBLISH\r\n$7\r\nchannel\r\n$5\r\nhello\r\n"
	pubReader := resp.NewReqReader(newMockReader(publishCmd))
	pubWriter := resp.NewRESPWriter(publisher.Writer)

	// Parse and execute PUBLISH
	command, _, err = pubReader.ParseCommand()
	if err != nil {
		t.Fatalf("Failed to parse PUBLISH command: %v", err)
	}

	cmdHash = hashString(command)
	err = engine.ExecuteCommand(pubCtx, cmdHash, command, pubReader, pubWriter)
	if err != nil {
		t.Fatalf("Failed to execute PUBLISH command: %v", err)
	}

	// Check publish response (should return 1 subscriber)
	pubOutput := publisher.Writer.(*mockWriter).String()
	if !strings.Contains(pubOutput, ":1\r\n") {
		t.Errorf("Expected publish to return 1 subscriber, got: %s", pubOutput)
	}

	// Wait for async message delivery
	time.Sleep(100 * time.Millisecond)

	// Check if subscriber received the message
	subOutput := subscriber.Writer.(*mockWriter).String()
	if !strings.Contains(subOutput, "message") || !strings.Contains(subOutput, "hello") {
		t.Errorf("Expected subscriber to receive message, got: %s", subOutput)
	}
}

func TestPubSubPatternMatching(t *testing.T) {
	// Create engine
	engine, err := engine.NewEngine(":memory:")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Register all commands
	if err := commands.RegisterAllCommands(engine.GetCommandRegistry()); err != nil {
		t.Fatalf("Failed to register commands: %v", err)
	}

	// Create mock subscriber
	subscriber := &transport.ConnInfo{
		ID:       "subscriber",
		Metadata: make(map[string]interface{}),
		Writer:   newMockWriter(),
	}

	subCtx := &transport.Context{ConnInfo: subscriber}

	// Test PSUBSCRIBE command
	psubscribeCmd := "*2\r\n$10\r\nPSUBSCRIBE\r\n$5\r\nnews*\r\n"
	subReader := resp.NewReqReader(newMockReader(psubscribeCmd))
	subWriter := resp.NewRESPWriter(subscriber.Writer)

	// Parse and execute PSUBSCRIBE
	command, _, err := subReader.ParseCommand()
	if err != nil {
		t.Fatalf("Failed to parse PSUBSCRIBE command: %v", err)
	}

	cmdHash := hashString(command)
	err = engine.ExecuteCommand(subCtx, cmdHash, command, subReader, subWriter)
	if err != nil {
		t.Fatalf("Failed to execute PSUBSCRIBE command: %v", err)
	}

	// Check pattern subscription confirmation
	output := subscriber.Writer.(*mockWriter).String()
	if !strings.Contains(output, "psubscribe") || !strings.Contains(output, "news*") {
		t.Errorf("Expected pattern subscription confirmation, got: %s", output)
	}
}

func TestPubSubUnsubscribe(t *testing.T) {
	// Create engine
	engine, err := engine.NewEngine(":memory:")
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer engine.Close()

	// Register all commands
	if err := commands.RegisterAllCommands(engine.GetCommandRegistry()); err != nil {
		t.Fatalf("Failed to register commands: %v", err)
	}

	// Create mock subscriber
	subscriber := &transport.ConnInfo{
		ID:       "subscriber",
		Metadata: make(map[string]interface{}),
		Writer:   newMockWriter(),
	}

	subCtx := &transport.Context{ConnInfo: subscriber}

	// First subscribe
	subscribeCmd := "*2\r\n$9\r\nSUBSCRIBE\r\n$7\r\nchannel\r\n"
	subReader := resp.NewReqReader(newMockReader(subscribeCmd))
	subWriter := resp.NewRESPWriter(subscriber.Writer)

	command, _, err := subReader.ParseCommand()
	if err != nil {
		t.Fatalf("Failed to parse SUBSCRIBE command: %v", err)
	}

	cmdHash := hashString(command)
	err = engine.ExecuteCommand(subCtx, cmdHash, command, subReader, subWriter)
	if err != nil {
		t.Fatalf("Failed to execute SUBSCRIBE command: %v", err)
	}

	// Clear buffer
	subscriber.Writer.(*mockWriter).buffer.Reset()

	// Then unsubscribe
	unsubscribeCmd := "*2\r\n$11\r\nUNSUBSCRIBE\r\n$7\r\nchannel\r\n"
	unsubReader := resp.NewReqReader(newMockReader(unsubscribeCmd))
	unsubWriter := resp.NewRESPWriter(subscriber.Writer)

	command, _, err = unsubReader.ParseCommand()
	if err != nil {
		t.Fatalf("Failed to parse UNSUBSCRIBE command: %v", err)
	}

	cmdHash = hashString(command)
	err = engine.ExecuteCommand(subCtx, cmdHash, command, unsubReader, unsubWriter)
	if err != nil {
		t.Fatalf("Failed to execute UNSUBSCRIBE command: %v", err)
	}

	// Check unsubscribe confirmation
	output := subscriber.Writer.(*mockWriter).String()
	if !strings.Contains(output, "unsubscribe") || !strings.Contains(output, ":0\r\n") {
		t.Errorf("Expected unsubscribe confirmation with 0 subscriptions, got: %s", output)
	}
}

// hashString computes a 32-bit FNV-1a hash of a string (same as in engine)
func hashString(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
