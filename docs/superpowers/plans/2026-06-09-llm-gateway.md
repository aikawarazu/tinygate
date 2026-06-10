# just-llm-gateway Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a lightweight Go LLM gateway that transparently proxies requests to downstream model providers, mapping API keys to shield clients from key changes.

**Architecture:** Single-binary Go HTTP server using `net/http/httputil.ReverseProxy`. YAML config defines routes (prefix → downstream URL + API key). Auth middleware validates upstream keys, Director rewrites requests (strip prefix, replace Authorization header), Transport handles timeout.

**Tech Stack:** Go (standard library only), YAML (gopkg.in/yaml.v3)

---

## File Structure

```
just-llm-gateway/
├── main.go                    # Entry point, server startup
├── config/
│   ├── config.go              # YAML parsing + env var injection
│   └── config_test.go         # Config parsing tests
├── gateway/
│   ├── auth.go                # Auth middleware
│   ├── auth_test.go           # Auth middleware tests
│   ├── router.go              # Route matching + prefix stripping
│   ├── router_test.go         # Router tests
│   ├── proxy.go               # ReverseProxy wrapper + logging
│   └── proxy_test.go          # Proxy tests
├── config.yaml                # Default config file
├── Dockerfile                 # Docker build
└── go.mod
```

---

### Task 1: Initialize Go Module

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Initialize Go module**

```bash
cd /root/workspace/just-llm-gateway
go mod init github.com/user/just-llm-gateway
```

- [ ] **Step 2: Add yaml dependency**

```bash
go get gopkg.in/yaml.v3
```

- [ ] **Step 3: Verify module is initialized**

```bash
cat go.mod
```

Expected: Module file with `module github.com/user/just-llm-gateway` and yaml dependency.

---

### Task 2: Config Parsing

**Files:**
- Create: `config/config.go`
- Create: `config/config_test.go`

- [ ] **Step 1: Write config struct and parsing tests**

```go
// config/config_test.go
package config

import (
	"os"
	"testing"
)

func TestParseConfig_BasicYAML(t *testing.T) {
	yaml := `
server:
  port: 39901
  timeout: 1200s
  health: true
gateway:
  api_keys:
    - "sk-key-1"
    - "sk-key-2"
routes:
  - prefix: "/zhipu"
    downstream_url: "https://open.bigmodel.cn"
    api_key: "sk-zhipu-key"
`
	cfg, err := ParseConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Port != 39901 {
		t.Errorf("expected port 39901, got %d", cfg.Server.Port)
	}
	if cfg.Server.Timeout != "1200s" {
		t.Errorf("expected timeout 1200s, got %s", cfg.Server.Timeout)
	}
	if !cfg.Server.Health {
		t.Error("expected health true")
	}
	if len(cfg.Gateway.APIKeys) != 2 {
		t.Errorf("expected 2 api keys, got %d", len(cfg.Gateway.APIKeys))
	}
	if cfg.Gateway.APIKeys[0] != "sk-key-1" {
		t.Errorf("expected sk-key-1, got %s", cfg.Gateway.APIKeys[0])
	}
	if len(cfg.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(cfg.Routes))
	}
	if cfg.Routes[0].Prefix != "/zhipu" {
		t.Errorf("expected prefix /zhipu, got %s", cfg.Routes[0].Prefix)
	}
	if cfg.Routes[0].DownstreamURL != "https://open.bigmodel.cn/api/paas" {
		t.Errorf("expected downstream_url https://open.bigmodel.cn/api/paas, got %s", cfg.Routes[0].DownstreamURL)
	}
	if cfg.Routes[0].APIKey != "sk-zhipu-key" {
		t.Errorf("expected api_key sk-zhipu-key, got %s", cfg.Routes[0].APIKey)
	}
}

func TestParseConfig_EnvVarInjection(t *testing.T) {
	os.Setenv("TEST_API_KEY", "sk-from-env")
	defer os.Unsetenv("TEST_API_KEY")

	yaml := `
