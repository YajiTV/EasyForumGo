package handler

import (
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
	"EasyForumGo/pkg/utils"
	"EasyForumGo/pkg/validator"
)

type SettingsHandler struct {
	users    *repository.UserRepository
	sessions *repository.SessionRepository
}

type SettingsPageData struct {
	User             *model.User
	HasLocalPassword bool
	Message          string
	Error            string
	ActiveSection    string
}

// NewSettingsHandler creates a new instance
func NewSettingsHandler(db *sql.DB) *SettingsHandler {
	return &SettingsHandler{
		users:    repository.NewUserRepository(db),
		sessions: repository.NewSessionRepository(db),
	}
}

// Show renders the settings page
func (h *SettingsHandler) Show(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	h.render(w, SettingsPageData{
		User:             user,
		HasLocalPassword: hasLocalPassword(user),
		Message:          settingsMessage(r.URL.Query().Get("status")),
	})
}

// UpdateEmail updates the current user's email address
func (h *SettingsHandler) UpdateEmail(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if !hasLocalPassword(user) {
		h.renderError(w, user, "email", "L'adresse e-mail de ce compte est gérée par Google.")
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	currentPassword := r.FormValue("current_password")
	if validationErrors := validator.ValidateEmailChange(email); validationErrors.HasErrors() {
		h.renderError(w, user, "email", firstValidationMessage(validationErrors))
		return
	}
	if !utils.CheckPasswordHash(currentPassword, user.Password) {
		h.renderError(w, user, "email", "Le mot de passe actuel est incorrect.")
		return
	}
	if existingUser, err := h.users.GetByEmail(email); err == nil && existingUser.ID != user.ID {
		h.renderError(w, user, "email", "Cette adresse e-mail est déjà utilisée.")
		return
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	if err := h.users.UpdateEmail(user.ID, email); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings?status=email-updated", http.StatusSeeOther)
}

// UpdatePassword updates the current user's password
func (h *SettingsHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if !hasLocalPassword(user) {
		h.renderError(w, user, "password", "Ce compte utilise Google pour se connecter.")
		return
	}

	currentPassword := r.FormValue("current_password")
	newPassword := r.FormValue("new_password")
	if !utils.CheckPasswordHash(currentPassword, user.Password) {
		h.renderError(w, user, "password", "Le mot de passe actuel est incorrect.")
		return
	}
	if utils.CheckPasswordHash(newPassword, user.Password) {
		h.renderError(w, user, "password", "Le nouveau mot de passe doit être différent de l'ancien.")
		return
	}
	if validationErrors := validator.ValidatePasswordChange(validator.PasswordChangeInput{
		Username:        user.Username,
		Email:           user.Email,
		Password:        newPassword,
		ConfirmPassword: r.FormValue("confirm_password"),
	}); validationErrors.HasErrors() {
		h.renderError(w, user, "password", firstValidationMessage(validationErrors))
		return
	}
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	if err := h.users.UpdatePasswordAndDeleteSessions(user.ID, hashedPassword); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	clearSettingsSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// DeleteAccount deletes the current user's account
func (h *SettingsHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if strings.TrimSpace(r.FormValue("confirmation")) != "SUPPRIMER" {
		h.renderError(w, user, "delete", "Saisissez SUPPRIMER pour confirmer.")
		return
	}
	if hasLocalPassword(user) && !utils.CheckPasswordHash(r.FormValue("current_password"), user.Password) {
		h.renderError(w, user, "delete", "Le mot de passe actuel est incorrect.")
		return
	}
	if err := h.users.Delete(user.ID); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	clearSettingsSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// renderError renders a settings section error
func (h *SettingsHandler) renderError(w http.ResponseWriter, user *model.User, section, message string) {
	h.render(w, SettingsPageData{
		User:             user,
		HasLocalPassword: hasLocalPassword(user),
		Error:            message,
		ActiveSection:    section,
	})
}

// render renders the settings page
func (h *SettingsHandler) render(w http.ResponseWriter, data SettingsPageData) {
	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "settings", "settings.html"),
	)
	if err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
	}
}

// userFromSession gets the user from the current session
func (h *SettingsHandler) userFromSession(r *http.Request) *model.User {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}
	session, err := h.sessions.GetByToken(cookie.Value)
	if err != nil || session.ExpiresAt.Before(time.Now()) {
		return nil
	}
	user, err := h.users.GetByID(session.UserID)
	if err != nil {
		return nil
	}
	return user
}

// hasLocalPassword reports whether a user can authenticate with a password
func hasLocalPassword(user *model.User) bool {
	return user != nil && !strings.HasPrefix(user.Password, "$oauth$")
}

// settingsMessage returns a safe confirmation message
func settingsMessage(status string) string {
	switch status {
	case "email-updated":
		return "Votre adresse e-mail a été mise à jour."
	default:
		return ""
	}
}

// clearSettingsSessionCookie clears the current session cookie
func clearSettingsSessionCookie(w http.ResponseWriter) {
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
