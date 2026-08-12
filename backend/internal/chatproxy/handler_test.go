package chatproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRewriteChatPath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/api/v1/chat/sessions", "/sessions"},
		{"/api/v1/chat/fs/browse", "/fs/browse"},
		{"/api/v1/chat/ws/sessions/abc-123", "/ws/sessions/abc-123"},
		{"/api/v1/chat", "/"},
		{"/api/v1/chat/", "/"},
		{"/api/v1/chat/healthz", "/healthz"},
	}
	for _, c := range cases {
		if got := rewriteChatPath(c.in); got != c.want {
			t.Errorf("rewriteChatPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTokenFromRequest(t *testing.T) {
	t.Run("header Authorization Bearer diutamakan", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions?access_token=fromquery", nil)
		r.Header.Set("Authorization", "Bearer fromheader")
		if got := tokenFromRequest(r); got != "fromheader" {
			t.Errorf("got %q, want fromheader", got)
		}
	})

	t.Run("query access_token dipakai kalau tidak ada header (jalur WS)", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/chat/ws/sessions/x?access_token=fromquery", nil)
		if got := tokenFromRequest(r); got != "fromquery" {
			t.Errorf("got %q, want fromquery", got)
		}
	})

	t.Run("kosong kalau tidak ada keduanya", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions", nil)
		if got := tokenFromRequest(r); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("header non-Bearer diabaikan, fallback ke query", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions?access_token=fromquery", nil)
		r.Header.Set("Authorization", "Basic abc")
		if got := tokenFromRequest(r); got != "fromquery" {
			t.Errorf("got %q, want fromquery", got)
		}
	})
}

// TestServeHTTPUnauthorized memastikan request tanpa token valid ditolak 401
// SEBELUM menyentuh service (keamanan: proxy adalah satu-satunya gerbang auth).
func TestServeHTTPUnauthorized(t *testing.T) {
	h := NewHandler("http://localhost:8090", "/tmp/repo")
	r := httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}
