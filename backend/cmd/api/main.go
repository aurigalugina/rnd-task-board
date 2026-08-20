package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"rndops/backend/internal/auth"
	"rndops/backend/internal/bigtask"
	"rndops/backend/internal/board"
	"rndops/backend/internal/changerequest"
	"rndops/backend/internal/chatproxy"
	"rndops/backend/internal/cheatsheet"
	"rndops/backend/internal/comment"
	"rndops/backend/internal/dailytask"
	"rndops/backend/internal/dataport"
	"rndops/backend/internal/notification"
	"rndops/backend/internal/referensi"
	"rndops/backend/internal/reviewqueue"
	"rndops/backend/internal/upload"
	"rndops/backend/internal/user"
	"rndops/backend/internal/weeklyplan"
)

func splitTrimmed(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://rndops:rndops@localhost:5432/rndops?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	allowedOrigins := []string{"http://localhost:5173", "http://localhost:8080"}
	if extra := os.Getenv("ALLOWED_ORIGINS"); extra != "" {
		for _, o := range splitTrimmed(extra) {
			allowedOrigins = append(allowedOrigins, o)
		}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/api/v1/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	authHandler := auth.NewHandler(pool)
	boardHandler := board.NewHandler(pool)
	bigTaskHandler := bigtask.NewHandler(pool)
	dailyTaskHandler := dailytask.NewHandler(pool)
	userHandler := user.NewHandler(pool)
	commentHandler := comment.NewHandler(pool)
	cheatSheetHandler := cheatsheet.NewHandler(pool)

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "/app/uploads"
	}
	uploadHandler := upload.NewHandler(uploadDir)
	weeklyPlanHandler := weeklyplan.NewHandler(pool, os.Getenv("MYAGENDA_SERVICE_URL"))
	reviewQueueHandler := reviewqueue.NewHandler(pool)
	referensiHandler := referensi.NewHandler(pool)
	changeRequestHandler := changerequest.NewHandler(pool)
	notificationHandler := notification.NewHandler(pool)
	dataportHandler := dataport.NewHandler(pool)

	// Background goroutine: kirim Telegram digest alert setiap 15 menit.
	// Goroutine ini mati kalau proses backend mati — acceptable buat internal tool.
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			notification.RunAlertJob(context.Background(), pool)
		}
	}()

	// chatProxy meneruskan /api/v1/chat/* ke claude-chat-service (Opsi B /
	// backend proxy). Auth diberlakukan di dalam proxy (header utk REST, query
	// ?access_token= utk WS) -- makanya dipasang DI LUAR grup protected.
	chatProxy := chatproxy.NewHandler(os.Getenv("CLAUDE_CHAT_SERVICE_URL"), os.Getenv("CHAT_DEFAULT_CWD"))

	r.Route("/api/v1", func(api chi.Router) {
		api.Post("/auth/login", authHandler.Login)
		api.Post("/auth/logout", authHandler.Logout)
		api.Post("/auth/refresh", authHandler.Refresh)

		// Chat proxy ke claude-chat-service (Vision §6). Auth internal ke proxy
		// (header/query) -- sengaja di luar grup protected yang cuma baca header.
		api.Handle("/chat/*", chatProxy)

		api.Group(func(protected chi.Router) {
			protected.Use(auth.RequireAuth)

			protected.Get("/boards", boardHandler.List)
			protected.Post("/boards", boardHandler.Create)
			protected.Get("/boards/{boardID}/summary", boardHandler.Summary)
			// Edit deskripsi/kategori/tim board -- super_user only, cek in-handler.
			// Lihat decision-log-boards-dashboard-enhancements-20260820.md.
			protected.Patch("/boards/{boardID}", boardHandler.Update)
			// Board Archive (super_user only) -- cek in-handler (bukan RequireRole,
			// pola sama seperti weekly-plan/team-status di bawah), lihat
			// decision-log-board-archive-20260812.md.
			protected.Get("/boards/archive", boardHandler.ListArchived)
			protected.Patch("/boards/{boardID}/archive", boardHandler.Archive)
			protected.Patch("/boards/{boardID}/unarchive", boardHandler.Unarchive)

			protected.Get("/boards/{boardID}/big-tasks", bigTaskHandler.ListByBoard)
			protected.Post("/boards/{boardID}/big-tasks", bigTaskHandler.Create)
			protected.Put("/big-tasks/{bigTaskID}/members", bigTaskHandler.SetMembers)
			// Edit judul/tanggal Big Task -- super_user only, cek in-handler (access_level
			// beda axis dari roles, pola sama seperti board archive/weekly-plan team-status).
			// Lihat decision-log-verdict-backfill-signoff-20260820.md.
			protected.Patch("/big-tasks/{bigTaskID}", bigTaskHandler.Update)

			protected.Group(func(spvOnly chi.Router) {
				spvOnly.Use(auth.RequireRole("spv"))
				spvOnly.Post("/big-tasks/{bigTaskID}/sign-off", bigTaskHandler.SignOff)
				spvOnly.Delete("/big-tasks/{bigTaskID}/sign-off", bigTaskHandler.UndoSignOff)
				spvOnly.Patch("/big-tasks/{bigTaskID}/on-hold", bigTaskHandler.ToggleOnHold)
			})

			// Review Queue: semua user login boleh akses, hasilnya di-filter per
			// user di handler (reviewer yang di-assign, atau fallback spv kalau
			// big task-nya belum di-assign reviewer sama sekali) -- lihat
			// decision-log-bigtask-reviewer-assignment-20260810.md.
			protected.Get("/review-queue", reviewQueueHandler.List)
			protected.Post("/review-queue/{itemType}/{itemID}/mark-reviewed", reviewQueueHandler.MarkReviewed)

			protected.Get("/big-tasks/{bigTaskID}/daily-tasks", dailyTaskHandler.ListByBigTask)
			protected.Post("/big-tasks/{bigTaskID}/daily-tasks", dailyTaskHandler.Create)
			protected.Post("/daily-tasks/{dailyTaskID}/clone-review", dailyTaskHandler.CloneReview)
			protected.Post("/daily-tasks/{dailyTaskID}/day-entries", dailyTaskHandler.AddDayEntry)
			protected.Patch("/day-entries/{dayEntryID}", dailyTaskHandler.UpdateDayEntry)
			protected.Delete("/day-entries/{dayEntryID}", dailyTaskHandler.DeleteDayEntry)

			protected.Get("/big-tasks/{bigTaskID}/comments", commentHandler.List)
			protected.Post("/big-tasks/{bigTaskID}/comments", commentHandler.Create)

			protected.Get("/boards/{boardID}/cheat-sheet", cheatSheetHandler.List)
			protected.Post("/boards/{boardID}/cheat-sheet", cheatSheetHandler.Create)
			protected.Patch("/boards/{boardID}/cheat-sheet/{itemID}", cheatSheetHandler.Update)
			protected.Delete("/boards/{boardID}/cheat-sheet/{itemID}", cheatSheetHandler.Delete)

			protected.Post("/uploads", uploadHandler.Create)
			protected.Get("/uploads/{filename}", uploadHandler.Serve)

			protected.Get("/weekly-plan", weeklyPlanHandler.List)
			protected.Post("/weekly-plan/push", weeklyPlanHandler.Push)
			// TeamStatus/List/Push semua cek super_user sendiri di handler (403
			// kalau bukan) -- bukan digate lewat RequireRole, karena access_level
			// itu konsep terpisah dari roles many-to-many (lihat
			// decision-log-hr-mapping-super-user-20260810.md).
			protected.Get("/weekly-plan/team-status", weeklyPlanHandler.TeamStatus)

			// Change Request (Vision §6): submit & list dibuka ke semua user
			// login; triase (PATCH) cuma spv + sa. Lihat
			// decision-log-change-request-integration-20260811.md.
			protected.Get("/change-requests", changeRequestHandler.List)
			protected.Post("/change-requests", changeRequestHandler.Create)
			protected.Group(func(triage chi.Router) {
				triage.Use(auth.RequireRole("spv", "sa"))
				triage.Patch("/change-requests/{id}", changeRequestHandler.Update)
			})

			// Data import/export (super_user only — cek in-handler).
			protected.Get("/admin/export", dataportHandler.Export)
			protected.Post("/admin/import", dataportHandler.Import)

			protected.Get("/notifications", notificationHandler.ListAlerts)
			protected.Get("/notifications/settings", notificationHandler.GetSettings)
			protected.Patch("/notifications/settings", notificationHandler.UpdateSettings)

			protected.Get("/users/me", userHandler.Me)
			protected.Patch("/users/me", userHandler.UpdateMe)
			protected.Get("/users/assignable", userHandler.ListAssignable)
			protected.Get("/users/progress-summary", userHandler.ProgressSummary)
			protected.Get("/roles", userHandler.ListRoles)
			// GET /referensi-tim dibuka ke SEMUA user login (bukan admin/spv only) --
			// cuma { id, name }, gak sensitif, dan dipakai Dashboard buat filter tim
			// (super_user belum tentu punya role admin/spv). Mutasi (Create/Update/
			// Delete) TETAP admin/spv only di grup adminOnly. Lihat
			// decision-log-boards-dashboard-enhancements-20260820.md.
			protected.Get("/referensi-tim", referensiHandler.ListTim)

			protected.Group(func(adminOnly chi.Router) {
				adminOnly.Use(auth.RequireRole("admin", "spv"))
				adminOnly.Get("/users", userHandler.List)
				adminOnly.Post("/users", userHandler.Create)
				adminOnly.Patch("/users/{userID}", userHandler.Update)
				adminOnly.Post("/referensi-tim", referensiHandler.CreateTim)
				adminOnly.Patch("/referensi-tim/{id}", referensiHandler.UpdateTim)
				adminOnly.Delete("/referensi-tim/{id}", referensiHandler.DeleteTim)
				adminOnly.Get("/referensi-user-hr", referensiHandler.ListUserHR)

				adminOnly.Get("/app-config/{key}", notificationHandler.GetAppConfig)
				adminOnly.Put("/app-config/{key}", notificationHandler.SetAppConfig)

				adminOnly.Post("/roles", userHandler.CreateRole)
				adminOnly.Patch("/roles/{code}", userHandler.UpdateRole)
				adminOnly.Delete("/roles/{code}", userHandler.DeleteRole)
			})
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	log.Printf("rnd-ops backend listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
