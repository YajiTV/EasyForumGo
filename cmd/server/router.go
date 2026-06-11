package main

import (
	"EasyForumGo/config"
	"EasyForumGo/internal/handler"
	"EasyForumGo/internal/middleware"
	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
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

	// shared renderer : injecte notifCount dans tous les templates via FuncMap
	notifRepo := repository.NewNotificationRepository(db)
	renderer := handler.NewPageRenderer(notifRepo)

	// error renderer
	errorRenderer := handler.NewErrorRenderer(cfg.TemplatesDir)
	errorRenderer.SetAuthRepositories(repository.NewSessionRepository(db), repository.NewUserRepository(db))
	errorRenderer.SetRenderer(renderer)

	// authentication routes
	authHandler := handler.NewAuthHandler(db, cfg.SessionDuration, errorRenderer, renderer)
	mux.HandleFunc("GET /login", authHandler.ShowLoginForm)
	mux.Handle("POST /login", loginLimiter.Wrap(http.HandlerFunc(authHandler.Login)))
	mux.HandleFunc("GET /register", authHandler.ShowRegisterForm)
	mux.HandleFunc("POST /register", authHandler.Signup)
	mux.HandleFunc("GET /register/password-strength", authHandler.PasswordStrength)
	mux.HandleFunc("POST /register/password-strength", authHandler.PasswordStrength)
	mux.HandleFunc("POST /logout", authHandler.Logout)

	// profile routes
	profileHandler := handler.NewProfileHandler(db, cfg.UploadDir, renderer)
	mux.HandleFunc("GET /profile", profileHandler.MyPosts)
	mux.HandleFunc("GET /profile/my-posts", profileHandler.MyPosts)
	mux.HandleFunc("GET /profile/liked-posts", profileHandler.LikedPosts)
	mux.HandleFunc("GET /profile/my-comments", profileHandler.MyComments)
	mux.HandleFunc("GET /profile/activity", profileHandler.Activity)
	mux.HandleFunc("GET /profile/edit", profileHandler.ShowEditForm)
	mux.HandleFunc("POST /profile/edit", profileHandler.UpdateProfile)

	// social profile routes
	socialHandler := handler.NewSocialHandler(db, renderer)
	mux.HandleFunc("GET /user/{username}", socialHandler.PublicProfile)
	mux.HandleFunc("GET /user/{username}/followers", socialHandler.Followers)
	mux.HandleFunc("GET /user/{username}/following", socialHandler.Following)
	mux.Handle("POST /user/{username}/follow", writeLimiter.Wrap(http.HandlerFunc(socialHandler.Follow)))
	mux.Handle("POST /user/{username}/unfollow", writeLimiter.Wrap(http.HandlerFunc(socialHandler.Unfollow)))
	mux.HandleFunc("GET /discover", socialHandler.Discover)
	mux.HandleFunc("GET /search", socialHandler.Search)

	// settings routes
	settingsHandler := handler.NewSettingsHandler(db, renderer)
	mux.HandleFunc("GET /settings", settingsHandler.Show)
	mux.Handle("POST /settings/email", writeLimiter.Wrap(http.HandlerFunc(settingsHandler.UpdateEmail)))
	mux.Handle("POST /settings/password", writeLimiter.Wrap(http.HandlerFunc(settingsHandler.UpdatePassword)))
	mux.Handle("POST /settings/social", writeLimiter.Wrap(http.HandlerFunc(settingsHandler.UpdateSocial)))
	mux.Handle("POST /settings/delete", writeLimiter.Wrap(http.HandlerFunc(settingsHandler.DeleteAccount)))

	// library routes
	libraryHandler := handler.NewLibraryHandler(db, renderer)
	mux.HandleFunc("GET /library", libraryHandler.Index)
	mux.HandleFunc("POST /library", libraryHandler.Create)
	mux.HandleFunc("GET /library/{id}", libraryHandler.Show)
	mux.HandleFunc("POST /library/{id}/rename", libraryHandler.Rename)
	mux.HandleFunc("POST /library/{id}/delete", libraryHandler.Delete)
	mux.HandleFunc("POST /library/{libraryID}/post/{postID}/remove", libraryHandler.RemovePost)
	mux.HandleFunc("POST /post/{postID}/library", libraryHandler.AddPost)

	// post routes
	postHandler := handler.NewPostHandler(db, cfg.UploadDir, renderer)
	mux.HandleFunc("GET /post/new", postHandler.ShowCreateForm)
	mux.Handle("POST /post/new", writeLimiter.Wrap(http.HandlerFunc(postHandler.CreatePost)))
	mux.HandleFunc("GET /post/{id}/edit", postHandler.ShowEditForm)
	mux.HandleFunc("POST /post/{id}/edit", postHandler.EditPost)
	mux.HandleFunc("GET /post/{id}/delete", postHandler.ShowDeleteConfirmation)
	mux.HandleFunc("POST /post/{id}/delete", postHandler.DeletePost)
	mux.HandleFunc("GET /post/{id}", postHandler.PostDetail)

	// comment routes
	commentHandler := handler.NewCommentHandler(db, renderer)
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
	categoryHandler := handler.NewCategoryHandler(db, renderer)
	mux.HandleFunc("GET /posts/category/{id}", categoryHandler.FilterByCategory)

	// static page routes
	pageHandler := handler.NewPageHandler(db, errorRenderer, renderer)
	mux.HandleFunc("GET /about", pageHandler.Page("about.html"))
	mux.HandleFunc("GET /rules", pageHandler.Page("rules.html"))
	mux.HandleFunc("GET /help", pageHandler.Page("help.html"))
	mux.HandleFunc("GET /legal", pageHandler.Page("legal.html"))
	mux.HandleFunc("GET /privacy", pageHandler.Page("privacy.html"))
	mux.HandleFunc("GET /terms", pageHandler.Page("terms.html"))
	mux.HandleFunc("GET /cookies", pageHandler.Page("cookies.html"))
	mux.HandleFunc("GET /contact", pageHandler.Page("contact.html"))

	// OAuth routes
	redirectURL := cfg.OAuthRedirectURL()
	oauthHandler := handler.NewOAuthHandler(db, cfg.OAuthClientID, cfg.OAuthClientSecret, redirectURL, cfg.SessionDuration, errorRenderer, renderer)
	mux.HandleFunc("GET /auth/google", oauthHandler.GoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", oauthHandler.GoogleCallback)
	mux.HandleFunc("GET /auth/complete-profile", oauthHandler.ShowCompleteProfile)
	mux.HandleFunc("POST /auth/complete-profile", oauthHandler.CompleteProfile)

	// notification routes
	notifHandler := handler.NewNotificationHandler(db, notifRepo, renderer)
	mux.HandleFunc("GET /notifications", notifHandler.ShowNotifications)

	// home route
	homeHandler := handler.NewHomeHandler(db, errorRenderer, renderer)
	mux.HandleFunc("/", homeHandler.Home)

	authMiddleware := middleware.NewAuthMiddleware(db)
	moderationHandler := handler.NewModerationHandler(db)
	mux.Handle("POST /post/{id}/report", authMiddleware.RequireAuth(http.HandlerFunc(moderationHandler.ReportPost)))
	mux.Handle("POST /comment/{id}/report", authMiddleware.RequireAuth(http.HandlerFunc(moderationHandler.ReportComment)))
	mux.Handle("POST /user/{id}/report", authMiddleware.RequireAuth(http.HandlerFunc(moderationHandler.ReportUser)))
	mux.Handle("GET /moderation", authMiddleware.RequireRole(model.RoleModerator, http.HandlerFunc(moderationHandler.Dashboard)))
	mux.Handle("POST /moderation/report/{id}/resolve", authMiddleware.RequireRole(model.RoleModerator, http.HandlerFunc(moderationHandler.ResolveReport)))
	mux.Handle("POST /moderation/report/{id}/dismiss", authMiddleware.RequireRole(model.RoleModerator, http.HandlerFunc(moderationHandler.DismissReport)))
	mux.Handle("POST /admin/user/{id}/role", authMiddleware.RequireRole(model.RoleAdmin, http.HandlerFunc(moderationHandler.ChangeUserRole)))

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
