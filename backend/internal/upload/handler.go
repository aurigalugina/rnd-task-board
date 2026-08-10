package upload

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const maxUploadSize = 10 << 20 // 10MB

type Handler struct {
	dir string
}

// NewHandler menyiapkan direktori upload — lihat
// docs/decision-log/decision-log-file-upload-storage-20260809.md untuk alasan
// disk lokal + volume docker (bukan cloud storage).
func NewHandler(dir string) *Handler {
	os.MkdirAll(dir, 0o755)
	return &Handler{dir: dir}
}

// Create mengimplementasikan POST /uploads (multipart/form-data, field "file").
// Dipakai Cheat Sheet tipe file (05-api-contract.md §10).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "file terlalu besar atau request tidak valid (maks 10MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "field 'file' wajib diisi", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// filepath.Base membuang komponen direktori apa pun dari nama asli —
	// mencegah path traversal lewat nama file yang dimanipulasi.
	safeName := filepath.Base(header.Filename)
	storedName := uuid.New().String() + "_" + safeName

	dst, err := os.Create(filepath.Join(h.dir, storedName))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"value": storedName})
}

// Serve mengimplementasikan GET /uploads/{filename} — mengambil kembali file
// yang sudah diupload (dipakai link unduh Cheat Sheet tipe file).
func (h *Handler) Serve(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(chi.URLParam(r, "filename"))
	http.ServeFile(w, r, filepath.Join(h.dir, filename))
}
