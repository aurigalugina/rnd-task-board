package myagenda

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

type upsertRequest struct {
	UserID       int     `json:"user_id"`
	JudulTask    string  `json:"judul_task"`
	TglRencana   string  `json:"tgl_rencana"` // YYYY-MM-DD
	UraianTask   string  `json:"uraian_task"`
	DueDate      string  `json:"due_date"` // YYYY-MM-DD
	Target       float64 `json:"target"`
	Capaian      float64 `json:"capaian"`
	IsPercentage bool    `json:"is_percentage"`
}

type upsertResponse struct {
	MyAgendaID int    `json:"my_agenda_id"`
	Action     string `json:"action"` // "created" | "updated"
}

// Upsert mengimplementasikan POST /my-agenda. DDL `my_agenda` (skema asli
// sistem HR, TIDAK dimodifikasi) tidak punya unique constraint di luar PK
// auto-increment -- upsert dilakukan di level aplikasi berdasarkan kombinasi
// (user_id, judul_task, tgl_rencana) sebagai identitas natural. Lihat
// docs/decision-log/decision-log-myagenda-hr-service-20260810.md di repo
// rnd-task-board untuk alasan lengkap & catatan bahwa user_id di sini masih
// PLACEHOLDER (belum ada mapping ke id pegawai HR asli).
func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	var req upsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.UserID == 0 || req.JudulTask == "" || req.TglRencana == "" || req.DueDate == "" {
		http.Error(w, "user_id, judul_task, tgl_rencana, due_date wajib diisi", http.StatusBadRequest)
		return
	}

	prosentaseCapaian := int(req.Capaian + 0.5)

	var existingID int
	err := h.db.QueryRow(`
		SELECT my_agenda_id FROM my_agenda
		WHERE user_id = ? AND judul_task = ? AND tgl_rencana = ?
	`, req.UserID, req.JudulTask, req.TglRencana).Scan(&existingID)

	switch {
	case err == sql.ErrNoRows:
		res, err := h.db.Exec(`
			INSERT INTO my_agenda
				(user_id, judul_task, tgl_rencana, uraian_task, due_date, target, capaian, is_percentage, prosentase_capaian, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, req.UserID, req.JudulTask, req.TglRencana, req.UraianTask, req.DueDate,
			req.Target, req.Capaian, req.IsPercentage, prosentaseCapaian, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		insertID, _ := res.LastInsertId()
		writeJSON(w, http.StatusCreated, upsertResponse{MyAgendaID: int(insertID), Action: "created"})
		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	default:
		_, err := h.db.Exec(`
			UPDATE my_agenda SET
				uraian_task = ?, due_date = ?, target = ?, capaian = ?, is_percentage = ?, prosentase_capaian = ?
			WHERE my_agenda_id = ?
		`, req.UraianTask, req.DueDate, req.Target, req.Capaian, req.IsPercentage, prosentaseCapaian, existingID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, upsertResponse{MyAgendaID: existingID, Action: "updated"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
