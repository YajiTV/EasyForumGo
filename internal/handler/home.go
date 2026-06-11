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
	IsFollowing  bool
}

type HomePageData struct {
	User            *model.User
	Posts           []PostWithMeta
	Categories      []model.Category
	CurrentCategory *model.Category
	FollowingFeed   bool
	Page            int
	PreviousPage    int
	NextPage        int
	HasPrevious     bool
	HasNext         bool
	PaginationBase  string
}

type HomeHandler struct {
	posts      *repository.PostRepository
	users      *repository.UserRepository
	sessions   *repository.SessionRepository
	likes      *repository.LikeRepository
	categories *repository.CategoryRepository
	follows    *repository.FollowRepository
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
		follows:    repository.NewFollowRepository(db),
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

	page := pageNumber(r)
	followingFeed := r.URL.Query().Get("feed") == "following"
	var posts []model.Post
	var err error
	if followingFeed {
		if currentUser == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		posts, err = h.posts.GetFollowing(currentUser.ID, socialPageSize+1, (page-1)*socialPageSize)
	} else {
		posts, err = h.posts.GetAllPaginated(socialPageSize+1, (page-1)*socialPageSize)
	}
	if err != nil {
		h.errors.InternalServerError(w)
		return
	}

	hasNext := len(posts) > socialPageSize
	if hasNext {
		posts = posts[:socialPageSize]
	}
	postsWithMeta := h.postsWithMeta(posts, currentUser)

	categories, err := h.categories.GetAll()
	if err != nil {
		categories = []model.Category{}
	}

	data := HomePageData{
		User:           currentUser,
		Posts:          postsWithMeta,
		Categories:     categories,
		FollowingFeed:  followingFeed,
		Page:           page,
		PreviousPage:   page - 1,
		NextPage:       page + 1,
		HasPrevious:    page > 1,
		HasNext:        hasNext,
		PaginationBase: "/?page=",
	}
	if followingFeed {
		data.PaginationBase = "/?feed=following&page="
	}

	h.renderer.Render(w, "home.html", data)
}

// postsWithMeta adds display metadata to posts
func (h *HomeHandler) postsWithMeta(posts []model.Post, currentUser *model.User) []PostWithMeta {
	postsWithMeta := make([]PostWithMeta, 0, len(posts))
	for _, post := range posts {
		user, err := h.users.GetByID(post.UserID)
		if err != nil {
			user = &model.User{Username: "Inconnu"}
		}
		likes, _ := h.likes.CountPostLikes(post.ID)
		dislikes, _ := h.likes.CountPostDislikes(post.ID)
		isFollowing := false
		if currentUser != nil {
			isFollowing, _ = h.follows.IsFollowing(currentUser.ID, post.UserID)
		}
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
			IsFollowing:  isFollowing,
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
