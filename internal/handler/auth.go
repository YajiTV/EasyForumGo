package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"ForumJS/pkg/utils"
	"ForumJS/pkg/validator"
)

const (
	sessionCookieName = "session_token"
)

type AuthHandler struct {
	db              *sql.DB
	sessionDuration time.Duration
	errors          *ErrorRenderer
}

type userCredentials struct {
	id             string
	hashedPassword string
}

func NewAuthHandler(db *sql.DB, sessionDuration time.Duration, errors *ErrorRenderer) *AuthHandler {
	if sessionDuration <= 0 {
		sessionDuration = 24 * time.Hour
	}
	if errors == nil {
		errors = NewErrorRenderer("web/templates")
	}

	return &AuthHandler{
		db:              db,
		sessionDuration: sessionDuration,
		errors:          errors,
	}
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errors.MethodNotAllowed(w, "Méthode non autorisée.")
		return
	}

	input := validator.AuthInput{
		Email:    strings.TrimSpace(strings.ToLower(r.FormValue("email"))),
		Username: strings.TrimSpace(r.FormValue("username")),
		Password: r.FormValue("password"),
	}

	if validationErrors := validator.ValidateSignup(input); validationErrors.HasErrors() {
		h.writeValidationError(w, validationErrors)
		return
	}

	exists, err := h.userExists(input.Email, input.Username)
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}
	if exists {
		h.errors.Render(w, http.StatusConflict, "Cette adresse e-mail ou ce nom d'utilisateur est déjà utilisé.")
		return
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	if err := h.createUser(input.Email, input.Username, hashedPassword); err != nil {
		h.errors.InternalServerError(w)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errors.MethodNotAllowed(w, "Méthode non autorisée.")
		return
	}

	input := validator.AuthInput{
		Email:    strings.TrimSpace(strings.ToLower(r.FormValue("email"))),
		Password: r.FormValue("password"),
	}

	if validationErrors := validator.ValidateLogin(input); validationErrors.HasErrors() {
		h.writeValidationError(w, validationErrors)
		return
	}

	user, err := h.findUserCredentials(input.Email)
	if errors.Is(err, sql.ErrNoRows) {
		h.errors.Unauthorized(w, "Identifiants invalides.")
		return
	}
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	if !utils.CheckPasswordHash(input.Password, user.hashedPassword) {
		h.errors.Unauthorized(w, "Identifiants invalides.")
		return
	}

	sessionToken, expiresAt, err := h.createSession(user.id)
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errors.MethodNotAllowed(w, "Méthode non autorisée.")
		return
	}

	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		_, _ = h.db.Exec("DELETE FROM sessions WHERE session_token = ?", cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) userExists(email, username string) (bool, error) {
	var id string
	err := h.db.QueryRow(
		"SELECT id FROM users WHERE email = ? OR username = ? LIMIT 1",
		email,
		username,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (h *AuthHandler) findUserCredentials(email string) (userCredentials, error) {
	var user userCredentials
	err := h.db.QueryRow(
		"SELECT id, password FROM users WHERE email = ? LIMIT 1",
		email,
	).Scan(&user.id, &user.hashedPassword)
	return user, err
}

func (h *AuthHandler) createSession(userID string) (string, time.Time, error) {
	sessionToken := utils.NewSessionToken()
	expiresAt := time.Now().Add(h.sessionDuration)

	tx, err := h.db.Begin()
	if err != nil {
		return "", time.Time{}, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM sessions WHERE user_id = ?", userID); err != nil {
		return "", time.Time{}, err
	}

	_, err = tx.Exec(
		"INSERT INTO sessions (id, user_id, session_token, expires_at) VALUES (?, ?, ?, ?)",
		utils.NewUUID(),
		userID,
		sessionToken,
		expiresAt.UTC(),
	)
	if err != nil {
		return "", time.Time{}, err
	}

	if err := tx.Commit(); err != nil {
		return "", time.Time{}, err
	}

	return sessionToken, expiresAt, nil
}

func (h *AuthHandler) createUser(email, username, hashedPassword string) error {
	_, err := h.db.Exec(
		"INSERT INTO users (id, email, username, password) VALUES (?, ?, ?, ?)",
		utils.NewUUID(),
		email,
		username,
		hashedPassword,
	)
	return err
}

func (h *AuthHandler) writeValidationError(w http.ResponseWriter, validationErrors validator.ValidationErrors) {
	for _, message := range validationErrors {
		h.errors.BadRequest(w, message)
		return
	}

	h.errors.BadRequest(w, "Formulaire invalide.")
}
