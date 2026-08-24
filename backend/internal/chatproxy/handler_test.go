package chatproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func TestIsSetupTokenPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/api/v1/chat/auth/setup-token", true},
		{"/api/v1/chat/auth/setup-token/abc-123/input", true},
		{"/api/v1/chat/auth/status", false},
		{"/api/v1/chat/sessions", false},
		{"/api/v1/chat/auth/setup-token-status", false}, // prefix mirip tapi bukan sub-path (tanpa "/")
	}
	for _, c := range cases {
		if got := isSetupTokenPath(c.path); got != c.want {
			t.Errorf("isSetupTokenPath(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

// signTestToken menandatangani access token pakai secret default dev (sama
// seperti jwtSecret() di package auth kalau JWT_SECRET tidak di-set) --
// dipakai buat mensimulasikan request terautentikasi lewat proxy tanpa perlu
// mock server backend/DB.
func signTestToken(t *testing.T, accessLevel string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":          "user-1",
		"roles":        []string{"dev"},
		"access_level": accessLevel,
		"exp":          time.Now().Add(time.Hour).Unix(),
	})
	s, err := tok.SignedString([]byte("dev-secret-change-me"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

// TestServeHTTPSetupTokenGating memastikan cuma super_user yang boleh
// nyentuh /auth/setup-token (provisioning ulang OAuth Claude, akun bersama) --
// regular_user WAJIB 403 walau access token-nya valid, karena claude-chat-service
// sendiri sengaja no-auth/full-trust (proxy ini satu-satunya gerbang otorisasi).
func TestServeHTTPSetupTokenGating(t *testing.T) {
	// Backend service palsu -- balikin 200 apa adanya, cuma buat mastiin
	// request BENERAN sampai diteruskan (bukan ditolak gate) buat kasus
	// super_user, tanpa nunggu dial-timeout ke port yang gak ada.
	fakeService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer fakeService.Close()

	h := NewHandler(fakeService.URL, "/tmp/repo")

	t.Run("regular_user ditolak 403", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/chat/auth/setup-token", nil)
		r.Header.Set("Authorization", "Bearer "+signTestToken(t, "regular_user"))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("regular_user ditolak 403 di sub-route input juga", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/chat/auth/setup-token/abc-123/input", nil)
		r.Header.Set("Authorization", "Bearer "+signTestToken(t, "regular_user"))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("super_user diteruskan ke proxy (bukan 403)", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/chat/auth/setup-token", nil)
		r.Header.Set("Authorization", "Bearer "+signTestToken(t, "super_user"))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200 (dari fake service, artinya lolos gate)", rec.Code)
		}
	})
}
