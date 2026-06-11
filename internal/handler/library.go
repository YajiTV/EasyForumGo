package handler

import (
	"database/sql"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
	"EasyForumGo/pkg/utils"
)

type LibraryHandler struct {
	libraries *repository.LibraryRepository
	posts     *repository.PostRepository
	users     *repository.UserRepository
	likes     *repository.LikeRepository
	sessions  *repository.SessionRepository
}

type LibraryPageData struct {
	User      *model.User
	Libraries []model.Library
	Selected  *model.Library
	Posts     []PostWithMeta
	Error     string
}

// NewLibraryHandler creates a new instance
func NewLibraryHandler(db *sql.DB) *LibraryHandler {
	return &LibraryHandler{
		libraries: repository.NewLibraryRepository(db),
		posts:     repository.NewPostRepository(db),
		users:     repository.NewUserRepository(db),
		likes:     repository.NewLikeRepository(db),
		sessions:  repository.NewSessionRepository(db),
	}
}

// Index renders the current user's libraries
func (h *LibraryHandler) Index(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	h.renderPage(w, LibraryPageData{User: user})
}

// Show renders one library and its posts
func (h *LibraryHandler) Show(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	library, err := h.libraries.GetByIDAndUserID(r.PathValue("id"), user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	posts, err := h.libraries.GetPosts(library.ID, user.ID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	h.renderPage(w, LibraryPageData{
		User:     user,
		Selected: library,
		Posts:    h.postsWithMeta(posts),
	})
}

// Create creates a library for the current user
func (h *LibraryHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if !validLibraryName(name) {
		h.renderPage(w, LibraryPageData{
			User:  user,
			Error: "Le nom doit contenir entre 1 et 60 caractères.",
		})
		return
	}

	library := &model.Library{
		ID:        utils.NewUUID(),
		UserID:    user.ID,
		Name:      name,
		CreatedAt: time.Now(),
	}
	if err := h.libraries.Create(library); err != nil {
		h.renderPage(w, LibraryPageData{
			User:  user,
			Error: "Une bibliothèque porte déjà ce nom.",
		})
		return
	}
	http.Redirect(w, r, "/library/"+library.ID, http.StatusSeeOther)
}

// Rename renames a library owned by the current user
func (h *LibraryHandler) Rename(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	libraryID := r.PathValue("id")
	if _, err := h.libraries.GetByIDAndUserID(libraryID, user.ID); err != nil {
		http.NotFound(w, r)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if !validLibraryName(name) {
		http.Error(w, "Nom de bibliothèque invalide", http.StatusBadRequest)
		return
	}
	if err := h.libraries.Rename(libraryID, user.ID, name); err != nil {
		http.Error(w, "Ce nom est déjà utilisé", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/library/"+libraryID, http.StatusSeeOther)
}

// Delete deletes a library owned by the current user
func (h *LibraryHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	libraryID := r.PathValue("id")
	if _, err := h.libraries.GetByIDAndUserID(libraryID, user.ID); err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.libraries.Delete(libraryID, user.ID); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/library", http.StatusSeeOther)
}

// AddPost adds a post to the selected library
func (h *LibraryHandler) AddPost(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	libraryID := strings.TrimSpace(r.FormValue("library_id"))
	postID := r.PathValue("postID")
	if _, err := h.libraries.GetByIDAndUserID(libraryID, user.ID); err != nil {
		http.NotFound(w, r)
		return
	}
	if _, err := h.posts.GetByID(postID); err != nil {
		http.NotFound(w, r)
		return
	}

	if err := h.libraries.AddPost(libraryID, user.ID, postID); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}

// RemovePost removes a post from a library page
func (h *LibraryHandler) RemovePost(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	libraryID := r.PathValue("libraryID")
	if _, err := h.libraries.GetByIDAndUserID(libraryID, user.ID); err != nil {
		http.NotFound(w, r)
		return
	}
	if err := h.libraries.RemovePost(libraryID, user.ID, r.PathValue("postID")); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/library/"+libraryID, http.StatusSeeOther)
}

// renderPage renders the library page
func (h *LibraryHandler) renderPage(w http.ResponseWriter, data LibraryPageData) {
	libraries, err := h.libraries.GetByUserID(data.User.ID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	data.Libraries = libraries

	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", "library", "library.html"),
	)
	if err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base", data)
}

// postsWithMeta adds display metadata to library posts
func (h *LibraryHandler) postsWithMeta(posts []model.Post) []PostWithMeta {
	postsWithMeta := make([]PostWithMeta, 0, len(posts))
	for _, post := range posts {
		author, err := h.users.GetByID(post.UserID)
		if err != nil {
			author = &model.User{Username: "Inconnu"}
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
			Username:     author.Username,
			LikeCount:    likes,
			DislikeCount: dislikes,
		})
	}
	return postsWithMeta
}

// userFromSession gets the user from the current session
func (h *LibraryHandler) userFromSession(r *http.Request) *model.User {
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

// validLibraryName validates a library name
func validLibraryName(name string) bool {
	return len([]rune(name)) >= 1 && len([]rune(name)) <= 60
}
