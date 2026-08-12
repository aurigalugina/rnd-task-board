package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const (
	userIDKey      contextKey = "userID"
	rolesKey       contextKey = "roles"
	accessLevelKey contextKey = "accessLevel"
)

const (
	accessTokenTTL  = 2 * time.Hour
	refreshTokenTTL = 7 * 24 * time.Hour
	refreshCookie   = "refresh_token"
	refreshPath     = "/api/v1/auth"
)

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

func jwtSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-me"
	}
	return []byte(secret)
}

// secureCookies menentukan flag Secure pada refresh token cookie — dimatikan di
// development supaya tetap jalan di http://localhost tanpa TLS (lihat
// docs/decision-log/decision-log-auth-rbac-implementation-20260808.md).
func secureCookies() bool {
	return os.Getenv("APP_ENV") == "production"
}

// issueAccessToken menyisipkan access_level ('super_user'/'regular_user') di
// samping roles -- konsep terpisah dari roles many-to-many, lihat
// docs/decision-log/decision-log-hr-mapping-super-user-20260810.md.
func issueAccessToken(userID string, roles []string, accessLevel string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":          userID,
		"roles":        roles,
		"access_level": accessLevel,
		"exp":          time.Now().Add(accessTokenTTL).Unix(),
	})
	return token.SignedString(jwtSecret())
}

func issueRefreshToken(userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"typ": "refresh",
		"exp": time.Now().Add(refreshTokenTTL).Unix(),
	})
	return token.SignedString(jwtSecret())
}

func setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookie,
		Value:    token,
		Path:     refreshPath,
		HttpOnly: true,
		Secure:   secureCookies(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(refreshTokenTTL.Seconds()),
	})
}

func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookie,
		Value:    "",
		Path:     refreshPath,
		HttpOnly: true,
		Secure:   secureCookies(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// FetchRoles mengambil kode role milik user, diurutkan supaya deterministik.
func FetchRoles(ctx context.Context, db *pgxpool.Pool, userID string) ([]string, error) {
	rows, err := db.Query(ctx, `
		SELECT r.code FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY r.code
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := []string{}
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		roles = append(roles, code)
	}
	return roles, rows.Err()
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login mengimplementasikan POST /auth/login (05-api-contract.md §2).
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var userID, passwordHash, displayName, initials, orgTeam, accessLevel string
	err := h.db.QueryRow(r.Context(), `
		SELECT id, password_hash, display_name, initials, org_team, access_level FROM users WHERE email = $1
	`, req.Email).Scan(&userID, &passwordHash, &displayName, &initials, &orgTeam, &accessLevel)
	if err != nil {
		writeAuthError(w)
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)) != nil {
		writeAuthError(w)
		return
	}

	roles, err := FetchRoles(r.Context(), h.db, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accessToken, err := issueAccessToken(userID, roles, accessLevel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	refreshToken, err := issueRefreshToken(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setRefreshCookie(w, refreshToken)

	json.NewEncoder(w).Encode(map[string]any{
		"access_token": accessToken,
		"user": map[string]any{
			"id":           userID,
			"display_name": displayName,
			"initials":     initials,
			"roles":        roles,
			"org_team":     orgTeam,
			"access_level": accessLevel,
		},
	})
}

func writeAuthError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":{"code":"invalid_credentials","message":"Email atau password salah"}}`))
}