gateway:
  api_keys:
    - "sk-static"
routes:
  - prefix: "/test"
    downstream_url: "https://example.com"
    api_key: "${TEST_API_KEY}"
`
	cfg, err := ParseConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Routes[0].APIKey != "sk-from-env" {
		t.Errorf("expected sk-from-env, got %s", cfg.Routes[0].APIKey)
	}
}

func TestParseConfig_EscapedDollar(t *testing.T) {
	yaml := `
routes:
  - prefix: "/test"
    downstream_url: "https://example.com"
    api_key: "$${NOT_VAR}"
`
	cfg, err := ParseConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Routes[0].APIKey != "${NOT_VAR}" {
		t.Errorf("expected ${NOT_VAR}, got %s", cfg.Routes[0].APIKey)
	}
}

func TestParseConfig_DefaultValues(t *testing.T) {
	yaml := `
routes:
  - prefix: "/test"
    downstream_url: "https://example.com"
    api_key: "sk-test"
`
	cfg, err := ParseConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Port != 39901 {
		t.Errorf("expected default port 39901, got %d", cfg.Server.Port)
	}
	if cfg.Server.Timeout != "1200s" {
		t.Errorf("expected timeout 1200s, got %s", cfg.Server.Timeout)
	}
	if !cfg.Server.Health {
		t.Error("expected health true")
	}
	if len(cfg.Gateway.APIKeys) != 2 {
		t.Errorf("expected 2 api keys, got %d", len(cfg.Gateway.APIKeys))
	}
	if !cfg.Server.Health {
		t.Error("expected default health true")
	}
	if cfg.Routes[0].AuthHeader != "Authorization" {
		t.Errorf("expected default auth_header Authorization, got %s", cfg.Routes[0].AuthHeader)
	}
	if cfg.Routes[0].AuthFormat != "Bearer ${api_key}" {
		t.Errorf("expected default auth_format Bearer ${api_key}, got %s", cfg.Routes[0].AuthFormat)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /root/workspace/just-llm-gateway
go test ./config/ -v
```

Expected: FAIL with "cannot find package" or similar.

- [ ] **Step 3: Implement config parsing**

```go
// config/config.go
package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Gateway GatewayConfig `yaml:"gateway"`
	Routes  []RouteConfig `yaml:"routes"`
}

type ServerConfig struct {
	Port    int    `yaml:"port"`
	Timeout string `yaml:"timeout"` // e.g., "1200s", "20m"
	Health  bool   `yaml:"health"`
}

type GatewayConfig struct {
	APIKeys []string `yaml:"api_keys"`
}

type RouteConfig struct {
	Prefix         string `yaml:"prefix"`
	DownstreamURL  string `yaml:"downstream_url"`
	APIKey         string `yaml:"api_key"`
	AuthHeader     string `yaml:"auth_header"`
	AuthFormat     string `yaml:"auth_format"`
}

func ParseConfig(data []byte) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:    39901,
			Timeout: "1200s",
			Health:  true,
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults for routes
	for i := range cfg.Routes {
		if cfg.Routes[i].AuthHeader == "" {
			cfg.Routes[i].AuthHeader = "Authorization"
		}
		if cfg.Routes[i].AuthFormat == "" {
			cfg.Routes[i].AuthFormat = "Bearer ${api_key}"
		}
		// Inject env vars
		cfg.Routes[i].APIKey = injectEnvVars(cfg.Routes[i].APIKey)
	}

	return cfg, nil
}

// ParseTimeout parses a duration string like "1200s" or "20m" to seconds
func ParseTimeout(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 1200, nil // default 20 minutes
	}

	// Handle "s" suffix
	if strings.HasSuffix(s, "s") {
		s = s[:len(s)-1]
		var sec int
		if _, err := fmt.Sscanf(s, "%d", &sec); err != nil {
			return 0, fmt.Errorf("invalid timeout: %s", s)
		}
		return sec, nil
	}

	// Handle "m" suffix
	if strings.HasSuffix(s, "m") {
		s = s[:len(s)-1]
		var min int
		if _, err := fmt.Sscanf(s, "%d", &min); err != nil {
			return 0, fmt.Errorf("invalid timeout: %s", s)
		}
		return min * 60, nil
	}

	// Plain number (assume seconds)
	var sec int
	if _, err := fmt.Sscanf(s, "%d", &sec); err != nil {
		return 0, fmt.Errorf("invalid timeout: %s", s)
	}
	return sec, nil
}

var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

func injectEnvVars(s string) string {
	// Handle escaped $$
	s = strings.ReplaceAll(s, "$$", "\x00ESCAPED_DOLLAR\x00")

	// Replace ${VAR} with env var value
	s = envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[2 : len(match)-1] // Remove ${ and }
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		return match // Keep original if env var not found
	})

	// Restore escaped $
	s = strings.ReplaceAll(s, "\x00ESCAPED_DOLLAR\x00", "$")
	return s
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /root/workspace/just-llm-gateway
go test ./config/ -v
```

