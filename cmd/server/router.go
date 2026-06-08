package main

import (
	"ForumJS/config"
	"ForumJS/internal/handler"
	"ForumJS/internal/middleware"
	"ForumJS/internal/repository"
	"database/sql"
	"net"
	"net/http"
)

// setupRouter configures the application routes
func setupRouter(cfg config.Config, db *sql.DB, loginLimiter, writeLimiter *middleware.RateLimiter) *http.ServeMux {
	mux := http.NewServeMux()

	// static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticDir))))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir))))

	// authentication routes
	errorRenderer := handler.NewErrorRenderer(cfg.TemplatesDir)
	errorRenderer.SetAuthRepositories(repository.NewSessionRepository(db), repository.NewUserRepository(db))
	authHandler := handler.NewAuthHandler(db, cfg.SessionDuration, errorRenderer)
	mux.HandleFunc("GET /login", authHandler.ShowLoginForm)
	mux.Handle("POST /login", loginLimiter.Wrap(http.HandlerFunc(authHandler.Login)))
	mux.HandleFunc("GET /register", authHandler.ShowRegisterForm)
	mux.HandleFunc("POST /register", authHandler.Signup)
	mux.HandleFunc("GET /register/password-strength", authHandler.PasswordStrength)
	mux.HandleFunc("POST /register/password-strength", authHandler.PasswordStrength)
	mux.HandleFunc("POST /logout", authHandler.Logout)

	// profile routes
	profileHandler := handler.NewProfileHandler(db, cfg.UploadDir)
	mux.HandleFunc("GET /profile", profileHandler.MyPosts)
	mux.HandleFunc("GET /profile/my-posts", profileHandler.MyPosts)
	mux.HandleFunc("GET /profile/liked-posts", profileHandler.LikedPosts)
	mux.HandleFunc("GET /profile/my-comments", profileHandler.MyComments)
	mux.HandleFunc("GET /profile/edit", profileHandler.ShowEditForm)
	mux.HandleFunc("POST /profile/edit", profileHandler.UpdateProfile)

	// post routes
	postHandler := handler.NewPostHandler(db, cfg.UploadDir)
	mux.HandleFunc("GET /post/new", postHandler.ShowCreateForm)
	mux.Handle("POST /post/new", writeLimiter.Wrap(http.HandlerFunc(postHandler.CreatePost)))
	mux.HandleFunc("GET /post/{id}/edit", postHandler.ShowEditForm)
	mux.HandleFunc("POST /post/{id}/edit", postHandler.EditPost)
	mux.HandleFunc("GET /post/{id}/delete", postHandler.ShowDeleteConfirmation)
	mux.HandleFunc("POST /post/{id}/delete", postHandler.DeletePost)
	mux.HandleFunc("GET /post/{id}", postHandler.PostDetail)

	// comment routes
	commentHandler := handler.NewCommentHandler(db)
	mux.Handle("POST /post/{id}/comment", writeLimiter.Wrap(http.HandlerFunc(commentHandler.CreateComment)))
	mux.HandleFunc("GET /comment/{id}/delete", commentHandler.ShowDeleteConfirmation)
	mux.HandleFunc("POST /comment/{id}/delete", commentHandler.DeleteComment)
	mux.HandleFunc("GET /comment/{id}/edit", commentHandler.ShowEditForm)
	mux.HandleFunc("POST /comment/{id}/edit", commentHandler.EditComment)

	// vote routes
	likeHandler := handler.NewLikeHandler(db)
	mux.HandleFunc("POST /post/{id}/like", likeHandler.LikePost)
	mux.HandleFunc("POST /post/{id}/dislike", likeHandler.DislikePost)
	mux.HandleFunc("POST /comment/{id}/like", likeHandler.LikeComment)
	mux.HandleFunc("POST /comment/{id}/dislike", likeHandler.DislikeComment)

	// category routes
	categoryHandler := handler.NewCategoryHandler(db)
	mux.HandleFunc("GET /posts/category/{id}", categoryHandler.FilterByCategory)

	// static page routes
	pageHandler := handler.NewPageHandler(db, cfg.TemplatesDir, errorRenderer)
	mux.HandleFunc("GET /about", pageHandler.Page("about.html"))
	mux.HandleFunc("GET /rules", pageHandler.Page("rules.html"))
	mux.HandleFunc("GET /help", pageHandler.Page("help.html"))
	mux.HandleFunc("GET /legal", pageHandler.Page("legal.html"))
	mux.HandleFunc("GET /privacy", pageHandler.Page("privacy.html"))
	mux.HandleFunc("GET /terms", pageHandler.Page("terms.html"))
	mux.HandleFunc("GET /cookies", pageHandler.Page("cookies.html"))
	mux.HandleFunc("GET /contact", pageHandler.Page("contact.html"))

	// OAuth routes
	redirectURL := "http://localhost:" + cfg.Port + "/auth/google/callback"
	oauthHandler := handler.NewOAuthHandler(db, cfg.OAuthClientID, cfg.OAuthClientSecret, redirectURL, cfg.SessionDuration, errorRenderer)
	mux.HandleFunc("GET /auth/google", oauthHandler.GoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", oauthHandler.GoogleCallback)
	mux.HandleFunc("GET /auth/complete-profile", oauthHandler.ShowCompleteProfile)
	mux.HandleFunc("POST /auth/complete-profile", oauthHandler.CompleteProfile)

	// home route
	homeHandler := handler.NewHomeHandler(db, errorRenderer)
	mux.HandleFunc("/", homeHandler.Home)

	return mux
}

// httpsRedirectHandler redirects requests to https
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
