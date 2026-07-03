package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/user/tinygate/config"
)

func TestProxy_DirectorRewritesRequest(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("expected path /v1/chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-downstream" {
			t.Errorf("expected Bearer sk-downstream, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Custom") != "custom-value" {
			t.Errorf("expected X-Custom to be preserved, got %s", r.Header.Get("X-Custom"))
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

	proxy := NewProxy(route, "300s")

	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer sk-client-key")
	req.Header.Set("X-Custom", "custom-value")
	req.Header.Set("Host", "localhost:39901")

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestProxy_VersionPrefixInsertion(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Request without /v1 in path should have /v1 inserted
		if r.URL.Path != "/zen/go/v1/chat/completions" {
			t.Errorf("expected path /zen/go/v1/chat/completions, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	route := config.RouteConfig{
		Prefix:        "/opencode",
		DownstreamURL: downstream.URL + "/zen/go",
		APIKey:        "sk-downstream",
		AuthHeader:    "Authorization",
		AuthFormat:    "Bearer ${api_key}",
		VersionPrefix: "/v1",
	}

	proxy := NewProxy(route, "300s")

	// Request without version prefix in path
	req := httptest.NewRequest("POST", "/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer sk-client-key")

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestProxy_VersionPrefix_ExactMatch(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/zen/go/v1" {
			t.Errorf("expected path /zen/go/v1, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	route := config.RouteConfig{
		Prefix:        "/opencode",
		DownstreamURL: downstream.URL + "/zen/go",
		APIKey:        "sk-downstream",
		AuthHeader:    "Authorization",
		AuthFormat:    "Bearer ${api_key}",
		VersionPrefix: "/v1",
	}

	proxy := NewProxy(route, "300s")

	// Request path equals version prefix exactly
	req := httptest.NewRequest("POST", "/v1", nil)
	req.Header.Set("Authorization", "Bearer sk-client-key")

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestProxy_VersionPrefix_DifferentVersionPrefixInserted(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /v12 should NOT match /v1, so version prefix IS inserted -> /v1/v12/...
		// This is correct behavior: if requesting a different version, trust the path as-is
		if r.URL.Path != "/zen/go/v1/v12/chat" {
			t.Errorf("expected path /zen/go/v1/v12/chat, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	route := config.RouteConfig{
		Prefix:        "/opencode",
		DownstreamURL: downstream.URL + "/zen/go",
		APIKey:        "sk-downstream",
		AuthHeader:    "Authorization",
		AuthFormat:    "Bearer ${api_key}",
		VersionPrefix: "/v1",
	}

	proxy := NewProxy(route, "300s")

	// Request with /v12 (different version) should still get /v1 inserted
	// because the segment-aware check correctly rejects /v12 as matching /v1
	req := httptest.NewRequest("POST", "/v12/chat", nil)
	req.Header.Set("Authorization", "Bearer sk-client-key")

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestProxy_VersionPrefixAlreadyPresent(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Request WITH /v1 in path should NOT have it inserted again
		if r.URL.Path != "/zen/go/v1/chat/completions" {
			t.Errorf("expected path /zen/go/v1/chat/completions, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer downstream.Close()

	route := config.RouteConfig{
		Prefix:        "/opencode",
		DownstreamURL: downstream.URL + "/zen/go",
		APIKey:        "sk-downstream",
		AuthHeader:    "Authorization",
		AuthFormat:    "Bearer ${api_key}",
		VersionPrefix: "/v1",
	}

	proxy := NewProxy(route, "300s")

	// Request WITH version prefix in path - should not duplicate
	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer sk-client-key")

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

	proxy := NewProxy(route, "300s")

	req := httptest.NewRequest("POST", "/v1/messages", nil)
	req.Header.Set("Authorization", "Bearer sk-client-key")

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
