package notification

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rndops/backend/internal/auth"
)

type Handler struct {
	db *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{db: db}
}

// UserSettings adalah setting notifikasi per user.
type UserSettings struct {
	TelegramChatID        string `json:"telegram_chat_id"`
	TelegramThreadID      string `json:"telegram_thread_id"`
	DeadlineThresholdDays int    `json:"deadline_threshold_days"`
	CooldownHours         int    `json:"cooldown_hours"`
	NotifySignOffReady    bool   `json:"notify_sign_off_ready"`
	NotifyVerdictLose     bool   `json:"notify_verdict_lose"`
	NotifyDeadlineSoon    bool   `json:"notify_deadline_soon"`
}

func defaultSettings() UserSettings {
	return UserSettings{
		DeadlineThresholdDays: 3,
		CooldownHours:         24,
		NotifySignOffReady:    true,
		NotifyVerdictLose:     true,
		NotifyDeadlineSoon:    true,
	}
}

// GetSettings — GET /notifications/settings
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	var s UserSettings
	err := h.db.QueryRow(r.Context(), `
		SELECT COALESCE(telegram_chat_id,''), COALESCE(telegram_thread_id,''),
		       deadline_threshold_days, cooldown_hours,
		       notify_sign_off_ready, notify_verdict_lose, notify_deadline_soon
		FROM notification_settings WHERE user_id = $1
	`, userID).Scan(&s.TelegramChatID, &s.TelegramThreadID,
		&s.DeadlineThresholdDays, &s.CooldownHours,
		&s.NotifySignOffReady, &s.NotifyVerdictLose, &s.NotifyDeadlineSoon)
	if err != nil {
		s = defaultSettings()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

// UpdateSettings — PATCH /notifications/settings
func (h *Handler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	var req UserSettings
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.DeadlineThresholdDays < 1 {
		req.DeadlineThresholdDays = 1
	}
	if req.CooldownHours < 1 {
		req.CooldownHours = 1
	}
	_, err := h.db.Exec(r.Context(), `
		INSERT INTO notification_settings
			(user_id, telegram_chat_id, telegram_thread_id,
			 deadline_threshold_days, cooldown_hours,
			 notify_sign_off_ready, notify_verdict_lose, notify_deadline_soon, updated_at)
		VALUES ($1, NULLIF($2,''), NULLIF($3,''), $4, $5, $6, $7, $8, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			telegram_chat_id        = NULLIF($2,''),
			telegram_thread_id      = NULLIF($3,''),
			deadline_threshold_days = $4,
			cooldown_hours          = $5,
			notify_sign_off_ready   = $6,
			notify_verdict_lose     = $7,
			notify_deadline_soon    = $8,
			updated_at              = NOW()
	`, userID, req.TelegramChatID, req.TelegramThreadID,
		req.DeadlineThresholdDays, req.CooldownHours,
		req.NotifySignOffReady, req.NotifyVerdictLose, req.NotifyDeadlineSoon)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListAlerts — GET /notifications (in-app, fresh compute, tanpa cooldown check)
func (h *Handler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	var s UserSettings
	err := h.db.QueryRow(r.Context(), `
		SELECT COALESCE(telegram_chat_id,''), COALESCE(telegram_thread_id,''),
		       deadline_threshold_days, cooldown_hours,
		       notify_sign_off_ready, notify_verdict_lose, notify_deadline_soon
		FROM notification_settings WHERE user_id = $1
	`, userID).Scan(&s.TelegramChatID, &s.TelegramThreadID,
		&s.DeadlineThresholdDays, &s.CooldownHours,
		&s.NotifySignOffReady, &s.NotifyVerdictLose, &s.NotifyDeadlineSoon)
	if err != nil {
		s = defaultSettings()
	}
	alerts, err := computeAlertsForUser(r.Context(), h.db, userID, s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if alerts == nil {
		alerts = []Alert{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// GetAppConfig — GET /app-config/{key} (admin only)
func (h *Handler) GetAppConfig(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	var value string
	h.db.QueryRow(r.Context(), `SELECT value FROM app_config WHERE key = $1`, key).Scan(&value)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"key": key, "value": value})
}

// SetAppConfig — PUT /app-config/{key} (admin only)
func (h *Handler) SetAppConfig(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	var req struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	_, err := h.db.Exec(r.Context(), `
		INSERT INTO app_config (key, value, updated_at) VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()
	`, key, req.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
