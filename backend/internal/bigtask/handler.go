package bigtask

import (
	"context"
	"encoding/json"
	"errors"
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

type BigTask struct {
	ID                  string     `json:"id"`
	BoardID             string     `json:"board_id"`
	Name                string     `json:"name"`
	Description         string     `json:"description"`
	Severity            string     `json:"severity"`
	StartDate           string     `json:"start_date"`
	Deadline            string     `json:"deadline"`
	DefaultPicUserID    *string    `json:"default_pic_user_id"`
	OnHold              bool       `json:"on_hold"`
	ActualPct           int        `json:"actual_pct"`
	ExpectedPct         int        `json:"expected_pct"`
	DaysLeft            int        `json:"days_left"`
	Verdict             string     `json:"verdict"`
	Signed              bool       `json:"signed"`
	SignedBy            *string    `json:"signed_by"`
	SignedAt            *time.Time `json:"signed_at"`
	SignedAtBackdatedBy *string    `json:"signed_at_backdated_by"`
	UpdatedBy           *string    `json:"updated_by"`
	MemberUserIDs       []string   `json:"member_user_ids"`
}

// validSeverities -- BRD/SRS tidak mendefinisikan level severity, ini
// keputusan produk baru 2026-09-02 (permintaan user: "critical high medium
// low"), 4 level tetap, disimpan sebagai TEXT + CHECK constraint di DB
// (migration 0028) supaya konsisten dgn validasi di sini. Lihat
// decision-log-bigtask-severity-description-20260902.md.
var validSeverities = map[string]bool{
	"critical": true,
	"high":     true,
	"medium":   true,
	"low":      true,
}

func isValidSeverity(s string) bool {
	return validSeverities[s]
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
//
// signedAt (bukan cuma bool) SENGAJA dipakai buat evaluasi win/lose -- kalau
// dibandingkan ke `now` (waktu baca/request), Big Task yang sign-off-nya SAH
// on-time bakal "berubah" jadi lose begitu kalender lewat deadline & dashboard
// dibuka lagi, padahal keputusan menang itu sudah final di masa lalu. Lihat
// decision-log-verdict-backfill-signoff-20260820.md.
func computeVerdict(deadline time.Time, signedAt *time.Time, now time.Time) (verdict string, daysLeft int) {
	if signedAt != nil {
		daysLeft = int(deadline.Sub(*signedAt).Hours() / 24)
		if daysLeft >= 0 {
			return "win", daysLeft
		}
		return "lose", daysLeft
	}
	daysLeft = int(deadline.Sub(now).Hours() / 24)
	if daysLeft < 0 {
		return "lose", daysLeft
	}
	return "on_progress", daysLeft
}

// ListByBoard mengimplementasikan GET /boards/{board_id}/big-tasks.
func (h *Handler) ListByBoard(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "boardID")

	rows, err := h.db.Query(r.Context(), `
		SELECT bt.id, bt.board_id, bt.name, bt.description, bt.severity, bt.start_date, bt.deadline,
		       bt.default_pic_user_id, bt.on_hold,
		       COALESCE(agg.actual_pct, 0) AS actual_pct,
		       so.signed_by, so.signed_at, so.signed_at_backdated_by, bt.updated_by,
		       COALESCE(mem.user_ids, ARRAY[]::text[]) AS member_user_ids
		FROM big_tasks bt
		LEFT JOIN (
			SELECT dt.big_task_id, ROUND(AVG(sub.pct)) AS actual_pct
			FROM daily_tasks dt
			JOIN (
				SELECT daily_task_id,
				       AVG(progress_pct) AS pct
				FROM day_entries
				GROUP BY daily_task_id
			) sub ON sub.daily_task_id = dt.id
			GROUP BY dt.big_task_id
		) agg ON agg.big_task_id = bt.id
		LEFT JOIN big_task_signoffs so ON so.big_task_id = bt.id
		LEFT JOIN (
			SELECT big_task_id, array_agg(user_id::text ORDER BY user_id) AS user_ids
			FROM big_task_members
			GROUP BY big_task_id
		) mem ON mem.big_task_id = bt.id
		WHERE bt.board_id = $1 AND bt.deleted_at IS NULL
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

		if err := rows.Scan(&bt.ID, &bt.BoardID, &bt.Name, &bt.Description, &bt.Severity, &startDate, &deadline,
			&bt.DefaultPicUserID, &bt.OnHold, &bt.ActualPct, &signedBy, &signedAt,
			&bt.SignedAtBackdatedBy, &bt.UpdatedBy, &bt.MemberUserIDs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		bt.StartDate = startDate.Format("2006-01-02")
		bt.Deadline = deadline.Format("2006-01-02")
		bt.Signed = signedBy != nil
		bt.SignedBy = signedBy
		bt.SignedAt = signedAt

		bt.ExpectedPct = computeExpectedPct(startDate, deadline, now)

		verdict, daysLeft := computeVerdict(deadline, signedAt, now)
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
		SELECT bt.id, bt.board_id, bt.name, bt.description, bt.severity, bt.start_date, bt.deadline,
		       bt.default_pic_user_id, bt.on_hold,
		       COALESCE(agg.actual_pct, 0) AS actual_pct,
		       so.signed_by, so.signed_at, so.signed_at_backdated_by, bt.updated_by,
		       COALESCE(mem.user_ids, ARRAY[]::text[]) AS member_user_ids
		FROM big_tasks bt
		LEFT JOIN (
			SELECT dt.big_task_id, ROUND(AVG(sub.pct)) AS actual_pct
			FROM daily_tasks dt
			JOIN (
				SELECT daily_task_id,
				       AVG(progress_pct) AS pct
				FROM day_entries
				GROUP BY daily_task_id
			) sub ON sub.daily_task_id = dt.id
			GROUP BY dt.big_task_id
		) agg ON agg.big_task_id = bt.id
		LEFT JOIN big_task_signoffs so ON so.big_task_id = bt.id
		LEFT JOIN (
			SELECT big_task_id, array_agg(user_id::text ORDER BY user_id) AS user_ids
			FROM big_task_members
			GROUP BY big_task_id
		) mem ON mem.big_task_id = bt.id
		WHERE bt.id = $1 AND bt.deleted_at IS NULL
	`, bigTaskID).Scan(&bt.ID, &bt.BoardID, &bt.Name, &bt.Description, &bt.Severity, &startDate, &deadline,
		&bt.DefaultPicUserID, &bt.OnHold, &bt.ActualPct, &signedBy, &signedAt,
		&bt.SignedAtBackdatedBy, &bt.UpdatedBy, &bt.MemberUserIDs)
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

	verdict, daysLeft := computeVerdict(deadline, signedAt, now)
	bt.Verdict = verdict
	bt.DaysLeft = daysLeft

	return bt, nil
}

type createBigTaskRequest struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Severity         string   `json:"severity"`
	StartDate        string   `json:"start_date"`
	Deadline         string   `json:"deadline"`
	DefaultPicUserID *string  `json:"default_pic_user_id"`
	MemberUserIDs    []string `json:"member_user_ids"`
	// BaselinePct: persentase progress awal (opsional), sama seperti
	// baseline_pct di PATCH /big-tasks/{id} -- bisa langsung diisi saat
	// create supaya gak perlu buka Edit lagi abis Big Task kebuat. super_user
	// only, konsisten dengan field ini di endpoint Update. Lihat
	// decision-log-bigtask-baseline-progress-20260824.md.
	BaselinePct *int `json:"baseline_pct"`
}

