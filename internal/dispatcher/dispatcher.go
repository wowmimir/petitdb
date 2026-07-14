package dispatcher

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	apperrors "github.com/wowmimir/petitdb/internal/errors"
	"github.com/wowmimir/petitdb/internal/protocol/resp"
	"github.com/wowmimir/petitdb/internal/pubsub"
	"github.com/wowmimir/petitdb/internal/storage"
)

// SupportedCommands lists all v1 commands – used in verbose errors.
var SupportedCommands = []string{
	"PING", "SET", "GET", "DEL", "EXISTS",
	"EXPIRE", "TTL", "SAVE", "SUBSCRIBE", "PUBLISH", "INFO",
}

// StatsProvider provides runtime statistics for the INFO command.
type StatsProvider interface {
	UptimeSeconds() int64
	ClientCount() int32
}

// Dispatcher routes commands to the appropriate subsystem.
type Dispatcher struct {
	store          *storage.Store
	pubsub         *pubsub.Broker
	saveFunc       func() error
	commandCounter int64
	statsProvider  StatsProvider
}

func NewDispatcher(store *storage.Store, pb *pubsub.Broker, saveFunc func() error) *Dispatcher {
	return &Dispatcher{
		store:    store,
		pubsub:   pb,
		saveFunc: saveFunc,
	}
}

// SetStatsProvider injects the StatsProvider (typically the Server) after creation.
func (d *Dispatcher) SetStatsProvider(provider StatsProvider) {
	d.statsProvider = provider
}

// CommandCount returns the total number of commands processed.
func (d *Dispatcher) CommandCount() int64 {
	return atomic.LoadInt64(&d.commandCounter)
}

// Dispatch processes a command and returns a result or an error.
// The clientCh parameter is used for SUBSCRIBE commands.
func (d *Dispatcher) Dispatch(cmd string, args [][]byte, clientCh chan []byte) (interface{}, error) {
	// Increment command counter for every parsed command
	atomic.AddInt64(&d.commandCounter, 1)

	// Input validation for key (if there is at least one argument)
	if len(args) > 0 {
		key := string(args[0])
		if len(key) == 0 {
			return nil, apperrors.ErrEmptyKey
		}
		if len(key) > 256 {
			return nil, apperrors.ErrKeyTooLong
		}
	}

	cmdUpper := strings.ToUpper(cmd)

	switch cmdUpper {
	case "SET":
		if len(args) != 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'SET' (expected 2, got %d)", len(args))
		}
		d.store.Set(string(args[0]), args[1])
		return "OK", nil

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

	case "EXPIRE":
		if len(args) != 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'EXPIRE' (expected 2, got %d)", len(args))
		}
		seconds, err := strconv.ParseInt(string(args[1]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("ERR value is not an integer or out of range")
		}
		ok := d.store.Expire(string(args[0]), seconds)
		return ok, nil

	case "TTL":
		if len(args) != 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'TTL' (expected 1, got %d)", len(args))
		}
		ttl := d.store.TTL(string(args[0]))
		return ttl, nil

	case "SUBSCRIBE":
		if len(args) < 1 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'SUBSCRIBE' (expected at least 1)")
		}

		// Subscribe to each topic and build confirmations
		confirmations := make([]interface{}, 0, len(args))
		for _, topicBytes := range args {
			topic := string(topicBytes)
			// Validate topic name
			if len(topic) == 0 {
				return nil, apperrors.ErrEmptyKey
			}
			if len(topic) > 256 {
				return nil, apperrors.ErrKeyTooLong
			}

			// Add subscription
			d.pubsub.Subscribe(topic, clientCh)

			// Get count for this topic after subscribing
			count := d.pubsub.SubscriberCountForTopic(topic)

			// Create confirmation: ["subscribe", topic, count]
			confirmations = append(confirmations, []interface{}{
				"subscribe",
				topic,
				count,
			})
		}

		return confirmations, nil

	case "PUBLISH":
		if len(args) != 2 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'PUBLISH' (expected 2, got %d)", len(args))
		}

		topic := string(args[0])
		message := args[1]

		// Serialize the push message once: ["message", topic, message]
		pushMessage := []interface{}{"message", topic, message}
		serialized := resp.SerializeArray(pushMessage)

		// Broadcast to all subscribers
		count := d.pubsub.Publish(topic, []byte(serialized))
		return count, nil

	case "SAVE":
		if len(args) != 0 {
			return nil, fmt.Errorf("ERR wrong number of arguments for 'SAVE' (expected 0, got %d)", len(args))
		}
		if err := d.saveFunc(); err != nil {
			return nil, fmt.Errorf("ERR failed to save snapshot: %v", err)
		}
		return "OK", nil

	case "INFO":
		return d.handleInfo()

	default:
		return nil, fmt.Errorf(
			"ERR unknown command '%s'. PetitDB v1 supports: %s",
			cmd,
			strings.Join(SupportedCommands, ", "),
		)
	}
}

// handleInfo collects and formats runtime statistics.
func (d *Dispatcher) handleInfo() (interface{}, error) {
	if d.statsProvider == nil {
		return nil, fmt.Errorf("ERR stats provider not available")
	}

	uptime := d.statsProvider.UptimeSeconds()
	clients := d.statsProvider.ClientCount()
	keys := d.store.Size()
	commands := d.CommandCount()

	// Return as []byte to get RESP bulk string (supports newlines)
	return []byte(fmt.Sprintf(
		"# Server\nuptime_seconds: %d\nconnected_clients: %d\ntotal_keys: %d\ncommands_processed: %d\n",
		uptime, clients, keys, commands,
	)), nil
}