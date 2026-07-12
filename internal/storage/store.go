package storage

import (
	"sync"
)

type Value struct {
	Data      []byte
	ExpiresAt int64
}

type Store struct {
	mu   sync.RWMutex
	data map[string]Value
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]Value),
	}
}

func (s *Store) Set(key string, val []byte) {

	s.mu.Lock() //Full lock because write operation

	defer s.mu.Unlock()

	value := Value{
		Data:      make([]byte, len(val)),
		ExpiresAt: 0,
	}

	copy(value.Data, val)

	s.data[key] = value
}

func (s *Store) Get(key string) ([]byte, bool) {

	s.mu.RLock() // Read Lock (everyone can read) -> if writer holds lock does not allow read

	defer s.mu.RUnlock()

	val, exists := s.data[key]
	if !exists {
		return nil, false
	}

	result := make([]byte, len(val.Data))
	copy(result, val.Data)

	return result, true
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()

	defer s.mu.Unlock()

	_, exists := s.data[key]

	if !exists {
		return false
	}

	delete(s.data, key)

	return true
}

func (s *Store) Exists(key string) bool {
	s.mu.RLock()

	defer s.mu.RUnlock()

	_, exists := s.data[key]

	return exists
}

func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))

	for key := range s.data {
		keys = append(keys, key)
	}

	return keys
}

func (s *Store) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}