Expected: All 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add config/
git commit -m "feat: add config parsing with env var injection"
```

---

### Task 3: Auth Middleware

**Files:**
- Create: `gateway/auth.go`
- Create: `gateway/auth_test.go`

- [ ] **Step 1: Write auth middleware tests**

```go
// gateway/auth_test.go
package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware_ValidKey(t *testing.T) {
	apiKeys := []string{"sk-key-1", "sk-key-2"}
	handler := AuthMiddleware(apiKeys, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer sk-key-1")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidKey(t *testing.T) {
	apiKeys := []string{"sk-key-1", "sk-key-2"}
	handler := AuthMiddleware(apiKeys, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer sk-invalid")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_NoHeader(t *testing.T) {
	apiKeys := []string{"sk-key-1"}
	handler := AuthMiddleware(apiKeys, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_EmptyAPIKeys(t *testing.T) {
	handler := AuthMiddleware([]string{}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer sk-any")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// With empty api_keys list, no key should pass
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /root/workspace/just-llm-gateway
go test ./gateway/ -v -run TestAuth
```

Expected: FAIL with "undefined: AuthMiddleware".

- [ ] **Step 3: Implement auth middleware**

```go
// gateway/auth.go
package gateway

import (
	"net/http"
	"strings"
)

func AuthMiddleware(apiKeys []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		token := parts[1]

		// Check if token is in api_keys
		for _, key := range apiKeys {
			if token == key {
				next.ServeHTTP(w, r)
				return
			}
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /root/workspace/just-llm-gateway
go test ./gateway/ -v -run TestAuth
```

Expected: All 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add gateway/auth.go gateway/auth_test.go
git commit -m "feat: add auth middleware for upstream API key validation"
```

---

### Task 4: Router

**Files:**
- Create: `gateway/router.go`
- Create: `gateway/router_test.go`

- [ ] **Step 1: Write router tests**

```go
// gateway/router_test.go
package gateway

import (
	"testing"

	"github.com/user/just-llm-gateway/config"
)

func TestRouter_MatchPrefix(t *testing.T) {
	routes := []config.RouteConfig{
		{Prefix: "/zhipu", DownstreamURL: "https://open.bigmodel.cn/api/paas", APIKey: "sk-zhipu"},
		{Prefix: "/mimo", DownstreamURL: "https://api.xiaomimimo.com", APIKey: "sk-mimo"},
	}
	router := NewRouter(routes)

	route, remainingPath, ok := router.Match("/zhipu/v4/chat/completions")
	if !ok {
		t.Fatal("expected match")
	}
	if route.Prefix != "/zhipu" {
		t.Errorf("expected prefix /zhipu, got %s", route.Prefix)
	}
	if remainingPath != "/v4/chat/completions" {
		t.Errorf("expected /v4/chat/completions, got %s", remainingPath)
	}
}

func TestRouter_NoMatch(t *testing.T) {
	routes := []config.RouteConfig{
		{Prefix: "/zhipu", DownstreamURL: "https://open.bigmodel.cn/api/paas", APIKey: "sk-zhipu"},
		{Prefix: "/mimo", DownstreamURL: "https://api.xiaomimimo.com", APIKey: "sk-mimo"},
	}
	router := NewRouter(routes)

	route, remainingPath, ok := router.Match("/zhipu/v4/chat/completions")
	if !ok {
		t.Fatal("expected match")
	}
	if route.Prefix != "/zhipu" {
		t.Errorf("expected prefix /zhipu, got %s", route.Prefix)
	}
	if remainingPath != "/v4/chat/completions" {
		t.Errorf("expected /v4/chat/completions, got %s", remainingPath)
	}
}

func TestRouter_NoMatch(t *testing.T) {
	routes := []config.RouteConfig{
		{Prefix: "/zhipu", DownstreamURL: "https://open.bigmodel.cn/api/paas", APIKey: "sk-zhipu"},
	}
	router := NewRouter(routes)

	_, _, ok := router.Match("/unknown/path")
	if ok {
		t.Error("expected no match")
	}
}

func TestRouter_LongestPrefix(t *testing.T) {
	routes := []config.RouteConfig{
		{Prefix: "/api", DownstreamURL: "https://example.com", APIKey: "sk-1"},
		{Prefix: "/api/v2", DownstreamURL: "https://v2.example.com", APIKey: "sk-2"},
	}
	router := NewRouter(routes)

	route, remainingPath, ok := router.Match("/api/v2/chat/completions")
	if !ok {
		t.Fatal("expected match")
	}
	if route.Prefix != "/api/v2" {
		t.Errorf("expected prefix /api/v2, got %s", route.Prefix)
	}
	if remainingPath != "/chat/completions" {
		t.Errorf("expected /chat/completions, got %s", remainingPath)
	}
}
	router := NewRouter(routes)

	_, _, ok := router.Match("/unknown/path")
	if ok {
		t.Error("expected no match")
	}
}

func TestRouter_LongestPrefix(t *testing.T) {
	routes := []config.RouteConfig{
		{Prefix: "/api", DownstreamURL: "https://example.com", APIKey: "sk-1"},
		{Prefix: "/api/v2", DownstreamURL: "https://v2.example.com", APIKey: "sk-2"},
	}
	router := NewRouter(routes)

	route, remainingPath, ok := router.Match("/api/v2/chat/completions")
	if !ok {
		t.Fatal("expected match")
	}
	if route.Prefix != "/api/v2" {
		t.Errorf("expected prefix /api/v2, got %s", route.Prefix)
	}
	if remainingPath != "/chat/completions" {
		t.Errorf("expected /chat/completions, got %s", remainingPath)
	}
}

func TestRouter_ExactMatch(t *testing.T) {
	routes := []config.RouteConfig{
		{Prefix: "/health", DownstreamURL: "https://example.com", APIKey: "sk-1"},
	}
	router := NewRouter(routes)

	route, remainingPath, ok := router.Match("/health")
	if !ok {
		t.Fatal("expected match")
	}
	if route.Prefix != "/health" {
		t.Errorf("expected prefix /health, got %s", route.Prefix)
	}
	if remainingPath != "" {
		t.Errorf("expected empty remaining path, got %s", remainingPath)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /root/workspace/just-llm-gateway
go test ./gateway/ -v -run TestRouter
```

Expected: FAIL with "undefined: NewRouter".

- [ ] **Step 3: Implement router**

```go
// gateway/router.go
package gateway

import (
	"sort"
	"strings"

	"github.com/user/just-llm-gateway/config"
)

type Router struct {
	routes []config.RouteConfig
}

func NewRouter(routes []config.RouteConfig) *Router {
	// Sort routes by prefix length (longest first) for longest-prefix matching
	sorted := make([]config.RouteConfig, len(routes))
	copy(sorted, routes)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Prefix) > len(sorted[j].Prefix)
	})
	return &Router{routes: sorted}
}

func (r *Router) Match(path string) (*config.RouteConfig, string, bool) {
	for i := range r.routes {
		prefix := r.routes[i].Prefix
		if strings.HasPrefix(path, prefix) {
			// Exact match or prefix followed by /
			remaining := strings.TrimPrefix(path, prefix)
			if remaining == "" || strings.HasPrefix(remaining, "/") {
				return &r.routes[i], remaining, true
			}
		}
	}
	return nil, "", false
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /root/workspace/just-llm-gateway
go test ./gateway/ -v -run TestRouter
```

Expected: All 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add gateway/router.go gateway/router_test.go
git commit -m "feat: add router with prefix matching and longest-prefix strategy"
```

---

### Task 5: Proxy with Director

**Files:**
- Create: `gateway/proxy.go`
- Create: `gateway/proxy_test.go`

- [ ] **Step 1: Write proxy tests**

```go
// gateway/proxy_test.go
package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/user/just-llm-gateway/config"
)

func TestProxy_DirectorRewritesRequest(t *testing.T) {
	// Create a test downstream server
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request was rewritten correctly
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("expected path /v1/chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-downstream" {
			t.Errorf("expected Bearer sk-downstream, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Custom") != "custom-value" {
			t.Errorf("expected X-Custom to be preserved, got %s", r.Header.Get("X-Custom"))
		}
		if r.Header.Get("Host") != "" {
			t.Errorf("expected Host to be stripped")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	route := config.RouteConfig{
		Prefix:        "/test",
		DownstreamURL: downstream.URL,
		APIKey:        "sk-downstream",
		AuthHeader:    "Authorization",
		AuthFormat:    "Bearer ${api_key}",
	}

	proxy := NewProxy(route, 300)

	// Create a request that looks like it came from the client
	req := httptest.NewRequest("POST", "/test/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer sk-client-key")
	req.Header.Set("X-Custom", "custom-value")
	req.Header.Set("Host", "localhost:39901")

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
		if r.Header.Get("Authorization") != "Bearer sk-zhipu" {
			t.Errorf("expected Bearer sk-zhipu, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Custom") != "custom-value" {
			t.Errorf("expected X-Custom to be preserved, got %s", r.Header.Get("X-Custom"))
		}
		if r.Header.Get("Host") != "" {
			t.Errorf("expected Host to be stripped")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	route := config.RouteConfig{
		Prefix:        "/zhipu",
		DownstreamURL: downstream.URL + "/api/paas",
		APIKey:        "sk-zhipu",
		AuthHeader:    "Authorization",
		AuthFormat:    "Bearer ${api_key}",
	}

	proxy := NewProxy(route, 300)

	// Create a request that looks like it came from the client
	req := httptest.NewRequest("POST", "/zhipu/v4/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer sk-client-key")
	req.Header.Set("X-Custom", "custom-value")
	req.Header.Set("Host", "localhost:39901")

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestProxy_CustomAuthHeader(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "sk-ant-key" {
			t.Errorf("expected x-api-key sk-ant-key, got %s", r.Header.Get("x-api-key"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	route := config.RouteConfig{
		Prefix:        "/anthropic",
		DownstreamURL: downstream.URL,
		APIKey:        "sk-ant-key",
		AuthHeader:    "x-api-key",
		AuthFormat:    "${api_key}",
	}

	proxy := NewProxy(route, 300)

	req := httptest.NewRequest("POST", "/anthropic/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer sk-client-key")

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd /root/workspace/just-llm-gateway
go test ./gateway/ -v -run TestProxy
```

Expected: FAIL with "undefined: NewProxy".

- [ ] **Step 3: Implement proxy**

```go
// gateway/proxy.go
package gateway

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/user/just-llm-gateway/config"
)

type Proxy struct {
	route  config.RouteConfig
	proxy  *httputil.ReverseProxy
}

func NewProxy(route config.RouteConfig, timeoutStr string) *Proxy {
	target, err := url.Parse(route.DownstreamURL)
	if err != nil {
		log.Fatalf("invalid downstream_url %s: %v", route.DownstreamURL, err)
	}

	// Parse timeout
	timeoutSeconds, err := config.ParseTimeout(timeoutStr)
	if err != nil {
		log.Fatalf("invalid timeout %s: %v", timeoutStr, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Custom director to rewrite the request
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Set the target path (remaining path after prefix stripping)
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host

		// Build auth value
		authValue := strings.ReplaceAll(route.AuthFormat, "${api_key}", route.APIKey)
		req.Header.Set(route.AuthHeader, authValue)

		// Remove Host header (will be set by transport)
		req.Header.Del("Host")
	}

	// Custom transport with timeout
	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: time.Duration(timeoutSeconds) * time.Second,
	}

	// Error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error: %s %s -> %v", r.Method, r.URL.Path, err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return &Proxy{
		route: route,
		proxy: proxy,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd /root/workspace/just-llm-gateway
go test ./gateway/ -v -run TestProxy
```

Expected: All 2 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add gateway/proxy.go gateway/proxy_test.go
git commit -m "feat: add reverse proxy with request rewriting"
```

---

### Task 6: Logging Middleware

**Files:**
- Modify: `gateway/proxy.go`

- [ ] **Step 1: Add logging middleware to proxy**

```go
// Add to gateway/proxy.go

// LoggingMiddleware logs request summary
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		log.Printf("%s %s %d %v",
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			time.Since(start),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
```

- [ ] **Step 2: Verify existing tests still pass**

```bash
cd /root/workspace/just-llm-gateway
go test ./gateway/ -v
```

Expected: All tests PASS.

- [ ] **Step 3: Commit**

```bash
git add gateway/proxy.go
git commit -m "feat: add request logging middleware"
```

---

### Task 7: Main Server

**Files:**
- Create: `main.go`

- [ ] **Step 1: Implement main server**

```go
// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/user/just-llm-gateway/config"
	"github.com/user/just-llm-gateway/gateway"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	// Read config file
	data, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	// Parse config
	cfg, err := config.ParseConfig(data)
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	// Create router
	router := gateway.NewRouter(cfg.Routes)

	// Create proxies for each route
	proxies := make(map[string]*gateway.Proxy)
	for _, route := range cfg.Routes {
		proxies[route.Prefix] = gateway.NewProxy(route, cfg.Server.Timeout)
	}

	// Create main handler
	mux := http.NewServeMux()

	// Health check endpoint
	if cfg.Server.Health {
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
	}

	// Proxy handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Match route
		route, remainingPath, ok := router.Match(r.URL.Path)
		if !ok {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		// Rewrite path (strip prefix)
		r.URL.Path = remainingPath

		// Get proxy for this route
		proxy, ok := proxies[route.Prefix]
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		proxy.ServeHTTP(w, r)
	})

	// Apply auth middleware (except for health check)
	handler := gateway.AuthMiddleware(cfg.Gateway.APIKeys, mux)

	// Apply logging middleware
	handler = gateway.LoggingMiddleware(handler)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 2: Create default config file**

```yaml
# config.yaml
server:
  port: 39901
  timeout: 1200s
  health: true

gateway:
  api_keys:
    - "sk-my-key-1"
    - "sk-my-key-2"

routes:
  - prefix: "/zhipu"
    downstream_url: "https://open.bigmodel.cn/api/paas"
    api_key: "${ZHIPU_API_KEY}"

  - prefix: "/mimo"
    downstream_url: "https://api.xiaomimimo.com"
    api_key: "${MIMO_API_KEY}"

  - prefix: "/opencode"
    downstream_url: "https://opencode.ai/zen/go"
    api_key: "${OPENCODE_GO_API_KEY}"
```

- [ ] **Step 3: Verify build succeeds**

```bash
cd /root/workspace/just-llm-gateway
go build -o just-llm-gateway .
```

Expected: Binary `just-llm-gateway` created.

- [ ] **Step 4: Run all tests**

```bash
cd /root/workspace/just-llm-gateway
go test ./... -v
```

Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add main.go config.yaml
git commit -m "feat: add main server with routing and health check"
```

---

### Task 8: Dockerfile

**Files:**
- Create: `Dockerfile`

- [ ] **Step 1: Create Dockerfile**

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o just-llm-gateway .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/just-llm-gateway .
COPY --from=builder /app/config.yaml .

EXPOSE 39901

CMD ["./just-llm-gateway", "-config", "config.yaml"]
```

- [ ] **Step 2: Commit**

```bash
git add Dockerfile
git commit -m "feat: add Dockerfile for containerized deployment"
```

---

### Task 9: Final Integration Test

**Files:**
- Create: `integration_test.go`

- [ ] **Step 1: Write integration test**

```go
// integration_test.go
package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/user/just-llm-gateway/config"
	"github.com/user/just-llm-gateway/gateway"
)

func TestIntegration_EndToEnd(t *testing.T) {
	// Create a mock downstream server
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request was rewritten correctly
		if r.Header.Get("Authorization") != "Bearer sk-downstream" {
			t.Errorf("expected Bearer sk-downstream, got %s", r.Header.Get("Authorization"))
		}

		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "Hello"}},
			},
		})
	}))
	defer downstream.Close()

	// Create config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:    39901,
			Timeout: "300s",
			Health:  true,
		},
		Gateway: config.GatewayConfig{
			APIKeys: []string{"sk-client"},
		},
		Routes: []config.RouteConfig{
			{
				Prefix:        "/test",
				DownstreamURL: downstream.URL,
				APIKey:        "sk-downstream",
				AuthHeader:    "Authorization",
				AuthFormat:    "Bearer ${api_key}",
			},
		},
	}

	// Create router and proxy
	router := gateway.NewRouter(cfg.Routes)
	proxy := gateway.NewProxy(cfg.Routes[0], cfg.Server.Timeout)

	// Create handler
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		route, remainingPath, ok := router.Match(r.URL.Path)
		if !ok {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		r.URL.Path = remainingPath
		_ = route
		proxy.ServeHTTP(w, r)
	})

	handler := gateway.AuthMiddleware(cfg.Gateway.APIKeys, mux)

	// Create test request
	body := bytes.NewBufferString(`{"model":"test","messages":[{"role":"user","content":"Hello"}]}`)
	req := httptest.NewRequest("POST", "/test/v1/chat/completions", body)
	req.Header.Set("Authorization", "Bearer sk-client")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	// Verify response body
	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	choices, ok := resp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		t.Error("expected choices in response")
	}
}
```

- [ ] **Step 2: Run integration test**

```bash
cd /root/workspace/just-llm-gateway
go test -v -run TestIntegration
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add integration_test.go
git commit -m "test: add integration test for end-to-end flow"
```

---

## Summary

| Task | Description | Files |
|------|-------------|-------|
| 1 | Init Go module | `go.mod` |
| 2 | Config parsing | `config/config.go`, `config/config_test.go` |
| 3 | Auth middleware | `gateway/auth.go`, `gateway/auth_test.go` |
| 4 | Router | `gateway/router.go`, `gateway/router_test.go` |
| 5 | Proxy | `gateway/proxy.go`, `gateway/proxy_test.go` |
| 6 | Logging | `gateway/proxy.go` (modify) |
| 7 | Main server | `main.go`, `config.yaml` |
| 8 | Dockerfile | `Dockerfile` |
| 9 | Integration test | `integration_test.go` |
