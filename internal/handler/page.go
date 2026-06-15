package handler

import (
	"database/sql"
	"net/http"

	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
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
		data := PageData{User: userFromSession(r, h.sessions, h.users)}
		h.renderer.Render(w, filename, data)
	}
}

