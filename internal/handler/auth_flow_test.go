package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"ForumJS/internal/middleware"
	"ForumJS/pkg/utils"
)

func TestAuthenticationFlow(t *testing.T) {
	t.Setenv(sessionDurationEnvKey, "1")

	db := newAuthTestDB(t)
	authHandler := NewAuthHandler(db)

	signupResponse := httptest.NewRecorder()
	signupRequest := newFormRequest(http.MethodPost, "/signup", url.Values{
		"email":    {"new@example.com"},
		"username": {"newuser"},
		"password": {"password123"},
	})
	authHandler.Signup(signupResponse, signupRequest)

	if signupResponse.Code != http.StatusSeeOther {
		t.Fatalf("expected signup status %d, got %d", http.StatusSeeOther, signupResponse.Code)
	}

	var storedPassword string
	if err := db.QueryRow("SELECT password FROM users WHERE email = ?", "new@example.com").Scan(&storedPassword); err != nil {
		t.Fatalf("select stored password: %v", err)
	}
	if storedPassword == "password123" {
		t.Fatal("expected stored password to be hashed")
	}

	firstCookie := loginAndGetSessionCookie(t, authHandler)
	if sessionCount(t, db) != 1 {
		t.Fatalf("expected one active session after first login, got %d", sessionCount(t, db))
	}

	secondCookie := loginAndGetSessionCookie(t, authHandler)
	if sessionCount(t, db) != 1 {
		t.Fatalf("expected one active session after second login, got %d", sessionCount(t, db))
	}
	if firstCookie.Value == secondCookie.Value {
		t.Fatal("expected second login to rotate the session token")
	}

	authMiddleware := middleware.NewAuthMiddleware(db)
	protectedResponse := httptest.NewRecorder()
	protectedRequest := httptest.NewRequest(http.MethodGet, "/private", nil)
	protectedRequest.AddCookie(secondCookie)

	authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userID, ok := middleware.CurrentUserID(r); !ok || userID == "" {
			t.Fatal("expected current user id in request context")
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(protectedResponse, protectedRequest)

	if protectedResponse.Code != http.StatusNoContent {
		t.Fatalf("expected protected status %d, got %d", http.StatusNoContent, protectedResponse.Code)
	}

	logoutResponse := httptest.NewRecorder()
	logoutRequest := httptest.NewRequest(http.MethodPost, "/logout", nil)
	logoutRequest.AddCookie(secondCookie)
	authHandler.Logout(logoutResponse, logoutRequest)

	if logoutResponse.Code != http.StatusSeeOther {
		t.Fatalf("expected logout status %d, got %d", http.StatusSeeOther, logoutResponse.Code)
	}
	if sessionCount(t, db) != 0 {
		t.Fatalf("expected no active session after logout, got %d", sessionCount(t, db))
	}

	expiredCookieFound := false
	for _, cookie := range logoutResponse.Result().Cookies() {
		if cookie.Name == "session_token" && cookie.MaxAge < 0 {
			expiredCookieFound = true
		}
	}
	if !expiredCookieFound {
		t.Fatal("expected logout to expire the session cookie")
	}
}

func TestRequireAuthRejectsExpiredSession(t *testing.T) {
	db := newAuthTestDB(t)
	userID := insertAuthTestUser(t, db, "expired@example.com", "expireduser", "password123")
	sessionToken := utils.NewSessionToken()
	_, err := db.Exec(
		"INSERT INTO sessions (id, user_id, session_token, expires_at) VALUES (?, ?, ?, ?)",
		utils.NewUUID(),
		userID,
		sessionToken,
		time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		t.Fatalf("insert expired session: %v", err)
	}

	authMiddleware := middleware.NewAuthMiddleware(db)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	request.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})

	authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("expired session should not reach protected handler")
	})).ServeHTTP(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, response.Code)
	}
	if sessionCount(t, db) != 0 {
		t.Fatalf("expected expired session to be deleted, got %d sessions", sessionCount(t, db))
	}
}

func loginAndGetSessionCookie(t *testing.T, authHandler *AuthHandler) *http.Cookie {
	t.Helper()

	response := httptest.NewRecorder()
	request := newFormRequest(http.MethodPost, "/login", url.Values{
		"email":    {"new@example.com"},
		"password": {"password123"},
	})
	authHandler.Login(response, request)

	if response.Code != http.StatusSeeOther {
		t.Fatalf("expected login status %d, got %d", http.StatusSeeOther, response.Code)
	}

	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "session_token" && cookie.Value != "" {
			return cookie
		}
	}

	t.Fatal("expected session cookie")
	return nil
}

func sessionCount(t *testing.T, db *sql.DB) int {
	t.Helper()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count); err != nil {
		t.Fatalf("count sessions: %v", err)
	}

	return count
}
