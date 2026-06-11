package handler

import (
	"database/sql"
	"net/http"
	"time"

	"ForumJS/internal/model"
	"ForumJS/internal/repository"
)

type PageHandler struct {
	sessions *repository.SessionRepository
	users    *repository.UserRepository
	errors   *ErrorRenderer
	renderer *PageRenderer
}

type PageData struct {
	User *model.User
}

// NewPageHandler creates a new instance
func NewPageHandler(db *sql.DB, errors *ErrorRenderer, renderer *PageRenderer) *PageHandler {
	return &PageHandler{
		sessions: repository.NewSessionRepository(db),
		users:    repository.NewUserRepository(db),
		errors:   errors,
		renderer: renderer,
	}
}

// Page returns a handler for an informational page
func (h *PageHandler) Page(filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := PageData{User: h.userFromSession(r)}
		h.renderer.Render(w, filename, data)
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
