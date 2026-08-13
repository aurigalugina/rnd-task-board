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
	MemberUserIDs    []string   `json:"member_user_ids"`
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
		       so.signed_by, so.signed_at,
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
			&bt.DefaultPicUserID, &bt.OnHold, &bt.ActualPct, &signedBy, &signedAt, &bt.MemberUserIDs); err != nil {
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
		       so.signed_by, so.signed_at,
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
		WHERE bt.id = $1
	`, bigTaskID).Scan(&bt.ID, &bt.BoardID, &bt.Name, &startDate, &deadline,
		&bt.DefaultPicUserID, &bt.OnHold, &bt.ActualPct, &signedBy, &signedAt, &bt.MemberUserIDs)
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
	Name             string   `json:"name"`
	StartDate        string   `json:"start_date"`
	Deadline         string   `json:"deadline"`
	DefaultPicUserID *string  `json:"default_pic_user_id"`
	MemberUserIDs    []string `json:"member_user_ids"`
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
	}

	id := uuid.New().String()
	tx, err := h.db.Begin(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	if _, err := tx.Exec(r.Context(), `
		INSERT INTO big_tasks (id, board_id, name, start_date, deadline, default_pic_user_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, boardID, req.Name, req.StartDate, req.Deadline, req.DefaultPicUserID); err != nil {
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
