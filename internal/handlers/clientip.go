package handlers

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// cloudflareRanges are Cloudflare's edge addresses
// (https://www.cloudflare.com/ips/, checked 2026-10-11).
var cloudflareRanges = func() []netip.Prefix {
	var out []netip.Prefix
	for _, s := range []string{
		"173.245.48.0/20", "103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22",
		"141.101.64.0/18", "108.162.192.0/18", "190.93.240.0/20", "188.114.96.0/20",
		"197.234.240.0/22", "198.41.128.0/17", "162.158.0.0/15", "104.16.0.0/13",
		"104.24.0.0/14", "172.64.0.0/13", "131.0.72.0/22",
		"2400:cb00::/32", "2606:4700::/32", "2803:f800::/32", "2405:b500::/32",
		"2405:8100::/32", "2a06:98c0::/29", "2c0f:f248::/32",
	} {
		out = append(out, netip.MustParsePrefix(s))
	}
	return out
}()

func fromCloudflare(ip netip.Addr) bool {
	for _, p := range cloudflareRanges {
		if p.Contains(ip.Unmap()) {
			return true
		}
	}
	return false
}

// clientIP sets r.RemoteAddr to the visitor's address, replacing chi's
// RealIP (which trusts headers anyone can send). It assumes the deploy
// stack is Cloudflare, then Traefik (Dokploy), which puts the address it saw
// (a Cloudflare edge) in X-Real-Ip. Only when that address is Cloudflare's is
// CF-Connecting-IP (the visitor) trusted. Behind another proxy, adapt this.
func clientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		peer := strings.TrimSpace(r.Header.Get("X-Real-Ip"))
		if peer == "" {
			peer, _, _ = net.SplitHostPort(r.RemoteAddr)
		}
		if addr, err := netip.ParseAddr(peer); err == nil {
			r.RemoteAddr = addr.String()
			if fromCloudflare(addr) {
				if visitor, err := netip.ParseAddr(strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))); err == nil {
					r.RemoteAddr = visitor.String()
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