func writeForbiddenError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"error":{"code":"forbidden","message":"Anda tidak memiliki akses untuk aksi ini"}}`))
}

// Logout menghapus refresh token cookie. Access token yang sudah terlanjur
// terbit tetap valid sampai expire — lihat decision log auth-rbac-implementation.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// Refresh mengimplementasikan POST /auth/refresh menggunakan refresh token cookie.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookie)
	if err != nil {
		writeAuthError(w)
		return
	}

	token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret(), nil
	})
	if err != nil || !token.Valid {
		writeAuthError(w)
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["typ"] != "refresh" {
		writeAuthError(w)
		return
	}
	userID, _ := claims["sub"].(string)
	if userID == "" {
		writeAuthError(w)
		return
	}

	roles, err := FetchRoles(r.Context(), h.db, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var accessLevel string
	if err := h.db.QueryRow(r.Context(), `SELECT access_level FROM users WHERE id = $1`, userID).Scan(&accessLevel); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	accessToken, err := issueAccessToken(userID, roles, accessLevel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"access_token": accessToken})
}

// AccessClaims adalah bentuk terurai dari access token (bukan refresh token).
type AccessClaims struct {
	UserID      string
	Roles       []string
	AccessLevel string
}

// ErrInvalidToken dikembalikan ParseAccessToken kalau token kosong, gagal
// verifikasi tanda tangan, kedaluwarsa, atau ternyata refresh token.
var ErrInvalidToken = errors.New("invalid access token")

// ParseAccessToken memverifikasi tanda tangan + masa berlaku access token dan
// mengurai claim-nya. Dipakai bareng oleh RequireAuth (header) dan chatproxy
// (WS, token dari query param) — logika verifikasi terpusat & bisa ditest.
func ParseAccessToken(tokenString string) (*AccessClaims, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret(), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["typ"] == "refresh" {
		return nil, ErrInvalidToken
	}
	userID, _ := claims["sub"].(string)
	if userID == "" {
		return nil, ErrInvalidToken
	}
	roles := []string{}
	if rawRoles, ok := claims["roles"].([]interface{}); ok {
		for _, rr := range rawRoles {
			if code, ok := rr.(string); ok {
				roles = append(roles, code)
			}
		}
	}
	accessLevel, _ := claims["access_level"].(string)
	return &AccessClaims{UserID: userID, Roles: roles, AccessLevel: accessLevel}, nil
}

// ContextWithClaims menyisipkan userID + roles + accessLevel ke context, sama
// seperti yang dilakukan RequireAuth — supaya handler downstream bisa pakai
// UserIDFromContext/RolesFromContext meski auth-nya lewat jalur lain (WS proxy).
func ContextWithClaims(ctx context.Context, c *AccessClaims) context.Context {
	ctx = context.WithValue(ctx, userIDKey, c.UserID)
	ctx = context.WithValue(ctx, rolesKey, c.Roles)
	ctx = context.WithValue(ctx, accessLevelKey, c.AccessLevel)
	return ctx
}

// RequireAuth adalah middleware chi yang memvalidasi JWT dari header Authorization
// dan menyisipkan userID + roles dari token ke context.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			writeAuthError(w)
			return
		}
		claims, err := ParseAccessToken(strings.TrimPrefix(authHeader, "Bearer "))
		if err != nil {
			writeAuthError(w)
			return
		}
		next.ServeHTTP(w, r.WithContext(ContextWithClaims(r.Context(), claims)))
	})
}

// RequireRole adalah middleware chi yang menolak request (403) apabila user
// yang sudah lolos RequireAuth tidak memegang salah satu role yang diizinkan.
// Harus dipasang setelah RequireAuth pada chain middleware.
func RequireRole(allowed ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoles := RolesFromContext(r.Context())
			for _, role := range userRoles {
				for _, want := range allowed {
					if role == want {
						next.ServeHTTP(w, r)
						return
					}
				}
			}
			writeForbiddenError(w)
		})
	}
}

// UserIDFromContext mengambil user id yang disisipkan RequireAuth.
func UserIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(userIDKey).(string)
	return userID
}

// RolesFromContext mengambil daftar role yang disisipkan RequireAuth.
func RolesFromContext(ctx context.Context) []string {
	roles, _ := ctx.Value(rolesKey).([]string)
	return roles
}

// AccessLevelFromContext mengambil access_level ('super_user'/'regular_user')
// yang disisipkan RequireAuth -- konsep terpisah dari roles, lihat
// docs/decision-log/decision-log-hr-mapping-super-user-20260810.md.
func AccessLevelFromContext(ctx context.Context) string {
	level, _ := ctx.Value(accessLevelKey).(string)
	return level
}

// IsSuperUser adalah shorthand AccessLevelFromContext(ctx) == "super_user".
func IsSuperUser(ctx context.Context) bool {
	return AccessLevelFromContext(ctx) == "super_user"
}
