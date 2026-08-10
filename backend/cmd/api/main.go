package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"rndops/backend/internal/auth"
	"rndops/backend/internal/bigtask"
	"rndops/backend/internal/board"
	"rndops/backend/internal/cheatsheet"
	"rndops/backend/internal/comment"
	"rndops/backend/internal/dailytask"
	"rndops/backend/internal/reviewqueue"
	"rndops/backend/internal/upload"
	"rndops/backend/internal/user"
	"rndops/backend/internal/weeklyplan"
)

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
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:8080"},
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
	weeklyPlanHandler := weeklyplan.NewHandler(pool)
	reviewQueueHandler := reviewqueue.NewHandler(pool)

	r.Route("/api/v1", func(api chi.Router) {
		api.Post("/auth/login", authHandler.Login)
		api.Post("/auth/logout", authHandler.Logout)
		api.Post("/auth/refresh", authHandler.Refresh)

		api.Group(func(protected chi.Router) {
			protected.Use(auth.RequireAuth)

			protected.Get("/boards", boardHandler.List)
			protected.Post("/boards", boardHandler.Create)
			protected.Get("/boards/{boardID}/summary", boardHandler.Summary)

			protected.Get("/boards/{boardID}/big-tasks", bigTaskHandler.ListByBoard)
			protected.Post("/boards/{boardID}/big-tasks", bigTaskHandler.Create)

			protected.Group(func(spvOnly chi.Router) {
				spvOnly.Use(auth.RequireRole("spv"))
				spvOnly.Post("/big-tasks/{bigTaskID}/sign-off", bigTaskHandler.SignOff)
				spvOnly.Delete("/big-tasks/{bigTaskID}/sign-off", bigTaskHandler.UndoSignOff)

				spvOnly.Get("/review-queue", reviewQueueHandler.List)
				spvOnly.Post("/review-queue/{itemType}/{itemID}/mark-reviewed", reviewQueueHandler.MarkReviewed)
			})

			protected.Get("/big-tasks/{bigTaskID}/daily-tasks", dailyTaskHandler.ListByBigTask)
			protected.Post("/big-tasks/{bigTaskID}/daily-tasks", dailyTaskHandler.Create)
			protected.Post("/daily-tasks/{dailyTaskID}/clone-review", dailyTaskHandler.CloneReview)
			protected.Patch("/day-entries/{dayEntryID}", dailyTaskHandler.UpdateDayEntry)

			protected.Get("/big-tasks/{bigTaskID}/comments", commentHandler.List)
			protected.Post("/big-tasks/{bigTaskID}/comments", commentHandler.Create)

			protected.Get("/boards/{boardID}/cheat-sheet", cheatSheetHandler.List)
			protected.Post("/boards/{boardID}/cheat-sheet", cheatSheetHandler.Create)

			protected.Post("/uploads", uploadHandler.Create)
			protected.Get("/uploads/{filename}", uploadHandler.Serve)

			protected.Get("/weekly-plan", weeklyPlanHandler.List)
			protected.Post("/weekly-plan/push", weeklyPlanHandler.Push)

			protected.Get("/users/me", userHandler.Me)
			protected.Patch("/users/me", userHandler.UpdateMe)
			protected.Get("/users/assignable", userHandler.ListAssignable)
			protected.Get("/roles", userHandler.ListRoles)

			protected.Group(func(adminOnly chi.Router) {
				adminOnly.Use(auth.RequireRole("admin", "spv"))
				adminOnly.Get("/users", userHandler.List)
				adminOnly.Post("/users", userHandler.Create)
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
