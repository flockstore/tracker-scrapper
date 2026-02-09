package proxy

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"tracker-scrapper/internal/core/logger"

	"github.com/elazarl/goproxy"
	"go.uber.org/zap"
)

// ForwardingProxy creates a local proxy that forwards requests to an upstream proxy with credentials.
// This solves Chromium's limitation of not supporting proxy authentication via command line.
type ForwardingProxy struct {
	localPort   int
	upstreamURL *url.URL
	server      *http.Server
	listener    net.Listener
	logger      *zap.Logger
	mu          sync.Mutex
	running     bool
}

// NewForwardingProxy creates a new forwarding proxy.
// upstreamURL should include credentials, e.g., "http://user:pass@host:port"
func NewForwardingProxy(upstreamURL string) (*ForwardingProxy, error) {
	parsed, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream proxy URL: %w", err)
	}

	return &ForwardingProxy{
		upstreamURL: parsed,
		logger:      logger.Get(),
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
	proxy.Verbose = true // Enable for debugging

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

	// Create a custom dial function that routes ALL connections through upstream proxy
	dialThroughProxy := func(network, addr string) (net.Conn, error) {
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
