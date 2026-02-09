package proxy

import "fmt"

// Settings contains proxy configuration for adapters.
type Settings struct {
	Enabled  bool
	Hostname string
	Port     int
	Username string
	Password string
}

// HasProxy returns true if proxy is enabled and configured.
func (p Settings) HasProxy() bool {
	return p.Enabled && p.Hostname != "" && p.Port > 0
}

// HostPort returns the proxy host:port string (e.g., "http://geo.iproyal.com:12321").
func (p Settings) HostPort() string {
	if !p.HasProxy() {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", p.Hostname, p.Port)
}

// FullURL returns the full proxy URL with credentials (for HTTP client).
func (p Settings) FullURL() string {
	if !p.HasProxy() {
		return ""
	}
	if p.Username != "" && p.Password != "" {
		return fmt.Sprintf("http://%s:%s@%s:%d", p.Username, p.Password, p.Hostname, p.Port)
	}
	return p.HostPort()
}
