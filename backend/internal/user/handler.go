package user

import (
	"context"
	"encoding/json"
	"net/http"

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
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	Initials    string   `json:"initials"`
	Email       string   `json:"email"`
	OrgTeam     string   `json:"org_team"`
	Roles       []string `json:"roles"`
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
		SELECT id, display_name, initials, email, org_team FROM users ORDER BY created_at
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.DisplayName, &u.Initials, &u.Email, &u.OrgTeam); err != nil {
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
}

// Create mengimplementasikan POST /users (otorisasi: admin/spv — FR-USR-04).
// `password` wajib di request — admin/SPV menentukan password awal user baru
// dan menyampaikannya out-of-band (docs/decision-log/decision-log-users-api-gaps-20260809.md).
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

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	orgTeam := req.OrgTeam
	if orgTeam == "" {
		orgTeam = "R&D"
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	id := uuid.New().String()
	_, err = tx.Exec(r.Context(), `
		INSERT INTO users (id, display_name, initials, email, password_hash, org_team)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, req.DisplayName, req.Initials, req.Email, string(hash), orgTeam)
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
	})
}

func (h *Handler) loadMe(ctx context.Context, userID string) (Me, error) {
	var m Me
	err := h.db.QueryRow(ctx, `
		SELECT id, display_name, initials, email, org_team, theme_preference FROM users WHERE id = $1
	`, userID).Scan(&m.ID, &m.DisplayName, &m.Initials, &m.Email, &m.OrgTeam, &m.ThemePreference)
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
