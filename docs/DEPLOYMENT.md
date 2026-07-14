# Deployment

PetitDB is designed to be lightweight and easy to deploy in various environments. This guide covers production‑ready setups.

---

## Table of Contents

- [Docker Compose](#docker-compose)
- [Render](#render)
- [Fly.io](#flyio)
- [Systemd (Linux)](#systemd-linux)
- [Reverse Proxy (nginx)](#reverse-proxy-nginx)
- [Performance Tuning](#performance-tuning)

---

## Docker Compose

Create a `docker-compose.yml` file:

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
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

Start the service:
```bash
docker-compose up -d
```

Check logs:
```bash
docker-compose logs -f
```

---

## Render

### Using the Render Dashboard
1. Create a new **Private Service**.
2. Choose **Image** as the source.
3. Enter `wowmimir/petitdb:latest` as the image.
4. Set environment variables (optional):
   - `PETITDB_BIND=0.0.0.0`
   - `PETITDB_DIR=/data`
5. Mount a persistent disk at `/data`.
6. Expose port `9379`.

### Using `render.yaml`
```yaml
services:
  - type: pserv
    name: petitdb
    runtime: image
    image:
      url: wowmimir/petitdb:latest
    disk:
      name: petitdb-data
      mountPath: /data
      sizeGB: 1
    envVars:
      - key: PETITDB_BIND
        value: 0.0.0.0
      - key: PETITDB_DIR
        value: /data
```

---

## Fly.io

Create a `fly.toml`:

```toml
app = "petitdb"

[build]
  image = "wowmimir/petitdb:latest"

[[services]]
  internal_port = 9379
  protocol = "tcp"
  [[services.ports]]
    port = 9379

[mounts]
  source = "petitdb_data"
  destination = "/data"

[env]
  PETITDB_BIND = "0.0.0.0"
  PETITDB_DIR = "/data"
```

Deploy:
```bash
flyctl launch
flyctl deploy
```

---

## Systemd (Linux)

Create a system user:
```bash
sudo adduser --system --group --no-create-home petitdb
sudo mkdir -p /var/lib/petitdb/data
sudo chown petitdb:petitdb /var/lib/petitdb/data
```

Create `/etc/systemd/system/petitdb.service`:

```ini
[Unit]
Description=PetitDB state server
After=network.target

[Service]
Type=simple
User=petitdb
Group=petitdb
WorkingDirectory=/var/lib/petitdb
ExecStart=/usr/local/bin/petitdb --bind 127.0.0.1 --dir /var/lib/petitdb/data
Restart=always
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable petitdb
sudo systemctl start petitdb
sudo systemctl status petitdb
```

---

## Reverse Proxy (nginx)

If you need TLS/SSL, use nginx as a TCP reverse proxy.

```nginx
stream {
    server {
        listen 6379 ssl;          # Redis port
        proxy_pass 127.0.0.1:9379;
        ssl_certificate /etc/nginx/ssl/petitdb.crt;
        ssl_certificate_key /etc/nginx/ssl/petitdb.key;
    }
}
```

Reload nginx:
```bash
sudo nginx -s reload
```

Now connect using `redis-cli -h yourdomain.com -p 6379 --tls`.

---

## Performance Tuning

- **File Descriptors:** Increase OS limits (`ulimit -n 65536`).
- **Memory:** Ensure enough RAM for your keys and values.
- **Snapshot Frequency:** Use `SAVE` sparingly to avoid I/O bottlenecks.
- **Network:** Place PetitDB close to your application (same VPC).
- **Monitoring:** Use `INFO` to track clients, keys, and command rates.

---

## Backup & Recovery

- **Snapshots:** Located in `--dir`. Back them up using standard file backup tools.
- **Disaster Recovery:** If the snapshot is corrupted, PetitDB starts empty; restore a backup by copying the snapshot file back into place and restarting.

---

## Common Issues

| Issue | Solution |
|-------|----------|
| `address already in use` | Change port with `--port` or kill existing process. |
| Permission denied on data dir | Ensure user has write access. |
| Snapshot corruption | Server handles it; restore from backup if needed. |
| High memory usage | Reduce keys or values; upgrade memory. |

---