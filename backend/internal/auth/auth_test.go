package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestRequireRole langsung menguji logic pencocokan role (tanpa DB/JWT) —
// bagian paling gampang salah diam-diam di middleware otorisasi, karena
// mengetes ini penting untuk security (403 harus benar-benar menolak akses).
func TestRequireRole(t *testing.T) {
	cases := []struct {
		name       string
		userRoles  []string
		allowed    []string
		wantStatus int
	}{
		{"user punya role yang diizinkan", []string{"dev", "spv"}, []string{"spv"}, http.StatusOK},
		{"user tidak punya role yang diizinkan", []string{"dev"}, []string{"spv"}, http.StatusForbidden},
		{"user cocok salah satu dari beberapa role yang diizinkan", []string{"admin"}, []string{"admin", "spv"}, http.StatusOK},
		{"user tanpa role sama sekali", []string{}, []string{"spv"}, http.StatusForbidden},
		{"user role nil (belum di-set RequireAuth)", nil, []string{"spv"}, http.StatusForbidden},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})
			handler := RequireRole(c.allowed...)(next)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			ctx := context.WithValue(req.Context(), rolesKey, c.userRoles)
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != c.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, c.wantStatus)
			}
			wantCalled := c.wantStatus == http.StatusOK
			if called != wantCalled {
				t.Errorf("next handler called = %v, want %v", called, wantCalled)
			}
		})
	}
}

// signToken menandatangani MapClaims dengan jwtSecret() default — dipakai
// untuk memverifikasi ParseAccessToken tanpa perlu menyentuh DB/handler login.
func signToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString(jwtSecret())
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return s
}

func TestParseAccessToken(t *testing.T) {
	valid := signToken(t, jwt.MapClaims{
		"sub":          "user-1",
		"roles":        []string{"spv", "dev"},
		"access_level": "super_user",
		"exp":          time.Now().Add(time.Hour).Unix(),
	})

	t.Run("valid access token diurai lengkap", func(t *testing.T) {
		c, err := ParseAccessToken(valid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c.UserID != "user-1" || c.AccessLevel != "super_user" {
			t.Errorf("claims = %+v, want user-1/super_user", c)
		}
		if len(c.Roles) != 2 || c.Roles[0] != "spv" || c.Roles[1] != "dev" {
			t.Errorf("roles = %v, want [spv dev]", c.Roles)
		}
	})

	t.Run("token kosong ditolak", func(t *testing.T) {
		if _, err := ParseAccessToken(""); err != ErrInvalidToken {
			t.Errorf("err = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("refresh token ditolak (bukan access token)", func(t *testing.T) {
		refresh := signToken(t, jwt.MapClaims{
			"sub": "user-1",
			"typ": "refresh",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		if _, err := ParseAccessToken(refresh); err != ErrInvalidToken {
			t.Errorf("err = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("token kedaluwarsa ditolak", func(t *testing.T) {
		expired := signToken(t, jwt.MapClaims{
			"sub": "user-1",
			"exp": time.Now().Add(-time.Hour).Unix(),
		})
		if _, err := ParseAccessToken(expired); err != ErrInvalidToken {
			t.Errorf("err = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("tanda tangan salah ditolak", func(t *testing.T) {
		other := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user-1",
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		bad, _ := other.SignedString([]byte("secret-yang-salah-berbeda"))
		if _, err := ParseAccessToken(bad); err != ErrInvalidToken {
			t.Errorf("err = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("token tanpa sub ditolak", func(t *testing.T) {
		noSub := signToken(t, jwt.MapClaims{"exp": time.Now().Add(time.Hour).Unix()})
		if _, err := ParseAccessToken(noSub); err != ErrInvalidToken {
			t.Errorf("err = %v, want ErrInvalidToken", err)
		}
	})
}

func TestRolesFromContext(t *testing.T) {
	t.Run("returns roles when set", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), rolesKey, []string{"dev", "qa"})
		got := RolesFromContext(ctx)
		if len(got) != 2 || got[0] != "dev" || got[1] != "qa" {
			t.Errorf("RolesFromContext = %v, want [dev qa]", got)
		}
	})

	t.Run("returns empty slice when not set", func(t *testing.T) {
		got := RolesFromContext(context.Background())
		if len(got) != 0 {
			t.Errorf("RolesFromContext on empty context = %v, want empty", got)
		}
	})
}
