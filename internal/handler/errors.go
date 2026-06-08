package handler

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"ForumJS/internal/model"
	"ForumJS/internal/repository"
)

type ErrorRenderer struct {
	templatesDir string
	sessions     *repository.SessionRepository
	users        *repository.UserRepository
}

type ErrorPageData struct {
	User       any
	Title      string
	StatusCode int
	Heading    string
	Message    string
}

// NewErrorRenderer creates a new instance
func NewErrorRenderer(templatesDir string) *ErrorRenderer {
	return &ErrorRenderer{templatesDir: templatesDir}
}

// SetAuthRepositories enables user data on error pages
func (r *ErrorRenderer) SetAuthRepositories(sessions *repository.SessionRepository, users *repository.UserRepository) {
	r.sessions = sessions
	r.users = users
}

// BadRequest renders a bad request error
func (r *ErrorRenderer) BadRequest(w http.ResponseWriter, message string) {
	r.Render(w, http.StatusBadRequest, message)
}

// Unauthorized renders an unauthorized error
func (r *ErrorRenderer) Unauthorized(w http.ResponseWriter, message string) {
	r.Render(w, http.StatusUnauthorized, message)
}

// Forbidden renders a forbidden error
func (r *ErrorRenderer) Forbidden(w http.ResponseWriter, message string) {
	r.Render(w, http.StatusForbidden, message)
}

// NotFound renders a not found error
func (r *ErrorRenderer) NotFound(w http.ResponseWriter, message string) {
	r.Render(w, http.StatusNotFound, message)
}

// MethodNotAllowed renders a method not allowed error
func (r *ErrorRenderer) MethodNotAllowed(w http.ResponseWriter, message string) {
	r.Render(w, http.StatusMethodNotAllowed, message)
}

// InternalServerError renders an internal server error
func (r *ErrorRenderer) InternalServerError(w http.ResponseWriter) {
	r.Render(w, http.StatusInternalServerError, "Une erreur est survenue.")
}

// Render renders an error page
func (r *ErrorRenderer) Render(w http.ResponseWriter, statusCode int, message string) {
	r.RenderWithUser(w, statusCode, message, nil)
}

// RenderWithRequest renders an error page with request data
func (r *ErrorRenderer) RenderWithRequest(w http.ResponseWriter, req *http.Request, statusCode int, message string) {
	r.RenderWithUser(w, statusCode, message, r.userFromRequest(req))
}

// RenderWithUser renders an error page with user data
func (r *ErrorRenderer) RenderWithUser(w http.ResponseWriter, statusCode int, message string, user any) {
	if message == "" {
		message = defaultErrorMessage(statusCode)
	}

	data := ErrorPageData{
		User:       user,
		Title:      fmt.Sprintf("%d - %s", statusCode, http.StatusText(statusCode)),
		StatusCode: statusCode,
		Heading:    errorHeading(statusCode),
		Message:    message,
	}

	tmpl, err := template.ParseFiles(
		filepath.Join(r.templatesDir, "layout", "base.html"),
		filepath.Join(r.templatesDir, "error", fmt.Sprintf("%d.html", statusCode)),
	)
	if err != nil {
		log.Printf("error template parse failed for status %d: %v", statusCode, err)
		http.Error(w, message, statusCode)
		return
	}

	w.WriteHeader(statusCode)
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("error template execute failed for status %d: %v", statusCode, err)
	}
}

// userFromRequest gets the user from the request session
func (r *ErrorRenderer) userFromRequest(req *http.Request) *model.User {
	if req == nil || r.sessions == nil || r.users == nil {
		return nil
	}
	cookie, err := req.Cookie("session_token")
	if err != nil {
		return nil
	}
	session, err := r.sessions.GetByToken(cookie.Value)
	if err != nil || session.ExpiresAt.Before(time.Now()) {
		return nil
	}
	user, err := r.users.GetByID(session.UserID)
	if err != nil {
		return nil
	}
	return user
}

// defaultErrorMessage gets the default error message
func defaultErrorMessage(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "La requête envoyée est invalide."
	case http.StatusUnauthorized:
		return "Vous devez être connecté pour accéder à cette page."
	case http.StatusForbidden:
		return "Vous n'avez pas l'autorisation d'accéder à cette page."
	case http.StatusNotFound:
		return "La page demandée est introuvable."
	case http.StatusMethodNotAllowed:
		return "Cette action n'est pas disponible avec cette méthode."
	case http.StatusInternalServerError:
		return "Une erreur est survenue."
	default:
		return "Une erreur est survenue."
	}
}

// errorHeading gets the error page heading
func errorHeading(statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return "Requête invalide"
	case http.StatusUnauthorized:
		return "Connexion requise"
	case http.StatusForbidden:
		return "Accès refusé"
	case http.StatusNotFound:
		return "Page introuvable"
	case http.StatusMethodNotAllowed:
		return "Méthode non autorisée"
	case http.StatusInternalServerError:
		return "Erreur serveur"
	default:
		return "Erreur"
	}
}
