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
	"sort"
	"strings"
	"time"

	"github.com/user/tinygate/config"
)

type Proxy struct {
	route config.RouteConfig
	proxy *httputil.ReverseProxy
}

func NewProxy(route config.RouteConfig, timeoutStr string, verbose bool) *Proxy {
	target, err := url.Parse(route.DownstreamURL)
	if err != nil {
		log.Fatalf("invalid downstream_url %s: %v", route.DownstreamURL, err)
	}

	timeoutSeconds, err := config.ParseTimeout(timeoutStr)
	if err != nil {
		log.Fatalf("invalid timeout %s: %v", timeoutStr, err)
	}

	authValue := strings.ReplaceAll(route.AuthFormat, "${api_key}", route.APIKey)

	transport := &http.Transport{
		ResponseHeaderTimeout: time.Duration(timeoutSeconds) * time.Second,
		ForceAttemptHTTP2:     false,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	var rt http.RoundTripper = transport
	if verbose {
		rt = &debugRoundTripper{next: transport}
	}

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
		},
		Transport: rt,
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

// debugRoundTripper logs all outgoing downstream requests and their responses
// when --debug flag is enabled. It logs the full request (URL, headers, body)
// before forwarding, and logs the response status/headers on non-2xx.
type debugRoundTripper struct {
	next http.RoundTripper
}

func (t *debugRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			log.Printf("=== DEBUG DOWNSTREAM: failed to read body: %v ===", err)
		}
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	log.Printf("=== DEBUG DOWNSTREAM REQUEST ===")
	log.Printf("Method:  %s", req.Method)
	log.Printf("URL:     %s", req.URL.String())
	log.Printf("Host:    %s", req.Host)
	log.Printf("--- Downstream Headers ---")
	keys := make([]string, 0, len(req.Header))
	for k := range req.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		log.Printf("  %s: %s", k, strings.Join(req.Header[k], ", "))
	}
	if len(bodyBytes) > 0 {
		log.Printf("--- downstream body: %d bytes (truncated) ---", len(bodyBytes))
	}
	log.Printf("=== END DEBUG DOWNSTREAM REQUEST ===")

	resp, err := t.next.RoundTrip(req)
	if err != nil {
		log.Printf("=== DEBUG DOWNSTREAM ERROR: %v ===", err)
		return resp, err
	}

	if resp.StatusCode >= 400 {
		log.Printf("=== DEBUG DOWNSTREAM RESPONSE ===")
		log.Printf("Status:  %d %s", resp.StatusCode, resp.Status)
		log.Printf("--- Response Headers ---")
		respKeys := make([]string, 0, len(resp.Header))
		for k := range resp.Header {
			respKeys = append(respKeys, k)
		}
		sort.Strings(respKeys)
		for _, k := range respKeys {
			log.Printf("  %s: %s", k, strings.Join(resp.Header[k], ", "))
		}
		log.Printf("=== END DEBUG DOWNSTREAM RESPONSE ===")
	}

	return resp, nil
}

func LoggingMiddleware(verbose bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if verbose {
			log.Printf("=== REQUEST ===")
			log.Printf("Method:  %s", r.Method)
			log.Printf("URL:     %s", r.URL.String())
			log.Printf("Proto:   %s", r.Proto)
			log.Printf("Host:    %s", r.Host)
			log.Printf("Remote:  %s", r.RemoteAddr)
			log.Printf("--- Request Headers ---")
			keys := make([]string, 0, len(r.Header))
			for k := range r.Header {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				log.Printf("  %s: %s", k, strings.Join(r.Header[k], ", "))
			}
			if r.Body != nil {
				b, err := io.ReadAll(r.Body)
				if err == nil {
					r.Body.Close()
					r.Body = io.NopCloser(bytes.NewBuffer(b))
					if len(b) > 0 {
						log.Printf("--- request body: %d bytes (truncated) ---", len(b))
					}
				}
			}
			log.Printf("=== END REQUEST ===")
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
