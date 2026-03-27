package transport

import (
    "fmt"
    "net"
    "net/url"
    "time"

    "golang.org/x/net/proxy"
)

// KDCDialer abstracts direct TCP vs SOCKS5-proxied TCP.
type KDCDialer interface {
    Dial(network, addr string) (net.Conn, error)
}

// DirectDialer connects to KDCs directly using net.Dialer.
type DirectDialer struct {
    Timeout time.Duration
}

func (d *DirectDialer) Dial(network, addr string) (net.Conn, error) {
    dialer := &net.Dialer{Timeout: d.Timeout}
    return dialer.Dial(network, addr)
}

// SOCKSDialer connects to KDCs through a SOCKS5 proxy.
type SOCKSDialer struct {
    ProxyAddr string        // e.g. "127.0.0.1:1080"
    Username  string        // optional SOCKS5 auth
    Password  string        // optional SOCKS5 auth
    Timeout   time.Duration // connection timeout
    inner     proxy.Dialer  // initialized on first use
}

func (d *SOCKSDialer) init() error {
    if d.inner != nil {
        return nil
    }

    var auth *proxy.Auth
    if d.Username != "" {
        auth = &proxy.Auth{
            User:     d.Username,
            Password: d.Password,
        }
    }

    // The forward dialer connects FROM us TO the SOCKS5 proxy itself.
    forward := &net.Dialer{Timeout: d.Timeout}

    dialer, err := proxy.SOCKS5("tcp", d.ProxyAddr, auth, forward)
    if err != nil {
        return fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
    }
    d.inner = dialer
    return nil
}

func (d *SOCKSDialer) Dial(network, addr string) (net.Conn, error) {
    if err := d.init(); err != nil {
        return nil, err
    }
    return d.inner.Dial("tcp", addr) // Force TCP through SOCKS5
}

// ParseProxyURL parses a proxy string into a SOCKSDialer.
func ParseProxyURL(raw string, timeout time.Duration) (*SOCKSDialer, error) {
    if !hasScheme(raw) {
        raw = "socks5://" + raw
    }

    u, err := url.Parse(raw)
    if err != nil {
        return nil, fmt.Errorf("invalid proxy URL %q: %w", raw, err)
    }

    if u.Scheme != "socks5" {
        return nil, fmt.Errorf("unsupported proxy scheme %q (only socks5 is supported)", u.Scheme)
    }

    d := &SOCKSDialer{
        ProxyAddr: u.Host,
        Timeout:   timeout,
    }

    if u.User != nil {
        d.Username = u.User.Username()
        d.Password, _ = u.User.Password()
    }

    return d, nil
}

func hasScheme(s string) bool {
    for i, c := range s {
        if c == ':' {
            return i > 0
        }
        if !isSchemeChar(c, i == 0) {
            return false
        }
    }
    return false
}

func isSchemeChar(c rune, first bool) bool {
    if first {
        return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
    }
    return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
        (c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.'
}
