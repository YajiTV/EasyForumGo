package handler

import (
	"database/sql"
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	"ForumJS/internal/model"
	"ForumJS/internal/repository"
)

type ProfileHandler struct {
	users    *repository.UserRepository
	likes    *repository.LikeRepository
	sessions *repository.SessionRepository
}

func NewProfileHandler(db *sql.DB) *ProfileHandler {
	return &ProfileHandler{
		users:    repository.NewUserRepository(db),
		likes:    repository.NewLikeRepository(db),
		sessions: repository.NewSessionRepository(db),
	}
}

type ProfilePageData struct {
	User  *model.User
	Posts []PostWithMeta // PostWithMeta existe déjà dans home.go
}

func (h *ProfileHandler) LikedPosts(w http.ResponseWriter, r *http.Request) {
	// 1. Vérifie que l'utilisateur est connecté
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// 2. Récupère les posts aimés via le JOIN SQL
	posts, err := h.likes.GetLikedPostsByUserID(user.ID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// 3. Enrichit chaque post avec username + compteurs de likes
	var postsWithMeta []PostWithMeta
	for _, p := range posts {
		author, err := h.users.GetByID(p.UserID)
		if err != nil {
			author = &model.User{Username: "Inconnu"}
		}
		likes, _ := h.likes.CountPostLikes(p.ID)
		dislikes, _ := h.likes.CountPostDislikes(p.ID)

		postsWithMeta = append(postsWithMeta, PostWithMeta{
			ID:           p.ID,
			UserID:       p.UserID,
			Title:        p.Title,
			Content:      p.Content,
			ImagePath:    p.ImagePath,
			CreatedAt:    p.CreatedAt,
			Username:     author.Username,
			LikeCount:    likes,
			DislikeCount: dislikes,
		})
	}

	// 4. Affiche le template
	data := ProfilePageData{
		User:  user,
		Posts: postsWithMeta,
	}

	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "profile", "profile.html"),
	)
	if err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base", data)
}

func (h *ProfileHandler) userFromSession(r *http.Request) *model.User {
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
