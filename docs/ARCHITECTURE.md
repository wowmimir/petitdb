# Architecture

A high‑level overview of PetitDB's internal design.

---

## Core Principles

- **Single binary, zero config** – defaults that work out‑of‑the‑box.
- **Ephemeral state server** – temporary data only.
- **Concurrent and safe** – uses goroutines and mutexes.
- **Simple and maintainable** – no external dependencies.

---

## System Overview

```
┌─────────────┐
│   TCP       │
│   Server    │──▶ RESP Parser ──▶ Dispatcher ──┬──▶ Storage Engine
│             │                                 ├──▶ Pub/Sub Engine
└─────────────┘                                 └──▶ Persistence Manager
      ▲
      │
 CLI (REPL)
```

---

## Components

### 1. TCP Server (`internal/server`)
- Listens on `--bind`:`--port` (default `127.0.0.1:9379`).
- One goroutine per client (`handleConn`).
- Graceful shutdown using `context.Context` and `sync.WaitGroup`.
- Logs client connects/disconnects.

### 2. RESP Protocol (`internal/protocol/resp`)
- **Parser:** Reads from `bufio.Reader`, parses RESP arrays, bulk strings, integers, and simple strings.
- **Serializer:** Converts Go types (`string`, `int`, `[]byte`, `error`, `nil`) into RESP responses.
- **Error Handling:** Returns clear messages for malformed input.

### 3. Dispatcher (`internal/dispatcher`)
- Routes commands based on a command map.
- Validates arguments (e.g., key length ≤ 256).
- Calls storage, expiration, pubsub, or persistence functions.
- Returns results as `interface{}` (typed later by serializer).

### 4. Storage Engine (`internal/storage`)
- **Data:** `map[string]Value` protected by `sync.RWMutex`.
- **Value:** `struct { Data []byte; ExpiresAt int64 }`.
- **Operations:** `Set`, `Get`, `Delete`, `Exists`.
- **Defensive copying:** Copies input on `Set`, returns a copy on `Get` to prevent external mutation.

### 5. Expiration (`internal/expiration`)
- Lazy expiration: checks TTL on `GET` and `EXISTS`.
- Background cleanup: ticker (1 second) scans all keys and deletes expired ones.

### 6. Pub/Sub (`internal/pubsub`)
- **Registry:** `map[string]map[chan []byte]bool` protected by `sync.RWMutex`.
- **Subscribe:** Adds a buffered channel (capacity 64) to the topic(s).
- **Unsubscribe:** Removes on client disconnect.
- **Publish:** Copies subscriber list under read lock, then sends non‑blocking messages to each channel.
- **Subscriber Mode:** Clients that `SUBSCRIBE` stop reading commands; they only receive messages.

### 7. Persistence (`internal/persistence`)
- **Snapshot Manager:** Handles reading/writing snapshots.
- **Atomic write:** Writes to `snapshot.tmp`, then `os.Rename` to `snapshot`.
- **Corruption handling:** On load failure, logs ASCII‑art warning, renames corrupt file to `snapshot.corrupt.<timestamp>`, and starts with empty state.
- **Manual SAVE:** User triggers `SAVE`; dispatcher calls `SnapshotManager.Save(store)`.

### 8. Metrics (`internal/metrics`)
- Tracks uptime, client count, key count, command count, pub/sub stats.
- Exposed via `INFO` command.

### 9. CLI (`internal/cli`)
- Interactive REPL with command history (cyclic buffer, up/down arrows).
- Pretty‑prints RESP responses.
- Auto‑reconnect on server drop.

### 10. Configuration (`internal/config`)
- Parses `--port`, `--bind`, `--dir` with sensible defaults.

---

## Concurrency Model

- **One goroutine per client** – idiomatic Go.
- **Shared state** (storage, pubsub registry) guarded by `sync.RWMutex`.
- **Write operations** (SET, DEL, PUBLISH) take a `Lock`.
- **Read operations** (GET, EXISTS, TTL) take an `RLock`.
- **Dispatcher** holds the lock for the entire command to ensure atomicity.

---

## Request Lifecycle

1. Client connects → TCP server spawns goroutine.
2. Goroutine reads RESP commands using `bufio.Reader`.
3. Parser converts raw bytes into tokens.
4. Dispatcher matches command and validates args.
5. Subsystem (storage, pubsub, etc.) executes the operation.
6. Result is returned as `interface{}`.
7. Serializer converts result to RESP bytes.
8. Written back to the client connection.

---

## Error Handling

- **Unknown commands:** Reply with a verbose list of supported commands.
- **Corrupted snapshots:** Loud warning, preserve corrupt file, start empty.
- **Invalid clients:** Disconnect with an error; server never panics.

---

## Data Flow Diagram

```
Client → TCP Listener → Conn Handler (goroutine)
        ↓
    RESP Parser
        ↓
    Dispatcher
        ↓
   ┌────┴────┐
   │         │
Storage  Pub/Sub
   │         │
   └────┬────┘
        ↓
  RESP Serializer
        ↓
    Conn Write
        ↓
     Client
```

---

## Persistence Flow

```
User SAVE → Dispatcher → SnapshotManager.Save(store)
    │
    ▼
Serialize each key: "key|base64(value)|expires_at\n"
    │
    ▼
Write to snapshot.tmp
    │
    ▼
os.Rename("snapshot.tmp", "snapshot")  ← atomic on most filesystems
```

---

## Why No External Dependencies?

- **Simplicity:** Easier to understand, audit, and fork.
- **Eternal buildability:** No third‑party changes can break the build.
- **Security:** Minimal attack surface.

---

## Performance Considerations

- **Lock contention:** Single mutex may become a bottleneck under very high concurrency; v1 prioritizes simplicity.
- **Memory:** All data lives in RAM; values are `[]byte` – no compression.
- **Pub/Sub:** Non‑blocking sends (drop on full buffer) to avoid slowing publishers.

---

## Future Extensibility

PetitDB v1 intentionally limits scope. Future versions might add:
- More data types (hashes, lists)
- Command batching (pipelining)
- Replication (read replicas)
- TLS support

But the architecture is designed to keep the core simple and add features only if they don't compromise the zero‑config philosophy.