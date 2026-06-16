package handler

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
	"EasyForumGo/pkg/utils"
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
			count, err := r.notifications.CountUnread(user.ID)
			if err != nil {
				log.Printf("count unread notifications for user %q: %v", user.ID, err)
				return 0
			}
			return count
		},
		"stripMD": utils.StripMarkdown,
	}

	tmpl, err := template.New("base").Funcs(funcMap).ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", name),
	)
	if err != nil {
		log.Printf("parse page template %q: %v", name, err)
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		log.Printf("execute page template %q: %v", name, err)
	}
}
