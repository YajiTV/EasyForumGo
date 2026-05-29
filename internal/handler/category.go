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

// CategoryHandler gère la page de filtrage par catégorie
type CategoryHandler struct {
	posts      *repository.PostRepository
	users      *repository.UserRepository
	likes      *repository.LikeRepository
	categories *repository.CategoryRepository
	sessions   *repository.SessionRepository
}

func NewCategoryHandler(db *sql.DB) *CategoryHandler {
	return &CategoryHandler{
		posts:      repository.NewPostRepository(db),
		users:      repository.NewUserRepository(db),
		likes:      repository.NewLikeRepository(db),
		categories: repository.NewCategoryRepository(db),
		sessions:   repository.NewSessionRepository(db),
	}
}

// CategoryPageData contient tout ce dont le template a besoin
type CategoryPageData struct {
	User            *model.User
	Posts           []PostWithMeta // PostWithMeta existe déjà dans home.go
	Categories      []model.Category
	CurrentCategory *model.Category // la catégorie sélectionnée (pour afficher son nom)
}

func (h *CategoryHandler) FilterByCategory(w http.ResponseWriter, r *http.Request) {
	categoryID := r.PathValue("id")

	// 1. On vérifie que la catégorie existe
	category, err := h.categories.GetByID(categoryID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// 2. On récupère les posts de cette catégorie (le JOIN est déjà dans le repository)
	posts, err := h.posts.GetByCategory(categoryID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// 3. Pour chaque post, on enrichit avec le username et les likes
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

	// 4. On récupère toutes les catégories pour la barre de filtres
	categories, _ := h.categories.GetAll()

	// 5. On récupère l'utilisateur connecté (nil si non connecté)
	user := h.userFromSession(r)

	data := CategoryPageData{
		User:            user,
		Posts:           postsWithMeta,
		Categories:      categories,
		CurrentCategory: category,
	}

	// 6. On affiche le template (on réutilise home.html !)
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

// userFromSession est dupliquée ici depuis post.go
// (on pourrait la mettre dans un fichier helpers.go plus tard)
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
