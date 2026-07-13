package dispatcher

import (
	"fmt"
	"strings"

	apperrors "github.com/wowmimir/petitdb/internal/errors"
	"github.com/wowmimir/petitdb/internal/storage"
)

// SupportedCommands lists all v1 commands – used in verbose errors.
var SupportedCommands = []string{
	"PING", "SET", "GET", "DEL", "EXISTS",
	"EXPIRE", "TTL", "SAVE", "SUBSCRIBE", "PUBLISH", "INFO",
}

// Dispatcher routes commands to the storage engine.
type Dispatcher struct {
	store *storage.Store
}

func NewDispatcher(store *storage.Store) *Dispatcher {
	return &Dispatcher{store: store}
}

// Dispatch processes a command and returns a result or an error.
func (d *Dispatcher) Dispatch(cmd string, args [][]byte) (interface{}, error) {
	// Input validation for the key (if there is at least one argument)
	if len(args) > 0 {
		key := string(args[0])
		if len(key) == 0 {
			return nil, apperrors.ErrEmptyKey
		}
		if len(key) > 256 {
			return nil, apperrors.ErrKeyTooLong
		}
	}

	// Normalise command to uppercase
	cmdUpper := strings.ToUpper(cmd)

	switch cmdUpper {
	case "SET":
		if len(args) != 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'SET' (expected 2, got %d)", len(args))
		}
		d.store.Set(string(args[0]), args[1])
		return "+OK", nil

	case "GET":
		if len(args) != 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'GET' (expected 1, got %d)", len(args))
		}
		val, ok := d.store.Get(string(args[0]))
		if !ok {
			return nil, nil // RESP null bulk string
		}
		return val, nil

	case "DEL":
		if len(args) != 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'DEL' (expected 1, got %d)", len(args))
		}
		deleted := d.store.Delete(string(args[0]))
		return deleted, nil

	case "EXISTS":
		if len(args) != 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'EXISTS' (expected 1, got %d)", len(args))
		}
		exists := d.store.Exists(string(args[0]))
		return exists, nil

	// We'll add EXPIRE, TTL, SAVE, etc. later
	case "SUBSCRIBE", "PUBLISH":
		// TODO: Route to pubsub broker (Phase 4)
		return nil, fmt.Errorf("ERR pubsub commands not yet implemented")

	default:
		// Verbose unknown command error – lists all supported commands
		return nil, fmt.Errorf(
			"ERR unknown command '%s'. PetitDB v1 supports: %s",
			cmd,
			strings.Join(SupportedCommands, ", "),
		)
	}
}
