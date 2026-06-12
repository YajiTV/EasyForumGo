package handler

import (
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"EasyForumGo/pkg/utils"
	"EasyForumGo/pkg/validator"
)

const (
	sessionCookieName = "session_token"
)

type AuthHandler struct {
	db              *sql.DB
	sessionDuration time.Duration
	errors          *ErrorRenderer
	renderer        *PageRenderer
}

type userCredentials struct {
	id             string
	hashedPassword string
}

type loginFormData struct {
	User  any
	Email string
	Error string
}

type registerFormData struct {
	User             any
	Username         string
	Email            string
	Error            string
	PasswordStrength validator.PasswordStrength
	PasswordChecked  bool
}

// NewAuthHandler creates a new instance
func NewAuthHandler(db *sql.DB, sessionDuration time.Duration, errors *ErrorRenderer, renderer *PageRenderer) *AuthHandler {
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
		renderer:        renderer,
	}
}

// ShowLoginForm renders the requested page
func (h *AuthHandler) ShowLoginForm(w http.ResponseWriter, r *http.Request) {
	renderAuthTemplate(h.renderer, w, "login.html", nil)
}

// ShowRegisterForm renders the requested page
func (h *AuthHandler) ShowRegisterForm(w http.ResponseWriter, r *http.Request) {
	renderAuthTemplate(h.renderer, w, "register.html", registerFormData{
		PasswordStrength: validator.EvaluatePasswordStrength("", "", ""),
	})
}

// PasswordStrength handles the request
func (h *AuthHandler) PasswordStrength(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		h.errors.MethodNotAllowed(w, "Méthode non autorisée.")
		return
	}

	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'self'; form-action 'self'; base-uri 'none'")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("Cache-Control", "no-store")

	input := validator.AuthInput{}
	checked := false
	if r.Method == http.MethodPost {
		checked = true
		input = validator.AuthInput{
			Email:    strings.TrimSpace(strings.ToLower(r.FormValue("email"))),
			Username: strings.TrimSpace(r.FormValue("username")),
			Password: r.FormValue("password"),
		}
	}

	data := registerFormData{
		Username:         input.Username,
		Email:            input.Email,
		PasswordStrength: validator.EvaluatePasswordStrength(input.Password, input.Username, input.Email),
		PasswordChecked:  checked,
	}

	tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "auth", "password_strength.html"))
	if err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
}

// Signup handles the request
func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errors.MethodNotAllowed(w, "Méthode non autorisée.")
		return
	}

	input := validator.AuthInput{
		Email:           strings.TrimSpace(strings.ToLower(r.FormValue("email"))),
		Username:        strings.TrimSpace(r.FormValue("username")),
		Password:        r.FormValue("password"),
		ConfirmPassword: r.FormValue("confirm_password"),
	}

	if validationErrors := validator.ValidateSignup(input); validationErrors.HasErrors() {
		h.renderRegisterError(w, input.Username, input.Email, firstValidationMessage(validationErrors))
		return
	}

	exists, err := h.userExists(input.Email, input.Username)
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}
	if exists {
		h.renderRegisterError(w, input.Username, input.Email, "Cette adresse e-mail ou ce nom d'utilisateur est déjà utilisé.")
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

// Login handles the request
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
		h.renderLoginError(w, input.Email, firstValidationMessage(validationErrors))
		return
	}

	user, err := h.findUserCredentials(input.Email)
	if errors.Is(err, sql.ErrNoRows) {
		h.renderLoginError(w, input.Email, "Identifiants invalides.")
		return
	}
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	if !utils.CheckPasswordHash(input.Password, user.hashedPassword) {
		h.renderLoginError(w, input.Email, "Identifiants invalides.")
		return
	}

	if banned, err := h.isUserBanned(user.id); err != nil {
		h.errors.InternalServerError(w)
		return
	} else if banned {
		h.renderLoginError(w, input.Email, "Votre compte a été banni de la plateforme.")
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

// Logout handles the request
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

// userExists checks whether an account already exists
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

// findUserCredentials finds the requested data
func (h *AuthHandler) findUserCredentials(email string) (userCredentials, error) {
	var user userCredentials
	err := h.db.QueryRow(
		"SELECT id, password FROM users WHERE email = ? LIMIT 1",
		email,
	).Scan(&user.id, &user.hashedPassword)
	return user, err
}

// createSession replaces the current user session
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

// isUserBanned checks whether an active ban exists for the given user
func (h *AuthHandler) isUserBanned(userID string) (bool, error) {
	var count int
	err := h.db.QueryRow(
		`SELECT COUNT(*) FROM user_restrictions
		 WHERE user_id = ? AND type = 'ban'
		 AND (expires_at IS NULL OR expires_at > datetime('now'))`,
		userID,
	).Scan(&count)
	return count > 0, err
}

// createUser stores a new user account
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

// renderLoginError renders the requested page
func (h *AuthHandler) renderLoginError(w http.ResponseWriter, email, message string) {
	if message == "" {
		message = "Formulaire invalide."
	}
	renderAuthTemplate(h.renderer, w, "login.html", loginFormData{
		Email: email,
		Error: message,
	})
}

// renderRegisterError renders the requested page
func (h *AuthHandler) renderRegisterError(w http.ResponseWriter, username, email, message string) {
	if message == "" {
		message = "Formulaire invalide."
	}
	renderAuthTemplate(h.renderer, w, "register.html", registerFormData{
		Username:         username,
		Email:            email,
		Error:            message,
		PasswordStrength: validator.EvaluatePasswordStrength("", username, email),
	})
}

// firstValidationMessage gets the first validation message
func firstValidationMessage(validationErrors validator.ValidationErrors) string {
	for _, message := range validationErrors {
		return message
	}

	return "Formulaire invalide."
}

// renderAuthTemplate renders an authentication template
func renderAuthTemplate(renderer *PageRenderer, w http.ResponseWriter, filename string, data any) {
	renderer.Render(w, filepath.Join("auth", filename), data)
}
