package board

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

type Board struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

// List mengimplementasikan GET /boards (05-api-contract.md §3).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), `SELECT id, name, tag FROM boards ORDER BY created_at`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []Board{}
	for rows.Next() {
		var b Board
		if err := rows.Scan(&b.ID, &b.Name, &b.Tag); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result = append(result, b)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type createBoardRequest struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

// Create mengimplementasikan POST /boards.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createBoardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	_, err := h.db.Exec(r.Context(), `INSERT INTO boards (id, name, tag) VALUES ($1, $2, $3)`, id, req.Name, req.Tag)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Board{ID: id, Name: req.Name, Tag: req.Tag})
}

// Summary mengimplementasikan GET /boards/{board_id}/summary — matriks dashboard
// per board (05-api-contract.md §3). project_status "done" hanya jika seluruh
// Big Task pada board tersebut ber-status signed (BRD RULE-08 / SRS FR-BRD-07).
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "boardID")

	var total, notStarted, running, done, onHold, won, lost int
	err := h.db.QueryRow(r.Context(), `
		WITH bt AS (
			SELECT bt.id, bt.on_hold, bt.deadline,
			       COALESCE(agg.pct, 0) AS actual_pct,
			       (so.big_task_id IS NOT NULL) AS signed
			FROM big_tasks bt
			LEFT JOIN (
				SELECT dt.big_task_id, ROUND(AVG(sub.pct)) AS pct
				FROM daily_tasks dt
				JOIN (
					SELECT daily_task_id, 100.0 * COUNT(*) FILTER (WHERE is_done) / COUNT(*) AS pct
					FROM day_entries GROUP BY daily_task_id
				) sub ON sub.daily_task_id = dt.id
				GROUP BY dt.big_task_id
			) agg ON agg.big_task_id = bt.id
			LEFT JOIN big_task_signoffs so ON so.big_task_id = bt.id
			WHERE bt.board_id = $1
		)
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE actual_pct = 0 AND NOT on_hold),
			COUNT(*) FILTER (WHERE actual_pct > 0 AND actual_pct < 100 AND NOT on_hold),
			COUNT(*) FILTER (WHERE signed),
			COUNT(*) FILTER (WHERE on_hold),
			COUNT(*) FILTER (WHERE signed AND deadline >= CURRENT_DATE),
			COUNT(*) FILTER (WHERE (signed AND deadline < CURRENT_DATE) OR (NOT signed AND deadline < CURRENT_DATE))
		FROM bt
	`, boardID).Scan(&total, &notStarted, &running, &done, &onHold, &won, &lost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	completionRate := 0
	if total > 0 {
		completionRate = (done * 100) / total
	}
	projectStatus := "in_progress"
	if total > 0 && done == total {
		projectStatus = "done"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"board_id":        boardID,
		"total_big_tasks": total,
		"not_started":     notStarted,
		"running":         running,
		"done":            done,
		"on_hold":         onHold,
		"won":             won,
		"lost":            lost,
		"completion_rate": completionRate,
		"project_status":  projectStatus,
	})
}
