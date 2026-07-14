# Makefile for PetitDB

# ------------------------------------------------------------
# Variables
# ------------------------------------------------------------

BINARY := petitdb
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-X main.Version=$(VERSION)"
MAIN_PKG := ./cmd/petitdb
GO := go

# Runtime configuration (can be overridden)
PORT ?= 9379
BIND ?= 127.0.0.1
DIR ?= ./data

# ------------------------------------------------------------
# Targets
# ------------------------------------------------------------

.PHONY: build
build:
	$(GO) build $(LDFLAGS) -o $(BINARY) $(MAIN_PKG)

.PHONY: test
test:
	$(GO) test -race -v ./...

.PHONY: clean
clean:
	rm -f $(BINARY)
	rm -f snapshot snapshot.tmp snapshot.corrupt.*
	rm -rf data/

.PHONY: run
run:
	$(GO) run $(MAIN_PKG) --port $(PORT) --bind $(BIND) --dir $(DIR)

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build  - Build the binary (version injected from git tag)"
	@echo "  test   - Run all tests with race detector"
	@echo "  clean  - Remove binaries, snapshots, and data/"
	@echo "  run    - Start the server (flags: PORT, BIND, DIR)"
	@echo "  fmt    - Format source code"
	@echo "  vet    - Run go vet"
	@echo ""
	@echo "Examples:"
	@echo "  make run PORT=9380 BIND=0.0.0.0 DIR=/tmp/data"
	@echo "  make build VERSION=v1.2.3   (override version)"