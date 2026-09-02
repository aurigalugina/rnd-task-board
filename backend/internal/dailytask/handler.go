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
	ID            string `json:"id"`
	EntryDate     string `json:"entry_date"`
	PlannedText   string `json:"planned_text"`
	RealisasiText string `json:"realisasi_text"`
	ProgressPct   int    `json:"progress_pct"`
	BlockerText   string `json:"blocker_text"`
	IsWeekend     bool   `json:"is_weekend"`
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
// Permission check: user harus super_user, atau scope='team', atau (scope='self' AND member dari big task ini).
func (h *Handler) ListByBigTask(w http.ResponseWriter, r *http.Request) {
	bigTaskID := chi.URLParam(r, "bigTaskID")
	userID := auth.UserIDFromContext(r.Context())

	// Permission check: if not super_user, cek task_scope_visibility
	if !auth.IsSuperUser(r.Context()) {
		var userScope string
		var isMember bool
		err := h.db.QueryRow(r.Context(), `
			SELECT u.task_scope_visibility,
			       EXISTS(SELECT 1 FROM big_task_members WHERE big_task_id = $1 AND user_id = $2)
			FROM users u WHERE u.id = $2
		`, bigTaskID, userID).Scan(&userScope, &isMember)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Kalau scope='self', user hanya boleh akses kalau dia member Big Task itu
		if userScope == "self" && !isMember {
			http.Error(w, "tidak punya akses ke daily task ini", http.StatusForbidden)
			return
		}
	}

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
		SELECT id, entry_date, planned_text, realisasi_text, progress_pct, blocker_text
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
		if err := rows.Scan(&d.ID, &entryDate, &d.PlannedText, &d.RealisasiText, &d.ProgressPct, &d.BlockerText); err != nil {
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
	PlannedText   *string `json:"planned_text"`
	RealisasiText *string `json:"realisasi_text"`
	ProgressPct   *int    `json:"progress_pct"`
	BlockerText   *string `json:"blocker_text"`
}

// canEditDayEntry adalah logic murni di balik permission check
// UpdateDayEntry -- diekstrak supaya testable tanpa DB. Aturan: pemilik
// (PIC daily task) entry ini SELALU boleh edit; selain itu, boleh edit
// HANYA kalau user punya akses "lihat semua orang" (super_user, atau
// task_scope_visibility='team'). User scope='self' yang bukan pemilik
// DITOLAK. Lihat decision-log-team-today-edit-permission-20260902.md.
func canEditDayEntry(userID, picUserID string, isSuperUser bool, userScope string) bool {
	if userID == picUserID {
		return true
	}
	return isSuperUser || userScope == "team"
}

// UpdateDayEntry mengimplementasikan PATCH /day-entries/{day_entry_id} —
// interaksi inline cepat tanpa form terpisah (SRS FR-DLY-05). `progress_pct`
// 0-100 menggantikan `is_done` boolean lama -- 0="Belum", 100="Selesai",
// 1-99="On Progress" (turunan murni di frontend, tidak ada state tersimpan
// terpisah). Lihat docs/decision-log/decision-log-day-entry-progress-pct-20260810.md.
// `realisasi_text` (2026-09-01) -- catatan realisasi aktual di lapangan,
// terpisah dari `planned_text` (rencana) -- lihat
// decision-log-day-entry-realisasi-field-20260901.md.
//
// Permission check (2026-09-02, dipicu oleh inline edit di Team Today):
// lihat canEditDayEntry(). Hanya PIC daily task pemilik entry, ATAU user
// dengan akses "lihat semua orang", yang boleh update. Lihat
// decision-log-team-today-edit-permission-20260902.md.
func (h *Handler) UpdateDayEntry(w http.ResponseWriter, r *http.Request) {
	dayEntryID := chi.URLParam(r, "dayEntryID")
	userID := auth.UserIDFromContext(r.Context())

	if !auth.IsSuperUser(r.Context()) {
		var picUserID, userScope string
		err := h.db.QueryRow(r.Context(), `
			SELECT dt.pic_user_id, u.task_scope_visibility
			FROM day_entries de
			JOIN daily_tasks dt ON dt.id = de.daily_task_id
			JOIN users u ON u.id = $2
			WHERE de.id = $1
		`, dayEntryID, userID).Scan(&picUserID, &userScope)
		if err != nil {
			http.Error(w, "day entry tidak ditemukan", http.StatusNotFound)
			return
		}
		if !canEditDayEntry(userID, picUserID, false, userScope) {
			http.Error(w, "tidak punya akses untuk mengubah day entry ini", http.StatusForbidden)
			return
		}
	}

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
			realisasi_text = COALESCE($3, realisasi_text),
			progress_pct = COALESCE($4, progress_pct),
			blocker_text = COALESCE($5, blocker_text),
			updated_at = now()
		WHERE id = $1
		RETURNING id, entry_date, planned_text, realisasi_text, progress_pct, blocker_text
	`, dayEntryID, req.PlannedText, req.RealisasiText, req.ProgressPct, req.BlockerText).Scan(
		&d.ID, &entryDate, &d.PlannedText, &d.RealisasiText, &d.ProgressPct, &d.BlockerText,
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
		RETURNING id, entry_date, planned_text, realisasi_text, progress_pct, blocker_text
	`, uuid.New().String(), dailyTaskID, req.EntryDate, req.PlannedText).Scan(
		&d.ID, &entryDate, &d.PlannedText, &d.RealisasiText, &d.ProgressPct, &d.BlockerText,
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

// DeleteDailyTask mengimplementasikan DELETE /daily-tasks/{daily_task_id} —
// hapus permanen (bukan soft-delete, konsisten dengan DeleteDayEntry).
// SENGAJA ditolak (409) selama daily task ini MASIH punya day_entries --
// permintaan user 2026-09-02: daily task cuma boleh dihapus setelah semua
// day entries-nya sendiri dikosongkan/dihapus dulu (lewat DeleteDayEntry
// satu-satu), mencegah hilangnya histori rencana/realisasi/progress secara
// tidak sengaja hanya dengan menghapus daily task-nya. Lihat
// decision-log-daily-task-delete-20260902.md.
//
// Child records lain (comments, weekly_push_log) sudah ON DELETE CASCADE
// di skema DB -- dihapus otomatis bersama daily task tanpa perlu ditangani
// manual di sini. review_of_daily_task_id (daily task lain yang me-review
// task ini) ON DELETE SET NULL -- tidak ikut terhapus, cukup relasinya
// diputus.
func (h *Handler) DeleteDailyTask(w http.ResponseWriter, r *http.Request) {
	dailyTaskID := chi.URLParam(r, "dailyTaskID")

	var entryCount int
	if err := h.db.QueryRow(r.Context(), `
		SELECT COUNT(*) FROM day_entries WHERE daily_task_id = $1
	`, dailyTaskID).Scan(&entryCount); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if entryCount > 0 {
		http.Error(w, "hapus dulu semua day entries sebelum bisa hapus daily task ini", http.StatusConflict)
		return
	}

	tag, err := h.db.Exec(r.Context(), `DELETE FROM daily_tasks WHERE id = $1`, dailyTaskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "daily task tidak ditemukan", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TeamTodayEntry adalah satu Day Entry milik seorang user pada tanggal
// terpilih, dilengkapi konteks board/big task/daily task supaya bisa
// ditampilkan langsung tanpa fetch tambahan di frontend.
type TeamTodayEntry struct {
	DayEntryID     string `json:"day_entry_id"`
	BoardID        string `json:"board_id"`
	BoardName      string `json:"board_name"`
	BigTaskID      string `json:"big_task_id"`
	BigTaskName    string `json:"big_task_name"`
	DailyTaskID    string `json:"daily_task_id"`
	DailyTaskTitle string `json:"daily_task_title"`
	PlannedText    string `json:"planned_text"`
	RealisasiText  string `json:"realisasi_text"`
	ProgressPct    int    `json:"progress_pct"`
	BlockerText    string `json:"blocker_text"`
}

// TeamTodayUser mengelompokkan seluruh TeamTodayEntry milik satu user pada
// tanggal terpilih -- satu baris per orang di /team-today, bukan satu baris
// per entry (lihat decision-log-team-today-menu-20260901.md untuk alasan POV
// "per orang" ini).
type TeamTodayUser struct {
	UserID      string           `json:"user_id"`
	DisplayName string           `json:"display_name"`
	Initials    string           `json:"initials"`
	OrgTeam     string           `json:"org_team"`
	Entries     []TeamTodayEntry `json:"entries"`
}

// TeamToday mengimplementasikan GET /team-today?date=YYYY-MM-DD -- "apa yang
// sedang dikerjakan tim hari ini", POV per orang (bukan per board/project
// seperti Dashboard, dan bukan per minggu seperti Weekly Plan). SEMUA user
// terautentikasi bisa lihat SEMUA orang (transparan, sama seperti Dashboard)
// -- keputusan eksplisit user, TIDAK di-scope task_scope_visibility seperti
// endpoint daily-task lain. Lihat decision-log-team-today-menu-20260901.md.
//
// Default date = hari ini (server time) kalau param tidak diisi/tidak valid.
// User TANPA entry di tanggal itu tetap muncul (Entries: []) supaya terlihat
// jelas siapa yang belum update, bukan diam-diam hilang dari daftar.
func (h *Handler) TeamToday(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		http.Error(w, "date wajib format YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT
			u.id, u.display_name, u.initials, u.org_team,
			de.id, b.id, b.name, bt.id, bt.name, dt.id, dt.title,
			de.planned_text, de.realisasi_text, de.progress_pct, de.blocker_text
		FROM users u
		LEFT JOIN daily_tasks dt ON dt.pic_user_id = u.id
		LEFT JOIN day_entries de ON de.daily_task_id = dt.id AND de.entry_date = $1
		LEFT JOIN big_tasks bt ON bt.id = dt.big_task_id
		LEFT JOIN boards b ON b.id = bt.board_id
		ORDER BY u.display_name, b.name, bt.name, dt.title
	`, date)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Preserve urutan display_name dari query (ORDER BY di atas) pakai slice
	// + index map, bukan map biasa (urutan map di Go tidak deterministik).
	order := []string{}
	byUser := map[string]*TeamTodayUser{}

	for rows.Next() {
		var userID, displayName, initials, orgTeam string
		var dayEntryID, boardID, boardName, bigTaskID, bigTaskName, dailyTaskID, dailyTaskTitle *string
		var plannedText, realisasiText, blockerText *string
		var progressPct *int
		if err := rows.Scan(
			&userID, &displayName, &initials, &orgTeam,
			&dayEntryID, &boardID, &boardName, &bigTaskID, &bigTaskName, &dailyTaskID, &dailyTaskTitle,
			&plannedText, &realisasiText, &progressPct, &blockerText,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		u, ok := byUser[userID]
		if !ok {
			u = &TeamTodayUser{UserID: userID, DisplayName: displayName, Initials: initials, OrgTeam: orgTeam, Entries: []TeamTodayEntry{}}
			byUser[userID] = u
			order = append(order, userID)
		}

		// LEFT JOIN tanpa match (user tidak punya daily task, atau punya tapi
		// tidak ada entry di tanggal ini) -- dayEntryID NULL, jangan append
		// entry kosong.
		if dayEntryID == nil {
			continue
		}
		u.Entries = append(u.Entries, TeamTodayEntry{
			DayEntryID:     *dayEntryID,
			BoardID:        *boardID,
			BoardName:      *boardName,
			BigTaskID:      *bigTaskID,
			BigTaskName:    *bigTaskName,
			DailyTaskID:    *dailyTaskID,
			DailyTaskTitle: *dailyTaskTitle,
			PlannedText:    *plannedText,
			RealisasiText:  *realisasiText,
			ProgressPct:    *progressPct,
			BlockerText:    *blockerText,
		})
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := make([]TeamTodayUser, 0, len(order))
	for _, userID := range order {
		result = append(result, *byUser[userID])
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
