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
)

type CommentHandler struct {
	comments *repository.CommentRepository
	posts    *repository.PostRepository
	sessions *repository.SessionRepository
	users    *repository.UserRepository
}

func NewCommentHandler(db *sql.DB) *CommentHandler {
	return &CommentHandler{
		comments: repository.NewCommentRepository(db),
		posts:    repository.NewPostRepository(db),
		sessions: repository.NewSessionRepository(db),
		users:    repository.NewUserRepository(db),
	}
}

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

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
		return
	}
	if len(content) > 2000 {
		http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
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

	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}

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

	if content == "" {
		renderErr("Le contenu ne peut pas être vide.")
		return
	}
	if len(content) > 2000 {
		renderErr("Le contenu ne peut pas dépasser 2000 caractères.")
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
