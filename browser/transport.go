package browser

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
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

// SafeTransport combines SSRF protection with genuine Chrome uTLS, HTTP/2 multiplexing, and automatic Cloudflare WARP tunnel fallback.
type SafeTransport struct {
	dialer               *net.Dialer
	blockPrivateNetworks bool
	tr2                  *http2.Transport
	connPool             sync.Map
	warpDomains          sync.Map // host -> bool (domains that require Cloudflare WARP to bypass datacenter IP blocks)
}

func getWARPProxyAddr() string {
	if env := strings.TrimSpace(os.Getenv("AI_GO_WARP_PROXY")); env != "" {
		return env
	}
	return "127.0.0.1:40000"
}

func (s *SafeTransport) dialViaWARPProxy(ctx context.Context, hostPort string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return nil, err
	}
	if s.blockPrivateNetworks {
		lowerHost := strings.ToLower(host)
		if lowerHost == "localhost" || strings.HasSuffix(lowerHost, ".localhost") || lowerHost == "metadata.google.internal" {
			return nil, fmt.Errorf("%w: %s", ErrSSRFBlocked, host)
		}
	}

	// Preferred path: pure-Go in-process WARP tunnel (no external process)
	if w := getEmbeddedWARP(); w != nil {
		return w.dial(ctx, hostPort)
	}

	proxyAddr := getWARPProxyAddr()
	proxyConn, err := s.dialer.DialContext(ctx, "tcp4", proxyAddr)
	if err != nil {
		return nil, err
	}

	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: Mozilla/5.0\r\n\r\n", hostPort, hostPort)
	if _, err := proxyConn.Write([]byte(connectReq)); err != nil {
		_ = proxyConn.Close()
		return nil, err
	}

	br := bufio.NewReader(proxyConn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		_ = proxyConn.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = proxyConn.Close()
		return nil, fmt.Errorf("warp proxy CONNECT failed with status %d", resp.StatusCode)
	}

	return proxyConn, nil
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
	host := strings.ToLower(req.URL.Hostname())
	if _, useWARP := s.warpDomains.Load(host); useWARP {
		if resp, err := s.roundTripOnce(req, true); err == nil {
			if resp.Header != nil {
				resp.Header.Set("X-AI-Go-WARP-Fallback", "cached-domain")
			}
			return resp, nil
		}
	}

	resp, err := s.roundTripOnce(req, false)
	if err != nil {
		if errors.Is(err, ErrSSRFBlocked) {
			return nil, err
		}
		if warpResp, warpErr := s.roundTripOnce(req, true); warpErr == nil {
			if warpResp.Header != nil {
				warpResp.Header.Set("X-AI-Go-WARP-Fallback", "error-recovery")
			}
			return warpResp, nil
		}
		return nil, err
	}

	// If WAF/Cloudflare/Datacenter IP block triggers (403, 429, 503, 999), automatically retry through local Cloudflare WARP tunnel
	if resp.StatusCode == 403 || resp.StatusCode == 429 || resp.StatusCode == 503 || resp.StatusCode == 999 {
		if warpResp, warpErr := s.roundTripOnce(req, true); warpErr == nil {
			if warpResp.StatusCode < 400 || warpResp.StatusCode != resp.StatusCode {
				_ = resp.Body.Close()
				if warpResp.StatusCode < 400 {
					s.warpDomains.Store(host, true)
				}
				if warpResp.Header != nil {
					warpResp.Header.Set("X-AI-Go-WARP-Fallback", "status-healed")
				}
				return warpResp, nil
			}
			_ = warpResp.Body.Close()
		}
	}

	return resp, nil
}

