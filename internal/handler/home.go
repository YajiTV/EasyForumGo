package handler

import (
	"html/template"
	"net/http"
	"path/filepath"
)

type HomeHandler struct{}

func NewHomeHandler() *HomeHandler {
	return &HomeHandler{}
}

func (h *HomeHandler) Home(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "home.html"),
	)
	if err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "base", nil)
}
