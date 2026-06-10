<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/deps-zero-blue?style=flat" alt="Zero Dependencies">
  <img src="https://img.shields.io/badge/binary-~8MB-green?style=flat" alt="Binary Size">
</p>

# TinyGate

> Tiny personal LLM gateway — one key, all models.

TinyGate is a zero-dependency HTTP reverse proxy between your applications and LLM providers. Configure your real API keys once, then all your apps talk to TinyGate with a single unified key. When a provider key changes, update TinyGate — not your apps.

## Why?

```
Before:                          After:
                                 
App A ──► sk-openai-real-xxx     App A ──┐
App B ──► sk-ant-real-yyy        App B ──┤
App C ──► sk-zhipu-real-zzz      App C ──┤── sk-gateway-key ──► TinyGate ──┬──► OpenAI
                                                                           ├──► Anthropic
When key rotates → update        When key rotates → update TinyGate         └──► Zhipu
every app config                 config only. Apps never notice.
```

## Features

- **Zero dependencies** — pure Go standard library, single 8MB binary
- **Transparent proxying** — fully transparent, provider-agnostic path forwarding
- **Key mapping** — unified upstream key → per-provider downstream key
- **Config-driven** — YAML config, zero code changes to add providers
- **Streaming** — SSE streaming transparently passed through
- **Multi-key upstream** — support multiple upstream keys for rotation
- **Custom auth** — override auth header/format per provider
- **Health check** — `GET /health`
- **Docker** — multi-stage build included

## Quick Start

```bash
# 1. Set your API keys (once)
export ZHIPU_API_KEY="your-key"
export MIMO_API_KEY="your-key"
export OPENCODE_GO_API_KEY="your-key"

# 2. One command
make all && make start

# 3. Done — use it
curl http://localhost:39901/opencode/v1/chat/completions \
  -H "Authorization: Bearer sk-gateway-key-1" \
  -H "Content-Type: application/json" \
  -d '{"model":"glm-5.1","messages":[{"role":"user","content":"Hello"}]}'
```

## Routing

TinyGate strips the route prefix and appends the remaining path to the downstream URL:

```
Client path                              → Downstream
──────────────────────────────────────────────────────────────────
POST /zhipu/v4/chat/completions          → https://open.bigmodel.cn/api/paas/v4/chat/completions
POST /mimo/v1/chat/completions           → https://api.xiaomimimo.com/v1/chat/completions
POST /opencode/v1/chat/completions       → https://opencode.ai/zen/go/v1/chat/completions
```

## Configuration

```yaml
server:
  port: 39901          # server port
  timeout: 1200s       # global timeout (supports "1200s" / "20m")
  health: true         # enable /health endpoint

gateway:
  api_keys:            # valid upstream keys (any of these works)
    - "sk-gateway-key-1"
    - "sk-gateway-key-2"

routes:
  - prefix: "/provider"           # URL prefix for this provider
    downstream_url: "https://..." # downstream base URL
    api_key: "${ENV_VAR}"         # real API key (env var or literal)
    auth_header: "Authorization"  # optional, default: Authorization
    auth_format: "Bearer ${api_key}" # optional, default
```

### Environment variables

Use `${VAR_NAME}` to reference environment variables. Use `$${NOT_VAR}` to escape.

```yaml
api_key: "${DEEPSEEK_KEY}"   # resolves from environment
api_key: "$${NOT_A_VAR}"     # literal "${NOT_A_VAR}"
api_key: "sk-raw-key"        # literal, no resolution needed
```

### Custom auth per provider

```yaml
# Default: Authorization: Bearer xxx
routes:
  - prefix: "/openai"
    downstream_url: "https://api.openai.com"
    api_key: "${OPENAI_KEY}"

# Anthropic-style
  - prefix: "/claude"
    downstream_url: "https://api.anthropic.com"
    api_key: "${ANTHROPIC_KEY}"
    auth_header: "x-api-key"
    auth_format: "${api_key}"
```

## Docker

```bash
# Copy example and fill in your keys
cp .env.example .env
vim .env

# Build and run
docker build -t tinygate .
docker run -p 39901:39901 --env-file .env tinygate
```

## License

MIT
