package serve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	logger := logging.NewLogger("text", httptest.NewRecorder())
	cfg := config.DefaultConfig()

	srv, err := New(dir, logger, cfg, nil, nil)
	if err != nil {
		t.Fatalf("New server: %v", err)
	}
	return srv
}

func TestServeHandlers(t *testing.T) {
	srv := newTestServer(t)

	// Test dashboard index
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("index status: got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("index content-type: got %q", ct)
	}

	// Test sessions endpoint
	req = httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
	rr = httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("sessions status: got %d", rr.Code)
	}
	var sessions []interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &sessions); err != nil {
		t.Fatalf("sessions JSON: %v", err)
	}

	// Test metrics endpoint
	req = httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	rr = httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("metrics status: got %d", rr.Code)
	}
}

func TestTrustEndpoint(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/trust", nil)
	rr := httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("trust status: got %d", rr.Code)
	}

	var entries []interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &entries); err != nil {
		t.Fatalf("trust JSON: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty trust entries, got %d", len(entries))
	}
}

func TestCVEEndpoint(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/cve", nil)
	rr := httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("cve status: got %d", rr.Code)
	}

	var summary map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &summary); err != nil {
		t.Fatalf("cve JSON: %v", err)
	}
}

func TestStatusEndpoint(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rr := httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status status: got %d", rr.Code)
	}

	var status statusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &status); err != nil {
		t.Fatalf("status JSON: %v", err)
	}
	if status.Version == "" {
		t.Fatal("expected version to be set")
	}
	if status.Features == nil {
		t.Fatal("expected features to be set")
	}
	if status.Uptime == "" {
		t.Fatal("expected uptime to be set")
	}
}

func TestSessionByIDNotFound(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/sessions/nonexistent", nil)
	rr := httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("session by id status: expected 404, got %d", rr.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	srv := newTestServer(t)

	endpoints := []string{"/api/sessions", "/api/metrics", "/api/trust", "/api/cve", "/api/status"}
	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodPost, ep, nil)
		rr := httptest.NewRecorder()
		srv.mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s POST: expected 405, got %d", ep, rr.Code)
		}
	}
}

func TestNotFoundPath(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rr := httptest.NewRecorder()
	srv.mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("not found: expected 404, got %d", rr.Code)
	}
}

func TestSessionIDPathTraversal(t *testing.T) {
	srv := newTestServer(t)

	// IDs with dangerous characters should be rejected with 400.
	// Note: Go's http.ServeMux already handles "../" with 301 redirects (defense-in-depth),
	// and Go's HTTP parser rejects null bytes and invalid URLs at the transport level.
	// Our regex validation catches everything that gets through those layers.
	maliciousIDs := []string{
		"session;rm",      // shell metacharacters
		"session<script>", // XSS attempt
		"$HOME",           // shell variable expansion
		"session%2f",      // encoded slash
		"session@host",    // at sign
		"session&cmd",     // ampersand
		"session|pipe",    // pipe
		// Very long ID (> 64 chars)
		"session-12345678901234567890123456789012345678901234567890-overflow",
	}

	for _, id := range maliciousIDs {
		path := "/api/sessions/" + id
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		srv.mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("session ID %q: expected 400, got %d", id, rr.Code)
		}
	}

	// Valid-looking session IDs should pass validation (but return 404 since they don't exist)
	validIDs := []string{
		"session-1234567890",
		"session-1709000000000000000",
		"abc-def-123",
	}
	for _, id := range validIDs {
		req := httptest.NewRequest(http.MethodGet, "/api/sessions/"+id, nil)
		rr := httptest.NewRecorder()
		srv.mux.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("session ID %q: expected 404 (not found), got %d", id, rr.Code)
		}
	}
}

func TestSecurityHeaders(t *testing.T) {
	srv := newTestServer(t)

	handler := securityHeaders(srv.mux)
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	checks := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "no-referrer",
	}

	for header, expected := range checks {
		got := rr.Header().Get(header)
		if got != expected {
			t.Errorf("header %s: expected %q, got %q", header, expected, got)
		}
	}

	csp := rr.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("expected Content-Security-Policy header to be set")
	}
}
