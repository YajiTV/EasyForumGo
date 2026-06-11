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
	CurrentFilter   string
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

	currentFilter := r.URL.Query().Get("filter")
	if currentFilter != "mine" && currentFilter != "liked" {
		currentFilter = ""
	}
	posts, err := h.postsForFilter(currentFilter, currentUser)
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}
	if (currentFilter == "mine" || currentFilter == "liked") && currentUser == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postsWithMeta := h.postsWithMeta(posts)

	categories, err := h.categories.GetAll()
	if err != nil {
		categories = []model.Category{}
	}

	data := HomePageData{
		User:          currentUser,
		Posts:         postsWithMeta,
		Categories:    categories,
		CurrentFilter: currentFilter,
	}

	h.renderer.Render(w, "home.html", data)
}

// postsForFilter gets posts matching the selected home filter
func (h *HomeHandler) postsForFilter(filter string, user *model.User) ([]model.Post, error) {
	switch {
	case filter == "mine" && user != nil:
		return h.posts.GetByUserID(user.ID)
	case filter == "liked" && user != nil:
		return h.likes.GetLikedPostsByUserID(user.ID)
	default:
		return h.posts.GetAll()
	}
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
