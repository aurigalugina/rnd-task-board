package board

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

type Board struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// List mengimplementasikan GET /boards (05-api-contract.md §3). Board yang
// sudah di-archive TIDAK ikut muncul di sini -- endpoint ini dipakai bareng
// oleh Dashboard (aggregasi) dan halaman Boards (tab list kerja), jadi board
// archived otomatis hilang dari keduanya sekaligus. Lihat ListArchived buat
// board yang sudah diarsipkan (super_user only). Decision log:
// decision-log-board-archive-20260812.md.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(r.Context(), `SELECT id, name, description FROM boards WHERE archived_at IS NULL ORDER BY created_at`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []Board{}
	for rows.Next() {
		var b Board
		if err := rows.Scan(&b.ID, &b.Name, &b.Description); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result = append(result, b)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type createBoardRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Create mengimplementasikan POST /boards. Field 'tag' lama diganti 'description'
// -- lihat decision-log-bigtask-members-refactor-20260811.md.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createBoardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name wajib diisi", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	_, err := h.db.Exec(r.Context(), `INSERT INTO boards (id, name, description) VALUES ($1, $2, $3)`, id, req.Name, req.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Board{ID: id, Name: req.Name, Description: req.Description})
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
					SELECT daily_task_id, AVG(progress_pct) AS pct
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

type ArchivedBoard struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	ArchivedAt     string `json:"archived_at"`
	ArchivedByName string `json:"archived_by_name"`
}

// ListArchived mengimplementasikan GET /boards/archive -- cuma super_user
// (access_level, konsep terpisah dari roles many-to-many) yang boleh lihat
// daftar board yang sudah diarsipkan. Cek in-handler, bukan RequireRole
// middleware -- pola sama seperti weeklyplan.TeamStatus, lihat
// decision-log-hr-mapping-super-user-20260810.md.
func (h *Handler) ListArchived(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperUser(r.Context()) {
		http.Error(w, "cuma super_user yang bisa akses board archive", http.StatusForbidden)
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT b.id, b.name, b.description, b.archived_at, COALESCE(u.display_name, '')
		FROM boards b
		LEFT JOIN users u ON u.id = b.archived_by
		WHERE b.archived_at IS NOT NULL
		ORDER BY b.archived_at DESC
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []ArchivedBoard{}
	for rows.Next() {
		var b ArchivedBoard
		var archivedAt time.Time
		if err := rows.Scan(&b.ID, &b.Name, &b.Description, &archivedAt, &b.ArchivedByName); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		b.ArchivedAt = archivedAt.Format(time.RFC3339)
		result = append(result, b)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Archive mengimplementasikan PATCH /boards/{boardID}/archive -- super_user
// only. Keberadaan archived_at = state diarsipkan (existence-pattern,
// konsisten big_task_signoffs) -- 409 kalau board tidak ada atau sudah
// diarsipkan, bukan diam-diam no-op.
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperUser(r.Context()) {
		http.Error(w, "cuma super_user yang bisa archive board", http.StatusForbidden)
		return
	}
	boardID := chi.URLParam(r, "boardID")
	userID := auth.UserIDFromContext(r.Context())

	tag, err := h.db.Exec(r.Context(), `
		UPDATE boards SET archived_at = now(), archived_by = $2
		WHERE id = $1 AND archived_at IS NULL
	`, boardID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "board tidak ditemukan atau sudah diarsipkan", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Unarchive mengimplementasikan PATCH /boards/{boardID}/unarchive -- super_user
// only, mengembalikan board ke daftar aktif (GET /boards, GET /boards/archive
// gak lagi menampilkannya).
func (h *Handler) Unarchive(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperUser(r.Context()) {
		http.Error(w, "cuma super_user yang bisa unarchive board", http.StatusForbidden)
		return
	}
	boardID := chi.URLParam(r, "boardID")

	tag, err := h.db.Exec(r.Context(), `
		UPDATE boards SET archived_at = NULL, archived_by = NULL
		WHERE id = $1 AND archived_at IS NOT NULL
	`, boardID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "board tidak ditemukan atau belum diarsipkan", http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
