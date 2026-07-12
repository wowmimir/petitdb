package storage

import (
	"sync"
	"testing"
)

func TestStore_SetAndGet(t *testing.T) {
	s := NewStore()

	// Test normal set/get
	s.Set("name", []byte("petit"))
	val, ok := s.Get("name")
	if !ok {
		t.Errorf("Get('name') expected ok=true, got false")
	}
	if string(val) != "petit" {
		t.Errorf("Get('name') expected 'petit', got '%s'", string(val))
	}

	// Test getting a non-existent key
	_, ok = s.Get("missing")
	if ok {
		t.Errorf("Get('missing') expected ok=false, got true")
	}

	// Test storing and retrieving empty value
	s.Set("empty", []byte{})
	val, ok = s.Get("empty")
	if !ok {
		t.Errorf("Get('empty') expected ok=true, got false")
	}
	if len(val) != 0 {
		t.Errorf("Get('empty') expected empty slice, got length %d", len(val))
	}
}

func TestStore_Delete(t *testing.T) {
	s := NewStore()

	// Delete existing key
	s.Set("key", []byte("val"))
	deleted := s.Delete("key")
	if !deleted {
		t.Errorf("Delete('key') expected true, got false")
	}

	// Verify it's gone
	_, ok := s.Get("key")
	if ok {
		t.Errorf("Get('key') after delete expected false, got true")
	}

	// Delete non-existent key
	deleted = s.Delete("missing")
	if deleted {
		t.Errorf("Delete('missing') expected false, got true")
	}
}

func TestStore_Exists(t *testing.T) {
	s := NewStore()

	// Exists on existing key
	s.Set("present", []byte("val"))
	if !s.Exists("present") {
		t.Errorf("Exists('present') expected true, got false")
	}

	// Exists on missing key
	if s.Exists("missing") {
		t.Errorf("Exists('missing') expected false, got true")
	}

	// Exists after delete
	s.Delete("present")
	if s.Exists("present") {
		t.Errorf("Exists('present') after delete expected false, got true")
	}
}

func TestStore_KeysAndSize(t *testing.T) {
	s := NewStore()

	// Empty store
	if s.Size() != 0 {
		t.Errorf("Size() expected 0, got %d", s.Size())
	}
	if len(s.Keys()) != 0 {
		t.Errorf("Keys() expected empty slice, got %v", s.Keys())
	}

	// Add some keys
	s.Set("a", []byte("1"))
	s.Set("b", []byte("2"))
	s.Set("c", []byte("3"))

	if s.Size() != 3 {
		t.Errorf("Size() expected 3, got %d", s.Size())
	}

	keys := s.Keys()
	if len(keys) != 3 {
		t.Errorf("Keys() length expected 3, got %d", len(keys))
	}

	// Verify all keys are present (order doesn't matter)
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}
	if !keyMap["a"] || !keyMap["b"] || !keyMap["c"] {
		t.Errorf("Keys() expected ['a','b','c'], got %v", keys)
	}
}

// THIS IS THE CRITICAL TEST FOR CONCURRENCY SAFETY
func TestStore_ConcurrentAccess(t *testing.T) {
	s := NewStore()
	var wg sync.WaitGroup

	// Launch 100 goroutines writing
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%26)) // cycles through a-z
			s.Set(key, []byte("value"))
		}(i)
	}

	// Launch 100 goroutines reading
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Get("a")
			s.Exists("a")
		}()
	}

	// Launch 100 goroutines deleting
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := string(rune('a' + i%26))
			s.Delete(key)
		}(i)
	}

	wg.Wait()
	// If we reach this point without a panic or data race,
	// the mutex is properly protecting the map.
	// Run with: go test -race ./internal/storage/
}