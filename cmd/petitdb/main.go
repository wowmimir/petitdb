package main

import (
    "fmt"
    "log"

    "github.com/wowmimir/petitdb/internal/config"
    "github.com/wowmimir/petitdb/internal/dispatcher"
    "github.com/wowmimir/petitdb/internal/storage"
)

func main() {
	cfg := config.NewConfig()
	
	// Verify it works
	log.Printf("PetitDB starting on %s:%d", cfg.Bind, cfg.Port)
	log.Printf("Data directory: %s", cfg.Dir)

	store := storage.NewStore()

    // 2. Create the dispatcher with the store
    disp := dispatcher.NewDispatcher(store)

    // 3. Test some commands
    testCommands(disp)
}

func testCommands(disp *dispatcher.Dispatcher) {
    // Helper to print results
    run := func(cmd string, args ...[]byte) {
        result, err := disp.Dispatch(cmd, args)
        if err != nil {
            fmt.Printf("❌ %s %v → Error: %v\n", cmd, argsToString(args), err)
            return
        }
        fmt.Printf("✅ %s %v → %v\n", cmd, argsToString(args), result)
    }

    // Test SET
    run("SET", []byte("name"), []byte("PetitDB"))

    // Test GET
    run("GET", []byte("name"))

    // Test GET on non‑existent key
    run("GET", []byte("unknown"))

    // Test EXISTS
    run("EXISTS", []byte("name"))

    // Test DEL
    run("DEL", []byte("name"))

    // Test EXISTS after delete
    run("EXISTS", []byte("name"))

    // Test unknown command (should give verbose error)
    run("HGET", []byte("name"))
}

// Helper to convert [][]byte to a string representation for logging
func argsToString(args [][]byte) string {
    if len(args) == 0 {
        return "[]"
    }
    out := "["
    for i, arg := range args {
        if i > 0 {
            out += " "
        }
        out += fmt.Sprintf("%q", string(arg))
    }
    out += "]"
    return out
}