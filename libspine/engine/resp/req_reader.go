package resp

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"io"
	"strconv"
	"strings"

	"spine-go/libspine/transport"
)

// ReqReader wraps a transport.Reader to provide RESP protocol parsing for Redis requests
type ReqReader struct {
	reader      *bufio.Reader
	command     string
	nargs       int
	hash        uint32 // Will be non-zero if command has been parsed
	parseErr    error  // Store any error that occurred during parsing
	argsRead    int    // Counter for parsed arguments
}

// NewReqReader creates a new RESP request reader
func NewReqReader(reader transport.Reader) *ReqReader {
	return &ReqReader{
		reader:   bufio.NewReader(reader),
		hash:     0,
		parseErr: nil,
		argsRead: 0,
	}
}

// Command returns the command name (lazy loaded)
func (r *ReqReader) Command() (string, error) {
	if err := r.parseCommandAndArgs(); err != nil {
		return "", err
	}
	return r.command, nil
}

// NArgs returns the number of arguments (lazy loaded)
func (r *ReqReader) NArgs() (int, error) {
	if err := r.parseCommandAndArgs(); err != nil {
		return 0, err
	}
	return r.nargs, nil
}

// Hash returns the hash of the command (lazy loaded)
func (r *ReqReader) Hash() uint32 {
	if err := r.parseCommandAndArgs(); err != nil {
		return 0
	}
	return r.hash
}

// ParseCommand parses the command and returns the command name and number of arguments
func (r *ReqReader) ParseCommand() (string, int, error) {
	if err := r.parseCommandAndArgs(); err != nil {
		return "", 0, err
	}
	return r.command, r.nargs, nil
}

// parseCommandAndArgs parses both the command and argument count in a single operation
func (r *ReqReader) parseCommandAndArgs() error {
	// If already parsed or had an error, return immediately
	if r.parseErr != nil || r.hash != 0 {
		return r.parseErr
	}

	// Read the first line to determine if it's an array
	line, err := r.readLine()
	if err != nil {
		r.parseErr = err
		return err
	}

	if len(line) == 0 {
		r.parseErr = fmt.Errorf("empty line")
		return r.parseErr
	}

	// Command should be an array
	if line[0] != '*' {
		r.parseErr = fmt.Errorf("expected array for command, got %c", line[0])
		return r.parseErr
	}

	// Parse array length
	length, err := strconv.Atoi(line[1:])
	if err != nil {
		r.parseErr = fmt.Errorf("invalid array length: %w", err)
		return r.parseErr
	}

	if length == 0 {
		r.parseErr = fmt.Errorf("empty command array")
		return r.parseErr
	}

	// Store number of arguments (total elements - command)
	r.nargs = length - 1

	// Read the command (first element of the array)
	cmdLine, err := r.readLine()
	if err != nil {
		r.parseErr = err
		return err
	}

	if len(cmdLine) < 1 || cmdLine[0] != '$' {
		r.parseErr = fmt.Errorf("expected bulk string for command")
		return r.parseErr
	}

	cmdLength, err := strconv.Atoi(cmdLine[1:])
	if err != nil {
		r.parseErr = fmt.Errorf("invalid command length: %w", err)
		return r.parseErr
	}

	if cmdLength < 0 {
		r.parseErr = fmt.Errorf("null command")
		return r.parseErr
	}

	// Read the command data
	cmdBuf := make([]byte, cmdLength)
	_, err = io.ReadFull(r.reader, cmdBuf)
	if err != nil {
		r.parseErr = fmt.Errorf("failed to read command data: %w", err)
		return r.parseErr
	}

	// Read trailing \r\n
	_, err = r.readLine()
	if err != nil {
		r.parseErr = fmt.Errorf("failed to read command trailer: %w", err)
		return r.parseErr
	}

	// Store command info
	r.command = strings.ToUpper(string(cmdBuf))

	// Calculate hash
	h := fnv.New32a()
	h.Write([]byte(r.command))
	r.hash = h.Sum32()

	return nil
}

// NextValue returns the next argument value with its type, tracking parsed arguments count
func (r *ReqReader) NextValue() (*RESPValue, error) {
	// Ensure command and args are parsed first
	if err := r.parseCommandAndArgs(); err != nil {
		return nil, err
	}

	// Check if we've already read all arguments
	if r.argsRead >= r.nargs {
		return nil, fmt.Errorf("no more arguments available (read %d of %d)", r.argsRead, r.nargs)
	}

	// Create a RESPReader to read the next value
	respReader := &RESPReader{
		reader: r.reader,
	}

	// Read the value
	value, err := respReader.ReadValue()
	if err != nil {
		return nil, fmt.Errorf("failed to read argument %d: %w", r.argsRead+1, err)
	}

	// Increment the args counter
	r.argsRead++

	return value, nil
}

// NextReader returns a RESPReader for reading the next argument (legacy method)
func (r *ReqReader) NextReader() (*RESPReader, error) {
	// 如果 command 存在多个 args , 则可通过调用多次 next reader 来读取
	// Ensure command and args are parsed first
	if err := r.parseCommandAndArgs(); err != nil {
		return nil, err
	}

	// Create a new RESPReader that shares the same underlying reader
	return &RESPReader{
		reader: r.reader,
	}, nil
}

// Reset resets the reader state for parsing a new command
func (r *ReqReader) Reset() {
	r.command = ""
	r.nargs = 0
	r.hash = 0
	r.parseErr = nil
	r.argsRead = 0
}

// readLine reads a line ending with \r\n
func (r *ReqReader) readLine() (string, error) {
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
