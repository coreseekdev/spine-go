package commands

import (
	"fmt"
	"strconv"

	"spine-go/libspine/transport"
)

// RESP3 readers (for parsing commands)

// ParseRESPCommand parses a RESP command from a reader
func ParseRESPCommand(reader transport.Reader) (string, []string, error) {
	// Read the first line to determine the type
	buf := make([]byte, 1024)
	n, err := reader.Read(buf)
	if err != nil {
		return "", nil, err
	}

	data := string(buf[:n])
	lines := splitRESPLines(data)
	if len(lines) == 0 {
		return "", nil, fmt.Errorf("empty command")
	}

	// Parse array format: *<count>\r\n
	if lines[0][0] != '*' {
		return "", nil, fmt.Errorf("expected array format")
	}

	count, err := strconv.Atoi(lines[0][1:])
	if err != nil {
		return "", nil, fmt.Errorf("invalid array count: %w", err)
	}

	if count == 0 {
		return "", nil, fmt.Errorf("empty command array")
	}

	// Parse command and arguments
	var command string
	var args []string
	lineIdx := 1

	for i := 0; i < count && lineIdx < len(lines); i++ {
		// Expect bulk string format: $<length>\r\n<data>\r\n
		if lineIdx >= len(lines) || lines[lineIdx][0] != '$' {
			return "", nil, fmt.Errorf("expected bulk string")
		}

		length, err := strconv.Atoi(lines[lineIdx][1:])
		if err != nil {
			return "", nil, fmt.Errorf("invalid bulk string length: %w", err)
		}

		lineIdx++
		if lineIdx >= len(lines) {
			return "", nil, fmt.Errorf("missing bulk string data")
		}

		value := lines[lineIdx]
		if len(value) != length {
			return "", nil, fmt.Errorf("bulk string length mismatch")
		}

		if i == 0 {
			command = value
		} else {
			args = append(args, value)
		}

		lineIdx++
	}

	return command, args, nil
}

// splitRESPLines splits RESP data into lines
func splitRESPLines(data string) []string {
	var lines []string
	var current string

	for i := 0; i < len(data); i++ {
		if i < len(data)-1 && data[i] == '\r' && data[i+1] == '\n' {
			lines = append(lines, current)
			current = ""
			i++ // Skip \n
		} else {
			current += string(data[i])
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}