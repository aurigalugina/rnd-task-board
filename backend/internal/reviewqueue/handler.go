package reviewqueue

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

type Item struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	Reviewed    bool   `json:"reviewed"`
	BigTaskID   string `json:"big_task_id"`
	BigTaskName string `json:"big_task_name"`
	BoardID     string `json:"board_id"`
	BoardName   string `json:"board_name"`
}

// List mengimplementasikan GET /review-queue (FR-NTF-03). Dibatasi role spv
// (RequireRole di main.go) dan cuma item_type "daily_task" pada iterasi ini —
// lihat docs/decision-log/decision-log-review-queue-scope-20260809.md.
// Cuma item yang BELUM ditinjau yang dikembalikan (sesuai teks FR-NTF-03).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), `
		SELECT dt.id, dt.title, bt.id, bt.name, b.id, b.name
		FROM daily_tasks dt
		JOIN big_tasks bt ON bt.id = dt.big_task_id
		JOIN boards b ON b.id = bt.board_id
		WHERE NOT EXISTS (
			SELECT 1 FROM item_reviews ir
			WHERE ir.item_type = 'daily_task' AND ir.item_id = dt.id
		)
		ORDER BY dt.created_at
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var it Item
		it.Type = "daily_task"
		if err := rows.Scan(&it.ID, &it.Title, &it.BigTaskID, &it.BigTaskName, &it.BoardID, &it.BoardName); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		it.Reviewed = false
		items = append(items, it)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

type markReviewedResponse struct {
	Reviewed   bool      `json:"reviewed"`
	ReviewedBy string    `json:"reviewed_by"`
	ReviewedAt time.Time `json:"reviewed_at"`
}

// MarkReviewed mengimplementasikan POST /review-queue/{item_type}/{item_id}/mark-reviewed
// (FR-NTF-02). Tidak mengubah field progress/status entitas terkait sama sekali
// — cuma insert/update baris item_reviews.
func (h *Handler) MarkReviewed(w http.ResponseWriter, r *http.Request) {
	itemType := chi.URLParam(r, "itemType")
	itemID := chi.URLParam(r, "itemID")
	reviewedBy := auth.UserIDFromContext(r.Context())

	if itemType != "daily_task" {
		http.Error(w, "item_type yang didukung saat ini cuma: daily_task", http.StatusBadRequest)
		return
	}

	var exists bool
	if err := h.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM daily_tasks WHERE id = $1)`, itemID).Scan(&exists); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "daily task tidak ditemukan", http.StatusNotFound)
		return
	}

	var resp markReviewedResponse
	err := h.db.QueryRow(r.Context(), `
		INSERT INTO item_reviews (id, item_type, item_id, reviewed_by, reviewed_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (item_type, item_id) DO UPDATE SET reviewed_by = $4, reviewed_at = now()
		RETURNING reviewed_by, reviewed_at
	`, uuid.New().String(), itemType, itemID, reviewedBy).Scan(&resp.ReviewedBy, &resp.ReviewedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Reviewed = true

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
