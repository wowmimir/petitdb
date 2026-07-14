package persistence

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wowmimir/petitdb/internal/storage"
)

// setupTest creates a temporary directory for test snapshots
func setupTest(t *testing.T) (string, func()) {
	t.Helper()
	
	tmpDir, err := os.MkdirTemp("", "petitdb-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	
	// Return cleanup function
	cleanup := func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Warning: failed to remove temp dir: %v", err)
		}
	}
	
	return tmpDir, cleanup
}

// createTestStore creates a store with some test data
func createTestStore() *storage.Store {
	store := storage.NewStore()
	
	// Add some test keys
	store.Set("key1", []byte("value1"))
	store.SetWithExpiration("key2", []byte("value2"), 60) // expires in 60s
	store.Set("key3", []byte("value3"))
	
	return store
}

// verifyStoreContent checks that the store matches expected values
func verifyStoreContent(t *testing.T, store *storage.Store) {
	t.Helper()
	
	// Check key1
	val, ok := store.Get("key1")
	if !ok || string(val) != "value1" {
		t.Errorf("key1: expected 'value1', got '%s' (exists: %v)", string(val), ok)
	}
	
	// Check key2
	val, ok = store.Get("key2")
	if !ok || string(val) != "value2" {
		t.Errorf("key2: expected 'value2', got '%s' (exists: %v)", string(val), ok)
	}
	
	// Check key3
	val, ok = store.Get("key3")
	if !ok || string(val) != "value3" {
		t.Errorf("key3: expected 'value3', got '%s' (exists: %v)", string(val), ok)
	}
	
	// Check TTL for key2 (should be ~60)
	ttl := store.TTL("key2")
	if ttl <= 0 || ttl > 60 {
		t.Errorf("key2 TTL: expected ~60, got %d", ttl)
	}
	
	// Check total size
	if store.Size() != 3 {
		t.Errorf("Store size: expected 3, got %d", store.Size())
	}
}

// TestSaveAndLoad tests basic save and load functionality
func TestSaveAndLoad(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()
	
	// Create store with test data
	store := createTestStore()
	
	// Create snapshot manager
	pm := NewSnapshotManager(tmpDir)
	
	// Save snapshot
	err := pm.Save(store)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	
	// Verify snapshot file exists
	snapshotPath := filepath.Join(tmpDir, snapshotFile)
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		t.Fatalf("Snapshot file not created at %s", snapshotPath)
	}
	
	// Load the snapshot
	loadedStore, wasLoaded, err := pm.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	if !wasLoaded {
		t.Error("Load should have loaded the snapshot, but wasLoaded is false")
	}
	
	// Verify content
	verifyStoreContent(t, loadedStore)
}

// TestLoadEmptySnapshot tests loading when no snapshot exists
func TestLoadEmptySnapshot(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()
	
	pm := NewSnapshotManager(tmpDir)
	
	// Load when no snapshot exists
	store, wasLoaded, err := pm.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	if wasLoaded {
		t.Error("Load should not have loaded anything, but wasLoaded is true")
	}
	
	if store.Size() != 0 {
		t.Errorf("Store should be empty, got %d keys", store.Size())
	}
}

// TestOverwriteSnapshot tests that saving overwrites existing snapshots
func TestOverwriteSnapshot(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()
	
	pm := NewSnapshotManager(tmpDir)
	
	// Save first snapshot
	store1 := storage.NewStore()
	store1.Set("key1", []byte("value1"))
	err := pm.Save(store1)
	if err != nil {
		t.Fatalf("First save failed: %v", err)
	}
	
	// Save second snapshot with different data
	store2 := storage.NewStore()
	store2.Set("key2", []byte("value2"))
	err = pm.Save(store2)
	if err != nil {
		t.Fatalf("Second save failed: %v", err)
	}
	
	// Load and verify only key2 exists
	loadedStore, wasLoaded, err := pm.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	if !wasLoaded {
		t.Error("Load should have loaded the snapshot")
	}
	
	if loadedStore.Size() != 1 {
		t.Errorf("Store should have 1 key, got %d", loadedStore.Size())
	}
	
	val, ok := loadedStore.Get("key2")
	if !ok || string(val) != "value2" {
		t.Errorf("key2: expected 'value2', got '%s' (exists: %v)", string(val), ok)
	}
	
	_, ok = loadedStore.Get("key1")
	if ok {
		t.Error("key1 should not exist in loaded store")
	}
}

