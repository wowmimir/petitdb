package storage

import (
	"sync"
	"time"
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

func (v Value) isExpired() bool{
	if v.ExpiresAt == 0{
		return false
	}

	return time.Now().UnixNano() > v.ExpiresAt
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

// SetWithExpiration stores a value with a TTL (in seconds)
func (s *Store) SetWithExpiration(key string, val []byte, ttlSeconds int64) {
    s.mu.Lock()
    defer s.mu.Unlock()

    var expiresAt int64 = 0
    if ttlSeconds > 0 {
        expiresAt = time.Now().Add(time.Duration(ttlSeconds) * time.Second).UnixNano()
    }

    value := Value{
        Data:      make([]byte, len(val)),
        ExpiresAt: expiresAt,
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

	if val.isExpired(){
		delete(s.data,key)
		return nil,false
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

func (s *Store) Expire(key string, ttlSeconds int64) bool{
	s.mu.Lock()

	defer s.mu.Unlock()

	val,exists := s.data[key]

	if !exists{
		return false 
	}

	if val.isExpired(){
		delete(s.data,key)
		return false
	}
	if ttlSeconds>0{
		val.ExpiresAt = time.Now().UnixNano() + ttlSeconds*1e9
	} else{
		val.ExpiresAt = 0
	}
	s.data[key]  = val

	return true

}

func (s *Store) TTL(key string) int64{
	s.mu.RLock()
	defer s.mu.RUnlock()

	val,exists := s.data[key]
	if !exists{
		return -2
	}
	if val.isExpired(){
		return -2
	}

	if val.ExpiresAt == 0{
		return -1
	}

	now := time.Now().UnixNano()	

	remaining := val.ExpiresAt - now

	if remaining <=0 {
		return -2
	}

	return remaining/1e9
}

func (s *Store) DeleteExpired() int{
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0

	for key,val := range(s.data){
		if val.isExpired(){
			delete(s.data,key)
			count++
		}
	}
	return count
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
