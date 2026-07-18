package gateway

import (
	"testing"

	"github.com/user/tinygate/config"
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

func TestRouter_VersionlessPathMatch(t *testing.T) {
	routes := []config.RouteConfig{
		{Prefix: "/opencode", DownstreamURL: "https://opencode.ai/zen/go", APIKey: "sk-opencode"},
	}
	router := NewRouter(routes)

	// Versionless path (without /v1) should still match the prefix
	route, remainingPath, ok := router.Match("/opencode/chat/completions")
	if !ok {
		t.Fatal("expected match for versionless path")
	}
	if route.Prefix != "/opencode" {
		t.Errorf("expected prefix /opencode, got %s", route.Prefix)
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

func TestRouter_DefaultRoute_FallsBack(t *testing.T) {
	routes := []config.RouteConfig{
		{Prefix: "/zhipu", DownstreamURL: "https://open.bigmodel.cn/api/paas", APIKey: "sk-zhipu"},
	}
	router := NewRouter(routes)
	defaultRoute := &config.RouteConfig{
		Prefix:        "",
		DownstreamURL: "https://opencode.ai/zen/go",
		APIKey:        "sk-default",
	}
	router.SetDefault(defaultRoute)

	// An unmatched path should hit the default route and preserve the
	// original path as the remaining path (no prefix to strip).
	route, remainingPath, ok := router.Match("/unknown/v1/chat")
	if !ok {
		t.Fatal("expected default route to match")
	}
	if route != defaultRoute {
		t.Error("expected returned route to be the default route pointer")
	}
	if route.DownstreamURL != "https://opencode.ai/zen/go" {
		t.Errorf("unexpected downstream_url: %s", route.DownstreamURL)
	}
	if remainingPath != "/unknown/v1/chat" {
		t.Errorf("expected original path preserved, got %s", remainingPath)
	}
}

func TestRouter_DefaultRoute_PrefersExplicitPrefix(t *testing.T) {
	routes := []config.RouteConfig{
		{Prefix: "/zhipu", DownstreamURL: "https://open.bigmodel.cn/api/paas", APIKey: "sk-zhipu"},
	}
	router := NewRouter(routes)
	router.SetDefault(&config.RouteConfig{
		DownstreamURL: "https://default.example.com",
		APIKey:        "sk-default",
	})

	route, remainingPath, ok := router.Match("/zhipu/v4/chat")
	if !ok {
		t.Fatal("expected match")
	}
	if route.Prefix != "/zhipu" {
		t.Errorf("expected /zhipu prefix, got %s", route.Prefix)
	}
	if remainingPath != "/v4/chat" {
		t.Errorf("expected /v4/chat, got %s", remainingPath)
	}
}

func TestRouter_NoDefault_NoMatch(t *testing.T) {
	routes := []config.RouteConfig{
		{Prefix: "/zhipu", DownstreamURL: "https://open.bigmodel.cn/api/paas", APIKey: "sk-zhipu"},
	}
	router := NewRouter(routes)

	_, _, ok := router.Match("/unknown/path")
	if ok {
		t.Error("expected no match when default route is not configured")
	}
	if router.DefaultRoute() != nil {
		t.Error("expected nil default route")
	}
}
