package commands

import (
	"fmt"
	"strconv"

	"spine-go/libspine/transport"
)

// RESP3 response writers

// writeRESP3SimpleString writes a RESP3 simple string
func writeRESP3SimpleString(writer transport.Writer, s string) error {
	response := fmt.Sprintf("+%s\r\n", s)
	_, err := writer.Write([]byte(response))
	return err
}

// writeRESP3Error writes a RESP3 error
func writeRESP3Error(writer transport.Writer, err string) error {
	response := fmt.Sprintf("-%s\r\n", err)
	_, err2 := writer.Write([]byte(response))
	return err2
}

// writeRESP3Integer writes a RESP3 integer
func writeRESP3Integer(writer transport.Writer, i int64) error {
	response := fmt.Sprintf(":%d\r\n", i)
	_, err := writer.Write([]byte(response))
	return err
}

// writeRESP3BulkString writes a RESP3 bulk string
func writeRESP3BulkString(writer transport.Writer, s string) error {
	response := fmt.Sprintf("$%d\r\n%s\r\n", len(s), s)
	_, err := writer.Write([]byte(response))
	return err
}

// writeRESP3Null writes a RESP3 null
func writeRESP3Null(writer transport.Writer) error {
	response := "$-1\r\n"
	_, err := writer.Write([]byte(response))
	return err
}

// writeRESP3Array writes a RESP3 array of strings
func writeRESP3Array(writer transport.Writer, arr []string) error {
	response := fmt.Sprintf("*%d\r\n", len(arr))
	for _, item := range arr {
		response += fmt.Sprintf("$%d\r\n%s\r\n", len(item), item)
	}
	_, err := writer.Write([]byte(response))
	return err
}

// writeRESP3Map writes a RESP3 map
func writeRESP3Map(writer transport.Writer, m map[string]interface{}) error {
	response := fmt.Sprintf("%%%d\r\n", len(m))
	
	for key, value := range m {
		// Write key as bulk string
		response += fmt.Sprintf("$%d\r\n%s\r\n", len(key), key)
		
		// Write value based on type
		switch v := value.(type) {
		case string:
			response += fmt.Sprintf("$%d\r\n%s\r\n", len(v), v)
		case int:
			response += fmt.Sprintf(":%d\r\n", v)
		case int64:
			response += fmt.Sprintf(":%d\r\n", v)
		case []string:
			response += fmt.Sprintf("*%d\r\n", len(v))
			for _, item := range v {
				response += fmt.Sprintf("$%d\r\n%s\r\n", len(item), item)
			}
		default:
			// Convert to string as fallback
			str := fmt.Sprintf("%v", v)
			response += fmt.Sprintf("$%d\r\n%s\r\n", len(str), str)
		}
	}
	
	_, err := writer.Write([]byte(response))
	return err
}

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