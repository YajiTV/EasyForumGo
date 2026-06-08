package handler

import (
	"database/sql"
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	"ForumJS/internal/model"
	"ForumJS/internal/repository"
)

type PageHandler struct {
	templatesDir string
	sessions     *repository.SessionRepository
	users        *repository.UserRepository
	errors       *ErrorRenderer
}

type PageData struct {
	User *model.User
}

// NewPageHandler creates a new instance
func NewPageHandler(db *sql.DB, templatesDir string, errors *ErrorRenderer) *PageHandler {
	return &PageHandler{
		templatesDir: templatesDir,
		sessions:     repository.NewSessionRepository(db),
		users:        repository.NewUserRepository(db),
		errors:       errors,
	}
}

// Page returns a handler for an informational page
func (h *PageHandler) Page(filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			filepath.Join(h.templatesDir, "layout", "base.html"),
			filepath.Join(h.templatesDir, filename),
		)
		if err != nil {
			h.errors.RenderWithRequest(w, r, http.StatusInternalServerError, "Une erreur est survenue.")
			return
		}

		data := PageData{User: h.userFromSession(r)}
		if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
			h.errors.RenderWithRequest(w, r, http.StatusInternalServerError, "Une erreur est survenue.")
		}
	}
}

// userFromSession gets the user from the current session
func (h *PageHandler) userFromSession(r *http.Request) *model.User {
	cookie, err := r.Cookie("session_token")
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
