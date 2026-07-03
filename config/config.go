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
	APIKeys string `yaml:"api_keys"`
}

func (g GatewayConfig) Keys() []string {
	if g.APIKeys == "" {
		return nil
	}
	parts := strings.Split(g.APIKeys, ",")
	var keys []string
	for _, p := range parts {
		if k := strings.TrimSpace(p); k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

type RouteConfig struct {
	Prefix         string `yaml:"prefix"`
	DownstreamURL  string `yaml:"downstream_url"`
	APIKey         string `yaml:"api_key"`
	AuthHeader     string `yaml:"auth_header"`
	AuthFormat     string `yaml:"auth_format"`
	VersionPrefix  string `yaml:"version_prefix"`
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

	cfg.Gateway.APIKeys = injectEnvVars(cfg.Gateway.APIKeys)

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