# Installation

PetitDB is distributed as a single Go binary with no external dependencies. Choose the method that suits your environment.

---

## Quick Links

- [Go Install](#go-install)
- [Pre‑built Binaries](#prebuilt-binaries)
- [Docker](#docker)
- [Build from Source](#build-from-source)
- [Verify Installation](#verify-installation)

---

## Go Install

If you have Go 1.26.5+ installed:

```bash
go install github.com/wowmimir/petitdb@latest
```

The binary will be placed in `$GOPATH/bin` (or `~/go/bin`). Ensure this directory is in your `PATH`.

---

## Pre‑built Binaries

Download the latest binary for your platform from the [GitHub Releases](https://github.com/wowmimir/petitdb/releases) page.

### Linux (amd64)
```bash
curl -LO https://github.com/wowmimir/petitdb/releases/latest/download/petitdb-linux-amd64
chmod +x petitdb-linux-amd64
sudo mv petitdb-linux-amd64 /usr/local/bin/petitdb
```

### Linux (arm64)
```bash
curl -LO https://github.com/wowmimir/petitdb/releases/latest/download/petitdb-linux-arm64
chmod +x petitdb-linux-arm64
sudo mv petitdb-linux-arm64 /usr/local/bin/petitdb
```

### macOS (amd64 / Intel)
```bash
curl -LO https://github.com/wowmimir/petitdb/releases/latest/download/petitdb-darwin-amd64
chmod +x petitdb-darwin-amd64
sudo mv petitdb-darwin-amd64 /usr/local/bin/petitdb
```

### macOS (arm64 / Apple Silicon)
```bash
curl -LO https://github.com/wowmimir/petitdb/releases/latest/download/petitdb-darwin-arm64
chmod +x petitdb-darwin-arm64
sudo mv petitdb-darwin-arm64 /usr/local/bin/petitdb
```

### Windows (amd64)
```powershell
Invoke-WebRequest -Uri https://github.com/wowmimir/petitdb/releases/latest/download/petitdb-windows-amd64.exe -OutFile petitdb.exe
```

Move the executable to a directory in your `PATH` (e.g., `C:\Program Files\PetitDB`).

---

## Docker

### Pull the image
```bash
docker pull wowmimir/petitdb:latest
```

### Run the container
```bash
docker run -p 9379:9379 -v petitdb_data:/data wowmimir/petitdb:latest
```

For detailed Docker configuration, see [Docker section in README](../README.md#docker).

---

## Build from Source

### Prerequisites
- Go 1.26.5+
- (Optional) `make` or `build.ps1` for convenience.

### Clone the repository
```bash
git clone https://github.com/wowmimir/petitdb.git
cd petitdb
```

### Build

**Using Make (Linux/macOS):**
```bash
make build
```

**Using PowerShell (Windows):**
```powershell
.\build.ps1 build
```

**Direct Go build:**
```bash
go build -o petitdb ./cmd/petitdb
```

The binary will be placed in the current directory.

---

## Verify Installation

Check the version:
```bash
petitdb --version
```

Start the server in the background:
```bash
petitdb &
```

Connect with the built‑in CLI:
```bash
petitdb cli
petitdb> PING
(string) "PONG"
```

If you see `PONG`, installation is successful.

---

## Uninstalling

- **Go install:** Delete the binary from `$GOPATH/bin`.
- **Pre‑built:** Remove the file from your `PATH`.
- **Docker:** `docker rmi wowmimir/petitdb:latest` and remove volumes.
- **Source:** Delete the cloned repository and the built binary.
---


