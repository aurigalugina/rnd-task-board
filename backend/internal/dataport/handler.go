package dataport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xuri/excelize/v2"

	"rndops/backend/internal/auth"
)

type Handler struct{ db *pgxpool.Pool }

func NewHandler(db *pgxpool.Pool) *Handler { return &Handler{db: db} }

// ---------- import result ----------

type importResult struct {
	BoardsCreated     int      `json:"boards_created"`
	BigTasksCreated   int      `json:"big_tasks_created"`
	DailyTasksCreated int      `json:"daily_tasks_created"`
	DayEntriesCreated int      `json:"day_entries_created"`
	Warnings          []string `json:"warnings"`
}

// Export handles GET /admin/export — streams all active project data as an XLSX workbook.
// Sheets: Boards | BigTasks | DailyTasks | DayEntries
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !auth.IsSuperUser(ctx) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	f := excelize.NewFile()
	defer f.Close()

	// ---- Sheets & headers ----
	f.SetSheetName("Sheet1", "Boards")
	f.SetSheetRow("Boards", "A1", &[]interface{}{"Name", "Description"})

	f.NewSheet("BigTasks")
	f.SetSheetRow("BigTasks", "A1", &[]interface{}{"Board", "Name", "StartDate", "Deadline", "OnHold", "Members"})

	f.NewSheet("DailyTasks")
	f.SetSheetRow("DailyTasks", "A1", &[]interface{}{"Board", "BigTask", "Title", "PICEmail", "StartDate", "EndDate"})

	f.NewSheet("DayEntries")
	f.SetSheetRow("DayEntries", "A1",
		&[]interface{}{"Board", "BigTask", "DailyTask", "EntryDate", "PlannedText", "ProgressPct", "BlockerText"})

	bRow, btRow, dtRow, deRow := 2, 2, 2, 2

	bRows, err := h.db.Query(ctx,
		`SELECT id, name, COALESCE(description,'') FROM boards WHERE archived_at IS NULL ORDER BY created_at`)
	if err != nil {
		http.Error(w, "db error boards: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer bRows.Close()

	for bRows.Next() {
		var boardID, boardName, boardDesc string
		if err := bRows.Scan(&boardID, &boardName, &boardDesc); err != nil {
			http.Error(w, "scan board: "+err.Error(), http.StatusInternalServerError)
			return
		}
		f.SetSheetRow("Boards", fmt.Sprintf("A%d", bRow), &[]interface{}{boardName, boardDesc})
		bRow++

		// ---- big tasks ----
		btRows, err := h.db.Query(ctx,
			`SELECT id, name, start_date::text, deadline::text, on_hold
			 FROM big_tasks WHERE board_id = $1 ORDER BY created_at`, boardID)
		if err != nil {
			http.Error(w, "db error big_tasks: "+err.Error(), http.StatusInternalServerError)
			return
		}

		for btRows.Next() {
			var btID, btName, btStart, btDeadline string
			var btOnHold bool
			if err := btRows.Scan(&btID, &btName, &btStart, &btDeadline, &btOnHold); err != nil {
				btRows.Close()
				http.Error(w, "scan big_task: "+err.Error(), http.StatusInternalServerError)
				return
			}
			onHoldStr := "Tidak"
			if btOnHold {
				onHoldStr = "Ya"
			}

			// members (comma-separated emails)
			var emails []string
			mRows, err := h.db.Query(ctx,
				`SELECT COALESCE(u.email,'')
				 FROM users u JOIN big_task_members m ON u.id = m.user_id
				 WHERE m.big_task_id = $1`, btID)
			if err != nil {
				btRows.Close()
				http.Error(w, "db error members: "+err.Error(), http.StatusInternalServerError)
				return
			}
			for mRows.Next() {
				var email string
				if err := mRows.Scan(&email); err != nil {
					mRows.Close()
					btRows.Close()
					http.Error(w, "scan member: "+err.Error(), http.StatusInternalServerError)
					return
				}
				emails = append(emails, email)
			}
			mRows.Close()

			f.SetSheetRow("BigTasks", fmt.Sprintf("A%d", btRow), &[]interface{}{
				boardName, btName, btStart, btDeadline, onHoldStr, strings.Join(emails, ", "),
			})
			btRow++

			// ---- daily tasks (non-review) ----
			dtRows, err := h.db.Query(ctx,
				`SELECT dt.id, dt.title, dt.start_date::text, dt.end_date::text,
				        COALESCE(u.email,'')
				 FROM daily_tasks dt
				 LEFT JOIN users u ON dt.pic_user_id = u.id
				 WHERE dt.big_task_id = $1 AND dt.review_of_daily_task_id IS NULL
				 ORDER BY dt.created_at`, btID)
			if err != nil {
				btRows.Close()
				http.Error(w, "db error daily_tasks: "+err.Error(), http.StatusInternalServerError)
				return
			}

			for dtRows.Next() {
				var dtID, dtTitle, dtStart, dtEnd, picEmail string
				if err := dtRows.Scan(&dtID, &dtTitle, &dtStart, &dtEnd, &picEmail); err != nil {
					dtRows.Close()
					btRows.Close()
					http.Error(w, "scan daily_task: "+err.Error(), http.StatusInternalServerError)
					return
				}

				f.SetSheetRow("DailyTasks", fmt.Sprintf("A%d", dtRow), &[]interface{}{
					boardName, btName, dtTitle, picEmail, dtStart, dtEnd,
				})
				dtRow++

				// ---- day entries ----
				deRows, err := h.db.Query(ctx,
					`SELECT entry_date::text, COALESCE(planned_text,''), progress_pct, COALESCE(blocker_text,'')
					 FROM day_entries WHERE daily_task_id = $1 ORDER BY entry_date, created_at`, dtID)
				if err != nil {
					dtRows.Close()
					btRows.Close()
					http.Error(w, "db error day_entries: "+err.Error(), http.StatusInternalServerError)
					return
				}
				for deRows.Next() {
					var entryDate, plannedText, blockerText string
					var progressPct int
					if err := deRows.Scan(&entryDate, &plannedText, &progressPct, &blockerText); err != nil {
						deRows.Close()
						dtRows.Close()
						btRows.Close()
						http.Error(w, "scan day_entry: "+err.Error(), http.StatusInternalServerError)
						return
					}
					f.SetSheetRow("DayEntries", fmt.Sprintf("A%d", deRow), &[]interface{}{
						boardName, btName, dtTitle, entryDate, plannedText, progressPct, blockerText,
					})
					deRow++
				}
				deRows.Close()
			}
			dtRows.Close()
		}
		btRows.Close()
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		http.Error(w, "excel write error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="rndops-export.xlsx"`)
	w.Write(buf.Bytes()) //nolint:errcheck
}

// Import handles POST /admin/import — inserts data from an uploaded XLSX file.
// Accepts multipart/form-data with a "file" field containing the xlsx workbook.
// Returns JSON importResult with counts and any warnings.
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if !auth.IsSuperUser(ctx) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

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

	// ---- build email → user ID map ----
	emailToUID := map[string]string{}
	uRows, err := h.db.Query(ctx, `SELECT id, email FROM users`)
	if err != nil {
		http.Error(w, "db error users: "+err.Error(), http.StatusInternalServerError)
		return
	}
	for uRows.Next() {
		var uid, email string
		if err := uRows.Scan(&uid, &email); err != nil {
			uRows.Close()
			http.Error(w, "scan users: "+err.Error(), http.StatusInternalServerError)
			return
		}
		emailToUID[email] = uid
	}
	uRows.Close()

	// ---- read all sheets (skip header row 0) ----
	boardRows, _ := f.GetRows("Boards")
	btSheetRows, _ := f.GetRows("BigTasks")
	dtSheetRows, _ := f.GetRows("DailyTasks")
	deSheetRows, _ := f.GetRows("DayEntries")

	tx, err := h.db.Begin(ctx)
	if err != nil {
		http.Error(w, "begin tx: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	result := importResult{Warnings: []string{}}

	// ---- Boards sheet ----
	// boardName → inserted boardID
	boardIDs := map[string]string{}
	for _, row := range skip1(boardRows) {
		if len(row) < 1 || row[0] == "" {
			continue
		}
		name := row[0]
		desc := ""
		if len(row) > 1 {
			desc = row[1]
		}
		var boardID string
		if err := tx.QueryRow(ctx,
			`INSERT INTO boards (name, description) VALUES ($1, $2) RETURNING id`,
			name, desc).Scan(&boardID); err != nil {
			http.Error(w, "insert board: "+err.Error(), http.StatusInternalServerError)
			return
		}
		boardIDs[name] = boardID
		result.BoardsCreated++
	}

	// ---- BigTasks sheet ----
	// key = boardName+"|"+btName → inserted btID
	btIDs := map[string]string{}
	for _, row := range skip1(btSheetRows) {
		if len(row) < 4 || row[0] == "" || row[1] == "" {
			continue
		}
		boardName, btName, startDate, deadline := row[0], row[1], row[2], row[3]
		onHold := len(row) > 4 && isYes(row[4])
		var memberEmails []string
		if len(row) > 5 && row[5] != "" {
			for _, e := range strings.Split(row[5], ",") {
				if em := strings.TrimSpace(e); em != "" {
					memberEmails = append(memberEmails, em)
				}
			}
		}

		boardID, ok := boardIDs[boardName]
		if !ok {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Big task '%s': board '%s' tidak ditemukan di file, di-skip", btName, boardName))
			continue
		}

		var btID string
		if err := tx.QueryRow(ctx,
			`INSERT INTO big_tasks (board_id, name, start_date, deadline, on_hold)
			 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			boardID, btName, startDate, deadline, onHold).Scan(&btID); err != nil {
			http.Error(w, "insert big_task: "+err.Error(), http.StatusInternalServerError)
			return
		}
		btIDs[boardName+"|"+btName] = btID
		result.BigTasksCreated++

		missed := 0
		for _, email := range memberEmails {
			uid, ok := emailToUID[email]
			if !ok {
				missed++
				continue
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO big_task_members (big_task_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				btID, uid); err != nil {
				http.Error(w, "insert member: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if missed > 0 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Big task '%s': %d anggota tidak ditemukan (tidak ada di environment ini)", btName, missed))
		}
	}

	// ---- DailyTasks sheet ----
	// key = boardName+"|"+btName+"|"+title → inserted dtID
	dtIDs := map[string]string{}
	for _, row := range skip1(dtSheetRows) {
		if len(row) < 6 || row[0] == "" || row[1] == "" || row[2] == "" {
			continue
		}
		boardName, btName, title, picEmail, startDate, endDate := row[0], row[1], row[2], row[3], row[4], row[5]
		btID, ok := btIDs[boardName+"|"+btName]
		if !ok {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Daily task '%s': big task '%s > %s' tidak ditemukan, di-skip", title, boardName, btName))
			continue
		}
		picUID, ok := emailToUID[picEmail]
		if !ok {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Daily task '%s' di big task '%s': PIC '%s' tidak ditemukan, di-skip", title, btName, picEmail))
			continue
		}
		var dtID string
		if err := tx.QueryRow(ctx,
			`INSERT INTO daily_tasks (big_task_id, title, pic_user_id, start_date, end_date)
			 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			btID, title, picUID, startDate, endDate).Scan(&dtID); err != nil {
			http.Error(w, "insert daily_task: "+err.Error(), http.StatusInternalServerError)
			return
		}
		dtIDs[boardName+"|"+btName+"|"+title] = dtID
		result.DailyTasksCreated++
	}

	// ---- DayEntries sheet ----
	for _, row := range skip1(deSheetRows) {
		if len(row) < 4 || row[0] == "" || row[1] == "" || row[2] == "" || row[3] == "" {
			continue
		}
		boardName, btName, dtTitle, entryDate := row[0], row[1], row[2], row[3]
		plannedText := cellStr(row, 4)
		progressPct := cellInt(row, 5)
		blockerText := cellStr(row, 6)

		dtID, ok := dtIDs[boardName+"|"+btName+"|"+dtTitle]
		if !ok {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Day entry %s: daily task '%s > %s > %s' tidak ditemukan, di-skip",
					entryDate, boardName, btName, dtTitle))
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO day_entries (daily_task_id, entry_date, planned_text, progress_pct, blocker_text)
			 VALUES ($1, $2, $3, $4, $5)`,
			dtID, entryDate, plannedText, progressPct, blockerText); err != nil {
			http.Error(w, "insert day_entry: "+err.Error(), http.StatusInternalServerError)
			return
		}
		result.DayEntriesCreated++
	}

	if err := tx.Commit(ctx); err != nil {
		http.Error(w, "commit: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// skip1 removes the header row from a sheet's rows slice.
func skip1(rows [][]string) [][]string {
	if len(rows) > 1 {
		return rows[1:]
	}
	return nil
}

// isYes returns true for "ya", "true", "1" (case-insensitive).
func isYes(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "ya" || s == "true" || s == "1"
}

// cellStr returns row[i] if it exists, else "".
func cellStr(row []string, i int) string {
	if len(row) > i {
		return row[i]
	}
	return ""
}

// cellInt parses row[i] as int, returning 0 on any error.
func cellInt(row []string, i int) int {
	if len(row) <= i {
		return 0
	}
	v, _ := strconv.Atoi(strings.TrimSpace(row[i]))
	return v
}
