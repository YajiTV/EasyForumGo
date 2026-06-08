package config

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultPort                 = "8080"
	defaultHTTPSPort            = "8443"
	defaultDBPath               = "./data/forum.db"
	defaultMigrationsDir        = "./migrations"
	defaultStaticDir            = "web/static"
	defaultTemplatesDir         = "web/templates"
	defaultUploadDir            = "./uploads"
	defaultSessionDurationHours = 24
)

type Config struct {
	Port            string
	HTTPSPort       string
	TLSCertFile     string
	TLSKeyFile      string
	DBPath          string
	MigrationsDir   string
	StaticDir       string
	TemplatesDir    string
	UploadDir       string
	SessionDuration time.Duration
}

// Load loads the application configuration
func Load() Config {
	return Config{
		Port:            stringEnv("PORT", defaultPort),
		HTTPSPort:       stringEnv("HTTPS_PORT", defaultHTTPSPort),
		TLSCertFile:     stringEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:      stringEnv("TLS_KEY_FILE", ""),
		DBPath:          stringEnv("DB_PATH", defaultDBPath),
		MigrationsDir:   stringEnv("MIGRATIONS_DIR", defaultMigrationsDir),
		StaticDir:       stringEnv("STATIC_DIR", defaultStaticDir),
		TemplatesDir:    stringEnv("TEMPLATES_DIR", defaultTemplatesDir),
		UploadDir:       stringEnv("UPLOAD_DIR", defaultUploadDir),
		SessionDuration: time.Duration(intEnv("SESSION_DURATION_H", defaultSessionDurationHours)) * time.Hour,
	}
}

// TLSEnabled checks whether tls is enabled
func (c Config) TLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != ""
}

// stringEnv gets a string environment value
func stringEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// intEnv gets an integer environment value
func intEnv(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
