package handler

import (
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"ForumJS/internal/model"
	"ForumJS/internal/repository"
	"ForumJS/pkg/utils"
	"ForumJS/pkg/validator"
)

type ProfileHandler struct {
	users     *repository.UserRepository
	likes     *repository.LikeRepository
	sessions  *repository.SessionRepository
	uploadDir string
}

func NewProfileHandler(db *sql.DB, uploadDir string) *ProfileHandler {
	return &ProfileHandler{
		users:     repository.NewUserRepository(db),
		likes:     repository.NewLikeRepository(db),
		sessions:  repository.NewSessionRepository(db),
		uploadDir: uploadDir,
	}
}

type ProfilePageData struct {
	User  *model.User
	Posts []PostWithMeta // PostWithMeta existe déjà dans home.go
}

type EditProfilePageData struct {
	User  *model.User
	Error string
}

func (h *ProfileHandler) LikedPosts(w http.ResponseWriter, r *http.Request) {
	// 1. Vérifie que l'utilisateur est connecté
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// 2. Récupère les posts aimés via le JOIN SQL
	posts, err := h.likes.GetLikedPostsByUserID(user.ID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// 3. Enrichit chaque post avec username + compteurs de likes
	var postsWithMeta []PostWithMeta
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

	// 4. Affiche le template
	data := ProfilePageData{
		User:  user,
		Posts: postsWithMeta,
	}

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

func (h *ProfileHandler) ShowEditForm(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	h.renderEditForm(w, EditProfilePageData{User: user})
}

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
