package middleware

import (
	"context"
	"database/sql"
	"net/http"
)

const sessionCookieName = "session_token"

type contextKey string

const currentUserIDKey contextKey = "current_user_id"

type AuthMiddleware struct {
	db *sql.DB
}

func NewAuthMiddleware(db *sql.DB) *AuthMiddleware {
	return &AuthMiddleware{db: db}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		userID, err := m.findSessionUserID(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), currentUserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func CurrentUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(currentUserIDKey).(string)
	return userID, ok && userID != ""
}

func (m *AuthMiddleware) findSessionUserID(sessionToken string) (string, error) {
	var userID string
	err := m.db.QueryRow(
		"SELECT user_id FROM sessions WHERE session_token = ? LIMIT 1",
		sessionToken,
	).Scan(&userID)
	return userID, err
}
