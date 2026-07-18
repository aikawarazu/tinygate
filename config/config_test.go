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
  api_keys: "sk-key-1,sk-key-2"
routes:
  - prefix: "/zhipu"
    downstream_url: "https://open.bigmodel.cn/api/paas"
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
	keys := cfg.Gateway.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 api keys, got %d", len(keys))
	}
	if keys[0] != "sk-key-1" {
		t.Errorf("expected sk-key-1, got %s", keys[0])
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
  api_keys: "sk-static"
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

func TestParseConfig_EnvVarAPIKeys(t *testing.T) {
	os.Setenv("TINYGATE_API_KEYS", "sk-a,sk-b,sk-c")
	defer os.Unsetenv("TINYGATE_API_KEYS")

	yaml := `
gateway:
  api_keys: "${TINYGATE_API_KEYS}"
`
	cfg, err := ParseConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	keys := cfg.Gateway.Keys()
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d: %v", len(keys), keys)
	}
	if keys[1] != "sk-b" {
		t.Errorf("expected sk-b, got %s", keys[1])
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

func TestParseConfig_VersionPrefix(t *testing.T) {
	yaml := `
gateway:
  api_keys: "sk-key-1"
routes:
  - prefix: "/opencode"
    downstream_url: "https://opencode.ai/zen/go"
    api_key: "sk-opencode"
    version_prefix: "/v1"
`
	cfg, err := ParseConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(cfg.Routes))
	}
	if cfg.Routes[0].VersionPrefix != "/v1" {
		t.Errorf("expected version_prefix /v1, got %s", cfg.Routes[0].VersionPrefix)
	}
}

func TestParseConfig_DefaultValues(t *testing.T) {
	yaml := `
server:
  health: true
gateway:
  api_keys: "sk-key-1,sk-key-2"
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
	keys := cfg.Gateway.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 api keys, got %d", len(keys))
	}
	if cfg.Routes[0].AuthHeader != "Authorization" {
		t.Errorf("expected default auth_header Authorization, got %s", cfg.Routes[0].AuthHeader)
	}
	if cfg.Routes[0].AuthFormat != "Bearer ${api_key}" {
		t.Errorf("expected default auth_format Bearer ${api_key}, got %s", cfg.Routes[0].AuthFormat)
	}
	if cfg.Routes[0].VersionPrefix != "" {
		t.Errorf("expected default version_prefix to be empty, got %s", cfg.Routes[0].VersionPrefix)
	}
}

func TestParseConfig_DefaultRoute(t *testing.T) {
	os.Setenv("DEFAULT_API_KEY", "sk-from-env")
	defer os.Unsetenv("DEFAULT_API_KEY")

	yaml := `
gateway:
  api_keys: "sk-key-1"
routes:
  - prefix: "/zhipu"
    downstream_url: "https://open.bigmodel.cn/api/paas"
    api_key: "sk-zhipu"
default_route:
  downstream_url: "https://opencode.ai/zen/go"
  api_key: "${DEFAULT_API_KEY}"
  version_prefix: ""
`
	cfg, err := ParseConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultRoute == nil {
		t.Fatal("expected default_route to be set")
	}
	if cfg.DefaultRoute.DownstreamURL != "https://opencode.ai/zen/go" {
		t.Errorf("unexpected downstream_url: %s", cfg.DefaultRoute.DownstreamURL)
	}
	if cfg.DefaultRoute.APIKey != "sk-from-env" {
		t.Errorf("expected env-injected api_key sk-from-env, got %s", cfg.DefaultRoute.APIKey)
	}
	// Defaults should be applied to the default route too.
	if cfg.DefaultRoute.AuthHeader != "Authorization" {
		t.Errorf("expected default auth_header Authorization, got %s", cfg.DefaultRoute.AuthHeader)
	}
	if cfg.DefaultRoute.AuthFormat != "Bearer ${api_key}" {
		t.Errorf("expected default auth_format, got %s", cfg.DefaultRoute.AuthFormat)
	}
}

func TestParseConfig_NoDefaultRoute(t *testing.T) {
	yaml := `
gateway:
  api_keys: "sk-key-1"
routes:
  - prefix: "/zhipu"
    downstream_url: "https://open.bigmodel.cn/api/paas"
    api_key: "sk-zhipu"
`
	cfg, err := ParseConfig([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultRoute != nil {
		t.Errorf("expected nil default_route, got %+v", cfg.DefaultRoute)
	}
}
