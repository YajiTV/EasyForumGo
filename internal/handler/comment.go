package handler

import (
	"database/sql"
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

const maxCommentFormSize = 16 << 10

type CommentHandler struct {
	comments      *repository.CommentRepository
	posts         *repository.PostRepository
	sessions      *repository.SessionRepository
	users         *repository.UserRepository
	notifications *repository.NotificationRepository
}

// NewCommentHandler creates a new instance
func NewCommentHandler(db *sql.DB) *CommentHandler {
	return &CommentHandler{
		comments:      repository.NewCommentRepository(db),
		posts:         repository.NewPostRepository(db),
		sessions:      repository.NewSessionRepository(db),
		users:         repository.NewUserRepository(db),
		notifications: repository.NewNotificationRepository(db),
	}
}

// CreateComment creates a new record
func (h *CommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	postID := r.PathValue("id")
	if _, err := h.posts.GetByID(postID); err != nil {
		http.NotFound(w, r)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxCommentFormSize)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if validationErrors := validator.ValidateComment(validator.CommentInput{Content: content}); validationErrors.HasErrors() {
		http.Error(w, firstValidationMessage(validationErrors), http.StatusBadRequest)
		return
	}

	now := time.Now()
	c := &model.Comment{
		ID:        utils.NewUUID(),
		PostID:    postID,
		UserID:    user.ID,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := h.comments.Create(c); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// notification : seulement si ce n'est pas son propre post
	if post, err := h.posts.GetByID(postID); err == nil && post.UserID != user.ID {
		n := &model.Notification{
			ID:        utils.NewUUID(),
			UserID:    post.UserID,
			ActorID:   user.ID,
			Type:      "comment",
			PostID:    postID,
			CommentID: c.ID,
			CreatedAt: time.Now(),
		}
		_ = h.notifications.Create(n)
	}

	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)

}

// DeleteComment deletes an existing record
func (h *CommentHandler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	commentID := r.PathValue("id")
	comment, err := h.comments.GetByID(commentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if comment.UserID != user.ID {
		http.Error(w, "Interdit", http.StatusForbidden)
		return
	}

	if err := h.comments.Delete(commentID); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/post/"+comment.PostID, http.StatusSeeOther)
}

type editCommentData struct {
	User    *model.User
	Comment *model.Comment
	Error   string
}

type deleteCommentData struct {
	User    *model.User
	Comment *model.Comment
}

// ShowDeleteConfirmation renders the requested page
func (h *CommentHandler) ShowDeleteConfirmation(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	commentID := r.PathValue("id")
	comment, err := h.comments.GetByID(commentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if comment.UserID != user.ID {
		http.Error(w, "Interdit", http.StatusForbidden)
		return
	}

	h.renderTemplate(w, "comment/delete_comment.html", deleteCommentData{
		User:    user,
		Comment: comment,
	})
}

// ShowEditForm renders the requested page
func (h *CommentHandler) ShowEditForm(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	commentID := r.PathValue("id")
	comment, err := h.comments.GetByID(commentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if comment.UserID != user.ID {
		http.Error(w, "Interdit", http.StatusForbidden)
		return
	}

	h.renderTemplate(w, "comment/edit_comment.html", editCommentData{
		User:    user,
		Comment: comment,
	})
}

// EditComment updates a comment from the author
func (h *CommentHandler) EditComment(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	commentID := r.PathValue("id")
	comment, err := h.comments.GetByID(commentID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if comment.UserID != user.ID {
		http.Error(w, "Interdit", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxCommentFormSize)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))

	renderErr := func(msg string) {
		h.renderTemplate(w, "comment/edit_comment.html", editCommentData{
			User:    user,
			Comment: comment,
			Error:   msg,
		})
	}

	if validationErrors := validator.ValidateComment(validator.CommentInput{Content: content}); validationErrors.HasErrors() {
		renderErr(firstValidationMessage(validationErrors))
		return
	}

	comment.Content = content
	comment.UpdatedAt = time.Now()

	if err := h.comments.Update(comment); err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/post/"+comment.PostID, http.StatusSeeOther)
}

// userFromSession gets the user from the current session
func (h *CommentHandler) userFromSession(r *http.Request) *model.User {
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

// renderTemplate renders the requested page
func (h *CommentHandler) renderTemplate(w http.ResponseWriter, name string, data any) {
	tmpl, err := template.ParseFiles(
		filepath.Join("web", "templates", "layout", "base.html"),
		filepath.Join("web", "templates", name),
	)
	if err != nil {
		http.Error(w, "Erreur template", http.StatusInternalServerError)
		return
	}
	tmpl.ExecuteTemplate(w, "base", data)
}
