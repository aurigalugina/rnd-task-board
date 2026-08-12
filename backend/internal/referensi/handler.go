package referensi

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

type Tim struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListTim mengimplementasikan GET /referensi-tim (otorisasi: admin/spv) —
// sumber dropdown "Tim/Org" di form user, gantiin free-text lama. Lihat
// docs/decision-log/decision-log-hr-mapping-super-user-20260810.md.
func (h *Handler) ListTim(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), `SELECT id, name FROM referensi_tim ORDER BY name`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tims := []Tim{}
	for rows.Next() {
		var t Tim
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tims = append(tims, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tims)
}

type createTimRequest struct {
	Name string `json:"name"`
}

// CreateTim mengimplementasikan POST /referensi-tim (otorisasi: admin/spv) —
// tambah nama tim baru ke daftar referensi. Bukan CRUD penuh (belum ada
// update/delete, belum ada kebutuhannya) -- lihat decision log.
func (h *Handler) CreateTim(w http.ResponseWriter, r *http.Request) {
	var req createTimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name wajib diisi", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	_, err := h.db.Exec(r.Context(), `INSERT INTO referensi_tim (id, name) VALUES ($1, $2)`, id, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Tim{ID: id, Name: req.Name})
}

type updateTimRequest struct {
	Name string `json:"name"`
}

func (h *Handler) UpdateTim(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateTimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name wajib diisi", http.StatusBadRequest)
		return
	}
	tag, err := h.db.Exec(r.Context(), `UPDATE referensi_tim SET name = $1 WHERE id = $2`, req.Name, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Tim{ID: id, Name: req.Name})
}

func (h *Handler) DeleteTim(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tag, err := h.db.Exec(r.Context(), `DELETE FROM referensi_tim WHERE id = $1`, id)
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

type UserHR struct {
	HRUserID    int    `json:"hr_user_id"`
	Email       string `json:"email"`
	Nip         string `json:"nip"`
	NamaLengkap string `json:"nama_lengkap"`
}

// ListUserHR mengimplementasikan GET /referensi-user-hr (otorisasi: admin/spv)
// — daftar pegawai HR asli buat mapping di Manajemen User. Data sensitif
// (email/NIP internal), TIDAK diekspos ke role lain.
func (h *Handler) ListUserHR(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), `
		SELECT hr_user_id, email, COALESCE(nip, ''), nama_lengkap FROM referensi_user_hr ORDER BY nama_lengkap
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	list := []UserHR{}
	for rows.Next() {
		var u UserHR
		if err := rows.Scan(&u.HRUserID, &u.Email, &u.Nip, &u.NamaLengkap); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		list = append(list, u)
	}
	if list == nil {
		list = []UserHR{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}