// dedupeMembers membuang id kosong & duplikat dari daftar anggota (fungsi murni,
// ditest) -- input dari checkbox frontend bisa saja punya duplikat/kosong.
func dedupeMembers(ids []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// Create mengimplementasikan POST /boards/{board_id}/big-tasks (SRS FR-BRD-02).
// member_user_ids = siapa saja yang terlibat/menangani Big Task ini (WAJIB
// minimal 2 orang) -- lihat decision-log-bigtask-members-refactor-20260811.md.
// Anggota ini yang jadi kandidat PIC Daily Task & reviewer clone-review.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "boardID")

	var req createBigTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	members := dedupeMembers(req.MemberUserIDs)
	if len(members) < 2 {
		http.Error(w, "Big Task minimal punya 2 anggota", http.StatusBadRequest)
		return
	}

	if !auth.IsSuperUser(r.Context()) {
		today := time.Now().Format("2006-01-02")
		if req.StartDate < today || req.Deadline < today {
			http.Error(w, "Tidak bisa input tanggal lampau", http.StatusBadRequest)
			return
		}
		if req.BaselinePct != nil {
			http.Error(w, "cuma super_user yang bisa isi persentase awal", http.StatusForbidden)
			return
		}
	}
	if req.BaselinePct != nil && (*req.BaselinePct < 0 || *req.BaselinePct > 100) {
		http.Error(w, "baseline_pct harus 0-100", http.StatusBadRequest)
		return
	}
	if req.Severity == "" {
		req.Severity = "medium"
	} else if !isValidSeverity(req.Severity) {
		http.Error(w, "severity harus salah satu dari: critical, high, medium, low", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	tx, err := h.db.Begin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	if _, err := tx.Exec(r.Context(), `
		INSERT INTO big_tasks (id, board_id, name, description, severity, start_date, deadline, default_pic_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, boardID, req.Name, req.Description, req.Severity, req.StartDate, req.Deadline, req.DefaultPicUserID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, memberID := range members {
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO big_task_members (big_task_id, user_id) VALUES ($1, $2)
		`, id, memberID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if req.BaselinePct != nil {
		// default_pic_user_id boleh NULL saat create -- upsertBaseline sudah
		// fallback ke anggota pertama (ORDER BY user_id) lewat COALESCE, jadi
		// aman dipanggil di sini walau DefaultPicUserID kosong.
		if err := upsertBaseline(r.Context(), tx, id, *req.BaselinePct); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

type setMembersRequest struct {
	MemberUserIDs []string `json:"member_user_ids"`
}

// SetMembers mengimplementasikan PUT /big-tasks/{big_task_id}/members — ganti
// seluruh daftar anggota (replace-set), tetap wajib minimal 2. Dipakai untuk
// menambah/mengurangi anggota Big Task yang sudah ada (termasuk merapikan data
// lama). Lihat decision-log-bigtask-members-refactor-20260811.md.
func (h *Handler) SetMembers(w http.ResponseWriter, r *http.Request) {
	bigTaskID := chi.URLParam(r, "bigTaskID")

	var req setMembersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	members := dedupeMembers(req.MemberUserIDs)
	if len(members) < 2 {
		http.Error(w, "Big Task minimal punya 2 anggota", http.StatusBadRequest)
		return
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	if _, err := tx.Exec(r.Context(), `DELETE FROM big_task_members WHERE big_task_id = $1`, bigTaskID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, memberID := range members {
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO big_task_members (big_task_id, user_id) VALUES ($1, $2)
		`, bigTaskID, memberID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
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

type signOffRequest struct {
	// SignedAt opsional, format "YYYY-MM-DD" -- CUMA dihormati kalau requester
	// super_user (403 kalau non-super_user isi field ini). Dipakai buat backfill
	// verdict data historical (sign-off "beneran" terjadi di masa lalu, bukan
	// hari ini). Lihat decision-log-verdict-backfill-signoff-20260820.md.
	SignedAt string `json:"signed_at"`
}

// SignOff mengimplementasikan POST /big-tasks/{big_task_id}/sign-off.
// Ditolak (409) apabila actual_pct belum 100 (BRD RULE-07 / SRS FR-BRD-06).
func (h *Handler) SignOff(w http.ResponseWriter, r *http.Request) {
	bigTaskID := chi.URLParam(r, "bigTaskID")
	userID := auth.UserIDFromContext(r.Context())

	var req signOffRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // body opsional, boleh kosong/tanpa signed_at

	if req.SignedAt != "" && !auth.IsSuperUser(r.Context()) {
		http.Error(w, "cuma super_user yang bisa set tanggal sign-off custom", http.StatusForbidden)
		return
	}

	var actualPct int
	err := h.db.QueryRow(r.Context(), `
		SELECT COALESCE(ROUND(AVG(sub.pct)), 0)
		FROM daily_tasks dt
		JOIN (
			SELECT daily_task_id, AVG(progress_pct) AS pct
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

	signedAt := time.Now()
	var backdatedBy *string

	if req.SignedAt != "" {
		if _, perr := time.Parse("2006-01-02", req.SignedAt); perr != nil {
			http.Error(w, "signed_at harus format YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		today := time.Now().Format("2006-01-02")
		if req.SignedAt > today {
			http.Error(w, "signed_at tidak boleh di masa depan", http.StatusBadRequest)
			return
		}

		var startDateStr string
		var lastEntryDate *string
		if err := h.db.QueryRow(r.Context(), `
			SELECT bt.start_date::text, (
				SELECT MAX(de.entry_date)::text
				FROM day_entries de
				JOIN daily_tasks dt ON dt.id = de.daily_task_id
				WHERE dt.big_task_id = bt.id
			)
			FROM big_tasks bt WHERE bt.id = $1
		`, bigTaskID).Scan(&startDateStr, &lastEntryDate); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if req.SignedAt < startDateStr {
			http.Error(w, "signed_at tidak boleh sebelum start_date Big Task", http.StatusBadRequest)
			return
		}
		if lastEntryDate != nil && req.SignedAt < *lastEntryDate {
			http.Error(w, "signed_at tidak boleh sebelum tanggal day entry terakhir", http.StatusBadRequest)
			return
		}

		parsed, _ := time.Parse("2006-01-02", req.SignedAt)
		signedAt = parsed
		backdated := userID
		backdatedBy = &backdated
	}

	_, err = h.db.Exec(r.Context(), `
		INSERT INTO big_task_signoffs (id, big_task_id, signed_by, signed_at, signed_at_backdated_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (big_task_id) DO UPDATE SET signed_by = $3, signed_at = $4, signed_at_backdated_by = $5
	`, uuid.New().String(), bigTaskID, userID, signedAt, backdatedBy)
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

// ToggleOnHold mengimplementasikan PATCH /big-tasks/{bigTaskID}/on-hold —
// toggle on_hold flag. Task yang on_hold tidak masuk hitungan status
// not_started/running tapi tidak di-drop dari progress agregat.
func (h *Handler) ToggleOnHold(w http.ResponseWriter, r *http.Request) {
	bigTaskID := chi.URLParam(r, "bigTaskID")
	_, err := h.db.Exec(r.Context(), `
		UPDATE big_tasks SET on_hold = NOT on_hold, updated_at = now() WHERE id = $1
	`, bigTaskID)
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

type updateBigTaskRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Severity    *string `json:"severity"`
	StartDate   *string `json:"start_date"`
	Deadline    *string `json:"deadline"`
	// BaselinePct/ClearBaseline: "Baseline Awal" -- input persentase progress
	// yang sudah berjalan di lapangan sebelum Big Task ini dicatat di sistem
	// (mis. migrasi data ke staging). super_user only, lihat
	// decision-log-bigtask-baseline-progress-20260824.md. Dua field terpisah
	// (bukan cuma *int nil-able) karena JSON tidak bisa membedakan "field
	// absen" dari "field di-null-kan" lewat satu pointer saja -- frontend
	// WAJIB kirim clear_baseline:true eksplisit untuk menghapus baseline.
	BaselinePct   *int `json:"baseline_pct"`
	ClearBaseline bool `json:"clear_baseline"`
}

// upsertBaseline membuat/mengupdate Daily Task khusus "Baseline Awal" (satu
// per Big Task, ditandai is_baseline) + satu day_entries di start_date Big
// Task itu dengan progress_pct = pct. Edit ulang (pct baru) meng-UPDATE
// day_entries yang sudah ada, TIDAK insert baris baru -- mencegah history
// menumpuk/AVG tercampur ganda. Daily Task/day_entries LAIN yang sudah ada
// (real, bukan baseline) tidak pernah disentuh oleh fungsi ini. Lihat
// decision-log-bigtask-baseline-progress-20260824.md.
func upsertBaseline(ctx context.Context, tx pgx.Tx, bigTaskID string, pct int) error {
	var picID string
	var startDate time.Time
	err := tx.QueryRow(ctx, `
		SELECT COALESCE(
			bt.default_pic_user_id,
			(SELECT user_id FROM big_task_members WHERE big_task_id = bt.id ORDER BY user_id LIMIT 1)
		), bt.start_date
		FROM big_tasks bt WHERE bt.id = $1
	`, bigTaskID).Scan(&picID, &startDate)
	if err != nil {
		return err
	}
	dateStr := startDate.Format("2006-01-02")

	var dailyTaskID string
	err = tx.QueryRow(ctx, `
		SELECT id FROM daily_tasks WHERE big_task_id = $1 AND is_baseline
	`, bigTaskID).Scan(&dailyTaskID)
	if errors.Is(err, pgx.ErrNoRows) {
		dailyTaskID = uuid.New().String()
		if _, err := tx.Exec(ctx, `
			INSERT INTO daily_tasks (id, big_task_id, title, pic_user_id, start_date, end_date, is_baseline)
			VALUES ($1, $2, 'Adjustment Percentage', $3, $4, $4, true)
		`, dailyTaskID, bigTaskID, picID, dateStr); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO day_entries (id, daily_task_id, entry_date, progress_pct)
			VALUES ($1, $2, $3, $4)
		`, uuid.New().String(), dailyTaskID, dateStr, pct)
		return err
	} else if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		UPDATE day_entries SET progress_pct = $2, updated_at = now() WHERE daily_task_id = $1
	`, dailyTaskID, pct)
	return err
}

// deleteBaseline menghapus Daily Task+day_entries "Baseline Awal" Big Task ini
// (kalau ada) -- dipanggil saat field baseline dikosongkan lagi di form edit.
func deleteBaseline(ctx context.Context, tx pgx.Tx, bigTaskID string) error {
	_, err := tx.Exec(ctx, `DELETE FROM daily_tasks WHERE big_task_id = $1 AND is_baseline`, bigTaskID)
	return err
}

// Update mengimplementasikan PATCH /big-tasks/{big_task_id} -- super_user only,
// partial update (name/start_date/deadline/baseline_pct). Dipakai a.l. buat
// koreksi data biasa, sebagai salah satu jalur remediasi verdict backfill
// (geser deadline sebelum sign-off) -- lihat
// decision-log-verdict-backfill-signoff-20260820.md -- dan buat set/adjust
// "Baseline Awal" progress migrasi data -- lihat
// decision-log-bigtask-baseline-progress-20260824.md. on_hold TETAP lewat
// endpoint terpisah yang sudah ada (ToggleOnHold), gak masuk sini.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperUser(r.Context()) {
		http.Error(w, "cuma super_user yang bisa edit big task", http.StatusForbidden)
		return
	}
	bigTaskID := chi.URLParam(r, "bigTaskID")

	var req updateBigTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == nil && req.Description == nil && req.Severity == nil && req.StartDate == nil && req.Deadline == nil && req.BaselinePct == nil && !req.ClearBaseline {
		http.Error(w, "tidak ada field yang diubah", http.StatusBadRequest)
		return
	}
	if req.Severity != nil && !isValidSeverity(*req.Severity) {
		http.Error(w, "severity harus salah satu dari: critical, high, medium, low", http.StatusBadRequest)
		return
	}
	if req.BaselinePct != nil && (*req.BaselinePct < 0 || *req.BaselinePct > 100) {
		http.Error(w, "baseline_pct harus 0-100", http.StatusBadRequest)
		return
	}
	if req.BaselinePct != nil && req.ClearBaseline {
		http.Error(w, "baseline_pct dan clear_baseline tidak bisa dipakai bersamaan", http.StatusBadRequest)
		return
	}

	userID := auth.UserIDFromContext(r.Context())

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	if _, err := tx.Exec(r.Context(), `
		UPDATE big_tasks SET
			name = COALESCE($2, name),
			description = COALESCE($3, description),
			severity = COALESCE($4, severity),
			start_date = COALESCE($5, start_date),
			deadline = COALESCE($6, deadline),
			updated_by = $7,
			updated_at = now()
		WHERE id = $1
	`, bigTaskID, req.Name, req.Description, req.Severity, req.StartDate, req.Deadline, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.BaselinePct != nil {
		if err := upsertBaseline(r.Context(), tx, bigTaskID, *req.BaselinePct); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if req.ClearBaseline {
		if err := deleteBaseline(r.Context(), tx, bigTaskID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
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

// Delete mengimplementasikan DELETE /big-tasks/{big_task_id} (soft delete, super_user only).
// Menandai Big Task sebagai "deleted" tanpa menghapus row fisik dari database.
// Child records (daily_tasks, comments, day_entries, dst) tetap ada, tetapi tidak
// ditampilkan di layer aplikasi karena query-nya mengecek deleted_at IS NULL.
// Safety pattern: mencegah data loss accidental sambil tetap maintaining referential integrity.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if !auth.IsSuperUser(r.Context()) {
		http.Error(w, "cuma super_user yang bisa delete big task", http.StatusForbidden)
		return
	}
	bigTaskID := chi.URLParam(r, "bigTaskID")
	userID := auth.UserIDFromContext(r.Context())

	_, err := h.db.Exec(r.Context(), `
		UPDATE big_tasks SET deleted_at = now(), deleted_by = $2, updated_at = now()
		WHERE id = $1
	`, bigTaskID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
