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

	data, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	cfg, err := config.ParseConfig(data)
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	router := gateway.NewRouter(cfg.Routes)

	proxies := make(map[string]*gateway.Proxy)
	for _, route := range cfg.Routes {
		proxies[route.Prefix] = gateway.NewProxy(route, cfg.Server.Timeout)
	}

	mux := http.NewServeMux()

	if cfg.Server.Health {
		mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
	}

	proxyMux := http.NewServeMux()
	proxyMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		route, remainingPath, ok := router.Match(r.URL.Path)
		if !ok {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		r.URL.Path = remainingPath

		proxy, ok := proxies[route.Prefix]
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		proxy.ServeHTTP(w, r)
	})

	authProxy := gateway.AuthMiddleware(cfg.Gateway.APIKeys, proxyMux)

	mux.Handle("/", authProxy)

	handler := gateway.LoggingMiddleware(mux)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("starting server on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
