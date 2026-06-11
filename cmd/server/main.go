package main

import (
	"EasyForumGo/config"
	"EasyForumGo/internal/middleware"
	"EasyForumGo/internal/repository"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// loadEnv loads variables from a .env file into the environment
func loadEnv() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if os.Getenv(key) == "" {
				os.Setenv(key, value)
			}
		}
	}
}

// main starts the application
func main() {
	loadEnv()
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	db, err := repository.InitDB(cfg.DBPath, cfg.MigrationsDir)
	if err != nil {
		log.Fatalf("DB init failed: %v", err)
	}
	defer db.Close()

	globalLimiter := middleware.NewRateLimiter(200, time.Minute)
	loginLimiter := middleware.NewRateLimiter(10, 15*time.Minute)
	writeLimiter := middleware.NewRateLimiter(20, time.Hour)

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
