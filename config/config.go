package config

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAppEnv               = "dev"
	defaultPort                 = "8080"
	defaultHTTPSPort            = "8443"
	defaultDBPath               = "./data/forum.db"
	defaultMigrationsDir        = "./migrations"
	defaultStaticDir            = "web/static"
	defaultTemplatesDir         = "web/templates"
	defaultUploadDir            = "./uploads"
	defaultMaxUploadMegabytes   = int64(20)
	defaultSessionDurationHours = 24
	defaultDBEncryptionKeyDev   = "Kx7!qP2$mR9#vN4@eL6w"
)

type Config struct {
	AppEnv            string
	Port              string
	HTTPSPort         string
	TLSCertFile       string
	TLSKeyFile        string
	DBPath            string
	DBEncryptionKey   string
	MigrationsDir     string
	StaticDir         string
	TemplatesDir      string
	UploadDir         string
	MaxUploadBytes    int64
	SessionDuration   time.Duration
	OAuthClientID     string
	OAuthClientSecret string
	GitHubOAuthID     string
	GitHubOAuthSecret string
	AppBasePath       string
	AppPublicURL      string
	TrustProxy        bool
}

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

// Load loads .env into the environment (if present), then reads configuration
func Load() Config {
	loadEnv()
	appEnv := strings.ToLower(strings.TrimSpace(stringEnv("APP_ENV", defaultAppEnv)))
	dbKeyDefault := ""
	if appEnv == "dev" {
		dbKeyDefault = defaultDBEncryptionKeyDev
	}
	return Config{
		AppEnv:            appEnv,
		Port:              stringEnv("PORT", defaultPort),
		HTTPSPort:         stringEnv("HTTPS_PORT", defaultHTTPSPort),
		TLSCertFile:       stringEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:        stringEnv("TLS_KEY_FILE", ""),
		DBPath:            stringEnv("DB_PATH", defaultDBPath),
		DBEncryptionKey:   stringEnv("DB_ENCRYPTION_KEY", dbKeyDefault),
		MigrationsDir:     stringEnv("MIGRATIONS_DIR", defaultMigrationsDir),
		StaticDir:         stringEnv("STATIC_DIR", defaultStaticDir),
		TemplatesDir:      stringEnv("TEMPLATES_DIR", defaultTemplatesDir),
		UploadDir:         stringEnv("UPLOAD_DIR", defaultUploadDir),
		MaxUploadBytes:    int64Env("MAX_UPLOAD_MB", defaultMaxUploadMegabytes) * 1024 * 1024,
		SessionDuration:   time.Duration(intEnv("SESSION_DURATION_H", defaultSessionDurationHours)) * time.Hour,
		OAuthClientID:     stringEnv("GOOGLE_OAUTH_ID", stringEnv("OAUTH_ID", "")),
		OAuthClientSecret: stringEnv("GOOGLE_OAUTH_KEY", stringEnv("OAUTH_KEY", "")),
		GitHubOAuthID:     stringEnv("GITHUB_OAUTH_ID", ""),
		GitHubOAuthSecret: stringEnv("GITHUB_OAUTH_KEY", ""),
		AppBasePath:       normalizeBasePath(stringEnv("APP_BASE_PATH", "")),
		AppPublicURL:      strings.TrimRight(strings.TrimSpace(stringEnv("APP_PUBLIC_URL", "")), "/"),
		TrustProxy:        boolEnv("TRUST_PROXY", false),
	}
}

// Validate checks whether the application configuration is coherent
func (c Config) Validate() error {
	if c.AppEnv != "dev" && c.AppEnv != "prod" {
		return fmt.Errorf("APP_ENV must be dev or prod")
	}
	if c.AppEnv == "prod" {
		if !strings.HasPrefix(strings.ToLower(c.AppPublicURL), "https://") {
			return fmt.Errorf("APP_PUBLIC_URL must use https in production")
		}
		if !c.TrustProxy {
			return fmt.Errorf("TRUST_PROXY must be true in production")
		}
		if c.DBEncryptionKey == "" {
			return fmt.Errorf("DB_ENCRYPTION_KEY must be set in production")
		}
	}
	return nil
}

// TLSEnabled checks whether tls is enabled
func (c Config) TLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != ""
}

// SecureCookies checks whether cookies must only be sent over https
func (c Config) SecureCookies() bool {
	return c.TLSEnabled() || strings.HasPrefix(strings.ToLower(c.AppPublicURL), "https://")
}

// OAuthRedirectURL builds a public oauth callback url
func (c Config) OAuthRedirectURL(provider string) string {
	if c.AppPublicURL != "" {
		return c.AppPublicURL + "/auth/" + provider + "/callback"
	}
	return "http://localhost:" + c.Port + path.Join(c.AppBasePath, "/auth/"+provider+"/callback")
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

// int64Env gets a 64 bit integer environment value
func int64Env(key string, fallback int64) int64 {
	value, err := strconv.ParseInt(os.Getenv(key), 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

// boolEnv gets a boolean environment value
func boolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

// normalizeBasePath normalizes an optional application url prefix
func normalizeBasePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "/" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.RawQuery != "" || parsed.Fragment != "" {
		return ""
	}
	return "/" + strings.Trim(path.Clean("/"+value), "/")
}
