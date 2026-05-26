package handler

import (
	"database/sql"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ForumJS/internal/model"
	"ForumJS/internal/repository"
	"ForumJS/pkg/utils"
)

const maxImageSize = 20 << 20 // 20 MB

type PostHandler struct {
	posts          *repository.PostRepository
	categories     *repository.CategoryRepository
	postCategories *repository.PostCategoryRepository
	sessions       *repository.SessionRepository
	users          *repository.UserRepository
	comments       *repository.CommentRepository
	likes          *repository.LikeRepository
}

func NewPostHandler(db *sql.DB) *PostHandler {
	return &PostHandler{
		posts:          repository.NewPostRepository(db),
		categories:     repository.NewCategoryRepository(db),
		postCategories: repository.NewPostCategoryRepository(db),
		sessions:       repository.NewSessionRepository(db),
		users:          repository.NewUserRepository(db),
		comments:       repository.NewCommentRepository(db),
		likes:          repository.NewLikeRepository(db),
	}
}

type createPostData struct {
	User       *model.User
	Categories []model.Category
	Error      string
	Title      string
	Content    string
}

func (h *PostHandler) ShowCreateForm(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	categories, _ := h.categories.GetAll()

	h.renderCreateForm(w, createPostData{
		User:       user,
		Categories: categories,
	})
}

func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImageSize)
	if err := r.ParseMultipartForm(maxImageSize); err != nil {
		h.renderError(w, r, user, "Fichier trop volumineux (max 20 Mo).")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))
	categoryIDs := r.Form["categories"]

	// Validation
	if title == "" {
		h.renderError(w, r, user, "Le titre est obligatoire.")
		return
	}
	if len(title) > 200 {
		h.renderError(w, r, user, "Le titre ne peut pas dépasser 200 caractères.")
		return
	}
	if content == "" {
		h.renderError(w, r, user, "Le contenu est obligatoire.")
		return
	}
	if len(categoryIDs) == 0 {
		h.renderError(w, r, user, "Sélectionnez au moins une catégorie.")
		return
	}

	// Image (optionnel)
	imagePath := ""
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
			h.renderError(w, r, user, "Format d'image invalide (JPEG, PNG, GIF uniquement).")
			return
		}

		filename := utils.NewUUID() + ext
		dst, err := os.Create(filepath.Join("uploads", filename))
		if err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
		defer dst.Close()
		io.Copy(dst, file)
		imagePath = filename
	}

	// Insertion post
	now := time.Now()
	post := &model.Post{
		ID:        utils.NewUUID(),
		UserID:    user.ID,
		Title:     title,
		Content:   content,
		ImagePath: imagePath,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.posts.Create(post); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// Association categories
	for _, catID := range categoryIDs {
		h.postCategories.AddCategory(post.ID, catID)
	}

	http.Redirect(w, r, "/post/"+post.ID, http.StatusSeeOther)
}

// --- helpers ---

func (h *PostHandler) userFromSession(r *http.Request) *model.User {
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

func (h *PostHandler) renderCreateForm(w http.ResponseWriter, data createPostData) {
	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "post", "create_post.html"),
	)
	if err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base", data)
}

func (h *PostHandler) renderError(w http.ResponseWriter, r *http.Request, user *model.User, msg string) {
	categories, _ := h.categories.GetAll()
	h.renderCreateForm(w, createPostData{
		User:       user,
		Categories: categories,
		Error:      msg,
		Title:      r.FormValue("title"),
		Content:    r.FormValue("content"),
	})
}

// --- PostDetail ---

type CommentWithAuthor struct {
	model.Comment
	Username string
}

type PostDetailData struct {
	User         *model.User
	Post         *model.Post
	Author       string
	Categories   []model.Category
	Comments     []CommentWithAuthor
	LikeCount    int
	DislikeCount int
}

func (h *PostHandler) PostDetail(w http.ResponseWriter, r *http.Request) {
	postID := r.PathValue("id")
	if postID == "" {
		http.NotFound(w, r)
		return
	}

	post, err := h.posts.GetByID(postID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	author, err := h.users.GetByID(post.UserID)
	if err != nil {
		author = &model.User{Username: "Inconnu"}
	}

	categories, _ := h.postCategories.GetCategoriesByPostID(postID)
	likes, _ := h.likes.CountPostLikes(postID)
	dislikes, _ := h.likes.CountPostDislikes(postID)

	rawComments, _ := h.comments.GetByPostID(postID)
	var comments []CommentWithAuthor
	for _, c := range rawComments {
		u, err := h.users.GetByID(c.UserID)
		username := "Inconnu"
		if err == nil {
			username = u.Username
		}
		comments = append(comments, CommentWithAuthor{Comment: c, Username: username})
	}

	data := PostDetailData{
		User:         h.userFromSession(r),
		Post:         post,
		Author:       author.Username,
		Categories:   categories,
		Comments:     comments,
		LikeCount:    likes,
		DislikeCount: dislikes,
	}

	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "post", "post_detail.html"),
	)
	if err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base", data)
}
