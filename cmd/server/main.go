package main

import (
	"ForumJS/config"
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

	mux := setupRouter(cfg, db)

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
