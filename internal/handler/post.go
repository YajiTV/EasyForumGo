package handler

import (
	"database/sql"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
	"EasyForumGo/pkg/utils"
	"EasyForumGo/pkg/validator"
)

type PostHandler struct {
	posts          *repository.PostRepository
	categories     *repository.CategoryRepository
	postCategories *repository.PostCategoryRepository
	sessions       *repository.SessionRepository
	users          *repository.UserRepository
	comments       *repository.CommentRepository
	likes          *repository.LikeRepository
	libraries      *repository.LibraryRepository
	uploadDir      string
	maxUploadBytes int64
	renderer       *PageRenderer
	follows        *repository.FollowRepository
	notifications  *repository.NotificationRepository
}

// NewPostHandler creates a new instance
func NewPostHandler(db *sql.DB, uploadDir string, maxUploadBytes int64, renderer *PageRenderer) *PostHandler {
	return &PostHandler{
		posts:          repository.NewPostRepository(db),
		categories:     repository.NewCategoryRepository(db),
		postCategories: repository.NewPostCategoryRepository(db),
		sessions:       repository.NewSessionRepository(db),
		users:          repository.NewUserRepository(db),
		comments:       repository.NewCommentRepository(db),
		likes:          repository.NewLikeRepository(db),
		libraries:      repository.NewLibraryRepository(db),
		uploadDir:      uploadDir,
		maxUploadBytes: maxUploadBytes,
		renderer:       renderer,
		follows:        repository.NewFollowRepository(db),
		notifications:  repository.NewNotificationRepository(db),
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

// ShowCreateForm renders the requested page
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

// CreatePost creates a new record
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, utils.MaxMultipartBodySize(h.maxUploadBytes))
	if err := r.ParseMultipartForm(h.maxUploadBytes); err != nil {
		h.renderError(w, r, user, "Fichier trop volumineux (max "+utils.UploadSizeLabel(h.maxUploadBytes)+").")
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
		filename, err := utils.SaveUploadedImage(file, header, h.uploadDir, h.maxUploadBytes)
		if errors.Is(err, utils.ErrInvalidMIME) || errors.Is(err, utils.ErrInvalidImage) || errors.Is(err, utils.ErrFileTooLarge) {
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

	approved := 0
	if user.IsModerator() {
		approved = 1
	}

	now := time.Now()
	post := &model.Post{
		ID:        utils.NewUUID(),
		UserID:    user.ID,
		Title:     title,
		Content:   content,
		ImagePath: imagePath,
		Approved:  approved,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.posts.CreateWithCategories(post, categoryIDs); err != nil {
		if imagePath != "" {
			if removeErr := os.Remove(filepath.Join(h.uploadDir, imagePath)); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				log.Printf("remove orphaned post image %q: %v", imagePath, removeErr)
			}
		}
		log.Printf("create post transaction failed: %v", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	h.notifyFollowers(post)

	http.Redirect(w, r, "/post/"+post.ID, http.StatusSeeOther)
}

// notifyFollowers notifies followers about a new post
func (h *PostHandler) notifyFollowers(post *model.Post) {
	followerIDs, err := h.follows.FollowerIDs(post.UserID)
	if err != nil {
		log.Printf("load followers for post notification: %v", err)
		return
	}
	for _, followerID := range followerIDs {
		if err := h.notifications.Create(&model.Notification{
			ID: utils.NewUUID(), UserID: followerID, ActorID: post.UserID,
			Type: "new_post", PostID: post.ID, CreatedAt: time.Now(),
		}); err != nil {
			log.Printf("create new-post notification for user %q: %v", followerID, err)
		}
	}
}

// DeletePost deletes an existing record
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

	if err := h.posts.Delete(postID); err != nil {
		log.Printf("delete post %q: %v", postID, err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	if post.ImagePath != "" {
		if err := os.Remove(filepath.Join(h.uploadDir, post.ImagePath)); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("remove deleted post image %q: %v", post.ImagePath, err)
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ShowDeleteConfirmation renders the requested page
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

	h.renderer.Render(w, "post/delete_post.html", deletePostData{
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

// ShowEditForm renders the requested page
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

// EditPost updates a post from the author
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

	r.Body = http.MaxBytesReader(w, r.Body, utils.MaxMultipartBodySize(h.maxUploadBytes))
	if err := r.ParseMultipartForm(h.maxUploadBytes); err != nil {
		http.Error(w, "Fichier trop volumineux (max "+utils.UploadSizeLabel(h.maxUploadBytes)+").", http.StatusBadRequest)
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

	previousImagePath := post.ImagePath
	newImagePath := ""
	file, header, err := r.FormFile("image")
	if err == nil {
		defer file.Close()
		filename, err := utils.SaveUploadedImage(file, header, h.uploadDir, h.maxUploadBytes)
		if errors.Is(err, utils.ErrInvalidMIME) || errors.Is(err, utils.ErrInvalidImage) || errors.Is(err, utils.ErrFileTooLarge) {
			renderErr(err.Error())
			return
		}
		if err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
		post.ImagePath = filename
		newImagePath = filename
	} else if !errors.Is(err, http.ErrMissingFile) {
		renderErr("L'image envoyée est invalide.")
		return
	}

	post.Title = title
	post.Content = content
	post.UpdatedAt = time.Now()

	if err := h.posts.UpdateWithCategories(post, categoryIDs); err != nil {
		if newImagePath != "" {
			if removeErr := os.Remove(filepath.Join(h.uploadDir, newImagePath)); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				log.Printf("remove orphaned replacement image %q: %v", newImagePath, removeErr)
			}
		}
		log.Printf("update post transaction failed for %q: %v", postID, err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	if newImagePath != "" && previousImagePath != "" {
		if err := os.Remove(filepath.Join(h.uploadDir, previousImagePath)); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Printf("remove replaced post image %q: %v", previousImagePath, err)
		}
	}

	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}

// renderEditForm renders the requested page
func (h *PostHandler) renderEditForm(w http.ResponseWriter, data editPostData) {
	h.renderer.Render(w, "post/edit_post.html", data)
}

// userFromSession gets the user from the current session
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

// ApprovePost approves a pending post (moderators and admins only)
func (h *PostHandler) ApprovePost(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil || !user.IsModerator() {
		http.Error(w, "Interdit", http.StatusForbidden)
		return
	}

	postID := r.PathValue("id")
	if err := h.posts.Approve(postID); err != nil {
		log.Printf("approve post %q: %v", postID, err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}

// renderCreateForm renders the requested page
func (h *PostHandler) renderCreateForm(w http.ResponseWriter, data createPostData) {
	h.renderer.Render(w, "post/create_post.html", data)
}

// renderError renders the requested page
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

// validCategoryIDs removes duplicate and unknown category ids
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

type CommentWithAuthor struct {
	model.Comment
	Username     string
	AvatarURL    string
	LikeCount    int
	DislikeCount int
}

type PostDetailData struct {
	User            *model.User
	Post            *model.Post
	RenderedContent template.HTML
	Author          string
	AuthorAvatarURL string
	AuthorBiography string
	AuthorTopics    []model.Category
	IsEdited        bool
	Categories      []model.Category
	Comments        []CommentWithAuthor
	Libraries       []model.Library
	LikeCount       int
	DislikeCount    int
}

// PostDetail handles the request
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
	authorTopics, _ := h.postCategories.GetPopularByUserID(post.UserID, 4)
	likes, _ := h.likes.CountPostLikes(postID)
	dislikes, _ := h.likes.CountPostDislikes(postID)

	rawComments, _ := h.comments.GetByPostID(postID)
	var comments []CommentWithAuthor
	for _, c := range rawComments {
		u, err := h.users.GetByID(c.UserID)
		username := "Inconnu"
		avatarURL := ""
		if err == nil {
			username = u.Username
			avatarURL = u.AvatarURL()
		}
		cLikes, _ := h.likes.CountCommentLikes(c.ID)
		cDislikes, _ := h.likes.CountCommentDislikes(c.ID)
		comments = append(comments, CommentWithAuthor{
			Comment:      c,
			Username:     username,
			AvatarURL:    avatarURL,
			LikeCount:    cLikes,
			DislikeCount: cDislikes,
		})
	}

	currentUser := h.userFromSession(r)
	libraries := []model.Library{}
	if currentUser != nil {
		libraries, _ = h.libraries.GetByUserID(currentUser.ID)
	}

	data := PostDetailData{
		User:            currentUser,
		Post:            post,
		RenderedContent: utils.RenderMarkdown(post.Content),
		Author:          author.Username,
		AuthorAvatarURL: author.AvatarURL(),
		AuthorBiography: author.Biography,
		AuthorTopics:    authorTopics,
		IsEdited:        post.UpdatedAt.After(post.CreatedAt.Add(time.Second)),
		Categories:      categories,
		Comments:        comments,
		Libraries:       libraries,
		LikeCount:       likes,
		DislikeCount:    dislikes,
	}

	h.renderer.Render(w, "post/post_detail.html", data)
}
