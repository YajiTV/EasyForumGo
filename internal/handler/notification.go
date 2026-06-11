package handler

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	"ForumJS/internal/model"
	"ForumJS/internal/repository"
)

type NotificationHandler struct {
	notifications *repository.NotificationRepository
	sessions      *repository.SessionRepository
	users         *repository.UserRepository
}

// NewNotificationHandler creates a new instance
func NewNotificationHandler(db *sql.DB) *NotificationHandler {
	return &NotificationHandler{
		notifications: repository.NewNotificationRepository(db),
		sessions:      repository.NewSessionRepository(db),
		users:         repository.NewUserRepository(db),
	}
}

type notificationPageData struct {
	User          *model.User
	Notifications []model.Notification
}

// ShowNotifications renders the notifications page
func (h *NotificationHandler) ShowNotifications(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	notifs, err := h.notifications.GetByUserID(user.ID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	_ = h.notifications.MarkAllRead(user.ID)

	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "notifications.html"),
	)
	if err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base", notificationPageData{
		User:          user,
		Notifications: notifs,
	})
}

// GetUnreadCount returns the unread notification count as JSON
func (h *NotificationHandler) GetUnreadCount(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"count":0}`))
		return
	}

	count, err := h.notifications.CountUnread(user.ID)
	if err != nil {
		count = 0
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}

// userFromSession gets the user from the current session
func (h *NotificationHandler) userFromSession(r *http.Request) *model.User {
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
