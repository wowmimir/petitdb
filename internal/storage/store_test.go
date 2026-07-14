package storage

import (
	"testing"
	"time"
)

func TestExpiration(t *testing.T) {
	s := NewStore()

	// Test Set and Get without expiration
	s.Set("key1", []byte("value1"))
	val, ok := s.Get("key1")
	if !ok || string(val) != "value1" {
		t.Errorf("Get failed: got %v, %v", string(val), ok)
	}

	// Test Expire and TTL
	s.Set("key2", []byte("value2"))
	ok = s.Expire("key2", 2)
	if !ok {
		t.Errorf("Expire failed for existing key")
	}

	ttl := s.TTL("key2")
	if ttl <= 0 || ttl > 2 {
		t.Errorf("TTL should be ~2, got %d", ttl)
	}

	// Test TTL on non-existent key
	ttl = s.TTL("nonexistent")
	if ttl != -2 {
		t.Errorf("TTL on nonexistent should be -2, got %d", ttl)
	}

	// Test expiration via Get
	time.Sleep(3 * time.Second)
	val, ok = s.Get("key2")
	if ok || val != nil {
		t.Errorf("Get after expiration should return nil, got %v, %v", string(val), ok)
	}

	// Test TTL after expiration
	ttl = s.TTL("key2")
	if ttl != -2 {
		t.Errorf("TTL after expiration should be -2, got %d", ttl)
	}
}

func TestDeleteExpired(t *testing.T) {
    s := NewStore()

    s.Set("key1", []byte("value1"))
    s.SetWithExpiration("key2", []byte("value2"), 1)
    s.SetWithExpiration("key3", []byte("value3"), 0)

    // Debug: print expiration times
    s.mu.RLock()
    for k, v := range s.data {
        t.Logf("Key: %s, ExpiresAt: %d", k, v.ExpiresAt)
    }
    s.mu.RUnlock()

    time.Sleep(3 * time.Second) // increase sleep to be safe

    // Debug: print current time and check again
    now := time.Now().UnixNano()
    s.mu.RLock()
    for k, v := range s.data {
        t.Logf("After sleep - Key: %s, ExpiresAt: %d, Now: %d, Expired: %v", k, v.ExpiresAt, now, v.isExpired())
    }
    s.mu.RUnlock()

    deleted := s.DeleteExpired()
    if deleted != 1 {
        t.Errorf("DeleteExpired should delete 1 key, got %d", deleted)
    }

    if s.Size() != 2 {
        t.Errorf("Size should be 2, got %d", s.Size())
    }
}

func TestConcurrentAccess(t *testing.T) {
	s := NewStore()

	// Run multiple goroutines concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(i int) {
			key := string(rune('a' + i))
			s.Set(key, []byte("value"))
			s.Expire(key, 10)
			s.TTL(key)
			s.Get(key)
			s.Exists(key)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
	// If we get here without race issues, test passes
}