package weeklyplan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"hash/crc32"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"rndops/backend/internal/auth"
)

type Handler struct {
	db                 *pgxpool.Pool
	myAgendaServiceURL string
	httpClient         *http.Client
}

// myAgendaServiceURL kosong = integrasi HR nonaktif (skip diam-diam) --
// default aman, tidak breaking kalau myagenda-service belum/tidak jalan.
// Lihat docs/decision-log/decision-log-myagenda-hr-service-20260810.md.
func NewHandler(db *pgxpool.Pool, myAgendaServiceURL string) *Handler {
	return &Handler{
		db:                 db,
		myAgendaServiceURL: myAgendaServiceURL,
		httpClient:         &http.Client{Timeout: 3 * time.Second},
	}
}

type PushInfo struct {
	CallbackID string    `json:"callback_id"`
	PushedAt   time.Time `json:"pushed_at"`
}

// Row adalah satu baris di tabel Weekly Plan, granularitas PER DAILY TASK.
// Board dan big_task_name ditampilkan di UI sebagai konteks tapi TIDAK dikirim
// ke HR. TglRencana/DueDate/UraianTask adalah preview persis payload yang akan
// dikirim ke HR — dipakai tombol "Lihat" di frontend buat tampilkan pratinjau.
type Row struct {
	DailyTaskID    string    `json:"daily_task_id"`
	DailyTaskTitle string    `json:"daily_task_title"`
	BigTaskName    string    `json:"big_task_name"`
	BoardID        string    `json:"board_id"`
	BoardName      string    `json:"board_name"`
	ActualPct      int       `json:"actual_pct"`
	ExpectedPct    int       `json:"expected_pct"`
	TglRencana     string    `json:"tgl_rencana"`
	DueDate        string    `json:"due_date"`
	UraianTask     string    `json:"uraian_task"`
	LastPush       *PushInfo `json:"last_push"`
}

// resolveTargetUserID adalah logic murni di balik resolveTargetUser --
// diekstrak supaya testable tanpa perlu bikin JWT/context asli. Default diri
// sendiri; asUserID CUMA dihormati kalau callerIsSuperUser, else ditolak
// (bukan diabaikan diam-diam -- fail-loud buat percobaan akses tidak sah).
// Lihat docs/decision-log/decision-log-hr-mapping-super-user-20260810.md.
func resolveTargetUserID(requesterID, asUserID string, callerIsSuperUser bool) (string, bool) {
	if asUserID == "" {
		return requesterID, true
	}
	if !callerIsSuperUser {
		return "", false
	}
	return asUserID, true
}

// resolveTargetUser menentukan Daily Task siapa yang mau dilihat/di-push --
// wrapper resolveTargetUserID yang ambil requester/access-level dari context
// request.
func resolveTargetUser(r *http.Request, asUserID string) (string, bool) {
	return resolveTargetUserID(auth.UserIDFromContext(r.Context()), asUserID, auth.IsSuperUser(r.Context()))
}

