package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

const sessionCookieName = "session_token"

type contextKey string

const currentUserIDKey contextKey = "current_user_id"

type AuthMiddleware struct {
	db *sql.DB
}

// NewAuthMiddleware creates a new instance
func NewAuthMiddleware(db *sql.DB) *AuthMiddleware {
	return &AuthMiddleware{db: db}
}

// RequireAuth protects routes from unauthenticated requests
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		userID, err := m.findValidSessionUserID(cookie.Value)
		if err != nil {
			clearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// make the user id available to protected handlers
		ctx := context.WithValue(r.Context(), currentUserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CurrentUserID gets the authenticated user id
func CurrentUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(currentUserIDKey).(string)
	return userID, ok && userID != ""
}

// findValidSessionUserID gets the user id from a valid session
func (m *AuthMiddleware) findValidSessionUserID(sessionToken string) (string, error) {
	var userID string
	var rawExpiresAt string
	err := m.db.QueryRow(
		"SELECT user_id, expires_at FROM sessions WHERE session_token = ? LIMIT 1",
		sessionToken,
	).Scan(&userID, &rawExpiresAt)
	if err != nil {
		return "", err
	}

	expiresAt, err := parseSQLiteTime(rawExpiresAt)
	if err != nil {
		return "", err
	}

	if !expiresAt.After(time.Now()) {
		// remove expired sessions before rejecting them
		_, _ = m.db.Exec("DELETE FROM sessions WHERE session_token = ?", sessionToken)
		return "", sql.ErrNoRows
	}

	return userID, nil
}

// parseSQLiteTime parses sqlite timestamp formats
func parseSQLiteTime(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}

	var lastErr error
	for _, layout := range layouts {
		parsedTime, err := time.Parse(layout, value)
		if err == nil {
			return parsedTime, nil
		}
		lastErr = err
	}

	return time.Time{}, lastErr
}

// clearSessionCookie expires the browser session cookie
func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
