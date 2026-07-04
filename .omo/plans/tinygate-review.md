# TinyGate Review Plan

## Final Architecture

### Gateway (`./tinygate`)
- Flags: `--config`, `--verbose`
- Always logs: `→ downstream: URL` + `POST /path STATUS DURATION`
- `--verbose`: adds request headers and body summary

### fsprovider (`./fsprovider`)
- Flag: `--debug` — all SSH config from env vars
- Env: `SSH_HOST`, `SSH_USER`, `SSH_KEY` (or `SSH_PASSWORD`), `LOCAL_PORT`, `REMOTE_HOST`, `REMOTE_PORT`, `SSH_PORT`

## Files

| File | Description |
|---|---|
| `gateway/proxy.go` | Director `log.Printf` downstream URL; `LoggingMiddleware(verbose, next)` |
| `main.go` | `--verbose` flag; `gateway.LoggingMiddleware(*verbose, http.Handler(mux))` |
| `cmd/fsprovider/remote.go` | SSH HTTP proxy; `--debug` flag; config from env |
| `go.mod` | `golang.org/x/crypto v0.28.0` |
