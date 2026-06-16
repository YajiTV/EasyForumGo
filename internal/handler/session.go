package handler

import (
	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
	"net/http"
	"time"
)

func userFromSession(r *http.Request, sessions *repository.SessionRepository, users *repository.UserRepository) *model.User {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return nil
	}
	session, err := sessions.GetByToken(cookie.Value)
	if err != nil || session.ExpiresAt.Before(time.Now()) {
		return nil
	}
	user, err := users.GetByID(session.UserID)
	if err != nil {
		return nil
	}
	return user
}
