package cheatsheet

import (
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

type Item struct {
	ID        string    `json:"id"`
	BoardID   string    `json:"board_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Value     string    `json:"value"`
	AuthorID  string    `json:"author_id"`
	CreatedAt time.Time `json:"created_at"`
}

// List mengimplementasikan GET /boards/{board_id}/cheat-sheet (FR-REF-01).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "boardID")

	rows, err := h.db.Query(r.Context(), `
		SELECT id, board_id, type, title, value, author_id, created_at
		FROM cheat_sheet_items WHERE board_id = $1 ORDER BY created_at
	`, boardID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.BoardID, &it.Type, &it.Title, &it.Value, &it.AuthorID, &it.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, it)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

type createItemRequest struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	Value string `json:"value"`
}

var validTypes = map[string]bool{"file": true, "url": true, "note": true}

// Create mengimplementasikan POST /boards/{board_id}/cheat-sheet (FR-REF-01/02/03/04).
// Untuk type "file", value berisi nama file hasil POST /uploads.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	boardID := chi.URLParam(r, "boardID")
	authorID := auth.UserIDFromContext(r.Context())

	var req createItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if !validTypes[req.Type] {
		http.Error(w, "type harus salah satu dari: file, url, note", http.StatusBadRequest)
		return
	}

	it := Item{
		ID:       uuid.New().String(),
		BoardID:  boardID,
		Type:     req.Type,
		Title:    req.Title,
		Value:    req.Value,
		AuthorID: authorID,
	}

	err := h.db.QueryRow(r.Context(), `
		INSERT INTO cheat_sheet_items (id, board_id, type, title, value, author_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`, it.ID, it.BoardID, it.Type, it.Title, it.Value, it.AuthorID).Scan(&it.CreatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(it)
}
