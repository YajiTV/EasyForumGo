package main

import (
	"ForumJS/config"
	"ForumJS/internal/handler"
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

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticDir))))
	errorRenderer := handler.NewErrorRenderer(cfg.TemplatesDir)
	authHandler := handler.NewAuthHandler(db, cfg.SessionDuration, errorRenderer)

	mux.HandleFunc("POST /signup", authHandler.Signup)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("POST /logout", authHandler.Logout)
	homeHandler := handler.NewHomeHandler(db, errorRenderer)
	mux.HandleFunc("/", homeHandler.Home)

	log.Printf("Server starting on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}
