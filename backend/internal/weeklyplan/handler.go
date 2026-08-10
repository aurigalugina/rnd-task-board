package weeklyplan

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

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

type PushInfo struct {
	CallbackID string    `json:"callback_id"`
	PushedAt   time.Time `json:"pushed_at"`
}

type Row struct {
	BigTaskID   string    `json:"big_task_id"`
	BigTaskName string    `json:"big_task_name"`
	BoardID     string    `json:"board_id"`
	BoardName   string    `json:"board_name"`
	ActualPct   int       `json:"actual_pct"`
	ExpectedPct int       `json:"expected_pct"`
	LastPush    *PushInfo `json:"last_push"`
}

// List mengimplementasikan GET /weekly-plan?week_start=YYYY-MM-DD (FR-WKL-01/02/03).
// Hanya Big Task dengan minimal satu Day Entry di rentang minggu terpilih yang
// muncul (dijamin oleh INNER JOIN day_entries yang di-filter WHERE, bukan LEFT JOIN).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	weekStart, weekEnd, err := parseWeekStart(r.URL.Query().Get("week_start"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(r.Context(), `
		SELECT
			bt.id, bt.name, b.id, b.name,
			COUNT(*) FILTER (WHERE de.entry_date BETWEEN $1 AND $2) AS total_week_days,
			COUNT(*) FILTER (WHERE de.entry_date BETWEEN $1 AND $2 AND de.is_done) AS done_week_days,
			COUNT(*) FILTER (WHERE de.entry_date BETWEEN $1 AND $2 AND de.entry_date <= CURRENT_DATE) AS elapsed_week_days,
			wpl.callback_id, wpl.pushed_at
		FROM big_tasks bt
		JOIN boards b ON b.id = bt.board_id
		JOIN daily_tasks dt ON dt.big_task_id = bt.id
		JOIN day_entries de ON de.daily_task_id = dt.id AND de.entry_date BETWEEN $1 AND $2
		LEFT JOIN weekly_push_log wpl ON wpl.big_task_id = bt.id AND wpl.week_start = $1
		GROUP BY bt.id, bt.name, b.id, b.name, wpl.callback_id, wpl.pushed_at
		ORDER BY b.name, bt.name
	`, weekStart, weekEnd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []Row{}
	for rows.Next() {
		var row Row
		var total, done, elapsed int
		var callbackID *string
		var pushedAt *time.Time
		if err := rows.Scan(
			&row.BigTaskID, &row.BigTaskName, &row.BoardID, &row.BoardName,
			&total, &done, &elapsed, &callbackID, &pushedAt,
		); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if total > 0 {
			row.ActualPct = (done * 100) / total
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
	BigTaskID string `json:"big_task_id"`
	WeekStart string `json:"week_start"`
}

// Push mengimplementasikan POST /weekly-plan/push (FR-WKL-04/05). Upsert
// berdasarkan (big_task_id, week_start) — callback_id baru cuma digenerate
// kalau belum ada baris itu; push berikutnya cuma update pushed_at & snapshot
// payload. Fase 2 (kirim HTTP beneran ke sistem HR eksternal) di luar cakupan
// — di sini cukup dicatat lokal (04-architecture.md §5.2).
func (h *Handler) Push(w http.ResponseWriter, r *http.Request) {
	pushedBy := auth.UserIDFromContext(r.Context())

	var req pushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	weekStart, weekEnd, err := parseWeekStart(req.WeekStart)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var actualPct, expectedPct int
	err = h.db.QueryRow(r.Context(), `
		SELECT
			COALESCE(ROUND(100.0 * COUNT(*) FILTER (WHERE de.entry_date BETWEEN $2 AND $3 AND de.is_done)
				/ NULLIF(COUNT(*) FILTER (WHERE de.entry_date BETWEEN $2 AND $3), 0)), 0),
			COALESCE(ROUND(100.0 * COUNT(*) FILTER (WHERE de.entry_date BETWEEN $2 AND $3 AND de.entry_date <= CURRENT_DATE)
				/ NULLIF(COUNT(*) FILTER (WHERE de.entry_date BETWEEN $2 AND $3), 0)), 0)
		FROM daily_tasks dt
		JOIN day_entries de ON de.daily_task_id = dt.id
		WHERE dt.big_task_id = $1
	`, req.BigTaskID, weekStart, weekEnd).Scan(&actualPct, &expectedPct)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var callbackID string
	var pushedAt time.Time
	err = h.db.QueryRow(r.Context(), `
		INSERT INTO weekly_push_log
			(id, big_task_id, week_start, callback_id, pushed_by, pushed_at, last_payload_actual_pct, last_payload_expected_pct)
		VALUES ($1, $2, $3, $4, $5, now(), $6, $7)
		ON CONFLICT (big_task_id, week_start) DO UPDATE SET
			pushed_by = $5,
			pushed_at = now(),
			last_payload_actual_pct = $6,
			last_payload_expected_pct = $7
		RETURNING callback_id, pushed_at
	`, uuid.New().String(), req.BigTaskID, weekStart, uuid.New().String(), pushedBy, actualPct, expectedPct).
		Scan(&callbackID, &pushedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PushInfo{CallbackID: callbackID, PushedAt: pushedAt})
}

func parseWeekStart(raw string) (start, end string, err error) {
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return "", "", errInvalidWeekStart
	}
	return t.Format("2006-01-02"), t.AddDate(0, 0, 6).Format("2006-01-02"), nil
}

var errInvalidWeekStart = errors.New("week_start wajib diisi, format YYYY-MM-DD")
