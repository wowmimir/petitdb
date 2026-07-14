# Changelog

All notable changes to PetitDB will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v1.0.0] – 2026-07-15

### Added
- Initial release.
- RESP (Redis Protocol) compatible TCP server.
- In‑memory storage with `sync.RWMutex` protection.
- Commands: `PING`, `SET`, `GET`, `DEL`, `EXISTS`, `EXPIRE`, `TTL`, `SAVE`, `SUBSCRIBE`, `PUBLISH`, `INFO`.
- Lazy expiration with background cleanup (1s ticker).
- Atomic snapshots (write to `snapshot.tmp`, rename to `snapshot`) for crash recovery.
- Loud corruption handling (ASCII warnings, rename corrupt file).
- Built‑in CLI (`petitdb cli`) with history and pretty‑printed responses.
- Verbose error messages for unknown commands (lists all v1 commands).
- Single binary with zero external dependencies.
- Docker image with multi‑arch support (`linux/amd64`, `linux/arm64`).
- Go 1.26.5+ support.
- Makefile and PowerShell build scripts.
- GitHub Actions for CI (test) and release (binaries + Docker push).
- Comprehensive documentation (README, INSTALL, DEPLOYMENT, COMMANDS, ARCHITECTURE, CHANGELOG, OPS).

### Known Limitations
- Only string values (no lists, hashes, sets).
- No authentication or TLS.
- No AOF/WAL – manual snapshots only.
- Single‑threaded locking; may be a bottleneck at very high concurrency.

---

## [Unreleased]

### Planned for future versions
- Pipelining (batch commands).
- Additional data types (hashes, lists).
- Replication (read replicas).
- TLS support.
- More metrics in `INFO`.