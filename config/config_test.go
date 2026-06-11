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
}
