package main

import (
	"log"

	"github.com/wowmimir/petitdb/internal/config"
)

func main() {
	cfg := config.NewConfig()
	
	// Verify it works
	log.Printf("PetitDB starting on %s:%d", cfg.Bind, cfg.Port)
	log.Printf("Data directory: %s", cfg.Dir)
}