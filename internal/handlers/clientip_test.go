package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name               string
		remote, realIP, cf string
		want               string
	}{
		{"through Cloudflare", "10.0.1.5:4242", "172.64.213.54", "203.0.113.7", "203.0.113.7"},
		{"Cloudflare over IPv6", "10.0.1.5:4242", "2606:4700::1", "2001:db8::42", "2001:db8::42"},
		{"spoofed header, not from Cloudflare", "10.0.1.5:4242", "198.51.100.9", "203.0.113.7", "198.51.100.9"},
		{"Cloudflare without the header", "10.0.1.5:4242", "172.64.213.54", "", "172.64.213.54"},
		{"garbage header", "10.0.1.5:4242", "172.64.213.54", "not-an-ip", "172.64.213.54"},
		{"no proxy (local)", "127.0.0.1:5555", "", "203.0.113.7", "127.0.0.1"},
	}
	for _, c := range cases {
		var got string
		h := clientIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = r.RemoteAddr }))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = c.remote
		if c.realIP != "" {
			req.Header.Set("X-Real-Ip", c.realIP)
		}
		if c.cf != "" {
			req.Header.Set("CF-Connecting-IP", c.cf)
		}
		h.ServeHTTP(httptest.NewRecorder(), req)
		if got != c.want {
			t.Errorf("%s: RemoteAddr = %q, want %q", c.name, got, c.want)
		}
	}
}
