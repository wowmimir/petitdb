FROM golang:1.26.5-alpine AS builder

WORKDIR /build

COPY go.mod go.sum* ./

RUN go mod download

COPY . .

ARG VERSION=dev

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o petitdb ./cmd/petitdb

# Stage 2: Final minimal image
FROM alpine:latest

# Create a non‑root user for security
RUN addgroup -S petitdb && adduser -S petitdb -G petitdb

# Copy the binary from builder
COPY --from=builder /build/petitdb /usr/local/bin/petitdb

# Ensure it's executable
RUN chmod +x /usr/local/bin/petitdb

# Switch to non‑root user
USER petitdb

# Expose the default port
EXPOSE 9379

# Mount point for persistent data
VOLUME ["/data"]

# Default command: bind to all interfaces and use /data directory
ENTRYPOINT ["petitdb"]
CMD ["--bind", "0.0.0.0", "--dir", "/data"]