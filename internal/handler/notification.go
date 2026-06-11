package handler

import (
	"database/sql"
	"net/http"
	"time"

	"ForumJS/internal/model"
	"ForumJS/internal/repository"
)

type NotificationHandler struct {
	notifications *repository.NotificationRepository
	sessions      *repository.SessionRepository
	users         *repository.UserRepository
	renderer      *PageRenderer
}

// NewNotificationHandler creates a new instance
func NewNotificationHandler(db *sql.DB, notifications *repository.NotificationRepository, renderer *PageRenderer) *NotificationHandler {
	return &NotificationHandler{
		notifications: notifications,
		sessions:      repository.NewSessionRepository(db),
		users:         repository.NewUserRepository(db),
		renderer:      renderer,
	}
}

type notificationPageData struct {
	User          *model.User
	Notifications []model.Notification
}

// ShowNotifications renders the notifications page and marks all as read
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

	h.renderer.Render(w, "notifications.html", notificationPageData{
		User:          user,
		Notifications: notifs,
	})
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
