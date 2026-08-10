package dailytask

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

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

type DayEntry struct {
	ID          string `json:"id"`
	EntryDate   string `json:"entry_date"`
	PlannedText string `json:"planned_text"`
	IsDone      bool   `json:"is_done"`
	BlockerText string `json:"blocker_text"`
	IsWeekend   bool   `json:"is_weekend"`
}

// isWeekend menghitung indikator "lembur" dari tanggal — tidak pernah
// disimpan sebagai kolom (SRS FR-DLY-04).
func isWeekend(t time.Time) bool {
	weekday := t.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

type DailyTask struct {
	ID        string     `json:"id"`
	BigTaskID string     `json:"big_task_id"`
	Title     string     `json:"title"`
	PicUserID string     `json:"pic_user_id"`
	StartDate string     `json:"start_date"`
	EndDate   string     `json:"end_date"`
	ActualPct int        `json:"actual_pct"`
	Days      []DayEntry `json:"days"`
}

// ListByBigTask mengimplementasikan GET /big-tasks/{big_task_id}/daily-tasks.
func (h *Handler) ListByBigTask(w http.ResponseWriter, r *http.Request) {
	bigTaskID := chi.URLParam(r, "bigTaskID")

	rows, err := h.db.Query(r.Context(), `
		SELECT id, big_task_id, title, pic_user_id, start_date, end_date
		FROM daily_tasks WHERE big_task_id = $1 ORDER BY created_at
	`, bigTaskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tasks := []DailyTask{}
	for rows.Next() {
		var dt DailyTask
		var start, end time.Time
		if err := rows.Scan(&dt.ID, &dt.BigTaskID, &dt.Title, &dt.PicUserID, &start, &end); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		dt.StartDate = start.Format("2006-01-02")
		dt.EndDate = end.Format("2006-01-02")
		tasks = append(tasks, dt)
	}

	for i := range tasks {
		days, pct, err := loadDays(r.Context(), h.db, tasks[i].ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tasks[i].Days = days
		tasks[i].ActualPct = pct
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func loadDays(ctx context.Context, db *pgxpool.Pool, dailyTaskID string) ([]DayEntry, int, error) {
	rows, err := db.Query(ctx, `
		SELECT id, entry_date, planned_text, is_done, blocker_text
		FROM day_entries WHERE daily_task_id = $1 ORDER BY entry_date
	`, dailyTaskID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	days := []DayEntry{}
	done := 0
	for rows.Next() {
		var d DayEntry
		var entryDate time.Time
		if err := rows.Scan(&d.ID, &entryDate, &d.PlannedText, &d.IsDone, &d.BlockerText); err != nil {
			return nil, 0, err
		}
		d.EntryDate = entryDate.Format("2006-01-02")
		d.IsWeekend = isWeekend(entryDate)
		if d.IsDone {
			done++
		}
		days = append(days, d)
	}

	pct := 0
	if len(days) > 0 {
		pct = (done * 100) / len(days)
	}
	return days, pct, nil
}

// loadDailyTask mengambil satu Daily Task lengkap (dengan days + actual_pct) —
// dipakai Create dan CloneReview supaya response-nya konsisten dengan bentuk
// GET (05-api-contract.md §5), bukan cuma { "id": ... }.
func loadDailyTask(ctx context.Context, db *pgxpool.Pool, id string) (DailyTask, error) {
	var dt DailyTask
	var start, end time.Time
	err := db.QueryRow(ctx, `
		SELECT id, big_task_id, title, pic_user_id, start_date, end_date
		FROM daily_tasks WHERE id = $1
	`, id).Scan(&dt.ID, &dt.BigTaskID, &dt.Title, &dt.PicUserID, &start, &end)
	if err != nil {
		return DailyTask{}, err
	}
	dt.StartDate = start.Format("2006-01-02")
	dt.EndDate = end.Format("2006-01-02")

	days, pct, err := loadDays(ctx, db, id)
	if err != nil {
		return DailyTask{}, err
	}
	dt.Days = days
	dt.ActualPct = pct

	return dt, nil
}

// insertDailyTaskWithDays insert baris daily_tasks + satu day_entries per
// tanggal kalender dalam rentang [start, end] inklusif (SRS FR-DLY-01/02).
func insertDailyTaskWithDays(
	ctx context.Context, db *pgxpool.Pool,
	id, bigTaskID, title, picUserID, startDate, endDate string,
	start, end time.Time,
) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO daily_tasks (id, big_task_id, title, pic_user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, bigTaskID, title, picUserID, startDate, endDate)
	if err != nil {
		return err
	}

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		_, err = tx.Exec(ctx, `
			INSERT INTO day_entries (id, daily_task_id, entry_date)
			VALUES ($1, $2, $3)
		`, uuid.New().String(), id, d.Format("2006-01-02"))
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func parseDateRange(startDate, endDate string) (start, end time.Time, err error) {
	start, err = time.Parse("2006-01-02", startDate)
	if err != nil {
		return
	}
	end, err = time.Parse("2006-01-02", endDate)
	if err != nil {
		return
	}
	if end.Before(start) {
		err = errEndBeforeStart
	}
	return
}

var errEndBeforeStart = errors.New("end_date must not be before start_date")

type createDailyTaskRequest struct {
	Title     string `json:"title"`
	PicUserID string `json:"pic_user_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// Create mengimplementasikan POST /big-tasks/{big_task_id}/daily-tasks.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	bigTaskID := chi.URLParam(r, "bigTaskID")

	var req createDailyTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	start, end, err := parseDateRange(req.StartDate, req.EndDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dailyTaskID := uuid.New().String()
	if err := insertDailyTaskWithDays(
		r.Context(), h.db, dailyTaskID, bigTaskID, req.Title, req.PicUserID, req.StartDate, req.EndDate, start, end,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dt, err := loadDailyTask(r.Context(), h.db, dailyTaskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dt)
}

type cloneReviewRequest struct {
	RoleTag   string `json:"role_tag"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// CloneReview mengimplementasikan POST /daily-tasks/{daily_task_id}/clone-review
// (FR-DLY-07). Detail & alasan keputusan (start_date/end_date di request,
// pemilihan PIC default): docs/decision-log/decision-log-clone-review-20260809.md.
func (h *Handler) CloneReview(w http.ResponseWriter, r *http.Request) {
	sourceID := chi.URLParam(r, "dailyTaskID")

	var req cloneReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var tag, roleCode string
	switch req.RoleTag {
	case "SPV":
		tag, roleCode = "[Review SPV]", "spv"
	case "QA":
		tag, roleCode = "[Review QA]", "qa"
	default:
		http.Error(w, "role_tag harus salah satu dari: SPV, QA", http.StatusBadRequest)
		return
	}

	start, end, err := parseDateRange(req.StartDate, req.EndDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var origTitle, bigTaskID string
	err = h.db.QueryRow(r.Context(), `
		SELECT title, big_task_id FROM daily_tasks WHERE id = $1
	`, sourceID).Scan(&origTitle, &bigTaskID)
	if err != nil {
		http.Error(w, "daily task sumber tidak ditemukan", http.StatusNotFound)
		return
	}

	var picUserID string
	err = h.db.QueryRow(r.Context(), `
		SELECT u.id FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		JOIN roles r ON r.id = ur.role_id
		WHERE r.code = $1
		ORDER BY u.display_name
		LIMIT 1
	`, roleCode).Scan(&picUserID)
	if err != nil {
		http.Error(w, "tidak ada user dengan role "+roleCode+" untuk dijadikan PIC default", http.StatusBadRequest)
		return
	}

	newID := uuid.New().String()
	newTitle := tag + " " + origTitle
	if err := insertDailyTaskWithDays(
		r.Context(), h.db, newID, bigTaskID, newTitle, picUserID, req.StartDate, req.EndDate, start, end,
	); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dt, err := loadDailyTask(r.Context(), h.db, newID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dt)
}

type updateDayEntryRequest struct {
	PlannedText *string `json:"planned_text"`
	IsDone      *bool   `json:"is_done"`
	BlockerText *string `json:"blocker_text"`
}

// UpdateDayEntry mengimplementasikan PATCH /day-entries/{day_entry_id} —
// interaksi inline cepat tanpa form terpisah (SRS FR-DLY-05).
func (h *Handler) UpdateDayEntry(w http.ResponseWriter, r *http.Request) {
	dayEntryID := chi.URLParam(r, "dayEntryID")

	var req updateDayEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	var d DayEntry
	var entryDate time.Time
	err := h.db.QueryRow(r.Context(), `
		UPDATE day_entries SET
			planned_text = COALESCE($2, planned_text),
			is_done = COALESCE($3, is_done),
			blocker_text = COALESCE($4, blocker_text),
			updated_at = now()
		WHERE id = $1
		RETURNING id, entry_date, planned_text, is_done, blocker_text
	`, dayEntryID, req.PlannedText, req.IsDone, req.BlockerText).Scan(
		&d.ID, &entryDate, &d.PlannedText, &d.IsDone, &d.BlockerText,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.EntryDate = entryDate.Format("2006-01-02")
	d.IsWeekend = isWeekend(entryDate)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d)
}
