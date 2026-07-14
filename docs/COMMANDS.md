# Command Reference

This document lists all supported commands in PetitDB v1.

---

## Command Categories

- [Utility](#utility)
- [Storage](#storage)
- [Expiration](#expiration)
- [Persistence](#persistence)
- [Messaging (Pub/Sub)](#messaging-pubsub)
- [Runtime](#runtime)

---

## Utility

### `PING`
Health check.

**Response:** `PONG` (simple string).

**Example:**
```
> PING
PONG
```

---

## Storage

### `SET key value`
Store a string value.

**Response:** `OK` (simple string) if successful.

**Example:**
```
> SET name Alice
OK
```

---

### `GET key`
Retrieve the value of a key.

**Response:** Bulk string containing the value, or `$-1` (null bulk) if key does not exist.

**Example:**
```
> GET name
"Alice"
```

---

### `DEL key`
Delete a key.

**Response:** Integer `1` if key existed and was deleted, or `0` if key did not exist.

**Example:**
```
> DEL name
(integer) 1
```

---

### `EXISTS key`
Check if a key exists.

**Response:** Integer `1` if key exists, `0` otherwise.

**Example:**
```
> EXISTS name
(integer) 1
```

---

## Expiration

### `EXPIRE key seconds`
Set a time-to-live (TTL) on a key in seconds.

**Response:** Integer `1` if TTL was set, `0` if key does not exist.

**Example:**
```
> EXPIRE session 3600
(integer) 1
```

**Note:** Expired keys are cleaned up lazily (on access) and periodically in the background.

---

### `TTL key`
Get the remaining TTL of a key in seconds.

**Response:** Integer representing seconds left, or `-1` if key has no expiry, or `-2` if key does not exist.

**Example:**
```
> TTL session
(integer) 3540
```

---

## Persistence

### `SAVE`
Manually trigger an atomic snapshot of the entire data store.

**Response:** `OK` (simple string) if successful.

**Example:**
```
> SAVE
OK
```

**Note:** Snapshots are written to `snapshot.tmp` and then atomically renamed to `snapshot`.

---

## Messaging (Pub/Sub)

### `SUBSCRIBE topic [topic ...]`
Subscribe to one or more topics. Once subscribed, the client enters subscriber mode: it will no longer accept commands and will forward published messages until disconnected.

**Response:** An array with subscription confirmation:
```
["subscribe", topic, number_of_subscriptions]
```

**Example:**
```
> SUBSCRIBE news alerts
*3
$9
subscribe
$4
news
:1
```

---

### `PUBLISH topic message`
Publish a message to a topic. All active subscribers receive the message.

**Response:** Integer indicating the number of subscribers that received the message.

**Example:**
```
> PUBLISH news "Hello world"
(integer) 3
```

**Subscriber receives:**
```
["message", "news", "Hello world"]
```

---

## Runtime

### `INFO`
Returns runtime statistics including:
- Uptime (seconds)
- Connected clients
- Total keys
- Total commands executed
- Pub/Sub channels and subscribers

**Response:** A multi‑line bulk string with key‑value pairs.

**Example:**
```
> INFO
# Server
uptime: 3600
clients: 5
keys: 42
commands: 1284
# Pub/Sub
channels: 3
subscribers: 7
```

---

## Unknown Commands

If you send an unsupported command, PetitDB replies with a verbose error listing all supported v1 commands:

```
-ERR unknown command 'INCR'. PetitDB v1 supports: PING, SET, GET, DEL, EXISTS, EXPIRE, TTL, SAVE, SUBSCRIBE, PUBLISH, INFO
```

---

## RESP Protocol Compatibility

PetitDB uses the Redis Serialization Protocol (RESP) v2:
- **Simple Strings:** `+OK\r\n`
- **Errors:** `-ERR ...\r\n`
- **Integers:** `:1\r\n`
- **Bulk Strings:** `$5\r\nHello\r\n`
- **Null Bulk:** `$-1\r\n`
- **Arrays:** `*2\r\n$4\r\nPING\r\n$4\r\nPONG\r\n`

All Redis clients (e.g., `redis-py`, `ioredis`, `go-redis`) can connect to PetitDB without modification (just change the port to 9379).