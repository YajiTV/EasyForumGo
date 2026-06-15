package handler

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
	"EasyForumGo/pkg/utils"
	"EasyForumGo/pkg/validator"
)

type SettingsHandler struct {
	users          *repository.UserRepository
	sessions       *repository.SessionRepository
	renderer       *PageRenderer
	follows        *repository.FollowRepository
	posts          *repository.PostRepository
	uploadDir      string
	maxUploadBytes int64
}

type SettingsPageData struct {
	User             *model.User
	HasLocalPassword bool
	Message          string
	Error            string
	ActiveSection    string
	SocialStats      model.SocialStats
}

// NewSettingsHandler creates a new instance
func NewSettingsHandler(db *sql.DB, uploadDir string, maxUploadBytes int64, renderer *PageRenderer) *SettingsHandler {
	return &SettingsHandler{
		users:          repository.NewUserRepository(db),
		sessions:       repository.NewSessionRepository(db),
		renderer:       renderer,
		follows:        repository.NewFollowRepository(db),
		posts:          repository.NewPostRepository(db),
		uploadDir:      uploadDir,
		maxUploadBytes: maxUploadBytes,
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
		SocialStats:      h.socialStats(user.ID),
	})
}

// UpdateProfile updates the current user's public name and picture
func (h *SettingsHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, utils.MaxMultipartBodySize(h.maxUploadBytes))
	if err := r.ParseMultipartForm(h.maxUploadBytes); err != nil {
		h.renderError(w, user, "profile", "Fichier trop volumineux (max "+utils.UploadSizeLabel(h.maxUploadBytes)+").")
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	if validationErrors := validator.ValidateProfile(validator.ProfileInput{Username: username}); validationErrors.HasErrors() {
		h.renderError(w, user, "profile", firstValidationMessage(validationErrors))
		return
	}
	if existingUser, err := h.users.GetByUsername(username); err == nil && existingUser.ID != user.ID {
		h.renderError(w, user, "profile", "Ce nom d'utilisateur est déjà utilisé.")
		return
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("check profile username availability: %v", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	profilePicture := user.ProfilePicture
	newProfilePicture := ""
	file, header, err := r.FormFile("profile_picture")
	if err == nil {
		defer file.Close()
		filename, saveErr := utils.SaveUploadedImage(file, header, h.uploadDir, h.maxUploadBytes)
		if errors.Is(saveErr, utils.ErrInvalidMIME) || errors.Is(saveErr, utils.ErrInvalidImage) || errors.Is(saveErr, utils.ErrFileTooLarge) {
			h.renderError(w, user, "profile", saveErr.Error())
			return
		}
		if saveErr != nil {
			h.renderError(w, user, "profile", "La photo de profil n'a pas pu être enregistrée.")
			return
		}
		profilePicture = filename
		newProfilePicture = filename
	} else if !errors.Is(err, http.ErrMissingFile) {
		h.renderError(w, user, "profile", "La photo de profil n'a pas pu être lue.")
		return
	}

	if err := h.users.UpdateProfile(user.ID, username, profilePicture); err != nil {
		if newProfilePicture != "" {
			h.removeLocalProfilePicture(newProfilePicture)
		}
		log.Printf("update profile for user %q: %v", user.ID, err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	if newProfilePicture != "" && user.ProfilePicture != "" && !strings.HasPrefix(user.ProfilePicture, "http") {
		h.removeLocalProfilePicture(user.ProfilePicture)
	}
	http.Redirect(w, r, "/settings?status=profile-updated#profile", http.StatusSeeOther)
}

func (h *SettingsHandler) removeLocalProfilePicture(filename string) {
	if err := os.Remove(filepath.Join(h.uploadDir, filepath.Base(filename))); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("remove profile picture %q: %v", filename, err)
	}
}

// UpdateSocial updates the current user's social preferences
func (h *SettingsHandler) UpdateSocial(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	biography := strings.TrimSpace(r.FormValue("biography"))
	if validationErrors := validator.ValidateSocialProfile(validator.SocialProfileInput{Biography: biography}); validationErrors.HasErrors() {
		h.renderError(w, user, "social", firstValidationMessage(validationErrors))
		return
	}
	followsVisible := r.FormValue("follows_visible") == "on"
	if err := h.users.UpdateSocialProfile(user.ID, biography, followsVisible); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings?status=social-updated#social", http.StatusSeeOther)
}

// UpdateEmail updates the current user's email address
func (h *SettingsHandler) UpdateEmail(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if !hasLocalPassword(user) {
		h.renderError(w, user, "email", "L'adresse e-mail de ce compte est gérée par son fournisseur OAuth.")
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
		h.renderError(w, user, "password", "Ce compte utilise un fournisseur OAuth pour se connecter.")
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
		SocialStats:      h.socialStats(user.ID),
	})
}

// render renders the settings page
func (h *SettingsHandler) render(w http.ResponseWriter, data SettingsPageData) {
	h.renderer.Render(w, "settings/settings.html", data)
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
	case "profile-updated":
		return "Votre profil public a été mis à jour."
	case "email-updated":
		return "Votre adresse e-mail a été mise à jour."
	case "social-updated":
		return "Vos préférences sociales ont été mises à jour."
	default:
		return ""
	}
}

// socialStats returns a user's social counters
func (h *SettingsHandler) socialStats(userID string) model.SocialStats {
	postCount, _ := h.posts.CountByUserID(userID)
	followerCount, _ := h.follows.CountFollowers(userID)
	followingCount, _ := h.follows.CountFollowing(userID)
	return model.SocialStats{PostCount: postCount, FollowerCount: followerCount, FollowingCount: followingCount}
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
