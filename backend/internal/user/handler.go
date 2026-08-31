package user

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"rndops/backend/internal/auth"
)

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

type User struct {
	ID                   string   `json:"id"`
	DisplayName          string   `json:"display_name"`
	Initials             string   `json:"initials"`
	Email                string   `json:"email"`
	OrgTeam              string   `json:"org_team"`
	Roles                []string `json:"roles"`
	AccessLevel          string   `json:"access_level"`
	TaskScopeVisibility  string   `json:"task_scope_visibility"` // 2026-08-31
	HRUserID             *int     `json:"hr_user_id"`
}

type Me struct {
	User
	ThemePreference string `json:"theme_preference"`
}

type Role struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// List mengimplementasikan GET /users (otorisasi: admin/spv — FR-USR-03).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), `
		SELECT id, display_name, initials, email, org_team, access_level, task_scope_visibility, hr_user_id FROM users ORDER BY created_at
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.DisplayName, &u.Initials, &u.Email, &u.OrgTeam, &u.AccessLevel, &u.TaskScopeVisibility, &u.HRUserID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	for i := range users {
		roles, err := auth.FetchRoles(r.Context(), h.db, users[i].ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users[i].Roles = roles
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// ListAssignable mengimplementasikan GET /users/assignable — daftar ringkas
// (tanpa email) untuk form assignment PIC, bisa diakses semua pengguna
// terautentikasi (FR-ASG-02, docs/decision-log/decision-log-pic-assignment-endpoint-20260809.md).
func (h *Handler) ListAssignable(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), `
		SELECT id, display_name, initials, org_team FROM users ORDER BY display_name
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type assignableUser struct {
		ID          string   `json:"id"`
		DisplayName string   `json:"display_name"`
		Initials    string   `json:"initials"`
		OrgTeam     string   `json:"org_team"`
		Roles       []string `json:"roles"`
	}

	users := []assignableUser{}
	for rows.Next() {
		var u assignableUser
		if err := rows.Scan(&u.ID, &u.DisplayName, &u.Initials, &u.OrgTeam); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	for i := range users {
		roles, err := auth.FetchRoles(r.Context(), h.db, users[i].ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users[i].Roles = roles
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

type createUserRequest struct {
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	Password    string   `json:"password"`
	Initials    string   `json:"initials"`
	OrgTeam     string   `json:"org_team"`
	RoleCodes   []string `json:"role_codes"`
	AccessLevel string   `json:"access_level"`
	HRUserID    *int     `json:"hr_user_id"`
}

func validAccessLevel(level string) bool {
	return level == "" || level == "super_user" || level == "regular_user"
}

// Create mengimplementasikan POST /users (otorisasi: admin/spv — FR-USR-04).
// `password` wajib di request — admin/SPV menentukan password awal user baru
// dan menyampaikannya out-of-band (docs/decision-log/decision-log-users-api-gaps-20260809.md).
// `access_level`/`hr_user_id` opsional -- lihat
// docs/decision-log/decision-log-hr-mapping-super-user-20260810.md.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "password wajib diisi", http.StatusBadRequest)
		return
	}
	if !validAccessLevel(req.AccessLevel) {
		http.Error(w, "access_level harus salah satu dari: super_user, regular_user", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	orgTeam := req.OrgTeam
	if orgTeam == "" {
		orgTeam = "R&D"
	}
	accessLevel := req.AccessLevel
	if accessLevel == "" {
		accessLevel = "regular_user"
	}

	var teamExists bool
	if err := h.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM referensi_tim WHERE name = $1)`, orgTeam).Scan(&teamExists); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !teamExists {
		http.Error(w, "org_team tidak ditemukan di referensi_tim", http.StatusBadRequest)
		return
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	id := uuid.New().String()
	_, err = tx.Exec(r.Context(), `
		INSERT INTO users (id, display_name, initials, email, password_hash, org_team, access_level, hr_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, req.DisplayName, req.Initials, req.Email, string(hash), orgTeam, accessLevel, req.HRUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, code := range req.RoleCodes {
		_, err = tx.Exec(r.Context(), `
			INSERT INTO user_roles (user_id, role_id)
			SELECT $1, id FROM roles WHERE code = $2
		`, id, code)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(User{
		ID:          id,
		DisplayName: req.DisplayName,
		Initials:    req.Initials,
		Email:       req.Email,
		OrgTeam:     orgTeam,
		Roles:       req.RoleCodes,
		AccessLevel: accessLevel,
		HRUserID:    req.HRUserID,
	})
}

type updateUserRequest struct {
	OrgTeam              *string  `json:"org_team"`
	AccessLevel          *string  `json:"access_level"`
	HRUserID             *int     `json:"hr_user_id"`
	ClearHR              bool     `json:"clear_hr_user_id"`
	RoleCodes            []string `json:"role_codes"`
	TaskScopeVisibility  *string  `json:"task_scope_visibility"` // 2026-08-31: 'self' or 'team'
}

// Update mengimplementasikan PATCH /users/{id} (otorisasi: admin/spv) — SEBELUM
// ini TIDAK ADA cara edit user yang sudah ada (cuma POST /users buat bikin
// baru). Ditambahkan supaya admin bisa map hr_user_id/access_level/org_team
// ke user existing (termasuk user seed lama), lihat
// docs/decision-log/decision-log-hr-mapping-super-user-20260810.md.
// role_codes, kalau dikirim, MENGGANTI SELURUH assignment role user itu
// (bukan menambah) -- klien wajib kirim daftar lengkap.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "userID")

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.AccessLevel != nil && !validAccessLevel(*req.AccessLevel) {
		http.Error(w, "access_level harus salah satu dari: super_user, regular_user", http.StatusBadRequest)
		return
	}
	if req.TaskScopeVisibility != nil && *req.TaskScopeVisibility != "self" && *req.TaskScopeVisibility != "team" {
		http.Error(w, "task_scope_visibility harus salah satu dari: self, team", http.StatusBadRequest)
		return
	}
	if req.OrgTeam != nil {
		var teamExists bool
		if err := h.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM referensi_tim WHERE name = $1)`, *req.OrgTeam).Scan(&teamExists); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !teamExists {
			http.Error(w, "org_team tidak ditemukan di referensi_tim", http.StatusBadRequest)
			return
		}
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	hrUserID := req.HRUserID
	if req.ClearHR {
		hrUserID = nil
	}
	_, err = tx.Exec(r.Context(), `
		UPDATE users SET
			org_team = COALESCE($2, org_team),
			access_level = COALESCE($3, access_level),
			task_scope_visibility = COALESCE($6, task_scope_visibility),
			hr_user_id = CASE WHEN $4 THEN NULL ELSE COALESCE($5, hr_user_id) END,
			updated_at = now()
		WHERE id = $1
	`, targetID, req.OrgTeam, req.AccessLevel, req.ClearHR, hrUserID, req.TaskScopeVisibility)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.RoleCodes != nil {
		if _, err := tx.Exec(r.Context(), `DELETE FROM user_roles WHERE user_id = $1`, targetID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, code := range req.RoleCodes {
			if _, err := tx.Exec(r.Context(), `
				INSERT INTO user_roles (user_id, role_id)
				SELECT $1, id FROM roles WHERE code = $2
			`, targetID, code); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var u User
	if err := h.db.QueryRow(r.Context(), `
		SELECT id, display_name, initials, email, org_team, access_level, hr_user_id FROM users WHERE id = $1
	`, targetID).Scan(&u.ID, &u.DisplayName, &u.Initials, &u.Email, &u.OrgTeam, &u.AccessLevel, &u.HRUserID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	roles, err := auth.FetchRoles(r.Context(), h.db, targetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u.Roles = roles

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}

func (h *Handler) loadMe(ctx context.Context, userID string) (Me, error) {
	var m Me
	err := h.db.QueryRow(ctx, `
		SELECT id, display_name, initials, email, org_team, theme_preference, access_level, hr_user_id FROM users WHERE id = $1
	`, userID).Scan(&m.ID, &m.DisplayName, &m.Initials, &m.Email, &m.OrgTeam, &m.ThemePreference, &m.AccessLevel, &m.HRUserID)
	if err != nil {
		return Me{}, err
	}

	roles, err := auth.FetchRoles(ctx, h.db, userID)
	if err != nil {
		return Me{}, err
	}
	m.Roles = roles

	return m, nil
}

// Me mengimplementasikan GET /users/me (docs/decision-log/decision-log-users-api-gaps-20260809.md).
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	me, err := h.loadMe(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(me)
}

type updateMeRequest struct {
	DisplayName     *string `json:"display_name"`
	Initials        *string `json:"initials"`
	CurrentPassword *string `json:"current_password"`
	Password        *string `json:"password"`
	ThemePreference *string `json:"theme_preference"`
}

func writeInvalidCredentials(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": "invalid_credentials", "message": message},
	})
}

// UpdateMe mengimplementasikan PATCH /users/me (FR-USR-01/02/05). Ganti password
// wajib menyertakan current_password yang valid — lihat decision log di atas.
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req updateMeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var newHash *string
	if req.Password != nil {
		if req.CurrentPassword == nil {
			writeInvalidCredentials(w, "current_password wajib diisi untuk ganti password")
			return
		}
		var currentHash string
		if err := h.db.QueryRow(r.Context(), `SELECT password_hash FROM users WHERE id = $1`, userID).
			Scan(&currentHash); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(currentHash), []byte(*req.CurrentPassword)) != nil {
			writeInvalidCredentials(w, "password saat ini tidak sesuai")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hashStr := string(hash)
		newHash = &hashStr
	}

	_, err := h.db.Exec(r.Context(), `
		UPDATE users SET
			display_name = COALESCE($2, display_name),
			initials = COALESCE($3, initials),
			password_hash = COALESCE($4, password_hash),
			theme_preference = COALESCE($5, theme_preference),
			updated_at = now()
		WHERE id = $1
	`, userID, req.DisplayName, req.Initials, newHash, req.ThemePreference)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	me, err := h.loadMe(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(me)
}

// ListRoles mengimplementasikan GET /roles (FR-ASG-02).
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), `SELECT code, label FROM roles ORDER BY code`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	roles := []Role{}
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.Code, &role.Label); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		roles = append(roles, role)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}

type createRoleRequest struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req createRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Code == "" || req.Label == "" {
		http.Error(w, "code dan label wajib diisi", http.StatusBadRequest)
		return
	}
	if _, err := h.db.Exec(r.Context(), `INSERT INTO roles (code, label) VALUES ($1, $2)`, req.Code, req.Label); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Role{Code: req.Code, Label: req.Label})
}

