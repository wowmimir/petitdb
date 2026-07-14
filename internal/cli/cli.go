// Package cli implements the interactive command-line interface for PetitDB.
package cli

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
	"strconv"

	"github.com/wowmimir/petitdb/internal/protocol/resp"
)

const (
	maxHistory            = 1000
	maxReconnectAttempts  = 5
	reconnectBaseDelay    = 1 * time.Second
)

// Run starts the CLI REPL connecting to the given host and port.
func Run(host string, port int) {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not connect to PetitDB at %s. Is the server running? Use 'petitdb' to start it.\n", addr)
		os.Exit(1)
	}
	defer conn.Close()

	// Enable raw terminal input
	fd := int(os.Stdin.Fd())
	oldState, err := enableRawMode(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not enable raw terminal: %v\n", err)
	} else {
		defer disableRawMode(fd, oldState)
	}

	history := make([]string, 0, maxHistory)
	histPos := 0
	currentLine := []rune{}
	cursorPos := 0

	fmt.Printf("PetitDB CLI (connected to %s)\nType 'exit' or Ctrl-C to quit.\n", addr)

	for {
		fmt.Print("petitdb> ")

		var line string
		var done bool
		for !done {
			b := make([]byte, 1)
			_, err := os.Stdin.Read(b)
			if err != nil {
				done = true
				line = ""
				break
			}
			c := b[0]

			switch c {
			case 3: // Ctrl+C
				fmt.Println("^C")
				restoreTerminal(fd, oldState)
				os.Exit(0)
			case 4: // Ctrl+D
				fmt.Println()
				done = true
				line = ""
			case 13, 10: // Enter
				fmt.Println()
				done = true
				line = string(currentLine)
				// Clear buffer for next command
				currentLine = []rune{}
				cursorPos = 0
			case 27: // Escape sequence
				seq := make([]byte, 2)
				if _, err := os.Stdin.Read(seq); err != nil {
					continue
				}
				if seq[0] == '[' {
					switch seq[1] {
					case 'A': // Up
						if histPos > 0 {
							histPos--
							if histPos < len(history) {
								currentLine = []rune(history[histPos])
								cursorPos = len(currentLine)
								fmt.Printf("\r\033[Kpetitdb> %s", string(currentLine))
							}
						}
					case 'B': // Down
						if histPos < len(history)-1 {
							histPos++
							currentLine = []rune(history[histPos])
							cursorPos = len(currentLine)
							fmt.Printf("\r\033[Kpetitdb> %s", string(currentLine))
						} else if histPos == len(history)-1 {
							histPos = len(history)
							currentLine = []rune{}
							cursorPos = 0
							fmt.Printf("\r\033[Kpetitdb> ")
						}
					}
				}
			case 127, 8: // Backspace
				if cursorPos > 0 {
					currentLine = append(currentLine[:cursorPos-1], currentLine[cursorPos:]...)
					cursorPos--
					fmt.Printf("\r\033[Kpetitdb> %s", string(currentLine))
					if cursorPos < len(currentLine) {
						fmt.Printf("\033[%dD", len(currentLine)-cursorPos)
					}
				}
			default:
				if c >= 32 && c <= 126 {
					inserted := make([]rune, len(currentLine)+1)
					copy(inserted[:cursorPos], currentLine[:cursorPos])
					inserted[cursorPos] = rune(c)
					copy(inserted[cursorPos+1:], currentLine[cursorPos:])
					currentLine = inserted
					cursorPos++
					fmt.Printf("\r\033[Kpetitdb> %s", string(currentLine))
					if cursorPos < len(currentLine) {
						fmt.Printf("\033[%dD", len(currentLine)-cursorPos)
					}
				}
			}
		}

		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			break
		}

		if len(history) == 0 || history[len(history)-1] != line {
			if len(history) >= maxHistory {
				history = history[1:]
			}
			history = append(history, line)
			histPos = len(history)
		}

		args, err := tokenize(line)
		if err != nil {
			fmt.Printf("(error) %v\n", err)
			continue
		}
		if len(args) == 0 {
			continue
		}

		cmdBytes := resp.SerializeCommand(args)
		_, err = conn.Write(cmdBytes)
		if err != nil {
			conn, err = reconnect(addr, conn)
			if err != nil {
				fmt.Printf("(error) connection lost: %v\n", err)
				conn = nil
				continue
			}
			_, err = conn.Write(cmdBytes)
			if err != nil {
				fmt.Printf("(error) failed to send command: %v\n", err)
				continue
			}
		}

		response, err := readResponse(conn)
		if err != nil {
			fmt.Printf("(error) reading response: %v\n", err)
			continue
		}
		printResponse(response)
	}
}

