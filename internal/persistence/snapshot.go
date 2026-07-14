package persistence

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wowmimir/petitdb/internal/storage"
)

const (
	snapshotFile     = "snapshot"
	snapshotTempFile = "snapshot.tmp"
	corruptPrefix    = "snapshot.corrupt"
)

// SnapshotManager handles persistence operations
type SnapshotManager struct {
	dir string
}

// NewSnapshotManager creates a new manager with the given data directory
func NewSnapshotManager(dataDir string) *SnapshotManager {
	return &SnapshotManager{
		dir: dataDir,
	}
}

// snapshotPath returns the full path to the snapshot file
func (sm *SnapshotManager) snapshotPath() string {
	return filepath.Join(sm.dir, snapshotFile)
}

// tempPath returns the full path to the temporary snapshot file
func (sm *SnapshotManager) tempPath() string {
	return filepath.Join(sm.dir, snapshotTempFile)
}

// Load attempts to load the snapshot from disk
// Returns (store, wasLoaded, error)
// If snapshot is corrupt, it logs ASCII art warnings, renames the file, and returns an empty store
func (sm *SnapshotManager) Load() (*storage.Store, bool, error) {
	snapshotPath := sm.snapshotPath()

	// Check if snapshot exists
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		log.Println("No snapshot found, starting with empty state")
		return storage.NewStore(), false, nil
	}

	// Attempt to load and parse
	store, err := sm.loadSnapshot(snapshotPath)
	if err != nil {
		// CORRUPTION DETECTED - Loud and unmissable
		sm.handleCorruption(snapshotPath)
		// Return empty store, but continue startup
		return storage.NewStore(), false, nil
	}

	log.Printf("Snapshot loaded successfully: %d keys restored", store.Size())
	return store, true, nil
}

// loadSnapshot reads and parses a snapshot file
func (sm *SnapshotManager) loadSnapshot(path string) (*storage.Store, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open snapshot: %w", err)
	}
	defer file.Close()

	store := storage.NewStore()
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // Skip empty lines
		}

		// Parse format: key|base64(value)|expires_at
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid format at line %d: expected 3 parts, got %d", lineNum, len(parts))
		}

		key := parts[0]
		if key == "" {
			return nil, fmt.Errorf("empty key at line %d", lineNum)
		}

		// Decode base64 value
		valueBytes, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid base64 at line %d: %w", lineNum, err)
		}

		// Parse expiration timestamp
		expiresAt, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid expiration at line %d: %w", lineNum, err)
		}

		// Store in the store (we'll use SetWithExpiration if needed, but we need to bypass
		// the automatic expiration check since we're loading)
		// We'll directly insert into the map
		store.SetRaw(key, valueBytes, expiresAt)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading snapshot: %w", err)
	}

	return store, nil
}

// handleCorruption logs ASCII art warnings and renames the corrupt file
func (sm *SnapshotManager) handleCorruption(snapshotPath string) {
	timestamp := time.Now().Unix()
	corruptPath := filepath.Join(sm.dir, fmt.Sprintf("%s.%d", corruptPrefix, timestamp))

	// Log loud ASCII art warning
	log.Println(`
!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
!                                                          !
!   ⚠️  WARNING: SNAPSHOT CORRUPTED  ⚠️                    !
!                                                          !
!   The snapshot file could not be loaded.                 !
!   Starting with EMPTY state.                             !
!   All previous data has been LOST.                       !
!                                                          !
!   Corrupt file renamed to: ` + filepath.Base(corruptPath) + `
!                                                          !
!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!`)

	// Rename corrupt file for forensic inspection
	if err := os.Rename(snapshotPath, corruptPath); err != nil {
		log.Printf("ERROR: Failed to rename corrupt snapshot: %v", err)
	} else {
		log.Printf("Corrupt snapshot preserved as: %s", corruptPath)
	}
}

// Save writes the current state to a snapshot atomically
func (sm *SnapshotManager) Save(store *storage.Store) error {
	tempPath := sm.tempPath()
	snapshotPath := sm.snapshotPath()

	// Write to temporary file first
	if err := sm.writeSnapshot(tempPath, store); err != nil {
		return fmt.Errorf("failed to write temporary snapshot: %w", err)
	}

	// Atomically rename temp file to final location
	if err := os.Rename(tempPath, snapshotPath); err != nil {
		return fmt.Errorf("failed to atomic rename snapshot: %w", err)
	}

	log.Printf("Snapshot saved: %d keys written", store.Size())
	return nil
}

// writeSnapshot writes the store data to a file
func (sm *SnapshotManager) writeSnapshot(path string, store *storage.Store) error {
	// Get all key-value pairs from the store
	entries := store.GetAll()

	// Create file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create snapshot file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Write each entry
	for _, entry := range entries {
		// Encode value as base64 to avoid issues with newlines and special characters
		encodedValue := base64.StdEncoding.EncodeToString(entry.Value)

		// Format: key|base64(value)|expires_at
		line := fmt.Sprintf("%s|%s|%d\n", entry.Key, encodedValue, entry.ExpiresAt)

		if _, err := writer.WriteString(line); err != nil {
			return fmt.Errorf("failed to write entry: %w", err)
		}
	}

	return nil
}
