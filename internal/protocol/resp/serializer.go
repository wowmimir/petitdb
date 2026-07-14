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
		return "+" + val + "\r\n"
	case int:
		return ":" + strconv.Itoa(val) + "\r\n"
	case int64:
		return ":" + strconv.FormatInt(val, 10) + "\r\n"
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
	case []interface{}:
		return SerializeArray(val)
	case error:
		return "-" + val.Error() + "\r\n"
	default:
		// Fallback: treat as string
		return fmt.Sprintf("+%v\r\n", val)
	}
}

// SerializeArray converts a slice of values to a RESP array string.
func SerializeArray(elements []interface{}) string {
	if elements == nil {
		return "*-1\r\n"
	}

	result := fmt.Sprintf("*%d\r\n", len(elements))
	for _, elem := range elements {
		result += Serialize(elem)
	}
	return result
}