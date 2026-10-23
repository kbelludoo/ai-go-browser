package browser

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

var (
	// ErrSSRFBlocked is returned when a destination resolves to a private or internal network.
	ErrSSRFBlocked = errors.New("blocked destination: private/internal network address not allowed (SSRF protection)")
	// ErrExternalBrowserDisabled is returned when an action attempts to spawn an external browser while in NativeOnly mode.
	ErrExternalBrowserDisabled = errors.New("external browser disabled: browser is operating in strict native-only Go mode")
)

// isPrivateIP checks if an IP belongs to private, loopback, link-local, or cloud metadata ranges.
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	// IPv4 loopback (127.0.0.0/8) or IPv6 loopback (::1)
	if ip.IsLoopback() {
		return true
	}

	// IPv4 private ranges (RFC 1918)
	// 10.0.0.0/8
	// 172.16.0.0/12
	// 192.168.0.0/16
	if ip.IsPrivate() {
		return true
	}

	// Link-local unicast (169.254.0.0/16 or fe80::/10)
	// 169.254.169.254 is the standard AWS / GCP / Azure / Oracle Cloud metadata IP!
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Unspecified (0.0.0.0 or ::)
	if ip.IsUnspecified() {
		return true
	}

	// Check for IPv4 mapped into IPv6
	if ip4 := ip.To4(); ip4 != nil {
		// Specific cloud metadata check: 169.254.169.254
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		// 100.64.0.0/10 (Carrier-Grade NAT / Shared Address Space)
		if ip4[0] == 100 && (ip4[1]&0xC0) == 64 {
			return true
		}
	}

	return false
}

// SafeTransport combines SSRF protection with genuine Chrome uTLS and HTTP/2 multiplexing.
type SafeTransport struct {
	dialer               *net.Dialer
	blockPrivateNetworks bool
	tr2                  *http2.Transport
	connPool             sync.Map
}

func (s *SafeTransport) dialSafe(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	if s.blockPrivateNetworks {
		lowerHost := strings.ToLower(host)
		if lowerHost == "localhost" || strings.HasSuffix(lowerHost, ".localhost") || lowerHost == "metadata.google.internal" {
			return nil, fmt.Errorf("%w: %s", ErrSSRFBlocked, host)
		}

		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("dns resolution failed for %s: %w", host, err)
		}

		for _, ip := range ips {
			if isPrivateIP(ip) {
				return nil, fmt.Errorf("%w: host %s resolved to private IP %s", ErrSSRFBlocked, host, ip.String())
			}
		}

		if len(ips) > 0 {
			addr = net.JoinHostPort(ips[0].String(), port)
		}
	}

	return s.dialer.DialContext(ctx, network, addr)
}

func (s *SafeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Hostname()
	port := req.URL.Port()
	if port == "" {
		if req.URL.Scheme == "http" {
			port = "80"
		} else {
			port = "443"
		}
	}
	addr := net.JoinHostPort(host, port)

	rawConn, err := s.dialSafe(req.Context(), "tcp4", addr)
	if err != nil {
		return nil, err
	}

	// Active deadline & context watcher to prevent hanging connections
	ctx := req.Context()
	stopWatcher := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = rawConn.Close()
		case <-stopWatcher:
		}
	}()

	// Plain HTTP/1.1
	if req.URL.Scheme == "http" {
		if err := req.Write(rawConn); err != nil {
			close(stopWatcher)
			rawConn.Close()
			return nil, err
		}
		resp, err := http.ReadResponse(bufio.NewReader(rawConn), req)
		if err != nil {
			close(stopWatcher)
			rawConn.Close()
			return nil, err
		}
		// Wrap body to stop watcher on close
		resp.Body = &closingBody{ReadCloser: resp.Body, conn: rawConn, stop: stopWatcher}
		return resp, nil
	}

	// HTTPS with genuine Chrome uTLS and GREASE
	uConfig := &utls.Config{
		ServerName: host,
	}
	uConn := utls.UClient(rawConn, uConfig, utls.HelloChrome_120)
	if err := uConn.HandshakeContext(req.Context()); err != nil {
		close(stopWatcher)
		rawConn.Close()
		return nil, err
	}

	// HTTP/2 negotiation
	if uConn.ConnectionState().NegotiatedProtocol == "h2" {
		h2Conn, err := s.tr2.NewClientConn(uConn)
		if err != nil {
			close(stopWatcher)
			uConn.Close()
			return nil, err
		}
		resp, err := h2Conn.RoundTrip(req)
		if err != nil {
			close(stopWatcher)
			uConn.Close()
			return nil, err
		}
		resp.Body = &closingBody{ReadCloser: resp.Body, conn: uConn, stop: stopWatcher}
		return resp, nil
	}

	// Fallback HTTP/1.1 over uTLS
	if err := req.Write(uConn); err != nil {
		close(stopWatcher)
		uConn.Close()
		return nil, err
	}
	resp, err := http.ReadResponse(bufio.NewReader(uConn), req)
	if err != nil {
		close(stopWatcher)
		uConn.Close()
		return nil, err
	}
	resp.Body = &closingBody{ReadCloser: resp.Body, conn: uConn, stop: stopWatcher}
	return resp, nil
}

type closingBody struct {
	io.ReadCloser
	conn net.Conn
	stop chan struct{}
	once sync.Once
}

func (c *closingBody) Close() error {
	c.once.Do(func() {
		close(c.stop)
		_ = c.conn.Close()
	})
	return c.ReadCloser.Close()
}

// NewSafeTransport returns a SafeTransport implementing http.RoundTripper with DNS resolution verification against SSRF.
func NewSafeTransport(blockPrivateNetworks bool) http.RoundTripper {
	return &SafeTransport{
		dialer: &net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		},
		blockPrivateNetworks: blockPrivateNetworks,
		tr2:                  &http2.Transport{},
	}
}
