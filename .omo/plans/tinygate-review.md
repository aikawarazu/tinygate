# TinyGate Review Plan

## Final Architecture

### Gateway (`./tinygate`)
- Flag: `--config` (default `/etc/tinygate`)
- Always logs: `→ downstream: URL` + `POST /path STATUS DURATION`
- No auto-config generation — config must exist or fatal error

### fsprovider (`./fsprovider`)
- Flag: `--debug` — all SSH config from env vars
- Env: `SSH_HOST`, `SSH_USER`, `SSH_KEY` (or `SSH_PASSWORD`), `LOCAL_PORT`, `REMOTE_HOST`, `REMOTE_PORT`, `SSH_PORT`

## Files

| File | Description |
|---|---|
| `gateway/proxy.go` | Director `log.Printf` downstream URL; `LoggingMiddleware(next)` always logs |
| `main.go` | `--config /etc/tinygate`; no auto-config generation |
| `cmd/fsprovider/remote.go` | SSH HTTP proxy; `--debug` flag; config from env |
| `go.mod` | `golang.org/x/crypto v0.28.0` |
