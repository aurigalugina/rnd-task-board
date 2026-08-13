// Package changerequest mengimplementasikan entitas change_requests (Vision
// Product §6 — Mekanisme Evolusi Produk). Usulan perubahan disusun lewat
// percakapan (claude-chat-service, diproksikan lewat chatproxy) lalu disimpan
// di sini untuk ditriase SPV & System Analyst. Lihat
// docs/decision-log/decision-log-change-request-integration-20260811.md.
package changerequest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"rndops/backend/internal/auth"
)

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

type ChangeRequest struct {
	ID              string  `json:"id"`
	SubmittedBy     string  `json:"submitted_by"`
	SubmittedByName string  `json:"submitted_by_name"`
	RawConversation string  `json:"raw_conversation"`
	DocumentMD      *string `json:"document_md"`
	Status          string  `json:"status"`
	ReviewedBy      *string `json:"reviewed_by"`
	ReviewedByName  *string `json:"reviewed_by_name"`
	ReviewedAt      *string `json:"reviewed_at"`
	CreatedAt       string  `json:"created_at"`
}

// validStatuses mengikuti CHECK constraint di migration 0006.
var validStatuses = map[string]bool{
	"pending":   true,
	"approved":  true,
	"rejected":  true,
	"scheduled": true,
}

// isValidStatusTransition menentukan apakah perpindahan status triase boleh.
// Saat ini semua transisi antar 4 status enum diizinkan (termasuk balik ke
// pending untuk batal triase) — target wajib salah satu enum yang valid.
// Diekstrak sebagai fungsi murni supaya mudah diperketat & ditest nanti.
func isValidStatusTransition(from, to string) bool {
	if !validStatuses[to] {
		return false
	}
	if from != "" && !validStatuses[from] {
		return false
	}
	return true
}

const selectCR = `
	SELECT cr.id, cr.submitted_by, su.display_name, cr.raw_conversation,
	       cr.document_md, cr.status, cr.reviewed_by, ru.display_name,
	       cr.reviewed_at, cr.created_at
	FROM change_requests cr
	JOIN users su ON su.id = cr.submitted_by
	LEFT JOIN users ru ON ru.id = cr.reviewed_by
`

func scanCR(row interface {
	Scan(dest ...any) error
}) (ChangeRequest, error) {
	var cr ChangeRequest
	var reviewedBy, reviewedByName *string
	var reviewedAt *time.Time
	var createdAt time.Time
	if err := row.Scan(&cr.ID, &cr.SubmittedBy, &cr.SubmittedByName, &cr.RawConversation,
		&cr.DocumentMD, &cr.Status, &reviewedBy, &reviewedByName,
		&reviewedAt, &createdAt); err != nil {
		return cr, err
	}
	cr.ReviewedBy = reviewedBy
	cr.ReviewedByName = reviewedByName
	if reviewedAt != nil {
		s := reviewedAt.Format(time.RFC3339)
		cr.ReviewedAt = &s
	}
	cr.CreatedAt = createdAt.Format(time.RFC3339)
	return cr, nil
}

// List mengimplementasikan GET /change-requests — dibuka ke semua user login
// (transparansi tim kecil; §6 "seluruh anggota tim"). Terbaru dulu.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), selectCR+` ORDER BY cr.created_at DESC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	list := []ChangeRequest{}
	for rows.Next() {
		cr, err := scanCR(rows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		list = append(list, cr)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

type createRequest struct {
	RawConversation string  `json:"raw_conversation"`
	DocumentMD      *string `json:"document_md"`
}

// Create mengimplementasikan POST /change-requests — semua user login boleh
// mengusulkan (§6). submitted_by SELALU dari JWT, status awal 'pending'.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.RawConversation == "" {
		http.Error(w, "raw_conversation wajib diisi", http.StatusBadRequest)
		return
	}

	userID := auth.UserIDFromContext(r.Context())
	id := uuid.New().String()
	_, err := h.db.Exec(r.Context(), `
		INSERT INTO change_requests (id, submitted_by, raw_conversation, document_md, status)
		VALUES ($1, $2, $3, $4, 'pending')
	`, id, userID, req.RawConversation, req.DocumentMD)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cr, err := h.loadOne(r, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cr)
}

type updateRequest struct {
	Status string `json:"status"`
}

// Update mengimplementasikan PATCH /change-requests/{id} — triase oleh SPV & SA
// (digate RequireRole di main.go). Mengubah status + set reviewed_by (dari JWT)
// & reviewed_at. Kembali ke 'pending' membatalkan jejak triase.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var current string
	if err := h.db.QueryRow(r.Context(), `SELECT status FROM change_requests WHERE id = $1`, id).Scan(&current); err != nil {
		http.Error(w, "change request tidak ditemukan", http.StatusNotFound)
		return
	}
	if !isValidStatusTransition(current, req.Status) {
		http.Error(w, "status transisi tidak valid", http.StatusBadRequest)
		return
	}

	reviewerID := auth.UserIDFromContext(r.Context())
	if req.Status == "pending" {
		// Batal triase — bersihkan jejak reviewer.
		_, err := h.db.Exec(r.Context(), `
			UPDATE change_requests SET status = 'pending', reviewed_by = NULL, reviewed_at = NULL WHERE id = $1
		`, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		_, err := h.db.Exec(r.Context(), `
			UPDATE change_requests SET status = $2, reviewed_by = $3, reviewed_at = now() WHERE id = $1
		`, id, req.Status, reviewerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	cr, err := h.loadOne(r, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cr)
}

func (h *Handler) loadOne(r *http.Request, id string) (ChangeRequest, error) {
	row := h.db.QueryRow(r.Context(), selectCR+` WHERE cr.id = $1`, id)
	return scanCR(row)
}
