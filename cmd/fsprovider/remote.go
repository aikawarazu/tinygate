package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
)

func main() {
	host := flag.String("host", "", "SSH server host")
	port := flag.Int("port", 22, "SSH server port")
	user := flag.String("user", "", "SSH user")
	sshArgs := flag.String("ssh-args", "", "path to SSH private key")
	localPort := flag.Int("local-port", 0, "local port to listen on")
	remoteHost := flag.String("remote-host", "localhost", "remote host to forward to")
	remotePort := flag.Int("remote-port", 0, "remote port to forward to")
	debug := flag.Bool("debug", false, "enable debug logging (print SSH connection details, request/response info)")
	httpOnly := flag.Bool("http-only", false, "only forward HTTP requests")
	flag.Parse()

	if *host == "" || *user == "" || *sshArgs == "" || *localPort == 0 || *remotePort == 0 {
		fmt.Fprintf(os.Stderr, "Usage: fsprovider-remote --host <host> --user <user> --ssh-args <path> --local-port <port> --remote-port <port> [--remote-host <host>] [--debug]\n")
		os.Exit(1)
	}

	keyBytes, err := os.ReadFile(*sshArgs)
	if err != nil {
		log.Fatalf("failed to read private key %s: %v", *sshArgs, err)
	}

	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		log.Fatalf("failed to parse private key: %v", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: *user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", *host, *port)
	if *debug {
		log.Printf("connecting to SSH server %s as %s", addr, *user)
	}

	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		log.Fatalf("SSH connection failed: %v", err)
	}
	defer sshClient.Close()

	if *debug {
		log.Printf("SSH connected: server version=%s", string(sshClient.ServerVersion()))
	}

	remoteAddr := fmt.Sprintf("%s:%d", *remoteHost, *remotePort)

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = remoteAddr
		},
		Transport: &sshRoundTripper{
			client:     sshClient,
			remoteAddr: remoteAddr,
			debug:      *debug,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error: %s %s -> %v", r.Method, r.URL.Path, err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		},
	}

	localAddr := fmt.Sprintf(":%d", *localPort)
	server := &http.Server{
		Addr:    localAddr,
		Handler: loggingHandler(proxy, *debug),
	}

	if *debug {
		mode := "SSH tunnel"
		if *httpOnly {
			mode = "SSH tunnel (HTTP only)"
		}
		log.Printf("proxy listening on %s, forwarding to %s via %s", localAddr, remoteAddr, mode)
	}
	fmt.Printf("listening on %s, forwarding to %s via SSH\n", localAddr, remoteAddr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		if *debug {
			log.Printf("received signal %v, shutting down", sig)
		}
		server.Close()
	}()

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func loggingHandler(next http.Handler, debug bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if debug {
			log.Printf("=== REQUEST ===")
			log.Printf("Method:  %s", r.Method)
			log.Printf("URL:     %s", r.URL.String())
			log.Printf("Host:    %s", r.Host)
			log.Printf("Remote:  %s", r.RemoteAddr)
			log.Printf("--- Headers ---")
			keys := make([]string, 0, len(r.Header))
			for k := range r.Header {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				log.Printf("  %s: %s", k, strings.Join(r.Header[k], ", "))
			}
			if r.Body != nil && r.ContentLength != 0 {
				b, err := io.ReadAll(r.Body)
				if err == nil {
					r.Body.Close()
					r.Body = io.NopCloser(bytes.NewBuffer(b))
					log.Printf("--- Body (%d bytes) ---", len(b))
					log.Printf("%s", string(b))
					log.Printf("--- End Body ---")
				}
			}
			log.Printf("=== END REQUEST ===")
		}

		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		if debug {
			log.Printf("%s %s -> %d %v", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
		} else {
			log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
		}
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

type sshRoundTripper struct {
	client     *ssh.Client
	remoteAddr string
	debug      bool
}

func (t *sshRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.debug {
		log.Printf("=== SSH TUNNEL ===")
		log.Printf("dialing remote %s via SSH", t.remoteAddr)
	}

	conn, err := t.client.Dial("tcp", t.remoteAddr)
	if err != nil {
		if t.debug {
			log.Printf("SSH dial failed: %v", err)
		}
		return nil, err
	}

	if t.debug {
		log.Printf("SSH tunnel connected to %s", t.remoteAddr)
	}

	if err := req.Write(conn); err != nil {
		conn.Close()
		return nil, err
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		conn.Close()
		return nil, err
	}

	resp.Body = &bodyCloser{conn: conn, ReadCloser: resp.Body}

	if t.debug {
		log.Printf("SSH tunnel response: %d %s", resp.StatusCode, resp.Status)
		log.Printf("=== END SSH TUNNEL ===")
	}

	return resp, nil
}

type bodyCloser struct {
	conn net.Conn
	io.ReadCloser
}

func (b *bodyCloser) Close() error {
	err := b.ReadCloser.Close()
	b.conn.Close()
	return err
}