// List mengimplementasikan GET /weekly-plan?week_start=YYYY-MM-DD (FR-WKL-01/02/03).
// "My Weekly Plan" -- di-filter ke Daily Task yang pic_user_id-nya target user
// (default diri sendiri; super_user bisa override lewat ?as_user_id=<uuid>).
// Granularitas PER DAILY TASK (bukan per big task) -- satu baris per daily task
// yang punya minimal satu day entry di rentang minggu terpilih.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := resolveTargetUser(r, r.URL.Query().Get("as_user_id"))
	if !ok {
		http.Error(w, "cuma super_user yang bisa pakai as_user_id", http.StatusForbidden)
		return
	}

	weekStart, weekEnd, err := parseWeekStart(r.URL.Query().Get("week_start"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT
			dt.id, dt.title, bt.name, b.id, b.name,
			COUNT(*) AS total_week_days,
			COALESCE(ROUND(AVG(de.progress_pct)), 0) AS actual_pct,
			COUNT(*) FILTER (WHERE de.entry_date <= CURRENT_DATE) AS elapsed_week_days,
			COALESCE(MIN(de.entry_date)::text, '') AS tgl_rencana,
			COALESCE(MAX(de.entry_date)::text, '') AS due_date,
			COALESCE(STRING_AGG(
				CASE WHEN COALESCE(de.planned_text, '') != '' THEN '- ' || de.planned_text ELSE NULL END,
				E'\n' ORDER BY de.entry_date
			), '') AS uraian_task,
			wpl.callback_id, wpl.pushed_at
		FROM daily_tasks dt
		JOIN big_tasks bt ON bt.id = dt.big_task_id
		JOIN boards b ON b.id = bt.board_id
		JOIN day_entries de ON de.daily_task_id = dt.id AND de.entry_date BETWEEN $1 AND $2
		LEFT JOIN weekly_push_log wpl ON wpl.daily_task_id = dt.id AND wpl.week_start = $1
		WHERE dt.pic_user_id = $3
		GROUP BY dt.id, dt.title, bt.name, b.id, b.name, wpl.callback_id, wpl.pushed_at
		ORDER BY b.name, bt.name, dt.title
	`, weekStart, weekEnd, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []Row{}
	for rows.Next() {
		var row Row
		var total, actualPct, elapsed int
		var callbackID *string
		var pushedAt *time.Time
		if err := rows.Scan(
			&row.DailyTaskID, &row.DailyTaskTitle, &row.BigTaskName, &row.BoardID, &row.BoardName,
			&total, &actualPct, &elapsed, &row.TglRencana, &row.DueDate, &row.UraianTask,
			&callbackID, &pushedAt,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		row.ActualPct = actualPct
		if total > 0 {
			row.ExpectedPct = (elapsed * 100) / total
		}
		if callbackID != nil {
			row.LastPush = &PushInfo{CallbackID: *callbackID, PushedAt: *pushedAt}
		}
		result = append(result, row)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type pushRequest struct {
	DailyTaskID string `json:"daily_task_id"`
	WeekStart   string `json:"week_start"`
	AsUserID    string `json:"as_user_id"`
}

// Push mengimplementasikan POST /weekly-plan/push (FR-WKL-04/05). Upsert
// berdasarkan (daily_task_id, week_start) -- callback_id baru cuma digenerate
// kalau belum ada baris; push berikutnya cuma update pushed_at & snapshot.
// `pushed_by` SELALU actor yang benar-benar menekan tombol (audit), BUKAN
// target user. Lihat decision-log-hr-mapping-super-user-20260810.md.
func (h *Handler) Push(w http.ResponseWriter, r *http.Request) {
	pushedBy := auth.UserIDFromContext(r.Context())

	var req pushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	targetUserID, ok := resolveTargetUser(r, req.AsUserID)
	if !ok {
		http.Error(w, "cuma super_user yang bisa pakai as_user_id", http.StatusForbidden)
		return
	}

	weekStart, weekEnd, err := parseWeekStart(req.WeekStart)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Verify daily task exists dan milik targetUser; hitung stats-nya sekaligus.
	var actualPct, expectedPct int
	err = h.db.QueryRow(r.Context(), `
		SELECT
			COALESCE(ROUND(AVG(de.progress_pct) FILTER (WHERE de.entry_date BETWEEN $2 AND $3)), 0),
			COALESCE(ROUND(100.0 * COUNT(*) FILTER (WHERE de.entry_date BETWEEN $2 AND $3 AND de.entry_date <= CURRENT_DATE)
				/ NULLIF(COUNT(*) FILTER (WHERE de.entry_date BETWEEN $2 AND $3), 0)), 0)
		FROM daily_tasks dt
		LEFT JOIN day_entries de ON de.daily_task_id = dt.id
		WHERE dt.id = $1 AND dt.pic_user_id = $4
		GROUP BY dt.id
	`, req.DailyTaskID, weekStart, weekEnd, targetUserID).Scan(&actualPct, &expectedPct)
	if err != nil {
		http.Error(w, "daily task tidak ditemukan atau bukan milik user ini", http.StatusNotFound)
		return
	}

	var callbackID string
	var pushedAt time.Time
	err = h.db.QueryRow(r.Context(), `
		INSERT INTO weekly_push_log
			(id, daily_task_id, week_start, callback_id, pushed_by, pushed_at, last_payload_actual_pct, last_payload_expected_pct)
		VALUES ($1, $2, $3, $4, $5, now(), $6, $7)
		ON CONFLICT (daily_task_id, week_start) DO UPDATE SET
			pushed_by = $5,
			pushed_at = now(),
			last_payload_actual_pct = $6,
			last_payload_expected_pct = $7
		RETURNING callback_id, pushed_at
	`, uuid.New().String(), req.DailyTaskID, weekStart, uuid.New().String(), pushedBy, actualPct, expectedPct).
		Scan(&callbackID, &pushedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Best-effort, TIDAK membatalkan response sukses di atas kalau gagal.
	h.pushToMyAgenda(r.Context(), req.DailyTaskID, weekStart, weekEnd, actualPct, expectedPct, targetUserID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PushInfo{CallbackID: callbackID, PushedAt: pushedAt})
}

// TeamStatus mengimplementasikan GET /weekly-plan/team-status?week_start=...
// (super_user only). Granularitas diubah ke per-daily-task (konsisten dengan
// List dan Push). Lihat decision-log-hr-mapping-super-user-20260810.md.
func (h *Handler) TeamStatus(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperUser(r.Context()) {
		http.Error(w, "cuma super_user yang bisa akses ringkasan tim", http.StatusForbidden)
		return
	}

	weekStart, weekEnd, err := parseWeekStart(r.URL.Query().Get("week_start"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT
			u.id, u.display_name, u.initials,
			COUNT(DISTINCT dt.id) FILTER (WHERE de.id IS NOT NULL) AS total_daily_tasks,
			COUNT(DISTINCT dt.id) FILTER (WHERE de.id IS NOT NULL AND wpl.daily_task_id IS NOT NULL) AS pushed_daily_tasks
		FROM users u
		LEFT JOIN daily_tasks dt ON dt.pic_user_id = u.id
		LEFT JOIN day_entries de ON de.daily_task_id = dt.id AND de.entry_date BETWEEN $1 AND $2
		LEFT JOIN weekly_push_log wpl ON wpl.daily_task_id = dt.id AND wpl.week_start = $1
		GROUP BY u.id, u.display_name, u.initials
		ORDER BY u.display_name
	`, weekStart, weekEnd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type teamStatusRow struct {
		UserID           string `json:"user_id"`
		DisplayName      string `json:"display_name"`
		Initials         string `json:"initials"`
		TotalDailyTasks  int    `json:"total_daily_tasks"`
		PushedDailyTasks int    `json:"pushed_daily_tasks"`
		AllPushed        bool   `json:"all_pushed"`
	}

	result := []teamStatusRow{}
	for rows.Next() {
		var row teamStatusRow
		if err := rows.Scan(&row.UserID, &row.DisplayName, &row.Initials, &row.TotalDailyTasks, &row.PushedDailyTasks); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		row.AllPushed = row.TotalDailyTasks > 0 && row.PushedDailyTasks == row.TotalDailyTasks
		result = append(result, row)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type myAgendaPayload struct {
	UserID       int     `json:"user_id"`
	JudulTask    string  `json:"judul_task"`
	TglRencana   string  `json:"tgl_rencana"`
	UraianTask   string  `json:"uraian_task"`
	DueDate      string  `json:"due_date"`
	Target       float64 `json:"target"`
	Capaian      float64 `json:"capaian"`
	IsPercentage bool    `json:"is_percentage"`
}

// pushToMyAgenda mengirim payload per DAILY TASK ke myagenda-service.
// Mapping:
//   - judul_task  = daily task title
//   - uraian_task = bullet list planned_text day entries di minggu ini
//   - tgl_rencana = tanggal terkecil day entry di minggu ini
//   - due_date    = tanggal terbesar day entry di minggu ini
//   - is_percentage = true, target = 100, capaian = actual_pct
//
// Board tidak dikirim (hanya tampil di UI sebagai konteks).
func (h *Handler) pushToMyAgenda(ctx context.Context, dailyTaskID, weekStart, weekEnd string, actualPct, expectedPct int, targetUserID string) {
	if h.myAgendaServiceURL == "" {
		return
	}

	var dailyTaskTitle string
	if err := h.db.QueryRow(ctx, `SELECT title FROM daily_tasks WHERE id = $1`, dailyTaskID).Scan(&dailyTaskTitle); err != nil {
		log.Printf("myagenda push: gagal ambil daily task %s: %v", dailyTaskID, err)
		return
	}

	deRows, err := h.db.Query(ctx, `
		SELECT entry_date, COALESCE(planned_text, '')
		FROM day_entries
		WHERE daily_task_id = $1 AND entry_date BETWEEN $2 AND $3
		ORDER BY entry_date
	`, dailyTaskID, weekStart, weekEnd)
	if err != nil {
		log.Printf("myagenda push: gagal ambil day entries untuk daily task %s: %v", dailyTaskID, err)
		return
	}
	defer deRows.Close()

	var minDate, maxDate string
	var lines []string
	for deRows.Next() {
		var entryDate, plannedText string
		if err := deRows.Scan(&entryDate, &plannedText); err != nil {
			continue
		}
		if minDate == "" {
			minDate = entryDate
		}
		maxDate = entryDate
		if plannedText != "" {
			lines = append(lines, "- "+plannedText)
		}
	}
	if deRows.Err() != nil {
		log.Printf("myagenda push: error iterating day entries untuk daily task %s: %v", dailyTaskID, deRows.Err())
		return
	}

	if minDate == "" {
		log.Printf("myagenda push: tidak ada day entries di minggu ini untuk daily task %s", dailyTaskID)
		return
	}

	uraianTask := strings.Join(lines, "\n")
	if uraianTask == "" {
		uraianTask = dailyTaskTitle
	}

	hrUserID := h.resolveHRUserID(ctx, targetUserID)

	payload := myAgendaPayload{
		UserID:       hrUserID,
		JudulTask:    dailyTaskTitle,
		TglRencana:   minDate,
		UraianTask:   uraianTask,
		DueDate:      maxDate,
		Target:       float64(expectedPct),
		Capaian:      float64(actualPct),
		IsPercentage: true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("myagenda push: gagal marshal payload: %v", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.myAgendaServiceURL+"/my-agenda", bytes.NewReader(body))
	if err != nil {
		log.Printf("myagenda push: gagal bikin request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		log.Printf("myagenda push: request gagal (service mungkin belum jalan, ini best-effort jadi tidak fatal): %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("myagenda push: service balikin status %d buat daily task %s", resp.StatusCode, dailyTaskID)
	}
}

// resolveHRUserID pakai users.hr_user_id ASLI kalau sudah di-mapping, fallback
// ke placeholder CRC32 (dengan warning log) kalau belum.
func (h *Handler) resolveHRUserID(ctx context.Context, userID string) int {
	var hrUserID *int
	if err := h.db.QueryRow(ctx, `SELECT hr_user_id FROM users WHERE id = $1`, userID).Scan(&hrUserID); err != nil {
		log.Printf("myagenda push: gagal ambil hr_user_id user %s, pakai placeholder: %v", userID, err)
		return placeholderHRUserID(userID)
	}
	if hrUserID == nil {
		log.Printf("myagenda push: user %s belum di-mapping ke referensi_user_hr, pakai placeholder", userID)
		return placeholderHRUserID(userID)
	}
	return *hrUserID
}

// placeholderHRUserID -- fallback kalau user BELUM di-mapping ke pegawai HR
// asli, derive angka stabil dari UUID supaya upsert myagenda tetap konsisten.
func placeholderHRUserID(userUUID string) int {
	return int(crc32.ChecksumIEEE([]byte(userUUID))%1_000_000) + 1
}

func parseWeekStart(raw string) (start, end string, err error) {
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return "", "", errInvalidWeekStart
	}
	return t.Format("2006-01-02"), t.AddDate(0, 0, 6).Format("2006-01-02"), nil
}

var errInvalidWeekStart = errors.New("week_start wajib diisi, format YYYY-MM-DD")
