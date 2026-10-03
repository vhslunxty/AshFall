package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Server struct {
	db *sql.DB
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s)/%s?parseTime=true&loc=UTC&charset=utf8mb4&collation=utf8mb4_unicode_ci",
		env("DB_USER", "survival"),
		env("DB_PASSWORD", "survival"),
		env("DB_HOST", "mysql:3306"),
		env("DB_NAME", "survival"),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	for i := 1; i <= 30; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		log.Printf("MySQL pas encore prêt (%d/30) : %v", i, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Impossible de joindre MySQL : %v", err)
	}
	log.Println("Connecté à MySQL ✅")

	s := &Server{db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /player/{id}/start", s.handleStart)
	mux.HandleFunc("GET /player/{id}", s.handleGet)
	mux.HandleFunc("POST /player/{id}/explore", s.handleExplore)
	mux.HandleFunc("POST /player/{id}/fight", s.handleFight)
	mux.HandleFunc("POST /player/{id}/eat", s.handleEat)
	mux.HandleFunc("GET /top", s.handleTop)

	addr := ":" + env("PORT", "8080")
	log.Printf("API en écoute sur %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
