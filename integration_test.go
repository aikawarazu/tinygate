package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/user/just-llm-gateway/config"
	"github.com/user/just-llm-gateway/gateway"
)

func TestIntegration_EndToEnd(t *testing.T) {
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk-downstream" {
			t.Errorf("expected Bearer sk-downstream, got %s", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "Hello"}},
			},
		})
	}))
	defer downstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Port: 39901, Timeout: "300s", Health: true},
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

	router := gateway.NewRouter(cfg.Routes)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		route, remainingPath, ok := router.Match(r.URL.Path)
		if !ok {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		r.URL.Path = remainingPath
		proxy := gateway.NewProxy(*route, cfg.Server.Timeout)
		proxy.ServeHTTP(w, r)
	})

	handler := gateway.AuthMiddleware(cfg.Gateway.APIKeys, mux)

	body := bytes.NewBufferString(`{"model":"test","messages":[{"role":"user","content":"Hello"}]}`)
	req := httptest.NewRequest("POST", "/test/v1/chat/completions", body)
	req.Header.Set("Authorization", "Bearer sk-client")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(rec.Body).Decode(&resp)
	choices, ok := resp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		t.Error("expected choices in response")
	}
}
