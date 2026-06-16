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
func setupRouter(cfg config.Config, db *sql.DB, globalLimiter, loginLimiter, writeLimiter *middleware.RateLimiter) *http.ServeMux {
	mux := http.NewServeMux()

	// static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.StaticDir))))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir))))

	// theme routes
	themeHandler := handler.NewThemeHandler()
	mux.HandleFunc("GET /theme.css", themeHandler.CSS)
	mux.HandleFunc("GET /theme/{theme}", themeHandler.Set)

	// shared renderer : injecte notifCount dans tous les templates via FuncMap
	notifRepo := repository.NewNotificationRepository(db)
	renderer := handler.NewPageRenderer(notifRepo)

	authMiddleware := middleware.NewAuthMiddleware(db)

	// error renderer
	errorRenderer := handler.NewErrorRenderer(cfg.TemplatesDir)
	errorRenderer.SetAuthRepositories(repository.NewSessionRepository(db), repository.NewUserRepository(db))
	errorRenderer.SetRenderer(renderer)

	tooManyRequests := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errorRenderer.TooManyRequests(w, r)
	})
	globalLimiter.WithErrorHandler(tooManyRequests)
	loginLimiter.WithErrorHandler(tooManyRequests)
	writeLimiter.WithErrorHandler(tooManyRequests)

	// authentication routes
	authHandler := handler.NewAuthHandler(db, cfg.SessionDuration, errorRenderer, renderer)
	mux.HandleFunc("GET /login", authHandler.ShowLoginForm)
	mux.Handle("POST /login", loginLimiter.Wrap(http.HandlerFunc(authHandler.Login)))
	mux.HandleFunc("GET /register", authHandler.ShowRegisterForm)
	mux.Handle("POST /register", loginLimiter.Wrap(http.HandlerFunc(authHandler.Signup)))
	mux.HandleFunc("GET /register/password-strength", authHandler.PasswordStrength)
	mux.Handle("POST /register/password-strength", loginLimiter.Wrap(http.HandlerFunc(authHandler.PasswordStrength)))
	mux.Handle("POST /logout", writeLimiter.Wrap(http.HandlerFunc(authHandler.Logout)))

	// profile routes
	profileHandler := handler.NewProfileHandler(db, renderer)
	mux.HandleFunc("GET /profile", profileHandler.MyPosts)
	mux.HandleFunc("GET /profile/my-posts", profileHandler.MyPosts)
	mux.HandleFunc("GET /profile/liked-posts", profileHandler.LikedPosts)
	mux.HandleFunc("GET /profile/my-comments", profileHandler.MyComments)
	mux.HandleFunc("GET /profile/activity", profileHandler.Activity)

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
	settingsHandler := handler.NewSettingsHandler(db, cfg.UploadDir, cfg.MaxUploadBytes, renderer)
	mux.HandleFunc("GET /settings", settingsHandler.Show)
	mux.Handle("POST /settings/profile", writeLimiter.Wrap(http.HandlerFunc(settingsHandler.UpdateProfile)))
	mux.Handle("POST /settings/email", writeLimiter.Wrap(http.HandlerFunc(settingsHandler.UpdateEmail)))
	mux.Handle("POST /settings/password", writeLimiter.Wrap(http.HandlerFunc(settingsHandler.UpdatePassword)))
	mux.Handle("POST /settings/social", writeLimiter.Wrap(http.HandlerFunc(settingsHandler.UpdateSocial)))
	mux.Handle("POST /settings/delete", writeLimiter.Wrap(http.HandlerFunc(settingsHandler.DeleteAccount)))

	// library routes
	libraryHandler := handler.NewLibraryHandler(db, renderer)
	mux.HandleFunc("GET /library", libraryHandler.Index)
	mux.Handle("POST /library", writeLimiter.Wrap(http.HandlerFunc(libraryHandler.Create)))
	mux.HandleFunc("GET /library/{id}", libraryHandler.Show)
	mux.Handle("POST /library/{id}/rename", writeLimiter.Wrap(http.HandlerFunc(libraryHandler.Rename)))
	mux.Handle("POST /library/{id}/delete", writeLimiter.Wrap(http.HandlerFunc(libraryHandler.Delete)))
	mux.Handle("POST /library/{libraryID}/post/{postID}/remove", writeLimiter.Wrap(http.HandlerFunc(libraryHandler.RemovePost)))
	mux.Handle("POST /post/{postID}/library", writeLimiter.Wrap(http.HandlerFunc(libraryHandler.AddPost)))

	// post routes
	postHandler := handler.NewPostHandler(db, cfg.UploadDir, cfg.MaxUploadBytes, renderer)
	mux.HandleFunc("GET /post/new", postHandler.ShowCreateForm)
	mux.Handle("POST /post/new", writeLimiter.Wrap(authMiddleware.RequireNotMuted(http.HandlerFunc(postHandler.CreatePost))))
	mux.HandleFunc("GET /post/{id}/edit", postHandler.ShowEditForm)
	mux.Handle("POST /post/{id}/edit", writeLimiter.Wrap(http.HandlerFunc(postHandler.EditPost)))
	mux.HandleFunc("GET /post/{id}/delete", postHandler.ShowDeleteConfirmation)
	mux.Handle("POST /post/{id}/delete", writeLimiter.Wrap(http.HandlerFunc(postHandler.DeletePost)))
	mux.HandleFunc("GET /post/{id}", postHandler.PostDetail)

	// comment routes
	commentHandler := handler.NewCommentHandler(db, renderer)
	mux.Handle("POST /post/{id}/comment", writeLimiter.Wrap(authMiddleware.RequireNotMuted(http.HandlerFunc(commentHandler.CreateComment))))
	mux.HandleFunc("GET /comment/{id}/delete", commentHandler.ShowDeleteConfirmation)
	mux.Handle("POST /comment/{id}/delete", writeLimiter.Wrap(http.HandlerFunc(commentHandler.DeleteComment)))
	mux.HandleFunc("GET /comment/{id}/edit", commentHandler.ShowEditForm)
	mux.Handle("POST /comment/{id}/edit", writeLimiter.Wrap(http.HandlerFunc(commentHandler.EditComment)))

	// vote routes
	likeHandler := handler.NewLikeHandler(db)
	mux.Handle("POST /post/{id}/like", writeLimiter.Wrap(http.HandlerFunc(likeHandler.LikePost)))
	mux.Handle("POST /post/{id}/dislike", writeLimiter.Wrap(http.HandlerFunc(likeHandler.DislikePost)))
	mux.Handle("POST /comment/{id}/like", writeLimiter.Wrap(http.HandlerFunc(likeHandler.LikeComment)))
	mux.Handle("POST /comment/{id}/dislike", writeLimiter.Wrap(http.HandlerFunc(likeHandler.DislikeComment)))

	// static page routes
	pageHandler := handler.NewPageHandler(db, errorRenderer, renderer)
	mux.HandleFunc("GET /about", pageHandler.Page("pages/about.html"))
	mux.HandleFunc("GET /rules", pageHandler.Page("pages/rules.html"))
	mux.HandleFunc("GET /help", pageHandler.Page("pages/help.html"))
	mux.HandleFunc("GET /legal", pageHandler.Page("pages/legal.html"))
	mux.HandleFunc("GET /privacy", pageHandler.Page("pages/privacy.html"))
	mux.HandleFunc("GET /terms", pageHandler.Page("pages/terms.html"))
	mux.HandleFunc("GET /cookies", pageHandler.Page("pages/cookies.html"))
	mux.HandleFunc("GET /contact", pageHandler.Page("pages/contact.html"))

	// OAuth routes
	oauthHandler := handler.NewOAuthHandler(db, handler.OAuthConfig{
		GoogleClientID:     cfg.OAuthClientID,
		GoogleClientSecret: cfg.OAuthClientSecret,
		GoogleRedirectURL:  cfg.OAuthRedirectURL("google"),
		GitHubClientID:     cfg.GitHubOAuthID,
		GitHubClientSecret: cfg.GitHubOAuthSecret,
		GitHubRedirectURL:  cfg.OAuthRedirectURL("github"),
	}, cfg.SessionDuration, errorRenderer, renderer)
	mux.HandleFunc("GET /auth/google", oauthHandler.GoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", oauthHandler.GoogleCallback)
	mux.HandleFunc("GET /auth/github", oauthHandler.GitHubLogin)
	mux.HandleFunc("GET /auth/github/callback", oauthHandler.GitHubCallback)
	mux.HandleFunc("GET /auth/complete-profile", oauthHandler.ShowCompleteProfile)
	mux.Handle("POST /auth/complete-profile", writeLimiter.Wrap(http.HandlerFunc(oauthHandler.CompleteProfile)))

	// notification routes
	notifHandler := handler.NewNotificationHandler(db, notifRepo, renderer)
	mux.HandleFunc("GET /notifications", notifHandler.ShowNotifications)

	// home route
	homeHandler := handler.NewHomeHandler(db, errorRenderer, renderer)
	mux.HandleFunc("/", homeHandler.Home)

	moderationHandler := handler.NewModerationHandler(db)
	mux.Handle("POST /post/{id}/report", writeLimiter.Wrap(authMiddleware.RequireAuth(http.HandlerFunc(moderationHandler.ReportPost))))
	mux.Handle("POST /comment/{id}/report", writeLimiter.Wrap(authMiddleware.RequireAuth(http.HandlerFunc(moderationHandler.ReportComment))))
	mux.Handle("POST /user/{id}/report", writeLimiter.Wrap(authMiddleware.RequireAuth(http.HandlerFunc(moderationHandler.ReportUser))))
	mux.Handle("GET /moderation", authMiddleware.RequireRole(model.RoleModerator, http.HandlerFunc(moderationHandler.Dashboard)))
	mux.Handle("GET /moderation/queue", authMiddleware.RequireRole(model.RoleModerator, http.HandlerFunc(moderationHandler.PendingQueue)))
	mux.Handle("POST /moderation/content/{id}/approve", authMiddleware.RequireRole(model.RoleModerator, http.HandlerFunc(moderationHandler.ApproveContent)))
	mux.Handle("POST /moderation/content/{id}/reject", authMiddleware.RequireRole(model.RoleModerator, http.HandlerFunc(moderationHandler.RejectContent)))
	mux.Handle("POST /moderation/report/{id}/resolve", writeLimiter.Wrap(authMiddleware.RequireRole(model.RoleModerator, http.HandlerFunc(moderationHandler.ResolveReport))))
	mux.Handle("POST /moderation/report/{id}/dismiss", writeLimiter.Wrap(authMiddleware.RequireRole(model.RoleModerator, http.HandlerFunc(moderationHandler.DismissReport))))
	mux.Handle("POST /admin/user/{id}/role", writeLimiter.Wrap(authMiddleware.RequireRole(model.RoleAdmin, http.HandlerFunc(moderationHandler.ChangeUserRole))))
	mux.Handle("POST /admin/user/{id}/ban", writeLimiter.Wrap(authMiddleware.RequireRole(model.RoleAdmin, http.HandlerFunc(moderationHandler.BanUser))))
	mux.Handle("POST /admin/user/{id}/mute", writeLimiter.Wrap(authMiddleware.RequireRole(model.RoleModerator, http.HandlerFunc(moderationHandler.MuteUser))))
	mux.Handle("POST /admin/user/{id}/unban", writeLimiter.Wrap(authMiddleware.RequireRole(model.RoleModerator, http.HandlerFunc(moderationHandler.UnbanUser))))
	mux.Handle("POST /admin/user/{id}/kick", writeLimiter.Wrap(authMiddleware.RequireRole(model.RoleModerator, http.HandlerFunc(moderationHandler.KickUser))))

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
