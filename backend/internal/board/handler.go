package board

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    *string  `json:"category"`
	TeamIDs     []string `json:"team_ids"`
}

var validCategories = map[string]bool{"project": true, "routine": true}

const boardSelectColumns = `
	b.id, b.name, b.description, b.category,
	COALESCE(bt.team_ids, ARRAY[]::text[]) AS team_ids
`
const boardTeamsJoin = `
	LEFT JOIN (
		SELECT board_id, array_agg(team_id::text ORDER BY team_id) AS team_ids
		FROM board_teams GROUP BY board_id
	) bt ON bt.board_id = b.id
`

func scanBoard(row interface{ Scan(dest ...any) error }) (Board, error) {
	var b Board
	err := row.Scan(&b.ID, &b.Name, &b.Description, &b.Category, &b.TeamIDs)
	return b, err
}

// List mengimplementasikan GET /boards (05-api-contract.md §3). Board yang
// sudah di-archive TIDAK ikut muncul di sini -- endpoint ini dipakai bareng
// oleh Dashboard (aggregasi) dan halaman Boards (tab list kerja), jadi board
// archived otomatis hilang dari keduanya sekaligus. Lihat ListArchived buat
// board yang sudah diarsipkan (super_user only). Decision log:
// decision-log-board-archive-20260812.md.
//
// Filter tim & kategori (2026-08-20, decision-log-boards-dashboard-enhancements):
// query ?category=project|routine (opsional, semua role). Regular user
// OTOMATIS dibatasi ke board yang board_teams-nya mengandung org_team dia
// (gak ada picker -- implisit); board TANPA tim ter-assign gak kelihatan sama
// sekali sampai super_user assign. super_user boleh ?team_id=<uuid> buat
// mempersempit ke 1 tim, atau kosongkan buat lihat semua tim. Level proteksi
// = filter query, BUKAN lockdown endpoint turunan (mis. /big-tasks) -- MVP
// trust-based, lihat decision log.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var category *string
	if c := r.URL.Query().Get("category"); c != "" {
		if !validCategories[c] {
			http.Error(w, "category harus salah satu dari: project, routine", http.StatusBadRequest)
			return
		}
		category = &c
	}

	var rows pgx.Rows
	var err error

	if auth.IsSuperUser(r.Context()) {
		var teamID *string
		if t := r.URL.Query().Get("team_id"); t != "" {
			teamID = &t
		}
		rows, err = h.db.Query(r.Context(), `
			SELECT `+boardSelectColumns+`
			FROM boards b
			`+boardTeamsJoin+`
			WHERE b.archived_at IS NULL
			  AND ($1::text IS NULL OR b.category = $1)
			  AND ($2::uuid IS NULL OR EXISTS (
			      SELECT 1 FROM board_teams x WHERE x.board_id = b.id AND x.team_id = $2
			  ))
			ORDER BY b.created_at
		`, category, teamID)
	} else {
		userID := auth.UserIDFromContext(r.Context())
		rows, err = h.db.Query(r.Context(), `
			SELECT `+boardSelectColumns+`
			FROM boards b
			`+boardTeamsJoin+`
			WHERE b.archived_at IS NULL
			  AND ($1::text IS NULL OR b.category = $1)
			  AND EXISTS (
			      SELECT 1 FROM board_teams x
			      JOIN referensi_tim rt ON rt.id = x.team_id
			      JOIN users u ON u.org_team = rt.name
			      WHERE x.board_id = b.id AND u.id = $2
			  )
			ORDER BY b.created_at
		`, category, userID)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []Board{}
	for rows.Next() {
		b, err := scanBoard(rows)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result = append(result, b)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type createBoardRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    *string `json:"category"`
}

// Create mengimplementasikan POST /boards. Field 'tag' lama diganti 'description'
// -- lihat decision-log-bigtask-members-refactor-20260811.md. `category`
// opsional, boleh dipilih user biasa (bukan super_user-only -- beda dari
// edit). Board baru OTOMATIS ke-assign ke org_team pembuatnya di board_teams,
// biar gak invisible dari filter tim sampai super_user assign manual. Lihat
// decision-log-boards-dashboard-enhancements-20260820.md.
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
	if req.Category != nil && !validCategories[*req.Category] {
		http.Error(w, "category harus salah satu dari: project, routine", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	userID := auth.UserIDFromContext(r.Context())

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	if _, err := tx.Exec(r.Context(), `
		INSERT INTO boards (id, name, description, category) VALUES ($1, $2, $3, $4)
	`, id, req.Name, req.Description, req.Category); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Auto-assign ke tim pembuatnya (kalau org_team dia match referensi_tim --
	// harusnya selalu match, org_team divalidasi terhadap referensi_tim saat
	// user dibuat, tapi jangan gagal keras kalau ternyata tidak ada).
	if _, err := tx.Exec(r.Context(), `
		INSERT INTO board_teams (board_id, team_id)
		SELECT $1, rt.id FROM referensi_tim rt
		JOIN users u ON u.org_team = rt.name
		WHERE u.id = $2
		ON CONFLICT DO NOTHING
	`, id, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := h.loadOne(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(b)
}

func (h *Handler) loadOne(ctx context.Context, id string) (Board, error) {
	row := h.db.QueryRow(ctx, `
		SELECT `+boardSelectColumns+`
		FROM boards b
		`+boardTeamsJoin+`
		WHERE b.id = $1
	`, id)
	return scanBoard(row)
}

type updateBoardRequest struct {
	Description *string   `json:"description"`
	Category    *string   `json:"category"`
	TeamIDs     *[]string `json:"team_ids"`
}

// Update mengimplementasikan PATCH /boards/{boardID} -- super_user only
// (in-handler check, pola sama archive/dst). Bisa ubah deskripsi, kategori,
// dan/atau REPLACE seluruh assignment tim (kalau team_ids dikirim -- nil
// berarti gak diubah, array kosong berarti dilepas dari semua tim). Lihat
// decision-log-boards-dashboard-enhancements-20260820.md.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperUser(r.Context()) {
		http.Error(w, "cuma super_user yang bisa edit board", http.StatusForbidden)
		return
	}
	boardID := chi.URLParam(r, "boardID")

	var req updateBoardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Category != nil && !validCategories[*req.Category] {
		http.Error(w, "category harus salah satu dari: project, routine", http.StatusBadRequest)
		return
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	if _, err := tx.Exec(r.Context(), `
		UPDATE boards SET
			description = COALESCE($2, description),
			category = COALESCE($3, category),
			updated_at = now()
		WHERE id = $1
	`, boardID, req.Description, req.Category); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.TeamIDs != nil {
		if _, err := tx.Exec(r.Context(), `DELETE FROM board_teams WHERE board_id = $1`, boardID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		for _, teamID := range *req.TeamIDs {
			if _, err := tx.Exec(r.Context(), `
				INSERT INTO board_teams (board_id, team_id) VALUES ($1, $2)
			`, boardID, teamID); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := h.loadOne(r.Context(), boardID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(b)
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
