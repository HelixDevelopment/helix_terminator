# Recording Service

HelixTerminator microservice — Assembles asciinema-format recordings from Kafka segments, Ed25519 signing, playback API, full-text search, MP4 export

## Features
- Asciinema-format recording assembly from Kafka segments
- Ed25519 signing for tamper evidence
- Playback API with seek support
- Full-text search across transcripts
- MP4 export for sharing

## Module Path

`helixterminator.io/services/recording`

## Database

PostgreSQL helixterm_recordings (metadata) + S3-compatible object storage

## Upstream Dependencies

terminal, audit

## API Endpoints

- `GET` `/api/v1/recordings` — List recordings
- `GET` `/api/v1/recordings/{recordingId}` — Get recording
- `GET` `/api/v1/recordings/{recordingId}/play` — Playback
- `GET` `/api/v1/recordings/{recordingId}/download` — Download asciinema
- `POST` `/api/v1/recordings/{recordingId}/export` — Export to MP4
- `GET` `/api/v1/recordings/search` — Full-text search
- `GET` `/api/v1/recordings/{recordingId}/transcript` — Get transcript

## Health Checks

- `GET /healthz` — Health check (200 = healthy)
- `GET /healthz/ready` — Readiness check (200 = ready, 503 = not ready)

## Running

```bash
export DATABASE_URL=postgres://user:pass@localhost/recording_service
export PORT=8080
go run ./cmd/recording
```

## Testing

```bash
go test -v -race -cover ./...
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | PostgreSQL connection string |
| `PORT` | No | 8080 | HTTP/gRPC port |
| `LOG_LEVEL` | No | info | Log level (debug/info/warn/error) |
| `KAFKA_BROKERS` | No | — | Kafka bootstrap servers |
| `REDIS_URL` | No | — | Redis connection string |

---

*HelixTerminator Recording Service — see `docs/research/mvp/final/implementation/backend/README.md` for canonical service registry*
