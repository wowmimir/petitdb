package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Port int
	Dir  string
	Bind string
}

func (c *Config) validate() {
	// Validate port range
	if c.Port < 1 || c.Port > 65535 {
		fmt.Printf("WARNING: Invalid port %d, using default 8080\n", c.Port)
		c.Port = 9379
	}

	// Validate directory exists
	if c.Dir != "" {
		absPath, err := filepath.Abs(c.Dir)
		if err == nil {
			c.Dir = absPath
		}

		if _, err := os.Stat(c.Dir); os.IsNotExist(err) {
			fmt.Printf("WARNING: Directory %s does not exist, creating...\n", c.Dir)
			if err := os.MkdirAll(c.Dir, 0755); err != nil {
				fmt.Printf("ERROR: Failed to create directory: %v\n", err)
			}
		}
	}
}

func NewConfig() *Config {
	cfg := &Config{}

	flag.IntVar(&cfg.Port, "port", 9379, "Server port to listen on")
	flag.StringVar(&cfg.Bind, "bind", "127.0.0.1", "IP address to bind to")
	flag.StringVar(&cfg.Dir, "dir", ".", "Working directory for the application")
	flag.Parse()

	cfg.validate()

	return cfg

}
