package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"EasyForumGo/internal/model"
)

const sessionCookieName = "session_token"

type contextKey string

const (
	currentUserIDKey   contextKey = "current_user_id"
	currentUserRoleKey contextKey = "current_user_role"
)

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

		userID, role, err := m.findValidSession(cookie.Value)
		if err != nil {
			clearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), currentUserIDKey, userID)
		ctx = context.WithValue(ctx, currentUserRoleKey, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) RequireRole(role model.Role, next http.Handler) http.Handler {
	return m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userRole, _ := CurrentUserRole(r)
		var allowed bool
		switch role {
		case model.RoleAdmin:
			allowed = userRole == model.RoleAdmin
		case model.RoleModerator:
			allowed = userRole == model.RoleModerator || userRole == model.RoleAdmin
		default:
			allowed = true
		}
		if !allowed {
			http.Error(w, "Accès interdit", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func CurrentUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(currentUserIDKey).(string)
	return userID, ok && userID != ""
}

func CurrentUserRole(r *http.Request) (model.Role, bool) {
	role, ok := r.Context().Value(currentUserRoleKey).(model.Role)
	return role, ok && role != ""
}

func (m *AuthMiddleware) findValidSession(sessionToken string) (string, model.Role, error) {
	var userID string
	var role model.Role
	var rawExpiresAt string
	err := m.db.QueryRow(
		`SELECT s.user_id, u.role, s.expires_at
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.session_token = ?
		 LIMIT 1`,
		sessionToken,
	).Scan(&userID, &role, &rawExpiresAt)
	if err != nil {
		return "", "", err
	}

	expiresAt, err := parseSQLiteTime(rawExpiresAt)
	if err != nil {
		return "", "", err
	}

	if !expiresAt.After(time.Now()) {
		_, _ = m.db.Exec("DELETE FROM sessions WHERE session_token = ?", sessionToken)
		return "", "", sql.ErrNoRows
	}

	return userID, role, nil
}

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
