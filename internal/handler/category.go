package handler

import (
	"database/sql"
	"net/http"
	"time"

	"ForumJS/internal/model"
	"ForumJS/internal/repository"
)

type CategoryHandler struct {
	posts      *repository.PostRepository
	users      *repository.UserRepository
	likes      *repository.LikeRepository
	categories *repository.CategoryRepository
	sessions   *repository.SessionRepository
	renderer   *PageRenderer
}

// NewCategoryHandler creates a new instance
func NewCategoryHandler(db *sql.DB, renderer *PageRenderer) *CategoryHandler {
	return &CategoryHandler{
		posts:      repository.NewPostRepository(db),
		users:      repository.NewUserRepository(db),
		likes:      repository.NewLikeRepository(db),
		categories: repository.NewCategoryRepository(db),
		sessions:   repository.NewSessionRepository(db),
		renderer:   renderer,
	}
}

type CategoryPageData struct {
	User            *model.User
	Posts           []PostWithMeta
	Categories      []model.Category
	CurrentCategory *model.Category
}

// FilterByCategory handles the request
func (h *CategoryHandler) FilterByCategory(w http.ResponseWriter, r *http.Request) {
	categoryID := r.PathValue("id")

	// reject unknown categories before loading posts
	category, err := h.categories.GetByID(categoryID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	posts, err := h.posts.GetByCategory(categoryID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// enrich posts with data required by the home template
	var postsWithMeta []PostWithMeta
	for _, p := range posts {
		user, err := h.users.GetByID(p.UserID)
		if err != nil {
			user = &model.User{Username: "Inconnu"}
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
			Username:     user.Username,
			LikeCount:    likes,
			DislikeCount: dislikes,
		})
	}

	categories, _ := h.categories.GetAll()

	user := h.userFromSession(r)

	data := CategoryPageData{
		User:            user,
		Posts:           postsWithMeta,
		Categories:      categories,
		CurrentCategory: category,
	}

	// reuse the home template with the selected category
	h.renderer.Render(w, "home.html", data)
}

// userFromSession gets the user from the current session
func (h *CategoryHandler) userFromSession(r *http.Request) *model.User {
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
