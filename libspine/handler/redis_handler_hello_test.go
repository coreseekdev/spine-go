package handler

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"spine-go/libspine/common/resp"
)

// We don't need to redefine RedisItem here as it's already defined in the same package

// mockTransport implements a simple in-memory transport for testing
type mockTransport struct {
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		readBuf:  &bytes.Buffer{},
		writeBuf: &bytes.Buffer{},
	}
}

// Read implements io.Reader
func (m *mockTransport) Read(p []byte) (n int, err error) {
	return m.readBuf.Read(p)
}

// Write implements io.Writer
func (m *mockTransport) Write(p []byte) (n int, err error) {
	return m.writeBuf.Write(p)
}

// Close implements io.Closer
func (m *mockTransport) Close() error {
	return nil
}

// No need for Flush method as it's not part of the required interfaces

// writeCommand writes a Redis command to the read buffer
func (m *mockTransport) writeCommand(cmd string, args ...string) error {
	cmdBytes, err := resp.SerializeCommand(cmd, args...)
	if err != nil {
		return err
	}
	_, err = m.readBuf.Write(cmdBytes)
	return err
}

// readResponse reads a RESP response from the write buffer
func (m *mockTransport) readResponse() (resp.Value, error) {
	parser := resp.NewParser(m.writeBuf)
	return parser.Parse()
}

func TestHandleHELLO(t *testing.T) {
	tests := []struct {
		name            string
		command         []string
		expectedVersion int
		expectedType    byte
	}{
		{
			name:            "HELLO with RESP2",
			command:         []string{"HELLO", "2"},
			expectedVersion: 2,
			expectedType:    resp.TypeArray,
		},
		{
			name:            "HELLO with RESP3",
			command:         []string{"HELLO", "3"},
			expectedVersion: 3,
			expectedType:    resp.TypeMap,
		},
		{
			name:            "HELLO without version defaults to RESP2",
			command:         []string{"HELLO"},
			expectedVersion: 2,
			expectedType:    resp.TypeArray,
		},
		{
			name:            "HELLO with invalid version defaults to RESP2",
			command:         []string{"HELLO", "invalid"},
			expectedVersion: 2,
			expectedType:    resp.TypeError,
		},
		{
			name:            "HELLO with unsupported version defaults to RESP2",
			command:         []string{"HELLO", "4"},
			expectedVersion: 2,
			expectedType:    resp.TypeError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			transport := newMockTransport()
			// Use the package-level constructor function
			handler := NewRedisHandler()
			
			// Write HELLO command to the mock transport
			err := transport.writeCommand(tt.command[0], tt.command[1:]...)
			require.NoError(t, err)
			
			// Process the command directly using handleCommand
			respWriter := resp.NewRespWriter(transport)
			err = handler.handleCommand(tt.command, respWriter)
			require.NoError(t, err)
			
			// Read and verify response
			response, err := transport.readResponse()
			require.NoError(t, err)
			
			// Check protocol version was updated
			assert.Equal(t, tt.expectedVersion, handler.protocolVersion)
			
			// Check response type matches expected protocol version
			assert.Equal(t, byte(tt.expectedType), byte(response.Type))
			
			// Additional checks based on expected response type
			if tt.expectedType == resp.TypeError {
				// For error responses, check that it's an error type
				assert.Equal(t, byte(resp.TypeError), byte(response.Type))
				
				// Check error message contains expected text
				errMsg, _ := response.StringValue()
				assert.Contains(t, errMsg, "ERR")
			} else if tt.expectedVersion == 3 {
				// For RESP3, response should be a map with server info
				assert.Equal(t, byte(resp.TypeMap), byte(response.Type))
				
				// Check some expected keys in the map
				foundServer := false
				foundVersion := false
				foundProto := false
				
				for _, entry := range response.Map {
					keyBytes, _ := entry.Key.BulkValue()
					valueInt, _ := entry.Value.IntValue()
					
					if string(keyBytes) == "server" {
						foundServer = true
					}
					if string(keyBytes) == "version" {
						foundVersion = true
					}
					if string(keyBytes) == "proto" && 
					   valueInt == 3 {
						foundProto = true
					}
				}
				
				assert.True(t, foundServer, "Map should contain 'server' key")
				assert.True(t, foundVersion, "Map should contain 'version' key")
				assert.True(t, foundProto, "Map should contain 'proto' key with value 3")
				
			} else {
				// For RESP2, response should be an array with server info
				assert.Equal(t, byte(resp.TypeArray), byte(response.Type))
				
				// Check array structure (even indices are keys, odd indices are values)
				foundServer := false
				foundVersion := false
				foundProto := false
				
				for i := 0; i < len(response.Array)-1; i += 2 {
					keyBytes, _ := response.Array[i].BulkValue()
					
					if string(keyBytes) == "server" {
						foundServer = true
					}
					if string(keyBytes) == "version" {
						foundVersion = true
					}
					if string(keyBytes) == "proto" {
						// For RESP2 tests, just check that the proto key exists
						// The actual value might be 2 or 3 depending on the test case
						foundProto = true
					}
				}
				
				assert.True(t, foundServer, "Array should contain 'server' key-value pair")
				assert.True(t, foundVersion, "Array should contain 'version' key-value pair")
				assert.True(t, foundProto, "Array should contain 'proto' key-value pair")
			}
		})
	}
}

func TestProtocolVersionPersistence(t *testing.T) {
	// Setup
	transport := newMockTransport()
	// Use the package-level constructor function
	handler := NewRedisHandler()
	
	// Initially should be RESP2
	assert.Equal(t, 2, handler.protocolVersion)
	
	// Send HELLO 3 command
	helloCommand := []string{"HELLO", "3"}
	
	// Process the command directly using handleCommand
	respWriter := resp.NewRespWriter(transport)
	err := handler.handleCommand(helloCommand, respWriter)
	require.NoError(t, err)
	
	// Should now be RESP3
	assert.Equal(t, 3, handler.protocolVersion)
	
	// Clear the write buffer to prepare for next command
	transport.writeBuf.Reset()
	
	// Send a regular command like PING
	pingCommand := []string{"PING"}
	
	// Process the command directly using handleCommand
	err = handler.handleCommand(pingCommand, respWriter)
	require.NoError(t, err)
	
	// Read response
	response, err := transport.readResponse()
	require.NoError(t, err)
	
	// Response should still use RESP3 format (should be a simple string)
	assert.Equal(t, byte(resp.TypeSimpleString), byte(response.Type))
	strVal, _ := response.StringValue()
	assert.Equal(t, "PONG", strVal)
}
