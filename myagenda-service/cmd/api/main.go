package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/go-sql-driver/mysql"

	"myagenda-service/internal/myagenda"
)

// Sub-project TERPISAH dari rnd-task-board -- dummy/lokal buat sekarang,
// production-nya nanti dipasang langsung di server aplikasi HR (di luar
// repo ini). Lihat docs/decision-log/decision-log-myagenda-hr-service-20260810.md
// di repo rnd-task-board untuk konteks lengkap.
func main() {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = "myagenda:myagenda@tcp(localhost:3307)/myagenda?parseTime=true"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("failed to open mysql connection: %v", err)
	}
	defer db.Close()

	// MySQL container butuh waktu inisialisasi (termasuk restart internal
	// sekali buat apply MYSQL_DATABASE/MYSQL_USER) -- `depends_on` compose
	// cuma nunggu container START, bukan READY, jadi ping pertama gampang
	// gagal saat startup bareng. Retry singkat daripada Fatal langsung.
	var pingErr error
	for i := 0; i < 15; i++ {
		if pingErr = db.Ping(); pingErr == nil {
			break
		}
		log.Printf("mysql belum siap (percobaan %d/15): %v", i+1, pingErr)
		time.Sleep(2 * time.Second)
	}
	if pingErr != nil {
		log.Fatalf("mysql ping failed setelah beberapa percobaan: %v", pingErr)
	}

	handler := myagenda.NewHandler(db)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	r.Post("/my-agenda", handler.Upsert)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	log.Printf("myagenda-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
