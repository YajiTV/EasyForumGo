package handler

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"ForumJS/internal/model"
	"ForumJS/internal/repository"
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
	User       *model.User
	Posts      []PostWithMeta
	Categories []model.Category
}

type HomeHandler struct {
	posts      *repository.PostRepository
	users      *repository.UserRepository
	likes      *repository.LikeRepository
	categories *repository.CategoryRepository
}

func NewHomeHandler(db *sql.DB) *HomeHandler {
	return &HomeHandler{
		posts:      repository.NewPostRepository(db),
		users:      repository.NewUserRepository(db),
		likes:      repository.NewLikeRepository(db),
		categories: repository.NewCategoryRepository(db),
	}
}

func (h *HomeHandler) Home(w http.ResponseWriter, r *http.Request) {
	posts, err := h.posts.GetAll()
	if err != nil {
		log.Printf("GetAll error: %v", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

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

	categories, err := h.categories.GetAll()
	if err != nil {
		categories = []model.Category{}
	}

	data := HomePageData{
		User:       nil,
		Posts:      postsWithMeta,
		Categories: categories,
	}

	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "home.html"),
	)
	if err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}

	tmpl.ExecuteTemplate(w, "base", data)
}