type updateRoleRequest struct {
	Label string `json:"label"`
}

func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	var req updateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Label == "" {
		http.Error(w, "label wajib diisi", http.StatusBadRequest)
		return
	}
	tag, err := h.db.Exec(r.Context(), `UPDATE roles SET label = $1 WHERE code = $2`, req.Label, code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Role{Code: code, Label: req.Label})
}

func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	var count int
	if err := h.db.QueryRow(r.Context(), `
		SELECT COUNT(*) FROM user_roles ur JOIN roles ro ON ro.id = ur.role_id WHERE ro.code = $1
	`, code).Scan(&count); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count > 0 {
		http.Error(w, "Jabatan masih dipakai user — hapus dari semua user dulu", http.StatusConflict)
		return
	}
	tag, err := h.db.Exec(r.Context(), `DELETE FROM roles WHERE code = $1`, code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type UserProgressSummary struct {
	UserID         string `json:"user_id"`
	Belum          int    `json:"belum"`
	OnProgress     int    `json:"on_progress"`
	Selesai        int    `json:"selesai"`
	CompletionRate int    `json:"completion_rate"`
}

// ProgressSummary mengimplementasikan GET /users/progress-summary.
// Agregasi Day Entry per PIC lintas semua board AKTIF, all-time -- board yang
// sudah di-archive TIDAK ikut terhitung (JOIN big_tasks/boards + filter
// archived_at IS NULL), konsisten dengan aturan "Dashboard hilangkan board
// archived total" (lihat decision-log-board-archive-20260812.md; section
// "Tim" ini juga bagian Dashboard, bukan laporan personal seperti Weekly Plan
// yang sengaja TIDAK difilter).
// super_user: semua user; regular_user: hanya dirinya sendiri.
// Dipakai section "Tim" di Dashboard buat nampilin metrik progress per orang
// tanpa N+1 fetch di frontend (lihat docs/decision-log change request).
func (h *Handler) ProgressSummary(w http.ResponseWriter, r *http.Request) {
	requesterID := auth.UserIDFromContext(r.Context())
	isSuperUser := auth.IsSuperUser(r.Context())

	query := `
		SELECT
			dt.pic_user_id,
			COUNT(*) FILTER (WHERE de.progress_pct = 0) AS belum,
			COUNT(*) FILTER (WHERE de.progress_pct BETWEEN 1 AND 99) AS on_progress,
			COUNT(*) FILTER (WHERE de.progress_pct = 100) AS selesai,
			COALESCE(ROUND(AVG(de.progress_pct)), 0) AS completion_rate
		FROM daily_tasks dt
		JOIN day_entries de ON de.daily_task_id = dt.id
		JOIN big_tasks bt ON bt.id = dt.big_task_id
		JOIN boards b ON b.id = bt.board_id
		WHERE b.archived_at IS NULL
		GROUP BY dt.pic_user_id
	`
	args := []any{}
	if !isSuperUser {
		query = `
			SELECT
				dt.pic_user_id,
				COUNT(*) FILTER (WHERE de.progress_pct = 0) AS belum,
				COUNT(*) FILTER (WHERE de.progress_pct BETWEEN 1 AND 99) AS on_progress,
				COUNT(*) FILTER (WHERE de.progress_pct = 100) AS selesai,
				COALESCE(ROUND(AVG(de.progress_pct)), 0) AS completion_rate
			FROM daily_tasks dt
			JOIN day_entries de ON de.daily_task_id = dt.id
			JOIN big_tasks bt ON bt.id = dt.big_task_id
			JOIN boards b ON b.id = bt.board_id
			WHERE b.archived_at IS NULL AND dt.pic_user_id = $1
			GROUP BY dt.pic_user_id
		`
		args = []any{requesterID}
	}

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []UserProgressSummary{}
	for rows.Next() {
		var s UserProgressSummary
		if err := rows.Scan(&s.UserID, &s.Belum, &s.OnProgress, &s.Selesai, &s.CompletionRate); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result = append(result, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
