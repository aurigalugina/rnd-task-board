package comment

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

type Comment struct {
	ID          string    `json:"id"`
	BigTaskID   string    `json:"big_task_id"`
	DailyTaskID *string   `json:"daily_task_id"`
	AuthorID    string    `json:"author_id"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
}

// List mengimplementasikan GET /big-tasks/{big_task_id}/comments?scope=all|general|{daily_task_id}
// (FR-CMT-01/02).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	bigTaskID := chi.URLParam(r, "bigTaskID")
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "all"
	}

	query := `
		SELECT id, big_task_id, daily_task_id, author_id, body, created_at
		FROM comments WHERE big_task_id = $1
	`
	args := []any{bigTaskID}
	switch scope {
	case "all":
		// tidak ada filter tambahan
	case "general":
		query += ` AND daily_task_id IS NULL`
	default:
		query += ` AND daily_task_id = $2`
		args = append(args, scope)
	}
	query += ` ORDER BY created_at`

	rows, err := h.db.Query(r.Context(), query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	comments := []Comment{}
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.BigTaskID, &c.DailyTaskID, &c.AuthorID, &c.Body, &c.CreatedAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		comments = append(comments, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

type createCommentRequest struct {
	DailyTaskID *string `json:"daily_task_id"`
	Body        string  `json:"body"`
}

// Create mengimplementasikan POST /big-tasks/{big_task_id}/comments. author_id
// selalu diambil dari token JWT, tidak pernah dari body request (FR-CMT-05).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	bigTaskID := chi.URLParam(r, "bigTaskID")
	authorID := auth.UserIDFromContext(r.Context())

	var req createCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	c := Comment{
		ID:          uuid.New().String(),
		BigTaskID:   bigTaskID,
		DailyTaskID: req.DailyTaskID,
		AuthorID:    authorID,
		Body:        req.Body,
	}

	err := h.db.QueryRow(r.Context(), `
		INSERT INTO comments (id, big_task_id, daily_task_id, author_id, body)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`, c.ID, c.BigTaskID, c.DailyTaskID, c.AuthorID, c.Body).Scan(&c.CreatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}
