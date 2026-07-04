# TinyGate Review Plan

## Final Architecture

### Gateway (`./tinygate`)
- **Zero flags** (only `--config`)
- Always logs:
  - `→ downstream: POST https://opencode.ai/zen/go/v1/chat/completions` (Director)
  - `POST /v1/chat/completions 200 1.234s` (LoggingMiddleware)

### fsprovider (`./fsprovider`)
- **Only `--debug` flag** — all SSH config from environment variables
- HTTP reverse proxy over SSH tunnel
- Env vars: `SSH_HOST`, `SSH_USER`, `SSH_KEY` (or `SSH_PASSWORD`), `LOCAL_PORT`, `REMOTE_HOST`, `REMOTE_PORT`, `SSH_PORT`

## Files

| File | Status | Description |
|---|---|---|
| `gateway/proxy.go` | ✅ done | Director `log.Printf` downstream URL; `LoggingMiddleware(next)` always logs METHOD PATH STATUS DURATION; no verbose/debug params |
| `main.go` | ✅ done | Only `--config` flag; `gateway.LoggingMiddleware(http.Handler(mux))` |
| `cmd/fsprovider/remote.go` | ✅ done | HTTP SSH tunnel proxy; `--debug` flag; config from env vars; `golang.org/x/crypto/ssh` |
| `go.mod` | ✅ done | Added `golang.org/x/crypto v0.28.0` |
