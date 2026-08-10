package bigtask

import (
	"context"
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

type BigTask struct {
	ID               string     `json:"id"`
	BoardID          string     `json:"board_id"`
	Name             string     `json:"name"`
	StartDate        string     `json:"start_date"`
	Deadline         string     `json:"deadline"`
	DefaultPicUserID *string    `json:"default_pic_user_id"`
	OnHold           bool       `json:"on_hold"`
	ActualPct        int        `json:"actual_pct"`
	ExpectedPct      int        `json:"expected_pct"`
	DaysLeft         int        `json:"days_left"`
	Verdict          string     `json:"verdict"`
	Signed           bool       `json:"signed"`
	SignedBy         *string    `json:"signed_by"`
	SignedAt         *time.Time `json:"signed_at"`
}

// computeExpectedPct menerapkan SRS FR-BRD-03: expected_pct = proporsi waktu
// berjalan terhadap total durasi komitmen (start_date-deadline), diklem ke
// [0, totalDays] supaya tidak negatif sebelum start_date atau >100% setelah deadline.
func computeExpectedPct(startDate, deadline, now time.Time) int {
	totalDays := int(deadline.Sub(startDate).Hours() / 24)
	if totalDays < 1 {
		totalDays = 1
	}
	elapsed := int(now.Sub(startDate).Hours() / 24)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > totalDays {
		elapsed = totalDays
	}
	return (elapsed * 100) / totalDays
}

// computeVerdict menerapkan BRD RULE-04/05/06: status "on_progress" bersifat
// netral selama tenggat waktu belum terlampaui, terlepas dari besar-kecilnya
// gap antara realisasi dan ekspektasi. Win/Lose hanya ditentukan pada titik
// keputusan (sign-off, atau tenggat terlampaui tanpa sign-off).
func computeVerdict(deadline time.Time, signed bool, now time.Time) (verdict string, daysLeft int) {
	daysLeft = int(deadline.Sub(now).Hours() / 24)
	if signed {
		if daysLeft >= 0 {
			return "win", daysLeft
		}
		return "lose", daysLeft
	}
	if daysLeft < 0 {
		return "lose", daysLeft
	}
	return "on_progress", daysLeft
}

// ListByBoard mengimplementasikan GET /boards/{board_id}/big-tasks.
func (h *Handler) ListByBoard(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "boardID")

	rows, err := h.db.Query(r.Context(), `
		SELECT bt.id, bt.board_id, bt.name, bt.start_date, bt.deadline,
		       bt.default_pic_user_id, bt.on_hold,
		       COALESCE(agg.actual_pct, 0) AS actual_pct,
		       so.signed_by, so.signed_at
		FROM big_tasks bt
		LEFT JOIN (
			SELECT dt.big_task_id, ROUND(AVG(sub.pct)) AS actual_pct
			FROM daily_tasks dt
			JOIN (
				SELECT daily_task_id,
				       100.0 * COUNT(*) FILTER (WHERE is_done) / COUNT(*) AS pct
				FROM day_entries
				GROUP BY daily_task_id
			) sub ON sub.daily_task_id = dt.id
			GROUP BY dt.big_task_id
		) agg ON agg.big_task_id = bt.id
		LEFT JOIN big_task_signoffs so ON so.big_task_id = bt.id
		WHERE bt.board_id = $1
		ORDER BY bt.created_at
	`, boardID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []BigTask{}
	now := time.Now()

	for rows.Next() {
		var bt BigTask
		var startDate, deadline time.Time
		var signedBy *string
		var signedAt *time.Time

		if err := rows.Scan(&bt.ID, &bt.BoardID, &bt.Name, &startDate, &deadline,
			&bt.DefaultPicUserID, &bt.OnHold, &bt.ActualPct, &signedBy, &signedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		bt.StartDate = startDate.Format("2006-01-02")
		bt.Deadline = deadline.Format("2006-01-02")
		bt.Signed = signedBy != nil
		bt.SignedBy = signedBy
		bt.SignedAt = signedAt

		bt.ExpectedPct = computeExpectedPct(startDate, deadline, now)

		verdict, daysLeft := computeVerdict(deadline, bt.Signed, now)
		bt.Verdict = verdict
		bt.DaysLeft = daysLeft

		result = append(result, bt)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// loadBigTask mengambil satu Big Task dengan field turunan yang sudah dihitung
// (verdict/expected_pct/dst) — bentuk sama seperti baris di ListByBoard, dipakai
// SignOff/UndoSignOff supaya response-nya sesuai kontrak (05-api-contract.md §4:
// "Response 200: Big Task object dengan signed: true").
func loadBigTask(ctx context.Context, db *pgxpool.Pool, bigTaskID string) (BigTask, error) {
	var bt BigTask
	var startDate, deadline time.Time
	var signedBy *string
	var signedAt *time.Time

	err := db.QueryRow(ctx, `
		SELECT bt.id, bt.board_id, bt.name, bt.start_date, bt.deadline,
		       bt.default_pic_user_id, bt.on_hold,
		       COALESCE(agg.actual_pct, 0) AS actual_pct,
		       so.signed_by, so.signed_at
		FROM big_tasks bt
		LEFT JOIN (
			SELECT dt.big_task_id, ROUND(AVG(sub.pct)) AS actual_pct
			FROM daily_tasks dt
			JOIN (
				SELECT daily_task_id,
				       100.0 * COUNT(*) FILTER (WHERE is_done) / COUNT(*) AS pct
				FROM day_entries
				GROUP BY daily_task_id
			) sub ON sub.daily_task_id = dt.id
			GROUP BY dt.big_task_id
		) agg ON agg.big_task_id = bt.id
		LEFT JOIN big_task_signoffs so ON so.big_task_id = bt.id
		WHERE bt.id = $1
	`, bigTaskID).Scan(&bt.ID, &bt.BoardID, &bt.Name, &startDate, &deadline,
		&bt.DefaultPicUserID, &bt.OnHold, &bt.ActualPct, &signedBy, &signedAt)
	if err != nil {
		return BigTask{}, err
	}

	bt.StartDate = startDate.Format("2006-01-02")
	bt.Deadline = deadline.Format("2006-01-02")
	bt.Signed = signedBy != nil
	bt.SignedBy = signedBy
	bt.SignedAt = signedAt

	now := time.Now()
	bt.ExpectedPct = computeExpectedPct(startDate, deadline, now)

	verdict, daysLeft := computeVerdict(deadline, bt.Signed, now)
	bt.Verdict = verdict
	bt.DaysLeft = daysLeft

	return bt, nil
}

type createBigTaskRequest struct {
	Name             string  `json:"name"`
	StartDate        string  `json:"start_date"`
	Deadline         string  `json:"deadline"`
	DefaultPicUserID *string `json:"default_pic_user_id"`
}

// Create mengimplementasikan POST /boards/{board_id}/big-tasks (SRS FR-BRD-02).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "boardID")

	var req createBigTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	_, err := h.db.Exec(r.Context(), `
		INSERT INTO big_tasks (id, board_id, name, start_date, deadline, default_pic_user_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, boardID, req.Name, req.StartDate, req.Deadline, req.DefaultPicUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

// SignOff mengimplementasikan POST /big-tasks/{big_task_id}/sign-off.
// Ditolak (409) apabila actual_pct belum 100 (BRD RULE-07 / SRS FR-BRD-06).
func (h *Handler) SignOff(w http.ResponseWriter, r *http.Request) {
	bigTaskID := chi.URLParam(r, "bigTaskID")
	userID := auth.UserIDFromContext(r.Context())

	var actualPct int
	err := h.db.QueryRow(r.Context(), `
		SELECT COALESCE(ROUND(AVG(sub.pct)), 0)
		FROM daily_tasks dt
		JOIN (
			SELECT daily_task_id, 100.0 * COUNT(*) FILTER (WHERE is_done) / COUNT(*) AS pct
			FROM day_entries GROUP BY daily_task_id
		) sub ON sub.daily_task_id = dt.id
		WHERE dt.big_task_id = $1
	`, bigTaskID).Scan(&actualPct)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if actualPct < 100 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error":{"code":"progress_incomplete","message":"Progress belum 100%"}}`))
		return
	}

	_, err = h.db.Exec(r.Context(), `
		INSERT INTO big_task_signoffs (id, big_task_id, signed_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (big_task_id) DO UPDATE SET signed_by = $3, signed_at = now()
	`, uuid.New().String(), bigTaskID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	bt, err := loadBigTask(r.Context(), h.db, bigTaskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bt)
}

// UndoSignOff mengimplementasikan DELETE /big-tasks/{big_task_id}/sign-off (SRS FR-BRD-08).
func (h *Handler) UndoSignOff(w http.ResponseWriter, r *http.Request) {
	bigTaskID := chi.URLParam(r, "bigTaskID")
	_, err := h.db.Exec(r.Context(), `DELETE FROM big_task_signoffs WHERE big_task_id = $1`, bigTaskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