// TestCorruptSnapshot tests handling of corrupted snapshots
func TestCorruptSnapshot(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()
	
	// Create a corrupt snapshot file
	snapshotPath := filepath.Join(tmpDir, snapshotFile)
	err := os.WriteFile(snapshotPath, []byte("corrupted data\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create corrupt snapshot: %v", err)
	}
	
	pm := NewSnapshotManager(tmpDir)
	
	// Attempt to load - should handle corruption gracefully
	store, wasLoaded, err := pm.Load()
	if err != nil {
		t.Fatalf("Load should not return error on corruption, got: %v", err)
	}
	
	if wasLoaded {
		t.Error("Load should not load a corrupt snapshot, but wasLoaded is true")
	}
	
	// Store should be empty
	if store.Size() != 0 {
		t.Errorf("Store should be empty, got %d keys", store.Size())
	}
	
	// Verify corrupt file was renamed
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}
	
	renamed := false
	for _, file := range files {
		if file.Name() != snapshotFile && len(file.Name()) > len(corruptPrefix) {
			// Check if it starts with corruptPrefix
			if len(file.Name()) >= len(corruptPrefix) &&
			   file.Name()[:len(corruptPrefix)] == corruptPrefix {
				renamed = true
				break
			}
		}
	}
	
	if !renamed {
		t.Errorf("Corrupt snapshot should be renamed to %s.*, but no such file found", corruptPrefix)
		t.Logf("Files in dir: %v", files)
	}
}

// TestCorruptSnapshotWithInvalidFormat tests various corruption scenarios
func TestCorruptSnapshotWithInvalidFormat(t *testing.T) {
	tests := []struct {
		name     string
		content  string
	}{
		{
			name:    "missing parts",
			content: "key|value\n", // missing expires_at
		},
		{
			name:    "invalid base64",
			content: "key|invalid!!base64|123\n",
		},
		{
			name:    "invalid expiration",
			content: "key|dmFsdWU=|notanumber\n", // base64 for "value"
		},
		{
			name:    "empty key",
			content: "|dmFsdWU=|123\n",
		},
		{
			name:    "random garbage",
			content: "this is just garbage\n",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := setupTest(t)
			defer cleanup()
			
			// Write corrupt content
			snapshotPath := filepath.Join(tmpDir, snapshotFile)
			err := os.WriteFile(snapshotPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create corrupt snapshot: %v", err)
			}
			
			pm := NewSnapshotManager(tmpDir)
			
			// Attempt to load - should handle gracefully
			store, wasLoaded, err := pm.Load()
			if err != nil {
				t.Fatalf("Load should not return error, got: %v", err)
			}
			
			if wasLoaded {
				t.Error("Load should not load a corrupt snapshot")
			}
			
			if store.Size() != 0 {
				t.Errorf("Store should be empty, got %d keys", store.Size())
			}
			
			// Verify corrupt file was renamed
			files, err := os.ReadDir(tmpDir)
			if err != nil {
				t.Fatalf("Failed to read temp dir: %v", err)
			}
			
			found := false
			for _, file := range files {
				if len(file.Name()) >= len(corruptPrefix) &&
				   file.Name()[:len(corruptPrefix)] == corruptPrefix {
					found = true
					break
				}
			}
			
			if !found {
				t.Errorf("Corrupt snapshot should be renamed to %s.*", corruptPrefix)
			}
		})
	}
}

