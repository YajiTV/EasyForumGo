package handler

import (
	"database/sql"
	"net/http"

	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
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
	user := userFromSession(r, h.sessions, h.users)
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

	h.renderer.Render(w, "social/notifications.html", notificationPageData{
		User:          user,
		Notifications: notifs,
	})
}

