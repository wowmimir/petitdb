package resp

import (
	"fmt"
	"strconv"
)

// Serialize converts a Go value to a RESP string.
func Serialize(v interface{}) string {
	if v == nil {
		return "$-1\r\n"
	}

	switch val := v.(type) {
	case string:
		// Simple string (e.g., "OK")
		return "+" + val + "\r\n"
	case int:
		return ":" + strconv.Itoa(val) + "\r\n"
	case bool:
		if val {
			return ":1\r\n"
		}
		return ":0\r\n"
	case []byte:
		if val == nil {
			return "$-1\r\n"
		}
		return fmt.Sprintf("$%d\r\n%s\r\n", len(val), string(val))
	case error:
		return "-" + val.Error() + "\r\n"
	default:
		// Fallback: treat as string
		return fmt.Sprintf("+%v\r\n", val)
	}
}