// TestAtomicRename tests that the temp file is handled correctly
func TestAtomicRename(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()
	
	pm := NewSnapshotManager(tmpDir)
	store := createTestStore()
	
	// Save snapshot
	err := pm.Save(store)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	
	// Verify temp file is gone (should have been renamed)
	tempPath := pm.tempPath()
	if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
		t.Errorf("Temp file should not exist after save, but it does: %v", err)
	}
	
	// Verify snapshot exists and is readable
	snapshotPath := pm.snapshotPath()
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("Failed to read snapshot: %v", err)
	}
	
	if len(data) == 0 {
		t.Error("Snapshot file is empty")
	}
}

// TestSavePerformance tests saving with many keys (optional, can skip for now)
func TestSavePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}
	
	tmpDir, cleanup := setupTest(t)
	defer cleanup()
	
	pm := NewSnapshotManager(tmpDir)
	store := storage.NewStore()
	
	// Add 10,000 keys
	for i := 0; i < 10000; i++ {
		key := string(rune('a' + (i % 26)))
		value := []byte("value" + string(rune(i)))
		store.Set(key, value)
		
		if i%2 == 0 {
			store.Expire(key, int64(i%100)) // some keys expire
		}
	}
	
	// Time the save
	start := time.Now()
	err := pm.Save(store)
	duration := time.Since(start)
	
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	
	t.Logf("Saved 10,000 keys in %v", duration)
	
	// Load and verify count
	loadedStore, wasLoaded, err := pm.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	if !wasLoaded {
		t.Error("Load should have loaded the snapshot")
	}
	
	if loadedStore.Size() != store.Size() {
		t.Errorf("Size mismatch: original %d, loaded %d", store.Size(), loadedStore.Size())
	}
}

// TestLoadWithoutSnapshot tests loading when no snapshot exists (additional test)
func TestLoadWithoutSnapshot(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()
	
	// Ensure directory exists but is empty
	pm := NewSnapshotManager(tmpDir)
	
	store, wasLoaded, err := pm.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	if wasLoaded {
		t.Error("wasLoaded should be false when no snapshot exists")
	}
	
	if store.Size() != 0 {
		t.Errorf("Store should be empty, got %d keys", store.Size())
	}
	
	// Verify snapshot file doesn't exist
	snapshotPath := filepath.Join(tmpDir, snapshotFile)
	if _, err := os.Stat(snapshotPath); !os.IsNotExist(err) {
		t.Error("Snapshot file should not exist after load")
	}
}

// TestSaveEmptyStore tests saving an empty store
func TestSaveEmptyStore(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()
	
	pm := NewSnapshotManager(tmpDir)
	store := storage.NewStore()
	
	// Save empty store
	err := pm.Save(store)
	if err != nil {
		t.Fatalf("Save of empty store failed: %v", err)
	}
	
	// Verify snapshot file exists and is valid
	snapshotPath := pm.snapshotPath()
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("Failed to read snapshot: %v", err)
	}
	
	// Empty snapshot should be a valid file (could be empty or contain no entries)
	// Our writeSnapshot creates an empty file which is fine
	if len(data) != 0 {
		t.Logf("Empty snapshot size: %d bytes (expected 0)", len(data))
	}
	
	// Load should work
	loadedStore, wasLoaded, err := pm.Load()
	if err != nil {
		t.Fatalf("Load of empty snapshot failed: %v", err)
	}
	
	if !wasLoaded {
		t.Error("Load should have loaded the empty snapshot")
	}
	
	if loadedStore.Size() != 0 {
		t.Errorf("Loaded store should be empty, got %d keys", loadedStore.Size())
	}
}

