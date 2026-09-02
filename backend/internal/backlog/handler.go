// Package backlog mengimplementasikan Board Backlog -- item planning mentah
// per board (judul + deskripsi) sebelum ditentukan Big Task/PIC/tanggal-nya.
// Reusable: sebuah backlog item bisa "dipromosikan" jadi Daily Task berkali-
// kali (mis. kerjaan recurring), item aslinya tidak hilang/habis sekali pakai.
// Lihat decision-log-board-backlog-20260902.md.
package backlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"

	"rndops/backend/internal/auth"
)

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

type Item struct {
	ID              string    `json:"id"`
	BoardID         string    `json:"board_id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	CreatedBy       string    `json:"created_by"`
	CreatedByName   string    `json:"created_by_name"`
	CreatedAt       time.Time `json:"created_at"`
	// PromotedCount = berapa kali item ini sudah dipromosikan jadi Daily
	// Task (via source_backlog_item_id) -- badge "dipakai di N daily task"
	// di UI. Item TETAP ADA setelah dipromosikan (reusable, bukan sekali
	// pakai), sesuai permintaan eksplisit user.
	PromotedCount int `json:"promoted_count"`
}

// canManageBacklog menerapkan aturan permission yang DISENGAJA TIDAK terikat
// role/access_level: super_user selalu boleh, selain itu HARUS flag
// can_manage_backlog di user (independen, mirip pola task_scope_visibility).
// Permintaan user eksplisit: "jangan terpaut sama role".
func canManageBacklog(isSuperUser, userCanManage bool) bool {
	return isSuperUser || userCanManage
}

// requireManagePermission mengembalikan true (dan sudah menulis 403) kalau
// user TIDAK boleh kelola backlog -- dipanggil di awal Create/Update/Delete.
func (h *Handler) requireManagePermission(w http.ResponseWriter, r *http.Request) bool {
	if auth.IsSuperUser(r.Context()) {
		return true
	}
	userID := auth.UserIDFromContext(r.Context())
	var flag bool
	if err := h.db.QueryRow(r.Context(), `
		SELECT can_manage_backlog FROM users WHERE id = $1
	`, userID).Scan(&flag); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	if !canManageBacklog(false, flag) {
		http.Error(w, "tidak punya akses kelola backlog", http.StatusForbidden)
		return false
	}
	return true
}

// ListByBoard mengimplementasikan GET /boards/{board_id}/backlog-items --
// SEMUA user login bisa lihat (transparan, permintaan eksplisit user),
// hanya Create/Update/Delete yang digate can_manage_backlog.
func (h *Handler) ListByBoard(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "boardID")

	rows, err := h.db.Query(r.Context(), `
		SELECT bi.id, bi.board_id, bi.title, bi.description, bi.created_by,
		       COALESCE(u.display_name, ''), bi.created_at,
		       COALESCE(pc.cnt, 0)
		FROM board_backlog_items bi
		LEFT JOIN users u ON u.id = bi.created_by
		LEFT JOIN (
			SELECT source_backlog_item_id, COUNT(*) AS cnt
			FROM daily_tasks
			WHERE source_backlog_item_id IS NOT NULL
			GROUP BY source_backlog_item_id
		) pc ON pc.source_backlog_item_id = bi.id
		WHERE bi.board_id = $1
		ORDER BY bi.created_at DESC
	`, boardID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []Item{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.BoardID, &it.Title, &it.Description,
			&it.CreatedBy, &it.CreatedByName, &it.CreatedAt, &it.PromotedCount); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		result = append(result, it)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type createItemRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Create mengimplementasikan POST /boards/{board_id}/backlog-items --
// digate can_manage_backlog (super_user ATAU flag, bukan role).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if !h.requireManagePermission(w, r) {
		return
	}
	boardID := chi.URLParam(r, "boardID")
	userID := auth.UserIDFromContext(r.Context())

	var req createItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Title == "" {
		http.Error(w, "title wajib diisi", http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	if _, err := h.db.Exec(r.Context(), `
		INSERT INTO board_backlog_items (id, board_id, title, description, created_by)
		VALUES ($1, $2, $3, $4, $5)
	`, id, boardID, req.Title, req.Description, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

type updateItemRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
}

// Update mengimplementasikan PATCH /backlog-items/{id} -- digate
// can_manage_backlog.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if !h.requireManagePermission(w, r) {
		return
	}
	itemID := chi.URLParam(r, "itemID")

	var req updateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Title != nil && *req.Title == "" {
		http.Error(w, "title tidak boleh kosong", http.StatusBadRequest)
		return
	}

	tag, err := h.db.Exec(r.Context(), `
		UPDATE board_backlog_items SET
			title = COALESCE($2, title),
			description = COALESCE($3, description),
			updated_at = now()
		WHERE id = $1
	`, itemID, req.Title, req.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "backlog item tidak ditemukan", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete mengimplementasikan DELETE /backlog-items/{id} -- digate
// can_manage_backlog. Daily Task yang sudah dipromosikan dari item ini
// TIDAK ikut terhapus (source_backlog_item_id ON DELETE SET NULL).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if !h.requireManagePermission(w, r) {
		return
	}
	itemID := chi.URLParam(r, "itemID")

	tag, err := h.db.Exec(r.Context(), `DELETE FROM board_backlog_items WHERE id = $1`, itemID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if tag.RowsAffected() == 0 {
		http.Error(w, "backlog item tidak ditemukan", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DownloadTemplate mengimplementasikan GET /boards/{board_id}/backlog-items/template
// -- workbook XLSX SIAP-ISI, dirancang SUPAYA user isi tanpa kesalahan
// SEBELUM import parser dipakai. Permintaan eksplisit user: "sediain dulu
// templatenya biar user bisa isi tanpa ada kesalahan" -- template harus ada
// LEBIH DULU sebelum Import dipakai, bukan reverse-engineer dari format
// import.
//
// SENGAJA 2 SHEET terpisah -- sheet "Backlog" HANYA berisi header + 1 baris
// contoh (murni data, kolom A/B), sheet "Petunjuk" berisi instruksi teks
// bebas. Import HANYA membaca sheet pertama (index 0) -- kalau instruksi
// ditaruh di sheet yang sama (baris di bawah data), parser akan salah
// mengira baris instruksi itu baris data (ditemukan & diperbaiki saat
// verifikasi manual, lihat decision-log-board-backlog-20260902.md).
func (h *Handler) DownloadTemplate(w http.ResponseWriter, r *http.Request) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Backlog"
	f.SetSheetName("Sheet1", sheet)
	f.SetSheetRow(sheet, "A1", &[]interface{}{"Judul", "Deskripsi"})
	f.SetSheetRow(sheet, "A2", &[]interface{}{
		"Contoh: Setup CI/CD pipeline",
		"Contoh: Buat pipeline build+test otomatis di GitHub Actions (opsional, boleh dikosongkan)",
	})
	f.SetColWidth(sheet, "A", "A", 35)
	f.SetColWidth(sheet, "B", "B", 60)

	boldStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	f.SetCellStyle(sheet, "A1", "B1", boldStyle)
	italicStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Italic: true, Color: "808080"}})
	f.SetCellStyle(sheet, "A2", "B2", italicStyle)

	instrSheet := "Petunjuk"
	f.NewSheet(instrSheet)
	f.SetSheetRow(instrSheet, "A1", &[]interface{}{"Petunjuk pengisian sheet 'Backlog':"})
	f.SetSheetRow(instrSheet, "A2", &[]interface{}{"1. Kolom Judul WAJIB diisi, jangan dikosongkan."})
	f.SetSheetRow(instrSheet, "A3", &[]interface{}{"2. Kolom Deskripsi boleh dikosongkan."})
	f.SetSheetRow(instrSheet, "A4", &[]interface{}{"3. Hapus baris contoh (baris 2 di sheet Backlog) sebelum upload, atau biarkan -- baris kosong di kolom Judul otomatis dilewati saat import."})
	f.SetSheetRow(instrSheet, "A5", &[]interface{}{"4. Jangan ubah urutan/nama kolom di baris 1 sheet Backlog (header)."})
	f.SetSheetRow(instrSheet, "A6", &[]interface{}{"5. Jangan tambah data di sheet ini (Petunjuk) -- hanya sheet 'Backlog' yang dibaca saat import."})
	f.SetColWidth(instrSheet, "A", "A", 90)
	f.SetCellStyle(instrSheet, "A1", "A1", boldStyle)

	sheetIdx, _ := f.GetSheetIndex(sheet)
	f.SetActiveSheet(sheetIdx)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		http.Error(w, "excel write error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="backlog-template.xlsx"`)
	w.Write(buf.Bytes()) //nolint:errcheck
}