// tokenize splits a command line into arguments, respecting double quotes.
func tokenize(line string) ([]string, error) {
	var args []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for _, ch := range line {
		switch {
		case escaped:
			current.WriteRune(ch)
			escaped = false
		case ch == '\\':
			escaped = true
		case ch == '"':
			inQuotes = !inQuotes
		case ch == ' ' && !inQuotes:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}
	if inQuotes {
		return nil, fmt.Errorf("unclosed quote")
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args, nil
}

// readResponse reads a complete RESP response from the connection.
func readResponse(conn net.Conn) (interface{}, error) {
	r := bufio.NewReader(conn)
	return parseValue(r)
}

func parseValue(r *bufio.Reader) (interface{}, error) {
	b, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	switch b {
	case '+':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		return string(line), nil
	case '-':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		return fmt.Errorf("%s", string(line)), nil
	case ':':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		var val int64
		_, err = fmt.Sscanf(string(line), "%d", &val)
		if err != nil {
			return nil, err
		}
		return val, nil
	case '$':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		var length int
		_, err = fmt.Sscanf(string(line), "%d", &length)
		if err != nil {
			return nil, err
		}
		if length < 0 {
			return nil, nil
		}
		data := make([]byte, length)
		_, err = r.Read(data)
		if err != nil {
			return nil, err
		}
		if _, err := readLine(r); err != nil {
			return nil, err
		}
		return data, nil
	case '*':
		line, err := readLine(r)
		if err != nil {
			return nil, err
		}
		var count int
		_, err = fmt.Sscanf(string(line), "%d", &count)
		if err != nil {
			return nil, err
		}
		if count < 0 {
			return nil, nil
		}
		arr := make([]interface{}, count)
		for i := 0; i < count; i++ {
			val, err := parseValue(r)
			if err != nil {
				return nil, err
			}
			arr[i] = val
		}
		return arr, nil
	default:
		return nil, fmt.Errorf("unknown RESP type: %c", b)
	}
}

func readLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if len(line) < 2 || line[len(line)-2] != '\r' {
		return nil, fmt.Errorf("invalid CRLF")
	}
	return line[:len(line)-2], nil
}

func printResponse(v interface{}) {
	switch val := v.(type) {
	case string:
		fmt.Printf("(string) \"%s\"\n", val)
	case error:
		fmt.Printf("(error) %s\n", val.Error())
	case int64:
		fmt.Printf("(integer) %d\n", val)
	case []byte:
		if val == nil {
			fmt.Println("(nil)")
		} else {
			fmt.Printf("(string) \"%s\"\n", string(val))
		}
	case []interface{}:
		fmt.Println("(array)")
		for i, elem := range val {
			fmt.Printf("  %d: ", i)
			printResponse(elem)
		}
	case nil:
		fmt.Println("(nil)")
	default:
		fmt.Printf("(unknown) %v\n", val)
	}
}

func reconnect(addr string, oldConn net.Conn) (net.Conn, error) {
	oldConn.Close()
	delay := reconnectBaseDelay
	for attempt := 1; attempt <= maxReconnectAttempts; attempt++ {
		time.Sleep(delay)
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			fmt.Printf("(reconnected to %s)\n", addr)
			return conn, nil
		}
		delay *= 2
	}
	return nil, fmt.Errorf("failed to reconnect after %d attempts", maxReconnectAttempts)
}

func restoreTerminal(fd int, oldState interface{}) {
	if oldState != nil {
		disableRawMode(fd, oldState)
	}
}