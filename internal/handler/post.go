package handler

import (
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ForumJS/internal/model"
	"ForumJS/internal/repository"
	"ForumJS/pkg/utils"
	"ForumJS/pkg/validator"
)

type PostHandler struct {
	posts          *repository.PostRepository
	categories     *repository.CategoryRepository
	postCategories *repository.PostCategoryRepository
	sessions       *repository.SessionRepository
	users          *repository.UserRepository
	comments       *repository.CommentRepository
	likes          *repository.LikeRepository
	uploadDir      string
}

func NewPostHandler(db *sql.DB, uploadDir string) *PostHandler {
	return &PostHandler{
		posts:          repository.NewPostRepository(db),
		categories:     repository.NewCategoryRepository(db),
		postCategories: repository.NewPostCategoryRepository(db),
		sessions:       repository.NewSessionRepository(db),
		users:          repository.NewUserRepository(db),
		comments:       repository.NewCommentRepository(db),
		likes:          repository.NewLikeRepository(db),
		uploadDir:      uploadDir,
	}
}

type createPostData struct {
	User       *model.User
	Categories []model.Category
	Error      string
	Title      string
	Content    string
}

type deletePostData struct {
	User *model.User
	Post *model.Post
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

	r.Body = http.MaxBytesReader(w, r.Body, utils.MaxUploadSize)
	if err := r.ParseMultipartForm(utils.MaxUploadSize); err != nil {
		h.renderError(w, r, user, "Fichier trop volumineux (max 20 Mo).")
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))
	categoryIDs := r.Form["categories"]

	if validationErrors := validator.ValidatePost(validator.PostInput{
		Title:       title,
		Content:     content,
		CategoryIDs: categoryIDs,
	}); validationErrors.HasErrors() {
		h.renderError(w, r, user, firstValidationMessage(validationErrors))
		return
	}

	categoryIDs, err := h.validCategoryIDs(categoryIDs)
	if err != nil {
		h.renderError(w, r, user, "Sélectionnez une catégorie valide.")
		return
	}

	imagePath := ""
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		filename, err := utils.SaveUploadedImage(file, header, h.uploadDir)
		if errors.Is(err, utils.ErrInvalidMIME) || errors.Is(err, utils.ErrFileTooLarge) {
			h.renderError(w, r, user, err.Error())
			return
		}
		if err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
		imagePath = filename
	} else if !errors.Is(err, http.ErrMissingFile) {
		h.renderError(w, r, user, "L'image envoyée est invalide.")
		return
	}

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

	for _, catID := range categoryIDs {
		if err := h.postCategories.AddCategory(post.ID, catID); err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/post/"+post.ID, http.StatusSeeOther)
}

func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID := strings.TrimSuffix(r.PathValue("id"), "/delete")
	post, err := h.posts.GetByID(postID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if post.UserID != user.ID {
		http.Error(w, "Interdit", http.StatusForbidden)
		return
	}

	if post.ImagePath != "" {
		os.Remove(filepath.Join(h.uploadDir, post.ImagePath))
	}

	if err := h.posts.Delete(postID); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *PostHandler) ShowDeleteConfirmation(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID := strings.TrimSuffix(r.PathValue("id"), "/delete")
	post, err := h.posts.GetByID(postID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if post.UserID != user.ID {
		http.Error(w, "Interdit", http.StatusForbidden)
		return
	}

	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "post", "delete_post.html"),
	)
	if err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base", deletePostData{
		User: user,
		Post: post,
	})
}

type editPostData struct {
	User               *model.User
	Post               *model.Post
	Categories         []model.Category
	SelectedCategories map[string]bool
	Error              string
}

