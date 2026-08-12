package notification

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	SignOffReady = "sign_off_ready"
	VerdictLose  = "verdict_lose"
	DeadlineSoon = "deadline_soon"
)

// Alert merepresentasikan satu kondisi yang harus di-alert ke user.
type Alert struct {
	Type        string `json:"type"`
	BigTaskID   string `json:"big_task_id"`
	BigTaskName string `json:"big_task_name"`
	BoardName   string `json:"board_name"`
	DaysLeft    int    `json:"days_left"`
	ActualPct   int    `json:"actual_pct"`
	ExpectedPct int    `json:"expected_pct"`
}

// computeAlertsForUser menghitung alert relevan untuk satu user berdasarkan
// settingnya (threshold, enabled flags). Dipanggil baik oleh in-app endpoint
// (fresh per request) maupun background Telegram job.
func computeAlertsForUser(ctx context.Context, db *pgxpool.Pool, userID string, s UserSettings) ([]Alert, error) {
	// Hitung actual_pct dan expected_pct langsung di SQL (konsisten dengan
	// bigtask.ListByBoard — computed at read time, tidak pernah disimpan).
	// days_left TIDAK di-clamp ke 0 supaya nilai negatif bisa detect verdict_lose.
	rows, err := db.Query(ctx, `
		WITH bt_stats AS (
			SELECT
				bt.id,
				bt.name,
				b.name                                                        AS board_name,
				bt.start_date,
				bt.deadline,
				bt.signed,
				bt.on_hold,
				FLOOR(EXTRACT(EPOCH FROM (bt.deadline - NOW())) / 86400)::int AS days_left,
				COALESCE(ROUND(AVG(de.progress_pct))::int, 0)                AS actual_pct
			FROM big_tasks bt
			JOIN boards b ON b.id = bt.board_id
			LEFT JOIN daily_tasks dt ON dt.big_task_id = bt.id
			LEFT JOIN day_entries de ON de.daily_task_id = dt.id
			WHERE bt.signed = false AND bt.on_hold = false
			GROUP BY bt.id, b.name
		)
		SELECT
			bts.id,
			bts.name,
			bts.board_name,
			bts.days_left,
			bts.actual_pct,
			LEAST(100, GREATEST(0,
				CASE
					WHEN EXTRACT(EPOCH FROM (bts.deadline - bts.start_date)) <= 0 THEN 100
					ELSE (100.0 * EXTRACT(EPOCH FROM (NOW() - bts.start_date)) /
					             EXTRACT(EPOCH FROM (bts.deadline - bts.start_date)))::int
				END
			)) AS expected_pct
		FROM bt_stats bts
		WHERE
			EXISTS (
				SELECT 1 FROM user_roles ur
				JOIN roles ro ON ro.id = ur.role_id
				WHERE ur.user_id = $1 AND ro.code = 'spv'
			)
			OR EXISTS (
				SELECT 1 FROM big_task_members
				WHERE big_task_id = bts.id AND user_id = $1
			)
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.BigTaskID, &a.BigTaskName, &a.BoardName,
			&a.DaysLeft, &a.ActualPct, &a.ExpectedPct); err != nil {
			return nil, err
		}

		// Prioritas: sign_off_ready > verdict_lose > deadline_soon
		// Satu big task hanya generate satu jenis alert.
		switch {
		case a.ActualPct >= 100 && s.NotifySignOffReady:
			a.Type = SignOffReady
			alerts = append(alerts, a)
		case a.DaysLeft < 0 && s.NotifyVerdictLose:
			a.Type = VerdictLose
			alerts = append(alerts, a)
		case a.DaysLeft >= 0 && a.DaysLeft <= s.DeadlineThresholdDays && s.NotifyDeadlineSoon:
			a.Type = DeadlineSoon
			alerts = append(alerts, a)
		}
	}
	return alerts, rows.Err()
}
