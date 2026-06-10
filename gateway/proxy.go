package gateway

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/user/just-llm-gateway/config"
)

type Proxy struct {
	route config.RouteConfig
	proxy *httputil.ReverseProxy
}

func NewProxy(route config.RouteConfig, timeoutStr string) *Proxy {
	target, err := url.Parse(route.DownstreamURL)
	if err != nil {
		log.Fatalf("invalid downstream_url %s: %v", route.DownstreamURL, err)
	}

	timeoutSeconds, err := config.ParseTimeout(timeoutStr)
	if err != nil {
		log.Fatalf("invalid timeout %s: %v", timeoutStr, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host

		authValue := strings.ReplaceAll(route.AuthFormat, "${api_key}", route.APIKey)
		req.Header.Set(route.AuthHeader, authValue)

		req.Header.Del("Host")
	}

	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: time.Duration(timeoutSeconds) * time.Second,
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("proxy error: %s %s -> %v", r.Method, r.URL.Path, err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return &Proxy{route: route, proxy: proxy}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
}

// LoggingMiddleware logs request summary
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
