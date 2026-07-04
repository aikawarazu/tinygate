package gateway

import (
	"bytes"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/user/tinygate/config"
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

	authValue := strings.ReplaceAll(route.AuthFormat, "${api_key}", route.APIKey)

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host

			// Insert version prefix if configured and request path lacks it
			reqPath := req.URL.Path
			if route.VersionPrefix != "" && !pathHasPrefixSegment(reqPath, route.VersionPrefix) {
				reqPath = route.VersionPrefix + reqPath
			}
			req.URL.Path = path.Join(target.Path, reqPath)

			req.Header.Set(route.AuthHeader, authValue)

			log.Printf("→ downstream: %s %s", req.Method, req.URL.String())
		},
		Transport: &http.Transport{
			ResponseHeaderTimeout: time.Duration(timeoutSeconds) * time.Second,
			ForceAttemptHTTP2:     false,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error: %s %s -> %v", r.Method, r.URL.Path, err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
	}

	return &Proxy{route: route, proxy: proxy}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.proxy.ServeHTTP(w, r)
}

// pathHasPrefixSegment checks if path has the given prefix as a full path segment.
// "/v1/chat" has prefix "/v1" => true, "/v12/chat" has prefix "/v1" => false.
func pathHasPrefixSegment(path, prefix string) bool {
	if path == prefix {
		return true
	}
	if strings.HasPrefix(path, prefix+"/") {
		return true
	}
	return false
}

func LoggingMiddleware(verbose bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if verbose {
			body := ""
			if r.Body != nil && r.Method == "POST" {
				b, err := io.ReadAll(r.Body)
				if err == nil {
					r.Body.Close()
					body = string(b)
					r.Body = io.NopCloser(bytes.NewBuffer(b))
					if len(body) > 200 {
						body = body[:200] + "..."
					}
				}
			}
			log.Printf("> %s %s", r.Method, r.URL.String())
			log.Printf("> headers: %v", r.Header)
			if body != "" {
				log.Printf("> body: %s", body)
			}
		}

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
