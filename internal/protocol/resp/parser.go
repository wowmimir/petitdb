package resp

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
	"fmt"
)

// ParseCommand reads a RESP array from the reader and returns (command, args, error).
func ParseCommand(r *bufio.Reader) (string, [][]byte, error) {
	// Read the first byte (should be '*')
	line, err := readLine(r)
	if err != nil {
		return "", nil, err
	}
	if len(line) == 0 || line[0] != '*' {
		return "", nil, fmt.Errorf("ERR invalid protocol: expected array, got %s", string(line))
	}

	// Parse array length
	count, err := parseInt(line[1:])
	if err != nil {
		return "", nil, fmt.Errorf("ERR invalid array length: %w", err)
	}
	if count < 1 {
		return "", nil, fmt.Errorf("ERR empty command array")
	}

	// Parse each bulk string element
	args := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		bulk, err := parseBulkString(r)
		if err != nil {
			return "", nil, err
		}
		args = append(args, bulk)
	}

	// First arg is the command name
	cmd := strings.ToUpper(string(args[0]))
	return cmd, args[1:], nil
}

// readLine reads until \r\n and returns the line (without the delimiter).
func readLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(line) < 2 || line[len(line)-2] != '\r' {
		return nil, fmt.Errorf("ERR invalid CRLF")
	}
	return line[:len(line)-2], nil
}

// parseBulkString reads a RESP bulk string: $<len>\r\n<data>\r\n
func parseBulkString(r *bufio.Reader) ([]byte, error) {
	line, err := readLine(r)
	if err != nil {
		return nil, err
	}
	if len(line) == 0 || line[0] != '$' {
		return nil, fmt.Errorf("ERR expected bulk string, got %s", string(line))
	}
	length, err := parseInt(line[1:])
	if err != nil {
		return nil, err
	}
	if length < 0 {
		return nil, nil // Null bulk string
	}
	data := make([]byte, length)
	_, err = r.Read(data)
	if err != nil {
		return nil, err
	}
	// Read trailing \r\n
	if _, err := readLine(r); err != nil {
		return nil, err
	}
	return data, nil
}

// parseInteger parses a base-10 integer from a byte slice.
func parseInt(b []byte) (int, error) {
	val, err := strconv.Atoi(string(bytes.TrimSpace(b)))
	if err != nil {
		return 0, err
	}
	return val, nil
}