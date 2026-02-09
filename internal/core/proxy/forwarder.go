package proxy

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"tracker-scrapper/internal/core/logger"

	"github.com/elazarl/goproxy"
	"go.uber.org/zap"
)

// ForwardingProxy creates a local proxy that forwards requests to an upstream proxy with credentials.
// This solves Chromium's limitation of not supporting proxy authentication via command line.
type ForwardingProxy struct {
	localPort      int
	upstreamURL    *url.URL
	server         *http.Server
	listener       net.Listener
	logger         *zap.Logger
	mu             sync.Mutex
	running        bool
	allowedDomains []string
}

// RedirectLogger adapts zap logger to goproxy.Logger interface
type RedirectLogger struct {
	logger *zap.Logger
}

func (l *RedirectLogger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	// Filter out verbose CONNECT handler logs if desired, or log as debug
	if strings.Contains(msg, "Running") && strings.Contains(msg, "CONNECT handlers") {
		return
	}
	l.logger.Debug("goproxy: " + msg)
}

// NewForwardingProxy creates a new forwarding proxy.
// upstreamURL should include credentials, e.g., "http://user:pass@host:port"
// allowedDomains is a list of domains to allow (e.g., "mobile.servientrega.com"). If empty, all domains are allowed.
func NewForwardingProxy(upstreamURL string, allowedDomains ...string) (*ForwardingProxy, error) {
	parsed, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream proxy URL: %w", err)
	}

	return &ForwardingProxy{
		upstreamURL:    parsed,
		logger:         logger.Get(),
		allowedDomains: allowedDomains,
	}, nil
}

// Start launches the local proxy server on a random available port.
// Returns the local address (e.g., "127.0.0.1:18080") for Chromium to use.
func (fp *ForwardingProxy) Start(ctx context.Context) (string, error) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if fp.running {
		return fp.LocalAddr(), nil
	}

	// Create goproxy server
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true // Keep verbose but redirect logging
	proxy.Logger = &RedirectLogger{logger: fp.logger}

	// Extract credentials from upstream URL
	var proxyAuth string
	if fp.upstreamURL.User != nil {
		username := fp.upstreamURL.User.Username()
		password, _ := fp.upstreamURL.User.Password()
		credentials := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		proxyAuth = "Basic " + credentials
	}

	// Build upstream host for the transport
	upstreamHost := fp.upstreamURL.Host
	log := fp.logger

	// Helper to check if a host is allowed
	isAllowed := func(host string) bool {
		if len(fp.allowedDomains) == 0 {
			return true
		}
		// Strip port if present
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		for _, domain := range fp.allowedDomains {
			if strings.HasSuffix(host, domain) {
				return true
			}
		}
		return false
	}

	// Create a custom dial function that routes ALL connections through upstream proxy
	dialThroughProxy := func(network, addr string) (net.Conn, error) {
		// Check allowlist
		if !isAllowed(addr) {
			log.Debug("Blocked connection to disallowed domain", zap.String("target", addr))
			return nil, fmt.Errorf("access denied to domain: %s", addr)
		}

		log.Debug("ConnectDial called",
			zap.String("network", network),
			zap.String("target", addr),
			zap.String("upstream", upstreamHost),
		)

		// Connect to upstream proxy
		conn, err := net.DialTimeout("tcp", upstreamHost, 30*time.Second)
		if err != nil {
			log.Error("Failed to dial upstream proxy",
				zap.String("upstream", upstreamHost),
				zap.Error(err),
			)
			return nil, fmt.Errorf("failed to connect to upstream proxy %s: %w", upstreamHost, err)
		}

		// Send CONNECT request to upstream proxy
		connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", addr, addr)
		if proxyAuth != "" {
			connectReq += fmt.Sprintf("Proxy-Authorization: %s\r\n", proxyAuth)
		}
		connectReq += "\r\n"

		log.Debug("Sending CONNECT to upstream", zap.String("target", addr))

		if _, err := conn.Write([]byte(connectReq)); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to send CONNECT request: %w", err)
		}

		// Read response from upstream proxy
		br := bufio.NewReader(conn)
		resp, err := http.ReadResponse(br, nil)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to read CONNECT response: %w", err)
		}

		if resp.StatusCode != 200 {
			conn.Close()
			log.Error("Upstream proxy rejected CONNECT",
				zap.Int("status", resp.StatusCode),
				zap.String("target", addr),
			)
			return nil, fmt.Errorf("upstream proxy CONNECT failed with status: %d", resp.StatusCode)
		}

		log.Debug("CONNECT tunnel established", zap.String("target", addr))
		return conn, nil
	}

	// Set ConnectDial for HTTPS CONNECT requests
	proxy.ConnectDial = dialThroughProxy

	// Also set Tr.Dial to route HTTP requests through the proxy tunnel
	proxy.Tr = &http.Transport{
		Dial: dialThroughProxy,
	}

	// Add Proxy-Authorization header for regular HTTP requests
	// AND filter requests based on allowlist
	proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		if !isAllowed(req.URL.Host) {
			log.Debug("Blocked HTTP request to disallowed domain", zap.String("url", req.URL.String()))
			return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusForbidden, "Access Denied")
		}

		if proxyAuth != "" {
			req.Header.Set("Proxy-Authorization", proxyAuth)
		}
		return req, nil
	})

	// Handle CONNECT (HTTPS) requests - reject if not allowed
	proxy.OnRequest().HandleConnectFunc(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		if !isAllowed(host) {
			log.Debug("Blocked CONNECT request to disallowed domain", zap.String("host", host))
			return goproxy.RejectConnect, host
		}
		return goproxy.OkConnect, host
	})

	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to find available port: %w", err)
	}
	fp.listener = listener
	fp.localPort = listener.Addr().(*net.TCPAddr).Port

	fp.server = &http.Server{
		Handler: proxy,
	}

	fp.logger.Debug("Starting local proxy forwarder",
		zap.String("local_addr", fp.LocalAddr()),
		zap.String("upstream", upstreamHost),
		zap.Strings("allowed_domains", fp.allowedDomains),
	)

	// Start server in background
	go func() {
		if err := fp.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fp.logger.Error("Local proxy server error", zap.Error(err))
		}
	}()

	fp.running = true

	// Give the server a moment to start
	time.Sleep(50 * time.Millisecond)

	return fp.LocalAddr(), nil
}

// Stop gracefully shuts down the local proxy server.
func (fp *ForwardingProxy) Stop() error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	if !fp.running {
		return nil
	}

	fp.logger.Debug("Stopping local proxy forwarder")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := fp.server.Shutdown(ctx); err != nil {
		fp.listener.Close()
		return err
	}

	fp.running = false
	return nil
}

// LocalAddr returns the local proxy address for Chromium to connect to.
// Returns format "http://127.0.0.1:<port>"
func (fp *ForwardingProxy) LocalAddr() string {
	return fmt.Sprintf("http://127.0.0.1:%d", fp.localPort)
}

// IsRunning returns whether the proxy server is currently running.
func (fp *ForwardingProxy) IsRunning() bool {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	return fp.running
}
