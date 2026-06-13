package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

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
	configPath := flag.String("config", "config.yaml", "path to config file")
	verbose := flag.Bool("verbose", false, "enable verbose logging (print request/response details)")
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

	authProxy := gateway.AuthMiddleware(cfg.Gateway.Keys(), proxyMux)

	mux.Handle("/", authProxy)

	handler := gateway.LoggingMiddleware(*verbose, mux)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("starting server on %s", addr)

	printQuickstart(cfg.Server.Port)

	if *verbose {
		log.Println("verbose logging enabled")
	}

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
