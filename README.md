# PetitDB

**Lightweight, Zero‑Config, Memory‑First State Server for Solo Devs & Small Teams**

PetitDB is a simple TCP state server for session storage, API caching, counters, and rate limiting — without the overhead of Redis or Memcached.

- **Single binary** – `go install` → `petitdb` → `SET key value` → done
- **Zero config** – no config files, no authentication
- **Redis‑compatible** – speaks RESP, works with `redis-cli` and Redis clients
- **Ephemeral** – built for temporary state, not persistent databases
- **Crash recovery** – atomic snapshots with loud corruption warnings
- **Pub/Sub** – real‑time messaging between clients
- **No dependencies** – pure Go, stdlib only

---

## Quick Start (30 seconds)

```bash
# Install
go install github.com/wowmimir/petitdb@latest

# Start the server
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

### Pre‑built binaries
Download from [GitHub Releases](https://github.com/wowmimir/petitdb/releases).

### Docker
```bash
docker pull wowmimir/petitdb:latest
docker run -p 9379:9379 -v petitdb_data:/data wowmimir/petitdb:latest
```

### From source
```bash
git clone https://github.com/wowmimir/petitdb.git
cd petitdb
make build
```

---

## Supported Commands

| Command | Description |
|---------|-------------|
| `PING` | Health check |
| `SET key value` | Store a string value |
| `GET key` | Retrieve a string value |
| `DEL key` | Delete a key |
| `EXISTS key` | Check if key exists |
| `EXPIRE key seconds` | Set TTL (seconds) |
| `TTL key` | Get remaining TTL |
| `SAVE` | Manual snapshot |
| `SUBSCRIBE topic...` | Subscribe to topics |
| `PUBLISH topic message` | Send a message |
| `INFO` | Runtime statistics |

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

## Deployment

See [DEPLOYMENT.md](./DEPLOYMENT.md) for Docker Compose, Render, Fly.io, and systemd examples.

---

## Philosophy

- **Not a database** – PostgreSQL/MySQL remain the source of truth
- **Ephemeral state** – session storage, API caching, rate limiting
- **Zero operations** – no config, no auth, no debugging headaches
- **Crash recovery, not durability** – snapshots for restart, not backups

---

## Documentation

- [Installation](./INSTALL.md)
- [Deployment](./DEPLOYMENT.md)
- [Commands](./COMMANDS.md)
- [Architecture](./ARCHITECTURE.md)
- [Changelog](./CHANGELOG.md)

---

## License

MIT © [wowmimir](https://github.com/wowmimir)