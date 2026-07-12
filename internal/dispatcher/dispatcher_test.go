package dispatcher

import (
	"strings"
	"testing"

	"github.com/wowmimir/petitdb/internal/storage"
)

func TestDispatcher_SetAndGet(t *testing.T) {
	store := storage.NewStore()
	d := NewDispatcher(store)

	// Test SET
	res, err := d.Dispatch("SET", [][]byte{[]byte("name"), []byte("petit")})
	if err != nil {
		t.Fatalf("SET unexpected error: %v", err)
	}
	if res != "OK" {
		t.Errorf("SET expected 'OK', got %v", res)
	}

	// Test GET existing key
	res, err = d.Dispatch("GET", [][]byte{[]byte("name")})
	if err != nil {
		t.Fatalf("GET unexpected error: %v", err)
	}
	val, ok := res.([]byte)
	if !ok {
		t.Fatalf("GET expected []byte, got %T", res)
	}
	if string(val) != "petit" {
		t.Errorf("GET expected 'petit', got '%s'", string(val))
	}

	// Test GET non-existent key (should return nil, nil)
	res, err = d.Dispatch("GET", [][]byte{[]byte("missing")})
	if err != nil {
		t.Fatalf("GET missing unexpected error: %v", err)
	}
	if res != nil {
		t.Errorf("GET missing expected nil, got %v", res)
	}

	// Test DEL
	res, err = d.Dispatch("DEL", [][]byte{[]byte("name")})
	if err != nil {
		t.Fatalf("DEL unexpected error: %v", err)
	}
	deleted, ok := res.(bool)
	if !ok {
		t.Fatalf("DEL expected bool, got %T", res)
	}
	if !deleted {
		t.Errorf("DEL expected true for existing key")
	}

	// Verify deletion
	res, err = d.Dispatch("GET", [][]byte{[]byte("name")})
	if err != nil {
		t.Fatalf("GET after DEL unexpected error: %v", err)
	}
	if res != nil {
		t.Errorf("GET after DEL expected nil, got %v", res)
	}

	// Test DEL on missing key
	res, err = d.Dispatch("DEL", [][]byte{[]byte("missing")})
	if err != nil {
		t.Fatalf("DEL missing unexpected error: %v", err)
	}
	deleted, ok = res.(bool)
	if !ok {
		t.Fatalf("DEL missing expected bool, got %T", res)
	}
	if deleted {
		t.Errorf("DEL missing expected false")
	}

	// Test EXISTS
	res, err = d.Dispatch("EXISTS", [][]byte{[]byte("name")})
	if err != nil {
		t.Fatalf("EXISTS unexpected error: %v", err)
	}
	exists, ok := res.(bool)
	if !ok {
		t.Fatalf("EXISTS expected bool, got %T", res)
	}
	if exists {
		t.Errorf("EXISTS after delete expected false")
	}
}

func TestDispatcher_KeyValidation(t *testing.T) {
	store := storage.NewStore()
	d := NewDispatcher(store)

	// Test empty key
	_, err := d.Dispatch("GET", [][]byte{[]byte("")})
	if err == nil {
		t.Errorf("Expected error for empty key, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("Empty key error should mention 'empty', got: %v", err)
	}

	// Test key too long (257 bytes)
	longKey := make([]byte, 257)
	for i := range longKey {
		longKey[i] = 'a'
	}
	_, err = d.Dispatch("GET", [][]byte{longKey})
	if err == nil {
		t.Errorf("Expected error for key longer than 256 chars, got nil")
	}
	if !strings.Contains(err.Error(), "256") {
		t.Errorf("Long key error should mention '256', got: %v", err)
	}

	// Test key exactly 256 chars (should pass)
	exactKey := make([]byte, 256)
	for i := range exactKey {
		exactKey[i] = 'a'
	}
	_, err = d.Dispatch("SET", [][]byte{exactKey, []byte("ok")})
	if err != nil {
		t.Errorf("Key of exactly 256 chars should be valid, got error: %v", err)
	}
}

func TestDispatcher_WrongArguments(t *testing.T) {
	store := storage.NewStore()
	d := NewDispatcher(store)

	tests := []struct {
		cmd  string
		args [][]byte
	}{
		{"SET", [][]byte{[]byte("key")}},            // SET needs 2 args
		{"GET", [][]byte{}},                          // GET needs 1 arg
		{"GET", [][]byte{[]byte("key"), []byte("x")}}, // GET needs 1 arg
		{"DEL", [][]byte{}},                          // DEL needs 1 arg
		{"EXISTS", [][]byte{[]byte("a"), []byte("b")}}, // EXISTS needs 1 arg
	}

	for _, tt := range tests {
		_, err := d.Dispatch(tt.cmd, tt.args)
		if err == nil {
			t.Errorf("Dispatch(%s, %d args) expected error, got nil", tt.cmd, len(tt.args))
		}
		if !strings.Contains(err.Error(), "wrong number") {
			t.Errorf("Dispatch(%s) error should mention 'wrong number', got: %v", tt.cmd, err)
		}
	}
}

func TestDispatcher_UnknownCommand(t *testing.T) {
	store := storage.NewStore()
	d := NewDispatcher(store)

	_, err := d.Dispatch("HGET", [][]byte{[]byte("key")})
	if err == nil {
		t.Errorf("Expected error for unknown command, got nil")
	}

	// Check the error contains the list of supported commands
	errMsg := err.Error()
	if !strings.Contains(errMsg, "unknown command") {
		t.Errorf("Error should say 'unknown command', got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "PING") {
		t.Errorf("Error should list 'PING' in supported commands, got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "SET") {
		t.Errorf("Error should list 'SET' in supported commands, got: %v", errMsg)
	}
}

func TestDispatcher_CaseInsensitivity(t *testing.T) {
	store := storage.NewStore()
	d := NewDispatcher(store)

	// Commands should work in any case
	testCases := []string{"set", "Set", "sEt", "SET"}
	for _, cmd := range testCases {
		res, err := d.Dispatch(cmd, [][]byte{[]byte("case"), []byte("test")})
		if err != nil {
			t.Errorf("Dispatch(%s) failed: %v", cmd, err)
		}
		if res != "OK" {
			t.Errorf("Dispatch(%s) expected OK, got %v", cmd, res)
		}
	}
}