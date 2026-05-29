package main

import (
	"ForumJS/config"
	"ForumJS/internal/repository"
	"log"
	"net/http"
)

func main() {
	cfg := config.Load()

	db, err := repository.InitDB(cfg.DBPath, cfg.MigrationsDir)
	if err != nil {
		log.Fatalf("DB init failed: %v", err)
	}
	defer db.Close()

	mux := setupRouter(cfg, db)

	log.Printf("Server starting on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}
