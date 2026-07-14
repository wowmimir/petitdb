# PetitDB

**Lightweight, Zero‑Config, Memory‑First State Server for Solo Devs & Small Teams**

[![CI](https://github.com/wowmimir/petitdb/actions/workflows/test.yml/badge.svg)](https://github.com/wowmimir/petitdb/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/badge/Go-1.26.5-blue)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

PetitDB is a simple TCP state server for session storage, API caching, counters, and rate limiting — without the overhead of Redis or Memcached.

- **Single binary** – `go install` → `petitdb` → `SET key value` → done
- **Zero config** – no config files, no authentication
- **Redis‑compatible** – speaks RESP, works with `redis-cli` and Redis clients
- **Ephemeral** – built for temporary state, not persistent databases
- **Crash recovery** – atomic snapshots with loud corruption warnings
- **Pub/Sub** – real‑time messaging between clients
- **No dependencies** – pure Go, stdlib only

---

## Table of Contents

- [Quick Start (30 seconds)](#quick-start-30-seconds)
- [Installation](#installation)
- [Docker](#docker)
- [Configuration](#configuration)
- [Built‑in CLI](#builtin-cli)
- [Supported Commands](#supported-commands)
- [Pub/Sub Example](#pubsub-example)
- [Persistence & Snapshots](#persistence--snapshots)
- [Error Handling & UX](#error-handling--ux)
- [Connect From Any Language](#connect-from-any-language)
- [Development](#development)
- [Deployment](#deployment)
- [Philosophy](#philosophy)
- [Limitations](#limitations)
- [FAQ](#faq)
- [Documentation](#documentation)
- [License](#license)

---

## Quick Start (30 seconds)

```bash
# Install
go install github.com/wowmimir/petitdb@latest

# Start the server (defaults: 127.0.0.1:9379, ./data)
petitdb

# In another terminal, connect with the built-in CLI
petitdb cli
petitdb> SET name "PetitDB"
(string) "OK"
petitdb> GET name
(string) "PetitDB"
```

Or with `redis-cli`:
```bash
redis-cli -p 9379
127.0.0.1:9379> SET name Redis
OK
127.0.0.1:9379> GET name
"Redis"
```

---

## Installation

### Go developers
```bash
go install github.com/wowmimir/petitdb@latest
```
This installs the `petitdb` binary in `$GOPATH/bin` (or `~/go/bin`).

### Pre‑built binaries
Download the latest binary for your platform from [GitHub Releases](https://github.com/wowmimir/petitdb/releases).

```bash
# Linux/macOS
curl -LO https://github.com/wowmimir/petitdb/releases/latest/download/petitdb-linux-amd64
chmod +x petitdb-linux-amd64
./petitdb-linux-amd64

# Windows (PowerShell)
Invoke-WebRequest -Uri https://github.com/wowmimir/petitdb/releases/latest/download/petitdb-windows-amd64.exe -OutFile petitdb.exe
.\petitdb.exe
```

### From source
```bash
git clone https://github.com/wowmimir/petitdb.git
cd petitdb

# Using Make (Linux/macOS)
make build

# Using PowerShell (Windows)
.\build.ps1 build
```

---

## Docker

### Quick Start

```bash
docker pull wowmimir/petitdb:latest
docker run -p 9379:9379 -v petitdb_data:/data wowmimir/petitdb:latest
```

### Customizing the Container

You can override any of the default flags by appending them to the `docker run` command:

```bash
# Custom port (container listens on 9380)
docker run -p 9380:9380 -v petitdb_data:/data wowmimir/petitdb:latest --port 9380

# Custom data directory (mount your own path)
docker run -p 9379:9379 -v /my/custom/path:/data wowmimir/petitdb:latest --dir /data

# Bind to localhost only (for security)
docker run -p 127.0.0.1:9379:9379 -v petitdb_data:/data wowmimir/petitdb:latest --bind 127.0.0.1

# Combine multiple customizations
docker run -d \
  --name petitdb \
  -p 9380:9380 \
  -v /home/user/petitdb_data:/data \
  wowmimir/petitdb:latest \
  --port 9380 \
  --bind 0.0.0.0 \
  --dir /data
```

### Understanding Port Mapping

| Setting | Meaning |
|---------|---------|
| `-p 9380:9379` | Host listens on port 9380, container listens on 9379 |
| `--port 9380` | Container listens on port 9380 (override default) |

If you change the container's internal port with `--port`, you must also update the port mapping:

```bash
# Container listens on 9380 → map host 9380 to container 9380
docker run -p 9380:9380 -v petitdb_data:/data wowmimir/petitdb:latest --port 9380
```

### Persisting Data

PetitDB stores snapshots in the directory specified by `--dir` (default: `/data`). To persist data across container restarts:

```bash
# Using a named volume
docker run -p 9379:9379 -v petitdb_data:/data wowmimir/petitdb:latest

# Using a host directory
docker run -p 9379:9379 -v /absolute/path/on/host:/data wowmimir/petitdb:latest

# Using a relative path (PowerShell/Unix)
docker run -p 9379:9379 -v $(pwd)/data:/data wowmimir/petitdb:latest
```

### Multi-Architecture Support

The Docker image supports both `linux/amd64` and `linux/arm64` architectures. Docker automatically selects the correct version for your system.

### Production Deployment with Docker Compose

Create `docker-compose.yml`:

```yaml
services:
  petitdb:
    image: wowmimir/petitdb:latest
    container_name: petitdb
    ports:
      - "9379:9379"
    volumes:
      - ./petitdb-data:/data
    restart: unless-stopped
    command: ["--bind", "0.0.0.0", "--dir", "/data"]
```

Start with:

```bash
docker-compose up -d
```

### Verify the Container

```bash
# Check logs
docker logs petitdb

# Test connection
redis-cli -p 9379 PING

# Check data directory
docker exec petitdb ls -la /data
```

---

## Configuration

PetitDB requires **no configuration file** – but you can override the default settings with command‑line flags:

```bash
petitdb --port 9380 --bind 0.0.0.0 --dir /var/lib/petitdb
```

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `9379` | TCP port to listen on (avoids Redis port 6379) |
| `--bind` | `127.0.0.1` | Address to bind to (use `0.0.0.0` to expose externally) |
| `--dir` | `./data` | Directory for snapshot files (relative or absolute path) |

> **Security:** Binding to `0.0.0.0` exposes PetitDB to your network. Consider firewall rules or use a reverse proxy if needed.

---

## Built‑in CLI

PetitDB includes a **first‑class REPL** for interactive use:

```bash
petitdb cli
```

Features:
- **Command history** – use ↑/↓ arrows to recall previous commands.
- **Pretty‑printed responses** – shows types and values clearly.
- **Auto‑reconnect** – if the server restarts, the CLI reconnects automatically.

Example session:
```
petitdb> SET counter 1
(integer) 1
petitdb> INCR counter
-ERR unknown command 'INCR'. PetitDB v1 supports: PING, SET, GET, DEL, EXISTS, EXPIRE, TTL, SAVE, SUBSCRIBE, PUBLISH, INFO
petitdb> GET counter
(string) "1"
petitdb> EXPIRE counter 10
(integer) 1
petitdb> TTL counter
(integer) 8
```

> **Note:** The CLI connects to `127.0.0.1:9379` by default. Use `petitdb cli --host 0.0.0.0 --port 9380` to connect to a custom server.

---

## Supported Commands

| Command | Description | Example |
|---------|-------------|---------|
| `PING` | Health check | `PING` → `"PONG"` |
| `SET key value` | Store a string value | `SET name Alice` |
| `GET key` | Retrieve a string value | `GET name` → `"Alice"` |
| `DEL key` | Delete a key | `DEL name` |
| `EXISTS key` | Check if key exists | `EXISTS name` → `1` (exists) / `0` |
| `EXPIRE key seconds` | Set TTL (seconds) | `EXPIRE name 60` |
| `TTL key` | Get remaining TTL | `TTL name` → `45` (seconds) or `-1` (no expiry) |
| `SAVE` | Manually trigger a snapshot | `SAVE` → `"OK"` |
| `SUBSCRIBE topic...` | Subscribe to one or more topics | `SUBSCRIBE news alerts` |
| `PUBLISH topic message` | Send a message to a topic | `PUBLISH news "Hello world"` |
| `INFO` | Runtime statistics (uptime, clients, keys, commands, pub/sub stats) | `INFO` → returns a multi‑line string |

---

## Pub/Sub Example

PetitDB supports real‑time messaging between clients.

**Terminal 1 (Subscriber):**
```bash
petitdb cli
petitdb> SUBSCRIBE news
(string) "SUBSCRIBE" "news" (integer) 1
# Waiting for messages...
```

**Terminal 2 (Publisher):**
```bash
petitdb cli
petitdb> PUBLISH news "Breaking: PetitDB v1.0 released!"
(integer) 1
```

**Terminal 1 receives:**
```
(string) "message" "news" "Breaking: PetitDB v1.0 released!"
```

> **Note:** Once a client subscribes, it enters "subscriber mode" – it stops accepting commands and only forwards messages. Use a separate connection for commands.

---

## Persistence & Snapshots

PetitDB uses **atomic snapshots** for crash recovery – not as a backup solution.

- **Manual save:** Use the `SAVE` command to trigger a snapshot at any time.
- **Atomic write:** Writes to `snapshot.tmp`, then atomically renames to `snapshot`.
- **Auto‑load on startup:** If a valid `snapshot` exists, it's loaded automatically.
- **Corruption handling:** If the snapshot is corrupted, the server:
  - Logs a **large ASCII‑art warning**.
  - Renames the corrupt file to `snapshot.corrupt.<timestamp>` for inspection.
  - Starts with an **empty state** – availability first.

**Why no auto‑save?**  
PetitDB is designed for **ephemeral** state. Auto‑save adds I/O overhead and complexity. You decide when to persist.

---

## Error Handling & UX

PetitDB is designed to be **friendly** and **helpful**.

- **Verbose unknown command errors:** If you type an unsupported command, the server replies with the full list of supported v1 commands – no more "broken Redis" confusion.
  ```
  -ERR unknown command 'INCR'. PetitDB v1 supports: PING, SET, GET, DEL, EXISTS, EXPIRE, TTL, SAVE, SUBSCRIBE, PUBLISH, INFO
  ```
- **Loud corruption warnings:** Snapshot corruption is impossible to miss – the server prints a block of `!` characters and a clear message.
- **No silent failures:** Invalid clients (malformed RESP) are disconnected with an error, but the server never crashes.

---

## Connect From Any Language

PetitDB speaks RESP (Redis Protocol). Use your favorite Redis client!

### Python
```python
import redis
r = redis.Redis(host='localhost', port=9379, decode_responses=True)
r.set('key', 'value')
print(r.get('key'))  # 'value'
```

### Node.js (ioredis)
```javascript
const Redis = require('ioredis');
const client = new Redis(9379, 'localhost');
await client.set('key', 'value');
console.log(await client.get('key')); // 'value'
```

### Go (go-redis)
```go
import "github.com/redis/go-redis/v9"

rdb := redis.NewClient(&redis.Options{
    Addr: "localhost:9379",
})
err := rdb.Set(ctx, "key", "value", 0).Err()
val, err := rdb.Get(ctx, "key").Result()
```

---

## Development

PetitDB is built with **zero external dependencies** – just Go.

### Prerequisites
- Go 1.26.5+
- (Optional) `make` for Linux/macOS, or use `build.ps1` on Windows.

### Build from source
```bash
# Using Make
make build

# Using PowerShell
.\build.ps1 build

# Directly with go
go build -o petitdb ./cmd/petitdb
```

### Run tests
```bash
make test      # or .\build.ps1 test
```

### Format and vet
```bash
make fmt vet   # or .\build.ps1 fmt, .\build.ps1 vet
```

### Build Docker image
```bash
docker build --build-arg VERSION=$(git describe --tags) -t wowmimir/petitdb:latest .
```

---

## Deployment

See [DEPLOYMENT.md](./DEPLOYMENT.md) for Docker Compose, Render, Fly.io, and systemd examples.

Quick production‑ready Docker Compose:
```yaml
services:
  petitdb:
    image: wowmimir/petitdb:latest
    ports:
      - "9379:9379"
    volumes:
      - ./petitdb-data:/data
    restart: unless-stopped
    command: ["--bind", "0.0.0.0", "--dir", "/data"]
```

---

## Philosophy

- **Not a database** – PostgreSQL/MySQL remain the source of truth.
- **Ephemeral state** – session storage, API caching, rate limiting.
- **Zero operations** – no config, no auth, no debugging headaches.
- **Crash recovery, not durability** – snapshots for restart, not backups.
- **Simplicity first** – no Lua scripting, no modules, no clustering.

---

## Limitations

| Aspect | Limit |
|--------|-------|
| Key length | ≤ 256 characters |
| Value size | Limited by available memory |
| Max concurrent clients | Limited by OS file descriptors (typically > 1000) |
| Data types | Strings only (no lists, hashes, sets) |
| Persistence | Manual snapshots only (no AOF/WAL) |
| Authentication | None (use network isolation) |

---

## FAQ

**Is PetitDB a database?**  
No – it's an ephemeral state server. Use a real database (PostgreSQL, MySQL) for long‑term storage.

**How do I persist data across restarts?**  
Use the `SAVE` command to create a snapshot. On startup, PetitDB loads the snapshot automatically.

**Can I use PetitDB in production?**  
Yes, for low‑stakes ephemeral state (sessions, caching, rate limiting). For critical data, use a proper database.

**What happens if the snapshot is corrupted?**  
PetitDB logs a loud warning, renames the corrupt file, and starts with an empty state – so your service stays up.

**Does PetitDB support TLS/SSL?**  
No – use a reverse proxy (like nginx) or a tunnel for encryption.

**Is there a web dashboard?**  
No – PetitDB is a minimalist TCP server.

---

## Documentation

- [Installation](./INSTALL.md)
- [Deployment](./DEPLOYMENT.md)
- [Commands](./COMMANDS.md)
- [Architecture](./ARCHITECTURE.md)
- [Changelog](./CHANGELOG.md)
- [Operations Handbook](./OPS.md) – for maintainers

---

## License

MIT © [wowmimir](https://github.com/wowmimir)