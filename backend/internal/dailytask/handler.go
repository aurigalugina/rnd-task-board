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

	"rndops/backend/internal/auth"
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
	ProgressPct int    `json:"progress_pct"`
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
	// IsBaseline: true kalau ini Daily Task khusus "Baseline Awal" (progress
	// awal migrasi data yang diisi lewat PATCH /big-tasks/{id} baseline_pct,
	// bukan Daily Task PIC beneran) -- lihat
	// decision-log-bigtask-baseline-progress-20260824.md.
	IsBaseline bool       `json:"is_baseline"`
	Days       []DayEntry `json:"days"`
}

// ListByBigTask mengimplementasikan GET /big-tasks/{big_task_id}/daily-tasks.
func (h *Handler) ListByBigTask(w http.ResponseWriter, r *http.Request) {
	bigTaskID := chi.URLParam(r, "bigTaskID")

	rows, err := h.db.Query(r.Context(), `
		SELECT id, big_task_id, title, pic_user_id, start_date, end_date, is_baseline
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
		if err := rows.Scan(&dt.ID, &dt.BigTaskID, &dt.Title, &dt.PicUserID, &start, &end, &dt.IsBaseline); err != nil {
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
		SELECT id, entry_date, planned_text, progress_pct, blocker_text
		FROM day_entries WHERE daily_task_id = $1 ORDER BY entry_date, created_at
	`, dailyTaskID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	days := []DayEntry{}
	sum := 0
	for rows.Next() {
		var d DayEntry
		var entryDate time.Time
		if err := rows.Scan(&d.ID, &entryDate, &d.PlannedText, &d.ProgressPct, &d.BlockerText); err != nil {
			return nil, 0, err
		}
		d.EntryDate = entryDate.Format("2006-01-02")
		d.IsWeekend = isWeekend(entryDate)
		sum += d.ProgressPct
		days = append(days, d)
	}

	pct := 0
	if len(days) > 0 {
		pct = sum / len(days)
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
		SELECT id, big_task_id, title, pic_user_id, start_date, end_date, is_baseline
		FROM daily_tasks WHERE id = $1
	`, id).Scan(&dt.ID, &dt.BigTaskID, &dt.Title, &dt.PicUserID, &start, &end, &dt.IsBaseline)
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

// isMemberOfBigTask cek apakah userID termasuk anggota big_task_members. Dipakai
// untuk validasi PIC Daily Task & reviewer clone-review wajib anggota Big Task
// -- lihat decision-log-bigtask-members-refactor-20260811.md.
func isMemberOfBigTask(ctx context.Context, db *pgxpool.Pool, bigTaskID, userID string) (bool, error) {
	var ok bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM big_task_members WHERE big_task_id = $1 AND user_id = $2)
	`, bigTaskID, userID).Scan(&ok)
	return ok, err
}

// insertDailyTaskWithDays insert baris daily_tasks + satu day_entries per
// tanggal kalender dalam rentang [start, end] inklusif (SRS FR-DLY-01/02).
// reviewOf != nil menandai ini task review dari daily task lain (clone-review).
func insertDailyTaskWithDays(
	ctx context.Context, db *pgxpool.Pool,
	id, bigTaskID, title, picUserID, startDate, endDate string,
	start, end time.Time, reviewOf *string,
) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO daily_tasks (id, big_task_id, title, pic_user_id, start_date, end_date, review_of_daily_task_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, id, bigTaskID, title, picUserID, startDate, endDate, reviewOf)
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

	if !auth.IsSuperUser(r.Context()) {
		today := time.Now().Format("2006-01-02")
		if req.StartDate < today || req.EndDate < today {
			http.Error(w, "Tidak bisa input tanggal lampau", http.StatusBadRequest)
			return
		}
	}

	// PIC wajib anggota Big Task (decision-log-bigtask-members-refactor-20260811).
	member, err := isMemberOfBigTask(r.Context(), h.db, bigTaskID, req.PicUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !member {
		http.Error(w, "PIC harus salah satu anggota Big Task", http.StatusBadRequest)
		return
	}

	dailyTaskID := uuid.New().String()
	if err := insertDailyTaskWithDays(
		r.Context(), h.db, dailyTaskID, bigTaskID, req.Title, req.PicUserID, req.StartDate, req.EndDate, start, end, nil,
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
	ReviewerUserID string `json:"reviewer_user_id"`
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
}

// CloneReview mengimplementasikan POST /daily-tasks/{daily_task_id}/clone-review
// (FR-DLY-07). Sekarang assign ORANG spesifik sebagai reviewer (wajib anggota
// Big Task), judul jadi "[Review <nama>] <judul asal>", dan menyimpan
// review_of_daily_task_id = daily task asal. Lihat
// decision-log-bigtask-members-refactor-20260811.md (menggantikan pemilihan
// role SPV/QA di decision-log-clone-review-20260809.md).
func (h *Handler) CloneReview(w http.ResponseWriter, r *http.Request) {
	sourceID := chi.URLParam(r, "dailyTaskID")

	var req cloneReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ReviewerUserID == "" {
		http.Error(w, "reviewer_user_id wajib diisi", http.StatusBadRequest)
		return
	}

	start, end, err := parseDateRange(req.StartDate, req.EndDate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !auth.IsSuperUser(r.Context()) {
		today := time.Now().Format("2006-01-02")
		if req.StartDate < today || req.EndDate < today {
			http.Error(w, "Tidak bisa input tanggal lampau", http.StatusBadRequest)
			return
		}
	}

	var origTitle, bigTaskID string
	err = h.db.QueryRow(r.Context(), `
		SELECT title, big_task_id FROM daily_tasks WHERE id = $1
	`, sourceID).Scan(&origTitle, &bigTaskID)
	if err != nil {
		http.Error(w, "daily task sumber tidak ditemukan", http.StatusNotFound)
		return
	}

	// Reviewer wajib anggota Big Task.
	member, err := isMemberOfBigTask(r.Context(), h.db, bigTaskID, req.ReviewerUserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !member {
		http.Error(w, "reviewer harus salah satu anggota Big Task", http.StatusBadRequest)
		return
	}

	var reviewerName string
	if err := h.db.QueryRow(r.Context(), `SELECT display_name FROM users WHERE id = $1`, req.ReviewerUserID).Scan(&reviewerName); err != nil {
		http.Error(w, "reviewer tidak ditemukan", http.StatusBadRequest)
		return
	}

	newID := uuid.New().String()
	newTitle := "[Review " + reviewerName + "] " + origTitle
	if err := insertDailyTaskWithDays(
		r.Context(), h.db, newID, bigTaskID, newTitle, req.ReviewerUserID, req.StartDate, req.EndDate, start, end, &sourceID,
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
	ProgressPct *int    `json:"progress_pct"`
	BlockerText *string `json:"blocker_text"`
}

// UpdateDayEntry mengimplementasikan PATCH /day-entries/{day_entry_id} —
// interaksi inline cepat tanpa form terpisah (SRS FR-DLY-05). `progress_pct`
// 0-100 menggantikan `is_done` boolean lama -- 0="Belum", 100="Selesai",
// 1-99="On Progress" (turunan murni di frontend, tidak ada state tersimpan
// terpisah). Lihat docs/decision-log/decision-log-day-entry-progress-pct-20260810.md.
func (h *Handler) UpdateDayEntry(w http.ResponseWriter, r *http.Request) {
	dayEntryID := chi.URLParam(r, "dayEntryID")

	var req updateDayEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ProgressPct != nil && (*req.ProgressPct < 0 || *req.ProgressPct > 100) {
		http.Error(w, "progress_pct harus antara 0-100", http.StatusBadRequest)
		return
	}

	var d DayEntry
	var entryDate time.Time
	err := h.db.QueryRow(r.Context(), `
		UPDATE day_entries SET
			planned_text = COALESCE($2, planned_text),
			progress_pct = COALESCE($3, progress_pct),
			blocker_text = COALESCE($4, blocker_text),
			updated_at = now()
		WHERE id = $1
		RETURNING id, entry_date, planned_text, progress_pct, blocker_text
	`, dayEntryID, req.PlannedText, req.ProgressPct, req.BlockerText).Scan(
		&d.ID, &entryDate, &d.PlannedText, &d.ProgressPct, &d.BlockerText,
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

type addDayEntryRequest struct {
	EntryDate   string `json:"entry_date"`
	PlannedText string `json:"planned_text"`
}

// AddDayEntry mengimplementasikan POST /daily-tasks/{daily_task_id}/day-entries
// — nambah SATU baris day_entries manual di luar generate otomatis
// (FR-DLY-01/02 tetap berlaku buat generate awal). Dipakai buat kasus PIC mau
// breakdown lebih dari satu task di tanggal yang sama — lihat
// docs/decision-log/decision-log-day-entry-add-delete-20260810.md. Tidak
// divalidasi terhadap rentang start_date/end_date Daily Task (sengaja,
// lihat decision log).
func (h *Handler) AddDayEntry(w http.ResponseWriter, r *http.Request) {
	dailyTaskID := chi.URLParam(r, "dailyTaskID")

	var req addDayEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	entryDate, err := time.Parse("2006-01-02", req.EntryDate)
	if err != nil {
		http.Error(w, "entry_date tidak valid", http.StatusBadRequest)
		return
	}

	var d DayEntry
	err = h.db.QueryRow(r.Context(), `
		INSERT INTO day_entries (id, daily_task_id, entry_date, planned_text)
		VALUES ($1, $2, $3, $4)
		RETURNING id, entry_date, planned_text, progress_pct, blocker_text
	`, uuid.New().String(), dailyTaskID, req.EntryDate, req.PlannedText).Scan(
		&d.ID, &entryDate, &d.PlannedText, &d.ProgressPct, &d.BlockerText,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.EntryDate = entryDate.Format("2006-01-02")
	d.IsWeekend = isWeekend(entryDate)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(d)
}

// DeleteDayEntry mengimplementasikan DELETE /day-entries/{day_entry_id} —
// hapus permanen (bukan soft-delete). actual_pct otomatis konsisten karena
// dihitung dari SEMUA day_entries yang ada saat dibaca (bukan disimpan).
func (h *Handler) DeleteDayEntry(w http.ResponseWriter, r *http.Request) {
	dayEntryID := chi.URLParam(r, "dayEntryID")

	tag, err := h.db.Exec(r.Context(), `DELETE FROM day_entries WHERE id = $1`, dayEntryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "day entry tidak ditemukan", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
