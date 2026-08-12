package notification

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func getBotToken(ctx context.Context, db *pgxpool.Pool) string {
	var token string
	db.QueryRow(ctx, `SELECT value FROM app_config WHERE key = 'telegram_bot_token'`).Scan(&token)
	return token
}

// withinCooldown cek apakah alert type+ref sudah dikirim ke user dalam window cooldown.
func withinCooldown(ctx context.Context, db *pgxpool.Pool, userID, alertType, refID string, cooldownHours int) bool {
	var count int
	db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notification_log
		WHERE user_id = $1 AND alert_type = $2 AND ref_id = $3
		  AND sent_at > NOW() - ($4 || ' hours')::interval
	`, userID, alertType, refID, cooldownHours).Scan(&count)
	return count > 0
}

func logAlerts(ctx context.Context, db *pgxpool.Pool, userID string, alerts []Alert) {
	for _, a := range alerts {
		db.Exec(ctx, `
			INSERT INTO notification_log (user_id, alert_type, ref_id, channel)
			VALUES ($1, $2, $3, 'telegram')
		`, userID, a.Type, a.BigTaskID)
	}
}

// formatDigest menyusun satu pesan Telegram yang merangkum semua alert.
func formatDigest(alerts []Alert, now time.Time) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "🔔 <b>R&amp;D Ops Alert</b>\n📅 %s\n", now.Format("02 Jan 2006, 15:04"))

	var signOff, lose, soon []Alert
	for _, a := range alerts {
		switch a.Type {
		case SignOffReady:
			signOff = append(signOff, a)
		case VerdictLose:
			lose = append(lose, a)
		case DeadlineSoon:
			soon = append(soon, a)
		}
	}

	if len(signOff) > 0 {
		fmt.Fprintf(&sb, "\n✅ <b>Sign-off siap (%d)</b>\n", len(signOff))
		for _, a := range signOff {
			fmt.Fprintf(&sb, "• %s — %s (%d%%)\n", a.BigTaskName, a.BoardName, a.ActualPct)
		}
	}
	if len(lose) > 0 {
		fmt.Fprintf(&sb, "\n⛔ <b>Verdict Lose (%d)</b>\n", len(lose))
		for _, a := range lose {
			fmt.Fprintf(&sb, "• %s — %s (deadline %d hari lalu, %d%%)\n",
				a.BigTaskName, a.BoardName, -a.DaysLeft, a.ActualPct)
		}
	}
	if len(soon) > 0 {
		fmt.Fprintf(&sb, "\n⏰ <b>Deadline Dekat (%d)</b>\n", len(soon))
		for _, a := range soon {
			label := "hari ini"
			if a.DaysLeft == 1 {
				label = "besok"
			} else if a.DaysLeft > 1 {
				label = fmt.Sprintf("%d hari lagi", a.DaysLeft)
			}
			fmt.Fprintf(&sb, "• %s — %s (%s, %d%% vs %d%% expected)\n",
				a.BigTaskName, a.BoardName, label, a.ActualPct, a.ExpectedPct)
		}
	}
	return sb.String()
}

func sendTelegram(botToken, chatID, threadID, text string) error {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	params := url.Values{
		"chat_id":    {chatID},
		"text":       {text},
		"parse_mode": {"HTML"},
	}
	if threadID != "" {
		params.Set("message_thread_id", threadID)
	}
	resp, err := http.PostForm(endpoint, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API %d: %s", resp.StatusCode, body)
	}
	return nil
}

// RunAlertJob dipanggil background goroutine setiap N menit.
// Cek semua user dengan Telegram dikonfigurasi, kirim digest kalau ada alert baru.
func RunAlertJob(ctx context.Context, db *pgxpool.Pool) {
	botToken := getBotToken(ctx, db)
	if botToken == "" {
		return
	}

	rows, err := db.Query(ctx, `
		SELECT user_id,
		       COALESCE(telegram_chat_id, ''),
		       COALESCE(telegram_thread_id, ''),
		       deadline_threshold_days,
		       cooldown_hours,
		       notify_sign_off_ready,
		       notify_verdict_lose,
		       notify_deadline_soon
		FROM notification_settings
		WHERE telegram_chat_id IS NOT NULL AND telegram_chat_id != ''
		  AND (notify_sign_off_ready OR notify_verdict_lose OR notify_deadline_soon)
	`)
	if err != nil {
		log.Printf("[notification] query users: %v", err)
		return
	}
	defer rows.Close()

	now := time.Now()
	for rows.Next() {
		var userID string
		var s UserSettings
		if err := rows.Scan(&userID, &s.TelegramChatID, &s.TelegramThreadID,
			&s.DeadlineThresholdDays, &s.CooldownHours,
			&s.NotifySignOffReady, &s.NotifyVerdictLose, &s.NotifyDeadlineSoon); err != nil {
			continue
		}

		alerts, err := computeAlertsForUser(ctx, db, userID, s)
		if err != nil {
			log.Printf("[notification] compute alerts user %s: %v", userID, err)
			continue
		}

		// Filter alert yang masih dalam cooldown window
		var toSend []Alert
		for _, a := range alerts {
			if !withinCooldown(ctx, db, userID, a.Type, a.BigTaskID, s.CooldownHours) {
				toSend = append(toSend, a)
			}
		}
		if len(toSend) == 0 {
			continue
		}

		text := formatDigest(toSend, now)
		if err := sendTelegram(botToken, s.TelegramChatID, s.TelegramThreadID, text); err != nil {
			log.Printf("[notification] send telegram user %s: %v", userID, err)
			continue
		}
		logAlerts(ctx, db, userID, toSend)
	}
}
