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
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	authHandler := handler.NewAuthHandler(db)

	mux.HandleFunc("GET /login", authHandler.ShowLoginForm)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("GET /register", authHandler.ShowRegisterForm)
	mux.HandleFunc("POST /register", authHandler.Signup)
	mux.HandleFunc("POST /logout", authHandler.Logout)

	profileHandler := handler.NewProfileHandler(db)
	mux.HandleFunc("GET /profile/liked-posts", profileHandler.LikedPosts)

	postHandler := handler.NewPostHandler(db)
	mux.HandleFunc("GET /post/new", postHandler.ShowCreateForm)
	mux.HandleFunc("POST /post/new", postHandler.CreatePost)
	mux.HandleFunc("GET /post/{id}", postHandler.PostDetail)

	homeHandler := handler.NewHomeHandler(db)
	mux.HandleFunc("/", homeHandler.Home)

	log.Printf("Server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
