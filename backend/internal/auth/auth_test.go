package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
