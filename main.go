package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/user/tinygate/config"
	"github.com/user/tinygate/gateway"
)

const banner = `
  _____ _               _____      _
 |_   _(_)_ __  _   _  / ____|    | |
   | | | | '_ \| | | || |  __ __ _| |_ ___ _ __
   | | | | | | | |_| || | |_ | _| | __/ _ \ '__|
   |_| |_|_| |_|\__, | \____|__|_|\__\___/_|
                __/ |
               |___/
`

func printQuickstart(port int) {
	fmt.Print(banner)
	fmt.Println(strings.Repeat("─", 55))
	fmt.Println("  Quick Start")
	fmt.Println(strings.Repeat("─", 55))
	fmt.Println()
	fmt.Printf("  # Health check\n  curl http://localhost:%d/health\n\n", port)
	fmt.Printf("  # List OpenCode Go models\n  curl -s http://localhost:%d/opencode/v1/models \\\n    -H \"Authorization: Bearer sk-gateway-key-1\"\n\n", port)
	fmt.Printf("  # Chat\n  curl -s http://localhost:%d/opencode/v1/chat/completions \\\n    -H \"Authorization: Bearer sk-gateway-key-1\" \\\n    -H \"Content-Type: application/json\" \\\n    -d '{\"model\":\"deepseek-v4-flash\",\"messages\":[{\"role\":\"user\",\"content\":\"你好\"}],\"max_tokens\":100}'\n\n", port)
	fmt.Printf("  # Stream\n  curl -sN http://localhost:%d/opencode/v1/chat/completions \\\n    -H \"Authorization: Bearer sk-gateway-key-1\" \\\n    -H \"Content-Type: application/json\" \\\n    -d '{\"model\":\"deepseek-v4-flash\",\"messages\":[{\"role\":\"user\",\"content\":\"你好\"}],\"max_tokens\":100,\"stream\":true}'\n\n", port)
	fmt.Println(strings.Repeat("─", 55))
}

func main() {
	configPath := flag.String("config", "/etc/tinygate", "path to config file")
	flag.Parse()

	exitReason := "normal shutdown"

	data, err := os.ReadFile(*configPath)
	if err != nil {
		log.Fatalf("failed to read config %s: %v", *configPath, err)
	}

	cfg, err := config.ParseConfig(data)
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	router := gateway.NewRouter(cfg.Routes)
	if cfg.DefaultRoute != nil {
		router.SetDefault(cfg.DefaultRoute)
	}

	proxies := make(map[string]*gateway.Proxy)
	for _, route := range cfg.Routes {
		proxies[route.Prefix] = gateway.NewProxy(route, cfg.Server.Timeout)
	}

	// The default route has no prefix entry in `proxies`; keep its proxy separate
	// and dispatch to it when the router falls back to the default.
	var defaultProxy *gateway.Proxy
	if cfg.DefaultRoute != nil {
		defaultProxy = gateway.NewProxy(*cfg.DefaultRoute, cfg.Server.Timeout)
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

		if route == router.DefaultRoute() {
			if defaultProxy == nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			defaultProxy.ServeHTTP(w, r)
			return
		}

		proxy, ok := proxies[route.Prefix]
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		proxy.ServeHTTP(w, r)
	})

	keys := cfg.Gateway.Keys()
	handler := http.Handler(proxyMux)
	if !(len(keys) == 1 && keys[0] == "noauth") {
		handler = gateway.AuthMiddleware(keys, handler)
	}
	mux.Handle("/", handler)

	handler = gateway.LoggingMiddleware(http.Handler(mux))

	addr := fmt.Sprintf(":%d", cfg.Server.Port)

	srv := &http.Server{Addr: addr, Handler: handler}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-quit
		exitReason = fmt.Sprintf("received signal: %v", sig)
		log.Printf("shutting down: %s", exitReason)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("starting server on %s", addr)
	printQuickstart(cfg.Server.Port)

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		exitReason = fmt.Sprintf("server error: %v", err)
		log.Fatalf("exit: %s", exitReason)
	}

	log.Printf("exit: %s", exitReason)
}
