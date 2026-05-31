package main

import (
	"ForumJS/config"
	"ForumJS/internal/handler"
	"ForumJS/internal/repository"
	"log"
	"net"
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
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir))))

	errorRenderer := handler.NewErrorRenderer(cfg.TemplatesDir)
	authHandler := handler.NewAuthHandler(db, cfg.SessionDuration, errorRenderer)

	mux.HandleFunc("GET /login", authHandler.ShowLoginForm)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("GET /register", authHandler.ShowRegisterForm)
	mux.HandleFunc("POST /register", authHandler.Signup)
	mux.HandleFunc("POST /logout", authHandler.Logout)

	postHandler := handler.NewPostHandler(db, cfg.UploadDir)
	mux.HandleFunc("GET /post/new", postHandler.ShowCreateForm)
	mux.HandleFunc("POST /post/new", postHandler.CreatePost)
	mux.HandleFunc("GET /post/{id}", postHandler.PostDetail)
	mux.HandleFunc("GET /post/{id}/edit", postHandler.ShowEditForm)
	mux.HandleFunc("POST /post/{id}/edit", postHandler.EditPost)
	mux.HandleFunc("POST /post/{id}/delete", postHandler.DeletePost)

	commentHandler := handler.NewCommentHandler(db)
	mux.HandleFunc("POST /post/{id}/comment", commentHandler.CreateComment)
	mux.HandleFunc("POST /comment/{id}/delete", commentHandler.DeleteComment)
	mux.HandleFunc("GET /comment/{id}/edit", commentHandler.ShowEditForm)
	mux.HandleFunc("POST /comment/{id}/edit", commentHandler.EditComment)

	likeHandler := handler.NewLikeHandler(db)
	mux.HandleFunc("POST /post/{id}/like", likeHandler.LikePost)
	mux.HandleFunc("POST /post/{id}/dislike", likeHandler.DislikePost)
	mux.HandleFunc("POST /comment/{id}/like", likeHandler.LikeComment)
	mux.HandleFunc("POST /comment/{id}/dislike", likeHandler.DislikeComment)

	homeHandler := handler.NewHomeHandler(db, errorRenderer)
	mux.HandleFunc("/", homeHandler.Home)

	if cfg.TLSEnabled() {
		go func() {
			log.Printf("HTTP server starting on :%s (redirecting to HTTPS)", cfg.Port)
			if err := http.ListenAndServe(":"+cfg.Port, httpsRedirectHandler(cfg.HTTPSPort)); err != nil {
				log.Fatalf("HTTP redirect server failed: %v", err)
			}
		}()
		log.Printf("HTTPS server starting on :%s", cfg.HTTPSPort)
		log.Fatal(http.ListenAndServeTLS(":"+cfg.HTTPSPort, cfg.TLSCertFile, cfg.TLSKeyFile, mux))
	} else {
		log.Printf("Server starting on :%s", cfg.Port)
		log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
	}
}

func httpsRedirectHandler(httpsPort string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if httpsPort != "443" {
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = net.JoinHostPort(h, httpsPort)
			} else {
				host = net.JoinHostPort(host, httpsPort)
			}
		}
		http.Redirect(w, r, "https://"+host+r.URL.RequestURI(), http.StatusMovedPermanently)
	})
}
