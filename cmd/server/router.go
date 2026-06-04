package main

import (
	"ForumJS/config"
	"ForumJS/internal/handler"
	"ForumJS/internal/repository"
	"database/sql"
	"html/template"
	"net/http"
	"path/filepath"
)

func setupRouter(cfg config.Config, db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()

	//statics
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticDir))))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir))))

	//Auth
	errorRenderer := handler.NewErrorRenderer(cfg.TemplatesDir)
	errorRenderer.SetAuthRepositories(repository.NewSessionRepository(db), repository.NewUserRepository(db))
	authHandler := handler.NewAuthHandler(db, cfg.SessionDuration, errorRenderer)
	mux.HandleFunc("GET /login", authHandler.ShowLoginForm)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.HandleFunc("GET /register", authHandler.ShowRegisterForm)
	mux.HandleFunc("POST /register", authHandler.Signup)
	mux.HandleFunc("GET /register/password-strength", authHandler.PasswordStrength)
	mux.HandleFunc("POST /register/password-strength", authHandler.PasswordStrength)
	mux.HandleFunc("POST /logout", authHandler.Logout)

	//Profile
	profileHandler := handler.NewProfileHandler(db, cfg.UploadDir)
	mux.HandleFunc("GET /profile", profileHandler.MyPosts)
	mux.HandleFunc("GET /profile/my-posts", profileHandler.MyPosts)
	mux.HandleFunc("GET /profile/liked-posts", profileHandler.LikedPosts)
	mux.HandleFunc("GET /profile/my-comments", profileHandler.MyComments)
	mux.HandleFunc("GET /profile/edit", profileHandler.ShowEditForm)
	mux.HandleFunc("POST /profile/edit", profileHandler.UpdateProfile)

	//Posts
	postHandler := handler.NewPostHandler(db, cfg.UploadDir)
	mux.HandleFunc("GET /post/new", postHandler.ShowCreateForm)
	mux.HandleFunc("POST /post/new", postHandler.CreatePost)
	mux.HandleFunc("GET /post/{id}/edit", postHandler.ShowEditForm)
	mux.HandleFunc("POST /post/{id}/edit", postHandler.EditPost)
	mux.HandleFunc("GET /post/{id}/delete", postHandler.ShowDeleteConfirmation)
	mux.HandleFunc("POST /post/{id}/delete", postHandler.DeletePost)
	mux.HandleFunc("GET /post/{id}", postHandler.PostDetail)

	//Commentaires
	commentHandler := handler.NewCommentHandler(db)
	mux.HandleFunc("POST /post/{id}/comment", commentHandler.CreateComment)
	mux.HandleFunc("GET /comment/{id}/delete", commentHandler.ShowDeleteConfirmation)
	mux.HandleFunc("POST /comment/{id}/delete", commentHandler.DeleteComment)
	mux.HandleFunc("GET /comment/{id}/edit", commentHandler.ShowEditForm)
	mux.HandleFunc("POST /comment/{id}/edit", commentHandler.EditComment)

	//Likes
	likeHandler := handler.NewLikeHandler(db)
	mux.HandleFunc("POST /post/{id}/like", likeHandler.LikePost)
	mux.HandleFunc("POST /post/{id}/dislike", likeHandler.DislikePost)
	mux.HandleFunc("POST /comment/{id}/like", likeHandler.LikeComment)
	mux.HandleFunc("POST /comment/{id}/dislike", likeHandler.DislikeComment)

	//Catégories
	categoryHandler := handler.NewCategoryHandler(db)
	mux.HandleFunc("GET /posts/category/{id}", categoryHandler.FilterByCategory)

	//Pages statiques
	mux.HandleFunc("GET /about", staticPage(cfg.TemplatesDir, "about.html"))
	mux.HandleFunc("GET /rules", staticPage(cfg.TemplatesDir, "rules.html"))
	mux.HandleFunc("GET /legal", staticPage(cfg.TemplatesDir, "legal.html"))
	mux.HandleFunc("GET /cookies", staticPage(cfg.TemplatesDir, "cookies.html"))

	//Accueil
	homeHandler := handler.NewHomeHandler(db, errorRenderer)
	mux.HandleFunc("/", homeHandler.Home)

	return mux
}

// staticPage
func staticPage(templatesDir, filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			filepath.Join(templatesDir, "layout", "base.html"),
			filepath.Join(templatesDir, filename),
		)
		if err != nil {
			http.Error(w, "Erreur template", http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "base", nil)
	}
}
