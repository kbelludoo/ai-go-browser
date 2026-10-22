package browser

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	utls "github.com/refraction-networking/utls"
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

// NewSafeTransport returns an http.Transport with DNS resolution verification against SSRF.
func NewSafeTransport(blockPrivateNetworks bool) *http.Transport {
	customDialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	safeDialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		// If private network blocking is enabled, resolve DNS and inspect IPs
		if blockPrivateNetworks {
			// Check direct localhost string matches
			lowerHost := strings.ToLower(host)
			if lowerHost == "localhost" || strings.HasSuffix(lowerHost, ".localhost") || lowerHost == "metadata.google.internal" {
				return nil, fmt.Errorf("%w: %s", ErrSSRFBlocked, host)
			}

			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, fmt.Errorf("dns resolution failed for %s: %w", host, err)
			}

			// Validate every resolved IP
			for _, ip := range ips {
				if isPrivateIP(ip) {
					return nil, fmt.Errorf("%w: host %s resolved to private IP %s", ErrSSRFBlocked, host, ip.String())
				}
			}

			// Dial the first valid resolved IP directly to prevent DNS rebinding attacks between check and connect
			if len(ips) > 0 {
				addr = net.JoinHostPort(ips[0].String(), port)
			}
		}

		return customDialer.DialContext(ctx, network, addr)
	}

	safeDialTLSContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		rawConn, err := safeDialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}

		uConfig := &utls.Config{
			ServerName: host,
		}
		uConn := utls.UClient(rawConn, uConfig, utls.HelloChrome_120)
		if err := uConn.HandshakeContext(ctx); err != nil {
			_ = rawConn.Close()
			return nil, err
		}
		return uConn, nil
	}

	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           safeDialContext,
		DialTLSContext:        safeDialTLSContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
}
