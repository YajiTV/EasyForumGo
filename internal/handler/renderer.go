package handler

import (
	"html/template"
	"net/http"
	"path/filepath"

	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
)

type PageRenderer struct {
	notifications *repository.NotificationRepository
}

// NewPageRenderer creates a new instance
func NewPageRenderer(notifications *repository.NotificationRepository) *PageRenderer {
	return &PageRenderer{notifications: notifications}
}

// Render parses and executes a template with notification count support
func (r *PageRenderer) Render(w http.ResponseWriter, name string, data any) {
	funcMap := template.FuncMap{
		"notifCount": func(user *model.User) int {
			if user == nil {
				return 0
			}
			count, _ := r.notifications.CountUnread(user.ID)
			return count
		},
	}

	tmpl, err := template.New("base").Funcs(funcMap).ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", name),
	)
	if err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base", data)
}
