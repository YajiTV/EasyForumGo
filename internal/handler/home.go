package handler

import (
	"database/sql"
	"net/http"
	"time"

	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
)

type PostWithMeta struct {
	ID           string
	UserID       string
	Title        string
	Content      string
	ImagePath    string
	CreatedAt    time.Time
	Username     string
	LikeCount    int
	DislikeCount int
}

type HomePageData struct {
	User            *model.User
	Posts           []PostWithMeta
	Categories      []model.Category
	CurrentCategory *model.Category
}

type HomeHandler struct {
	posts      *repository.PostRepository
	users      *repository.UserRepository
	sessions   *repository.SessionRepository
	likes      *repository.LikeRepository
	categories *repository.CategoryRepository
	errors     *ErrorRenderer
	renderer   *PageRenderer
}

// NewHomeHandler creates a new instance
func NewHomeHandler(db *sql.DB, errors *ErrorRenderer, renderer *PageRenderer) *HomeHandler {
	if errors == nil {
		errors = NewErrorRenderer("web/templates")
	}

	return &HomeHandler{
		posts:      repository.NewPostRepository(db),
		users:      repository.NewUserRepository(db),
		sessions:   repository.NewSessionRepository(db),
		likes:      repository.NewLikeRepository(db),
		categories: repository.NewCategoryRepository(db),
		errors:     errors,
		renderer:   renderer,
	}
}

// Home handles the request
func (h *HomeHandler) Home(w http.ResponseWriter, r *http.Request) {
	currentUser := h.userFromSession(r)

	if r.URL.Path != "/" {
		h.errors.RenderWithRequest(w, r, http.StatusNotFound, "La page demandée est introuvable.")
		return
	}

	posts, err := h.posts.GetAll()
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	postsWithMeta := h.postsWithMeta(posts)

	categories, err := h.categories.GetAll()
	if err != nil {
		categories = []model.Category{}
	}

	data := HomePageData{
		User:       currentUser,
		Posts:      postsWithMeta,
		Categories: categories,
	}

	h.renderer.Render(w, "home.html", data)
}

// postsWithMeta adds display metadata to posts
func (h *HomeHandler) postsWithMeta(posts []model.Post) []PostWithMeta {
	postsWithMeta := make([]PostWithMeta, 0, len(posts))
	for _, post := range posts {
		user, err := h.users.GetByID(post.UserID)
		if err != nil {
			user = &model.User{Username: "Inconnu"}
		}
		likes, _ := h.likes.CountPostLikes(post.ID)
		dislikes, _ := h.likes.CountPostDislikes(post.ID)
		postsWithMeta = append(postsWithMeta, PostWithMeta{
			ID:           post.ID,
			UserID:       post.UserID,
			Title:        post.Title,
			Content:      post.Content,
			ImagePath:    post.ImagePath,
			CreatedAt:    post.CreatedAt,
			Username:     user.Username,
			LikeCount:    likes,
			DislikeCount: dislikes,
		})
	}
	return postsWithMeta
}

// userFromSession gets the user from the current session
func (h *HomeHandler) userFromSession(r *http.Request) *model.User {
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
