package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"ForumJS/pkg/utils"

	_ "github.com/mattn/go-sqlite3"
)

func TestSignupRejectsInvalidInput(t *testing.T) {
	db := newAuthTestDB(t)
	authHandler := NewAuthHandler(db)

	response := httptest.NewRecorder()
	request := newFormRequest(http.MethodPost, "/signup", url.Values{
		"email":    {"bad-email"},
		"username": {"bo"},
		"password": {"short"},
	})

	authHandler.Signup(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func TestSignupRejectsDuplicateUser(t *testing.T) {
	db := newAuthTestDB(t)
	authHandler := NewAuthHandler(db)
	insertAuthTestUser(t, db, "used@example.com", "usedname", "password123")

	response := httptest.NewRecorder()
	request := newFormRequest(http.MethodPost, "/signup", url.Values{
		"email":    {"used@example.com"},
		"username": {"newname"},
		"password": {"password123"},
	})

	authHandler.Signup(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d", http.StatusConflict, response.Code)
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	db := newAuthTestDB(t)
	authHandler := NewAuthHandler(db)
	insertAuthTestUser(t, db, "user@example.com", "username", "password123")

	response := httptest.NewRecorder()
	request := newFormRequest(http.MethodPost, "/login", url.Values{
		"email":    {"user@example.com"},
		"password": {"wrong-password"},
	})

	authHandler.Login(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func newAuthTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	schema := `
	CREATE TABLE users (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		username TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL UNIQUE,
		session_token TEXT NOT NULL UNIQUE,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return db
}

func insertAuthTestUser(t *testing.T, db *sql.DB, email, username, password string) string {
	t.Helper()

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userID := utils.NewUUID()
	_, err = db.Exec(
		"INSERT INTO users (id, email, username, password) VALUES (?, ?, ?, ?)",
		userID,
		email,
		username,
		hashedPassword,
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	return userID
}

func newFormRequest(method, target string, values url.Values) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return request
}