func (s *SafeTransport) roundTripOnce(req *http.Request, useWARP bool) (*http.Response, error) {
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

	var (
		rawConn net.Conn
		err     error
	)
	if useWARP {
		rawConn, err = s.dialViaWARPProxy(req.Context(), addr)
	} else {
		rawConn, err = s.dialSafe(req.Context(), "tcp4", addr)
	}
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

	// HTTPS with genuine Chrome uTLS and GREASE (strip ALPS extension so standard x/net/http2 works seamlessly with Akamai/Cloudflare)
	uConfig := &utls.Config{
		ServerName: host,
	}
	uConn := buildChromeUConn(rawConn, uConfig)
	if err := uConn.HandshakeContext(req.Context()); err != nil {
		// Auto-fallback for government/academic portals with ICP-Brasil or self-signed x509 chains
		if strings.Contains(err.Error(), "x509:") {
			_ = rawConn.Close()
			var (
				rawConn2 net.Conn
				dialErr  error
			)
			if useWARP {
				rawConn2, dialErr = s.dialViaWARPProxy(req.Context(), addr)
			} else {
				rawConn2, dialErr = s.dialSafe(req.Context(), "tcp4", addr)
			}
			if dialErr == nil {
				rawConn = rawConn2
				uConfigInsecure := &utls.Config{
					ServerName:         host,
					InsecureSkipVerify: true,
				}
				uConn = buildChromeUConn(rawConn, uConfigInsecure)
				err = uConn.HandshakeContext(req.Context())
			}
		}
		if err != nil {
			close(stopWatcher)
			_ = rawConn.Close()
			return nil, err
		}
	}

	alpnProto := uConn.ConnectionState().NegotiatedProtocol
	if alpnProto == "" {
		alpnProto = uConn.HandshakeState.ServerHello.AlpnProtocol
	}

	// HTTP/2 negotiation
	if alpnProto == "h2" {
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
			// Auto-fallback to HTTP/1.1 over uTLS when WAF/CDN (e.g. Akamai) drops H2 frame fingerprint
			if strings.Contains(err.Error(), "closed network connection") || strings.Contains(err.Error(), "EOF") || strings.Contains(err.Error(), "reset") {
				return s.roundTripHTTP11TLS(req, host, addr)
			}
			return nil, err
		}
		resp.Body = &closingBody{ReadCloser: resp.Body, conn: nil, stop: stopWatcher}
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

func buildChromeUConn(rawConn net.Conn, uConfig *utls.Config) *utls.UConn {
	uConn := utls.UClient(rawConn, uConfig, utls.HelloCustom)
	if spec, err := utls.UTLSIdToSpec(utls.HelloChrome_120); err == nil {
		filtered := make([]utls.TLSExtension, 0, len(spec.Extensions))
		for _, ext := range spec.Extensions {
			if _, isALPS := ext.(*utls.ApplicationSettingsExtension); isALPS {
				continue
			}
			filtered = append(filtered, ext)
		}
		spec.Extensions = filtered
		_ = uConn.ApplyPreset(&spec)
		return uConn
	}
	return utls.UClient(rawConn, uConfig, utls.HelloChrome_120)
}

func (s *SafeTransport) roundTripHTTP11TLS(req *http.Request, host, addr string) (*http.Response, error) {
	rawConn, err := s.dialSafe(req.Context(), "tcp4", addr)
	if err != nil {
		return nil, err
	}
	ctx := req.Context()
	stopWatcher := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = rawConn.Close()
		case <-stopWatcher:
		}
	}()

	uConfig := &utls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
	}
	uConn := utls.UClient(rawConn, uConfig, utls.HelloCustom)
	if spec, specErr := utls.UTLSIdToSpec(utls.HelloChrome_120); specErr == nil {
		for _, ext := range spec.Extensions {
			if alpn, ok := ext.(*utls.ALPNExtension); ok {
				alpn.AlpnProtocols = []string{"http/1.1"}
			}
		}
		_ = uConn.ApplyPreset(&spec)
	}
	if err := uConn.HandshakeContext(ctx); err != nil {
		close(stopWatcher)
		_ = rawConn.Close()
		return nil, err
	}
	if err := req.Write(uConn); err != nil {
		close(stopWatcher)
		_ = uConn.Close()
		return nil, err
	}
	resp, err := http.ReadResponse(bufio.NewReader(uConn), req)
	if err != nil {
		close(stopWatcher)
		_ = uConn.Close()
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
	err := c.ReadCloser.Close()
	c.once.Do(func() {
		close(c.stop)
		if c.conn != nil {
			_ = c.conn.Close()
		}
	})
	return err
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