// TestSaveWithExpiration tests that expiration times are preserved
func TestSaveWithExpiration(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()
	
	pm := NewSnapshotManager(tmpDir)
	store := storage.NewStore()
	
	// Add keys with various expiration states
	store.SetWithExpiration("expires", []byte("value"), 100)
	store.Set("noexpire", []byte("value"))
	store.SetWithExpiration("expired", []byte("value"), 1)
	
	// Wait for "expired" to expire
	time.Sleep(2 * time.Second)
	
	// Save should only save non-expired keys
	err := pm.Save(store)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	
	// Load and verify
	loadedStore, wasLoaded, err := pm.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	if !wasLoaded {
		t.Error("Load should have loaded the snapshot")
	}
	
	// Should have 2 keys: "expires" and "noexpire"
	if loadedStore.Size() != 2 {
		t.Errorf("Loaded store should have 2 keys, got %d", loadedStore.Size())
	}
	
	// Check TTL for "expires" (should be about 100 seconds)
	ttl := loadedStore.TTL("expires")
	if ttl <= 0 || ttl > 100 {
		t.Errorf("TTL for 'expires': expected ~100, got %d", ttl)
	}
	
	// "noexpire" should have no TTL
	ttl = loadedStore.TTL("noexpire")
	if ttl != -1 {
		t.Errorf("TTL for 'noexpire': expected -1, got %d", ttl)
	}
	
	// "expired" should not exist
	_, exists := loadedStore.Get("expired")
	if exists {
		t.Error("Expired key should not exist after load")
	}
}

// TestConcurrentSave tests saving while other operations are happening
func TestConcurrentSave(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}
	
	tmpDir, cleanup := setupTest(t)
	defer cleanup()
	
	pm := NewSnapshotManager(tmpDir)
	store := storage.NewStore()
	
	// Start goroutines that continuously modify the store
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := string(rune('a' + id))
				store.Set(key, []byte("value"))
				store.Get(key)
				if j%10 == 0 {
					store.Expire(key, 10)
				}
			}
			done <- true
		}(i)
	}
	
	// Wait a bit for modifications to start
	time.Sleep(100 * time.Millisecond)
	
	// Save while operations are in progress
	err := pm.Save(store)
	if err != nil {
		t.Fatalf("Save during concurrent operations failed: %v", err)
	}
	
	// Wait for goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Load and verify we have some data
	loadedStore, wasLoaded, err := pm.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	if !wasLoaded {
		t.Error("Load should have loaded the snapshot")
	}
	
	// Should have at least some keys
	if loadedStore.Size() == 0 {
		t.Error("Loaded store should not be empty")
	}
}

// TestMultipleSaves tests saving multiple times
func TestMultipleSaves(t *testing.T) {
	tmpDir, cleanup := setupTest(t)
	defer cleanup()
	
	pm := NewSnapshotManager(tmpDir)
	store := storage.NewStore()
	
	// Save initial state
	store.Set("key1", []byte("value1"))
	err := pm.Save(store)
	if err != nil {
		t.Fatalf("First save failed: %v", err)
	}
	
	// Modify and save again
	store.Set("key2", []byte("value2"))
	err = pm.Save(store)
	if err != nil {
		t.Fatalf("Second save failed: %v", err)
	}
	
	// Modify and save a third time
	store.Set("key3", []byte("value3"))
	store.Delete("key1")
	err = pm.Save(store)
	if err != nil {
		t.Fatalf("Third save failed: %v", err)
	}
	
	// Load and verify final state
	loadedStore, wasLoaded, err := pm.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	if !wasLoaded {
		t.Error("Load should have loaded the snapshot")
	}
	
	if loadedStore.Size() != 2 {
		t.Errorf("Store should have 2 keys, got %d", loadedStore.Size())
	}
	
	_, ok := loadedStore.Get("key1")
	if ok {
		t.Error("key1 should not exist (was deleted)")
	}
	
	_, ok = loadedStore.Get("key2")
	if !ok {
		t.Error("key2 should exist")
	}
	
	_, ok = loadedStore.Get("key3")
	if !ok {
		t.Error("key3 should exist")
	}
}