type importResult struct {
	Created  int      `json:"created"`
	Warnings []string `json:"warnings"`
}

// Import mengimplementasikan POST /boards/{board_id}/backlog-items/import --
// digate can_manage_backlog (sama seperti Create). Menerima multipart/form-
// data field "file" berisi workbook XLSX sesuai format DownloadTemplate.
// Baris dengan Judul kosong dilewati (tidak dianggap error) -- mengakomodasi
// baris contoh di template yang mungkin sengaja tidak dihapus user.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	if !h.requireManagePermission(w, r) {
		return
	}
	boardID := chi.URLParam(r, "boardID")
	userID := auth.UserIDFromContext(r.Context())

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "gagal parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "field 'file' tidak ditemukan: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "gagal baca file: "+err.Error(), http.StatusBadRequest)
		return
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		http.Error(w, "format Excel tidak valid: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer f.Close()

	// Baca sheet "Backlog" secara eksplisit (bukan index 0) -- lebih robust
	// kalau user tidak sengaja mengubah urutan sheet di Excel. Fallback ke
	// sheet pertama kalau nama "Backlog" tidak ditemukan (mis. user rename
	// sheet-nya), supaya tetap coba proses alih-alih langsung gagal.
	sheetName := "Backlog"
	if idx, _ := f.GetSheetIndex(sheetName); idx == -1 {
		sheetName = f.GetSheetName(0)
	}
	rows, err := f.GetRows(sheetName)
	if err != nil {
		http.Error(w, "gagal baca sheet: "+err.Error(), http.StatusBadRequest)
		return
	}

	result := importResult{Warnings: []string{}}
	if len(rows) <= 1 {
		json.NewEncoder(w).Encode(result) //nolint:errcheck
		return
	}

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		http.Error(w, "begin tx: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context()) //nolint:errcheck

	for i, row := range rows[1:] { // skip header row
		title := cellStr(row, 0)
		if title == "" {
			continue // baris kosong (mis. dibiarkan dari template) -- bukan error, dilewati saja
		}
		desc := cellStr(row, 1)

		if _, err := tx.Exec(r.Context(), `
			INSERT INTO board_backlog_items (id, board_id, title, description, created_by)
			VALUES ($1, $2, $3, $4, $5)
		`, uuid.New().String(), boardID, title, desc, userID); err != nil {
			http.Error(w, fmt.Sprintf("baris %d: %s", i+2, err.Error()), http.StatusInternalServerError)
			return
		}
		result.Created++
	}

	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "commit: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result) //nolint:errcheck
}

// cellStr returns row[i] if it exists (trimmed), else "".
func cellStr(row []string, i int) string {
	if len(row) <= i {
		return ""
	}
	return strings.TrimSpace(row[i])
}
