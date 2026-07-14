# Installation

PetitDB is a single binary with no dependencies. You can use it immediately after downloading – no installation required unless you want it in your `PATH`.

---

## Quick Links

- [Download a Pre‑built Binary](#prebuilt-binaries)
- [Install via Go](#go-install)
- [Use Docker](#docker)
- [Build from Source](#build-from-source)
- [Verify Installation](#verify-installation)

---

## Pre‑built Binaries

### Quick Start (Run Without Installing)

**Linux/macOS:**
```bash
curl -LO https://github.com/wowmimir/petitdb/releases/latest/download/petitdb-linux-amd64
chmod +x petitdb-linux-amd64
./petitdb-linux-amd64
```

**Windows (PowerShell):**
```powershell
Invoke-WebRequest -Uri https://github.com/wowmimir/petitdb/releases/latest/download/petitdb-windows-amd64.exe -OutFile petitdb.exe
.\petitdb.exe
```

### Make It Globally Available (Optional)

To run `petitdb` from anywhere without `./`:

**Linux/macOS:**
```bash
# Move to a directory in your PATH (e.g., /usr/local/bin)
sudo mv petitdb-linux-amd64 /usr/local/bin/petitdb

# Or add the current directory to PATH (temporary)
export PATH=$PATH:$(pwd)

# Or add to PATH permanently (add to ~/.bashrc or ~/.zshrc)
echo 'export PATH=$PATH:/path/to/petitdb' >> ~/.bashrc
```

**Windows:**
1. Move `petitdb.exe` to a folder like `C:\Program Files\PetitDB\`
2. Add that folder to your system PATH:
   - Open System Properties → Environment Variables
   - Add `C:\Program Files\PetitDB` to `Path`

### Platform-Specific Downloads

| Platform | Architecture | Download |
|----------|--------------|----------|
| Linux | amd64 | `petitdb-linux-amd64` |
| Linux | arm64 | `petitdb-linux-arm64` |
| macOS | amd64 (Intel) | `petitdb-darwin-amd64` |
| macOS | arm64 (Apple Silicon) | `petitdb-darwin-arm64` |
| Windows | amd64 | `petitdb-windows-amd64.exe` |

---

## Go Install

If you have Go 1.26.5+ installed:

```bash
go install github.com/wowmimir/petitdb/cmd/petitdb@latest
```

This places the binary in `$GOPATH/bin` (or `~/go/bin`). **This directory is usually in your PATH** – so you can run `petitdb` immediately.

If it's not in your PATH, add it:
```bash
# Linux/macOS
export PATH=$PATH:$(go env GOPATH)/bin

# Windows (PowerShell)
$env:Path += ";$(go env GOPATH)\bin"
```

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

No `PATH` needed – Docker handles everything.

---

## Build from Source

### Prerequisites
- Go 1.26.5+
- (Optional) `make` or `build.ps1` for convenience.

### Clone and Build
```bash
git clone https://github.com/wowmimir/petitdb.git
cd petitdb

# Using Make (Linux/macOS)
make build

# Using PowerShell (Windows)
.\build.ps1 build

# Direct Go build
# Linux/macOS
go build -o petitdb ./cmd/petitdb
# Windows
go build -o petitdb.exe ./cmd/petitdb
```

The binary is now in your current directory. Run it with `./petitdb` (or `.\petitdb.exe` on Windows).

---

## Verify Installation

### Quick Test (Run Without Installing)

**Linux/macOS:**
```bash
./petitdb --version
./petitdb &
./petitdb cli
petitdb> PING
(string) "PONG"
```

**Windows:**
```powershell
.\petitdb.exe --version
Start-Process .\petitdb.exe
.\petitdb.exe cli
petitdb> PING
(string) "PONG"
```

### If Installed Globally (in PATH)

```bash
petitdb --version
petitdb &
petitdb cli
petitdb> PING
```

---

## Uninstalling

- **Binary:** Delete the file.
- **Go install:** `rm $(go env GOPATH)/bin/petitdb`
- **Docker:** `docker rmi wowmimir/petitdb:latest`

---

## Summary

| Method | Global? | PATH Required? |
|--------|---------|----------------|
| Go install | ✅ Yes (if `$GOPATH/bin` in PATH) | No, usually already in PATH |
| Download + `./` | ❌ No | No |
| Download + move to `/usr/local/bin` | ✅ Yes | Yes (but it's already there) |
| Docker | ❌ No (container-only) | No |
---