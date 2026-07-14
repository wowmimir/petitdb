package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/wowmimir/petitdb/internal/cli"
	"github.com/wowmimir/petitdb/internal/config"
	"github.com/wowmimir/petitdb/internal/dispatcher"
	"github.com/wowmimir/petitdb/internal/persistence"
	"github.com/wowmimir/petitdb/internal/pubsub"
	"github.com/wowmimir/petitdb/internal/server"
	"github.com/wowmimir/petitdb/internal/storage"
)

// Version is set at build time via -ldflags
var Version = "dev"

func main() {
	// Check for version flag (before anything else)
	if len(os.Args) >= 2 {
		arg := os.Args[1]
		if arg == "--version" || arg == "-v" {
			fmt.Printf("petitdb version %s\n", Version)
			os.Exit(0)
		}
	}

	// Check for CLI subcommand
	if len(os.Args) >= 2 && os.Args[1] == "cli" {
		cliFlags := flag.NewFlagSet("cli", flag.ExitOnError)
		host := cliFlags.String("host", "127.0.0.1", "server host")
		port := cliFlags.Int("port", 9379, "server port")
		cliFlags.Parse(os.Args[2:])
		cli.Run(*host, *port)
		return
	}

	// Otherwise start server
	cfg := config.NewConfig()

	pm := persistence.NewSnapshotManager(cfg.Dir)

	// Create the pub/sub broker
	broker := pubsub.NewBroker()

	// Load snapshot
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

	// Create dispatcher with save function and broker
	disp := dispatcher.NewDispatcher(store, broker, func() error {
		return pm.Save(store)
	})

	// Create server with broker
	srv := server.NewServer(cfg, disp, store, broker)

	// Inject server as StatsProvider for INFO command
	disp.SetStatsProvider(srv)

	// Start server
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	// Graceful shutdown
	srv.Shutdown()
}
