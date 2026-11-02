package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// These tests exercise the router and middleware without a database: none
// of the routes they hit run a query.

func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// HEAD requests get the GET route's status and headers instead of 405.
func TestHead(t *testing.T) {
	h := New(Deps{}).Routes()
	for path, want := range map[string]int{
		"/robots.txt":  http.StatusOK,
		"/nope":        http.StatusNotFound,
		"/counter/inc": http.StatusMethodNotAllowed, // POST only
	} {
		if rec := serve(h, httptest.NewRequest(http.MethodHead, path, nil)); rec.Code != want {
			t.Errorf("HEAD %s: %d, want %d", path, rec.Code, want)
		}
	}
}

// Pages are gzipped for clients that accept it.
func TestCompress(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := serve(New(Deps{}).Routes(), req)
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("404 page: Content-Encoding %q, want gzip", rec.Header().Get("Content-Encoding"))
	}
}

// Unknown paths get the 404 page with the layout, not a bare text error.
func TestNotFound(t *testing.T) {
	rec := serve(New(Deps{}).Routes(), httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rec.Code != http.StatusNotFound || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("/nope: %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "Page not found") || !strings.Contains(rec.Body.String(), "<!doctype html>") {
		t.Errorf("404 page lacks the layout or message:\n%s", rec.Body.String())
	}
}

func TestRobots(t *testing.T) {
	get := func(d Deps) *httptest.ResponseRecorder {
		rec := serve(New(d).Routes(), httptest.NewRequest(http.MethodGet, "/robots.txt", nil))
		if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/plain") {
			t.Fatalf("robots.txt: %d %q", rec.Code, rec.Header().Get("Content-Type"))
		}
		return rec
	}

	prod := get(Deps{})
	if strings.Contains(prod.Body.String(), "Disallow: /\n") || prod.Header().Get("X-Robots-Tag") != "" {
		t.Errorf("production blocks crawlers: %q, X-Robots-Tag %q", prod.Body.String(), prod.Header().Get("X-Robots-Tag"))
	}

	staging := get(Deps{NoIndex: true})
	if staging.Body.String() != "User-agent: *\nDisallow: /\n" {
		t.Errorf("staging robots.txt = %q, want the whole site disallowed", staging.Body.String())
	}
	if staging.Header().Get("X-Robots-Tag") != "noindex, nofollow" {
		t.Errorf("staging X-Robots-Tag = %q", staging.Header().Get("X-Robots-Tag"))
	}
}

// Cross-site form posts from browsers are refused before the handler runs.
func TestCrossOriginProtection(t *testing.T) {
	h := New(Deps{}).Routes()
	req := httptest.NewRequest(http.MethodPost, "/counter/inc", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	if rec := serve(h, req); rec.Code != http.StatusForbidden {
		t.Errorf("cross-site POST: %d, want 403", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/counter/inc", nil)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	if rec := serve(h, req); rec.Code != http.StatusOK {
		t.Errorf("same-origin POST: %d, want 200", rec.Code)
	}
}

// Hashed static URLs are cached for a year; bare ones are revalidated.
func TestStaticCaching(t *testing.T) {
	h := staticFiles(http.FS(fstest.MapFS{"app.css": {Data: []byte("body{}")}}))
	for url, want := range map[string]string{
		"/app.css?v=abc123": "public, max-age=31536000, immutable",
		"/app.css":          "no-cache",
	} {
		rec := serve(h, httptest.NewRequest(http.MethodGet, url, nil))
		if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") != want {
			t.Errorf("%s: %d, Cache-Control %q, want %q", url, rec.Code, rec.Header().Get("Cache-Control"), want)
		}
	}
}
