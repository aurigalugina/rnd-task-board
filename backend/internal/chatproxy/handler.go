// Package chatproxy mem-reverse-proxy /api/v1/chat/* ke claude-chat-service
// (Opsi B / backend proxy, lihat decision-log-change-request-integration-20260811.md).
// Frontend cuma tahu satu base URL; service (yang sengaja no-auth) tidak pernah
// di-reach langsung dari browser. Auth JWT diberlakukan DI SINI: REST lewat
// header Authorization, WebSocket lewat query param ?access_token= (browser
// WebSocket tidak bisa menyetel header custom, token kita in-memory bukan cookie).
package chatproxy

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"rndops/backend/internal/auth"
)

const (
	chatPrefix      = "/api/v1/chat"
	localConfigPath = chatPrefix + "/local/config"
	setupTokenPath  = chatPrefix + "/auth/setup-token"
)

type Handler struct {
	target     *url.URL
	proxy      *httputil.ReverseProxy
	defaultCwd string
}

// NewHandler membuat proxy ke serviceURL (default http://localhost:8090).
// defaultCwd diekspos lewat GET /api/v1/chat/local/config supaya frontend tidak
// meng-hardcode path repo.
func NewHandler(serviceURL, defaultCwd string) *Handler {
	if serviceURL == "" {
		serviceURL = "http://localhost:8090"
	}
	target, err := url.Parse(serviceURL)
	if err != nil {
		log.Fatalf("chatproxy: CLAUDE_CHAT_SERVICE_URL tidak valid: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	origDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		// Rewrite path SEBELUM director bawaan menyusun URL tujuan (director
		// bawaan cuma set scheme/host karena target.Path kosong).
		r.URL.Path = rewriteChatPath(r.URL.Path)
		origDirector(r)
		r.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, e error) {
		log.Printf("chatproxy: gagal reach service: %v", e)
		writeJSONError(w, http.StatusBadGateway, "claude-chat-service tidak dapat dihubungi")
	}

	return &Handler{target: target, proxy: proxy, defaultCwd: defaultCwd}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := tokenFromRequest(r)
	claims, err := auth.ParseAccessToken(token)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "token tidak valid")
		return
	}

	// Intercept endpoint lokal — TIDAK diteruskan ke service.
	if r.Method == http.MethodGet && r.URL.Path == localConfigPath {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"default_cwd": h.defaultCwd})
		return
	}

	// Provisioning ulang token OAuth Claude (akun subscription BERSAMA, dipakai
	// semua sesi chat) -- operasi sensitif level infra, bukan sekadar aksi tim
	// biasa. Digate super_user DI SINI (bukan di service, yang sengaja
	// no-auth/full-trust -- lihat catatan "TIDAK ADA RBAC" di CLAUDE.md),
	// supaya endpoint ini gak bisa dipanggil user manapun cuma karena punya
	// access token valid (service sendiri gak pernah di-reach langsung dari
	// browser, jadi ini satu-satunya titik enforcement).
	if isSetupTokenPath(r.URL.Path) && claims.AccessLevel != "super_user" {
		writeJSONError(w, http.StatusForbidden, "hanya super_user yang boleh provisioning ulang login Claude")
		return
	}

	// Jangan bocorkan access_token ke service via query string.
	if q := r.URL.Query(); q.Get("access_token") != "" {
		q.Del("access_token")
		r.URL.RawQuery = q.Encode()
	}
	r.Header.Set("X-Chat-User-Id", claims.UserID)
	r = r.WithContext(auth.ContextWithClaims(r.Context(), claims))
	h.proxy.ServeHTTP(w, r)
}

// rewriteChatPath membuang prefix /api/v1/chat sebelum diteruskan ke service.
//
//	/api/v1/chat/sessions          -> /sessions
//	/api/v1/chat/ws/sessions/abc   -> /ws/sessions/abc
//	/api/v1/chat  atau  /api/v1/chat/ -> /
func rewriteChatPath(p string) string {
	stripped := strings.TrimPrefix(p, chatPrefix)
	if stripped == "" || stripped == "/" {
		return "/"
	}
	if !strings.HasPrefix(stripped, "/") {
		return "/" + stripped
	}
	return stripped
}

// isSetupTokenPath cocokin path setup-token beserta sub-route-nya
// (.../auth/setup-token/{sessionId}/input, buat kirim balik authorization
// code) -- exact match ATAU prefix diikuti "/".
func isSetupTokenPath(p string) bool {
	return p == setupTokenPath || strings.HasPrefix(p, setupTokenPath+"/")
}

// tokenFromRequest mengambil access token dari header Authorization (REST) atau
// query param access_token (WebSocket, karena browser WS tak bisa set header).
func tokenFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return r.URL.Query().Get("access_token")
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
