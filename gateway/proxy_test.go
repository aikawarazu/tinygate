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
