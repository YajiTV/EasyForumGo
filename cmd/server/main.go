package main

import (
	"EasyForumGo/config"
	"EasyForumGo/internal/middleware"
	"EasyForumGo/internal/repository"
	"log"
	"net/http"
	"time"
)

// main starts the application
func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	db, err := repository.InitDB(cfg.DBPath, cfg.MigrationsDir, cfg.DBEncryptionKey)
	if err != nil {
		log.Fatalf("DB init failed: %v", err)
	}
	defer db.Close()

	globalLimiter := middleware.NewSessionRateLimiter(db, cfg.TrustProxy, 200, time.Minute)
	loginLimiter := middleware.NewSessionRateLimiter(db, cfg.TrustProxy, 10, 15*time.Minute)
	writeLimiter := middleware.NewSessionRateLimiter(db, cfg.TrustProxy, 20, time.Hour)

	mux := setupRouter(cfg, db, loginLimiter, writeLimiter)
	var root http.Handler = mux
	root = middleware.BasePath(cfg.AppBasePath, cfg.SecureCookies(), root)
	root = globalLimiter.Wrap(root)
	root = middleware.ProxyHeaders(cfg.TrustProxy, root)

	if cfg.TLSEnabled() {
		go func() {
			log.Printf("HTTP server starting on :%s (redirecting to HTTPS)", cfg.Port)
			if err := http.ListenAndServe(":"+cfg.Port, httpsRedirectHandler(cfg.HTTPSPort)); err != nil {
				log.Fatalf("HTTP redirect server failed: %v", err)
			}
		}()
		log.Printf("HTTPS server starting on :%s", cfg.HTTPSPort)
		log.Fatal(http.ListenAndServeTLS(":"+cfg.HTTPSPort, cfg.TLSCertFile, cfg.TLSKeyFile, root))
	} else {
		log.Printf("Server starting on :%s", cfg.Port)
		log.Fatal(http.ListenAndServe(":"+cfg.Port, root))
	}
}
