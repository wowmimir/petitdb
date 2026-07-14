package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wowmimir/petitdb/internal/config"
	"github.com/wowmimir/petitdb/internal/dispatcher"
	"github.com/wowmimir/petitdb/internal/persistence"
	"github.com/wowmimir/petitdb/internal/server"
	"github.com/wowmimir/petitdb/internal/storage"
)

func main() {
	// Parse flags and load config
	cfg := config.NewConfig()

	// Create persistence manager
	pm := persistence.NewSnapshotManager(cfg.Dir)

	// Load snapshot (or start empty)
	store, wasLoaded, err := pm.Load()
	if err != nil {
		log.Printf("Warning: unexpected error loading snapshot: %v", err)
		store = storage.NewStore()
	}

	if wasLoaded {
		log.Printf("Restored %d keys from snapshot", store.Size())
	} else {
		log.Println("Started with empty state")
	}

	// Create dispatcher with save function
	disp := dispatcher.NewDispatcher(store, func() error {
		return pm.Save(store)
	})

	// Create server
	srv := server.NewServer(cfg, disp, store)

	// Start server in a goroutine so we can listen for signals
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal (Ctrl+C)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	// Graceful shutdown
	srv.Shutdown()
}