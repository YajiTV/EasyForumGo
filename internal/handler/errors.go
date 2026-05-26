package handler

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

type ErrorRenderer struct {
	templatesDir string
}

type ErrorPageData struct {
	User       any
	Title      string
	StatusCode int
	Heading    string
	Message    string
}

func NewErrorRenderer(templatesDir string) *ErrorRenderer {
	return &ErrorRenderer{templatesDir: templatesDir}
}

func (r *ErrorRenderer) BadRequest(w http.ResponseWriter, message string) {
	r.Render(w, http.StatusBadRequest, message)
}

func (r *ErrorRenderer) Unauthorized(w http.ResponseWriter, message string) {
	r.Render(w, http.StatusUnauthorized, message)
}

func (r *ErrorRenderer) Forbidden(w http.ResponseWriter, message string) {
	r.Render(w, http.StatusForbidden, message)
}

func (r *ErrorRenderer) NotFound(w http.ResponseWriter, message string) {
	r.Render(w, http.StatusNotFound, message)
}

func (r *ErrorRenderer) MethodNotAllowed(w http.ResponseWriter, message string) {
	r.Render(w, http.StatusMethodNotAllowed, message)
}

func (r *ErrorRenderer) InternalServerError(w http.ResponseWriter) {
	r.Render(w, http.StatusInternalServerError, "Une erreur est survenue.")
}

func (r *ErrorRenderer) Render(w http.ResponseWriter, statusCode int, message string) {
	if message == "" {
		message = defaultErrorMessage(statusCode)
	}

	data := ErrorPageData{
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
