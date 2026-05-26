package main

import (
	"ForumJS/internal/handler"
	"ForumJS/internal/repository"
	"log"
	"net/http"
	"os"
)

func main() {
	db, err := repository.InitDB("./migrations")
	if err != nil {
		log.Fatalf("DB init failed: %v", err)
	}
	defer db.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	authHandler := handler.NewAuthHandler(db)

	mux.HandleFunc("POST /signup", authHandler.Signup)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("POST /logout", authHandler.Logout)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Forum is running"))
	})

	log.Printf("Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
