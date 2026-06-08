package handler

import (
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
	"EasyForumGo/pkg/utils"
	"EasyForumGo/pkg/validator"
)

type ProfileHandler struct {
	users     *repository.UserRepository
	posts     *repository.PostRepository
	comments  *repository.CommentRepository
	likes     *repository.LikeRepository
	sessions  *repository.SessionRepository
	uploadDir string
}

// NewProfileHandler creates a new instance
func NewProfileHandler(db *sql.DB, uploadDir string) *ProfileHandler {
	return &ProfileHandler{
		users:     repository.NewUserRepository(db),
		posts:     repository.NewPostRepository(db),
		comments:  repository.NewCommentRepository(db),
		likes:     repository.NewLikeRepository(db),
		sessions:  repository.NewSessionRepository(db),
		uploadDir: uploadDir,
	}
}

type ProfileCommentWithPost struct {
	ID        string
	PostID    string
	PostTitle string
	Content   string
	CreatedAt time.Time
}

type ProfilePageData struct {
	User         *model.User
	Posts        []PostWithMeta
	Comments     []ProfileCommentWithPost
	ActiveTab    string
	SectionTitle string
	EmptyTitle   string
	EmptyMessage string
}

type EditProfilePageData struct {
	User  *model.User
	Error string
}

// LikedPosts renders posts liked by the current user
func (h *ProfileHandler) LikedPosts(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	posts, err := h.likes.GetLikedPostsByUserID(user.ID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	postsWithMeta := h.postsWithMeta(posts)

	h.renderProfile(w, ProfilePageData{
		User:         user,
		Posts:        postsWithMeta,
		ActiveTab:    "liked-posts",
		SectionTitle: "Posts aimés",
		EmptyTitle:   "Aucun post aimé",
		EmptyMessage: "Les posts que vous aimez apparaîtront ici.",
	})
}

// MyPosts handles the request
func (h *ProfileHandler) MyPosts(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	posts, err := h.posts.GetByUserID(user.ID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	h.renderProfile(w, ProfilePageData{
		User:         user,
		Posts:        h.postsWithMeta(posts),
		ActiveTab:    "my-posts",
		SectionTitle: "Mes posts",
		EmptyTitle:   "Aucun post publié",
		EmptyMessage: "Vos publications apparaîtront ici.",
	})
}

// MyComments handles the request
func (h *ProfileHandler) MyComments(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	comments, err := h.comments.GetByUserID(user.ID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	commentsWithPost := make([]ProfileCommentWithPost, 0, len(comments))
	for _, comment := range comments {
		postTitle := "Post supprimé"
		if post, err := h.posts.GetByID(comment.PostID); err == nil {
			postTitle = post.Title
		}

		commentsWithPost = append(commentsWithPost, ProfileCommentWithPost{
			ID:        comment.ID,
			PostID:    comment.PostID,
			PostTitle: postTitle,
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt,
		})
	}

	h.renderProfile(w, ProfilePageData{
		User:         user,
		Comments:     commentsWithPost,
		ActiveTab:    "my-comments",
		SectionTitle: "Mes commentaires",
		EmptyTitle:   "Aucun commentaire",
		EmptyMessage: "Vos commentaires apparaîtront ici.",
	})
}

// userFromSession gets the user from the current session
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

// ShowEditForm renders the requested page
func (h *ProfileHandler) ShowEditForm(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	h.renderEditForm(w, EditProfilePageData{User: user})
}

// UpdateProfile updates an existing record
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, utils.MaxUploadSize)
	if err := r.ParseMultipartForm(utils.MaxUploadSize); err != nil {
		h.renderEditForm(w, EditProfilePageData{
			User:  user,
			Error: "Fichier trop volumineux (max 20 Mo).",
		})
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	if validationErrors := validator.ValidateProfile(validator.ProfileInput{Username: username}); validationErrors.HasErrors() {
		h.renderEditForm(w, EditProfilePageData{
			User:  user,
			Error: firstValidationMessage(validationErrors),
		})
		return
	}

	if existingUser, err := h.users.GetByUsername(username); err == nil && existingUser.ID != user.ID {
		h.renderEditForm(w, EditProfilePageData{
			User:  user,
			Error: "Ce nom d'utilisateur est déjà utilisé.",
		})
		return
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	profilePicture := user.ProfilePicture
	file, header, err := r.FormFile("profile_picture")
	if err == nil {
		defer file.Close()
		filename, err := utils.SaveUploadedImage(file, header, h.uploadDir)
		if errors.Is(err, utils.ErrInvalidMIME) || errors.Is(err, utils.ErrFileTooLarge) {
			h.renderEditForm(w, EditProfilePageData{
				User:  user,
				Error: err.Error(),
			})
			return
		}
		if err != nil {
			h.renderEditForm(w, EditProfilePageData{
				User:  user,
				Error: "La photo de profil n'a pas pu être enregistrée.",
			})
			return
		}
		profilePicture = filename
	}

	if err := h.users.UpdateProfile(user.ID, username, profilePicture); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

// renderEditForm renders the requested page
func (h *ProfileHandler) renderEditForm(w http.ResponseWriter, data EditProfilePageData) {
	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "profile", "edit_profile.html"),
	)
	if err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base", data)
}

// renderProfile renders the requested page
func (h *ProfileHandler) renderProfile(w http.ResponseWriter, data ProfilePageData) {
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

// postsWithMeta adds display metadata to posts
func (h *ProfileHandler) postsWithMeta(posts []model.Post) []PostWithMeta {
	postsWithMeta := make([]PostWithMeta, 0, len(posts))

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

	return postsWithMeta
}
