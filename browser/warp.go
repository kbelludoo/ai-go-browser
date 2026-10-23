package browser

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

// embeddedWARP is a pure-Go, in-process Cloudflare WARP (WireGuard userspace) tunnel.
// It replaces the external wireproxy/warp-cli process: no extra daemon, no extra service.
type embeddedWARP struct {
	tnet *netstack.Net
	dev  *device.Device
}

var (
	warpOnce   sync.Once
	warpTunnel *embeddedWARP
)

// getEmbeddedWARP lazily brings the tunnel up from a wgcf-profile.conf file.
// Path comes from AI_GO_WARP_CONF (default ./wgcf-profile.conf). Returns nil when unavailable.
func getEmbeddedWARP() *embeddedWARP {
	warpOnce.Do(func() {
		path := strings.TrimSpace(os.Getenv("AI_GO_WARP_CONF"))
		if path == "" {
			path = "wgcf-profile.conf"
		}
		t, err := startEmbeddedWARP(path)
		if err != nil {
			return
		}
		warpTunnel = t
	})
	return warpTunnel
}

func b64ToHex(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func startEmbeddedWARP(confPath string) (*embeddedWARP, error) {
	f, err := os.Open(confPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var priv, pub, endpoint string
	var addrs, dnsList []netip.Addr
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch strings.ToLower(k) {
		case "privatekey":
			priv = v
		case "publickey":
			pub = v
		case "endpoint":
			endpoint = v
		case "address":
			for _, p := range strings.Split(v, ",") {
				if pf, err := netip.ParsePrefix(strings.TrimSpace(p)); err == nil {
					addrs = append(addrs, pf.Addr())
				}
			}
		case "dns":
			for _, p := range strings.Split(v, ",") {
				if a, err := netip.ParseAddr(strings.TrimSpace(p)); err == nil {
					dnsList = append(dnsList, a)
				}
			}
		}
	}
	if priv == "" || pub == "" || endpoint == "" || len(addrs) == 0 {
		return nil, fmt.Errorf("incomplete WARP profile %s", confPath)
	}
	if len(dnsList) == 0 {
		dnsList = []netip.Addr{netip.MustParseAddr("1.1.1.1")}
	}

	privHex, err := b64ToHex(priv)
	if err != nil {
		return nil, err
	}
	pubHex, err := b64ToHex(pub)
	if err != nil {
		return nil, err
	}
	udpAddr, err := net.ResolveUDPAddr("udp4", endpoint)
	if err != nil {
		return nil, err
	}

	tdev, tnet, err := netstack.CreateNetTUN(addrs, dnsList, 1280)
	if err != nil {
		return nil, err
	}
	dev := device.NewDevice(tdev, conn.NewDefaultBind(), device.NewLogger(device.LogLevelSilent, ""))
	ipc := fmt.Sprintf("private_key=%s\npublic_key=%s\nallowed_ip=0.0.0.0/0\nallowed_ip=::/0\nendpoint=%s\npersistent_keepalive_interval=25\n",
		privHex, pubHex, udpAddr.String())
	if err := dev.IpcSet(ipc); err != nil {
		dev.Close()
		return nil, err
	}
	if err := dev.Up(); err != nil {
		dev.Close()
		return nil, err
	}
	return &embeddedWARP{tnet: tnet, dev: dev}, nil
}

func (w *embeddedWARP) dial(ctx context.Context, hostPort string) (net.Conn, error) {
	return w.tnet.DialContext(ctx, "tcp", hostPort)
}