func (h *PostHandler) ShowEditForm(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID := strings.TrimSuffix(r.PathValue("id"), "/edit")
	post, err := h.posts.GetByID(postID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if post.UserID != user.ID {
		http.Error(w, "Interdit", http.StatusForbidden)
		return
	}

	categories, _ := h.categories.GetAll()
	selected, _ := h.postCategories.GetCategoriesByPostID(postID)

	selectedMap := make(map[string]bool)
	for _, c := range selected {
		selectedMap[c.ID] = true
	}

	h.renderEditForm(w, editPostData{
		User:               user,
		Post:               post,
		Categories:         categories,
		SelectedCategories: selectedMap,
	})
}

func (h *PostHandler) EditPost(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID := strings.TrimSuffix(r.PathValue("id"), "/edit")
	post, err := h.posts.GetByID(postID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if post.UserID != user.ID {
		http.Error(w, "Interdit", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, utils.MaxUploadSize)
	if err := r.ParseMultipartForm(utils.MaxUploadSize); err != nil {
		http.Error(w, "Fichier trop volumineux", http.StatusBadRequest)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))
	categoryIDs := r.Form["categories"]

	categories, _ := h.categories.GetAll()
	selected, _ := h.postCategories.GetCategoriesByPostID(postID)
	selectedMap := make(map[string]bool)
	for _, c := range selected {
		selectedMap[c.ID] = true
	}

	renderErr := func(msg string) {
		h.renderEditForm(w, editPostData{
			User:               user,
			Post:               post,
			Categories:         categories,
			SelectedCategories: selectedMap,
			Error:              msg,
		})
	}

	if validationErrors := validator.ValidatePost(validator.PostInput{
		Title:       title,
		Content:     content,
		CategoryIDs: categoryIDs,
	}); validationErrors.HasErrors() {
		renderErr(firstValidationMessage(validationErrors))
		return
	}

	categoryIDs, err = h.validCategoryIDs(categoryIDs)
	if err != nil {
		renderErr("Sélectionnez une catégorie valide.")
		return
	}

	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		filename, err := utils.SaveUploadedImage(file, header, h.uploadDir)
		if errors.Is(err, utils.ErrInvalidMIME) || errors.Is(err, utils.ErrFileTooLarge) {
			renderErr(err.Error())
			return
		}
		if err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
		if post.ImagePath != "" {
			os.Remove(filepath.Join(h.uploadDir, post.ImagePath))
		}
		post.ImagePath = filename
	} else if !errors.Is(err, http.ErrMissingFile) {
		renderErr("L'image envoyée est invalide.")
		return
	}

	post.Title = title
	post.Content = content
	post.UpdatedAt = time.Now()

	if err := h.posts.Update(post); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	if err := h.postCategories.DeleteByPostID(postID); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	for _, catID := range categoryIDs {
		if err := h.postCategories.AddCategory(postID, catID); err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}

func (h *PostHandler) renderEditForm(w http.ResponseWriter, data editPostData) {
	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "post", "edit_post.html"),
	)
	if err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base", data)
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

func (h *PostHandler) validCategoryIDs(categoryIDs []string) ([]string, error) {
	seen := make(map[string]bool, len(categoryIDs))
	validIDs := make([]string, 0, len(categoryIDs))

	for _, categoryID := range categoryIDs {
		categoryID = strings.TrimSpace(categoryID)
		if categoryID == "" || seen[categoryID] {
			continue
		}
		if _, err := h.categories.GetByID(categoryID); err != nil {
			return nil, err
		}
		seen[categoryID] = true
		validIDs = append(validIDs, categoryID)
	}

	if len(validIDs) == 0 {
		return nil, errors.New("no valid category selected")
	}

	return validIDs, nil
}

// --- PostDetail ---

type CommentWithAuthor struct {
	model.Comment
	Username     string
	LikeCount    int
	DislikeCount int
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
	if strings.HasSuffix(postID, "/edit") {
		r.SetPathValue("id", strings.TrimSuffix(postID, "/edit"))
		h.ShowEditForm(w, r)
		return
	}
	if strings.HasSuffix(postID, "/delete") {
		r.SetPathValue("id", strings.TrimSuffix(postID, "/delete"))
		h.ShowDeleteConfirmation(w, r)
		return
	}
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
		cLikes, _ := h.likes.CountCommentLikes(c.ID)
		cDislikes, _ := h.likes.CountCommentDislikes(c.ID)
		comments = append(comments, CommentWithAuthor{
			Comment:      c,
			Username:     username,
			LikeCount:    cLikes,
			DislikeCount: cDislikes,
		})
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
