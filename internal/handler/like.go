package handler

import (
	"database/sql"
	"net/http"
	"time"

	"ForumJS/internal/model"
	"ForumJS/internal/repository"
	"ForumJS/pkg/utils"
)

type LikeHandler struct {
	likes    *repository.LikeRepository
	posts    *repository.PostRepository
	comments *repository.CommentRepository
	sessions *repository.SessionRepository
	users    *repository.UserRepository
}

// NewLikeHandler creates a new instance
func NewLikeHandler(db *sql.DB) *LikeHandler {
	return &LikeHandler{
		likes:    repository.NewLikeRepository(db),
		posts:    repository.NewPostRepository(db),
		comments: repository.NewCommentRepository(db),
		sessions: repository.NewSessionRepository(db),
		users:    repository.NewUserRepository(db),
	}
}

// LikePost handles a like request
func (h *LikeHandler) LikePost(w http.ResponseWriter, r *http.Request) {
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

	h.togglePostVote(w, r, user.ID, postID, true)
}

// DislikePost handles a dislike request
func (h *LikeHandler) DislikePost(w http.ResponseWriter, r *http.Request) {
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

	h.togglePostVote(w, r, user.ID, postID, false)
}

// togglePostVote toggles the requested vote
func (h *LikeHandler) togglePostVote(w http.ResponseWriter, r *http.Request, userID, postID string, isLike bool) {
	existing, err := h.likes.GetUserPostLike(postID, userID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	if existing == nil {
		l := &model.PostLike{
			ID:        utils.NewUUID(),
			PostID:    postID,
			UserID:    userID,
			IsLike:    isLike,
			CreatedAt: time.Now(),
		}
		if err := h.likes.CreatePostLike(l); err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
	} else if existing.IsLike == isLike {
		if err := h.likes.DeletePostLike(postID, userID); err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.likes.UpdatePostLike(postID, userID, isLike); err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}

// LikeComment handles a like request
func (h *LikeHandler) LikeComment(w http.ResponseWriter, r *http.Request) {
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

	h.toggleCommentVote(w, r, user.ID, commentID, comment.PostID, true)
}

// DislikeComment handles a dislike request
func (h *LikeHandler) DislikeComment(w http.ResponseWriter, r *http.Request) {
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

	h.toggleCommentVote(w, r, user.ID, commentID, comment.PostID, false)
}

// toggleCommentVote toggles the requested vote
func (h *LikeHandler) toggleCommentVote(w http.ResponseWriter, r *http.Request, userID, commentID, postID string, isLike bool) {
	existing, err := h.likes.GetUserCommentLike(commentID, userID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	if existing == nil {
		l := &model.CommentLike{
			ID:        utils.NewUUID(),
			CommentID: commentID,
			UserID:    userID,
			IsLike:    isLike,
			CreatedAt: time.Now(),
		}
		if err := h.likes.CreateCommentLike(l); err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
	} else if existing.IsLike == isLike {
		if err := h.likes.DeleteCommentLike(commentID, userID); err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
	} else {
		if err := h.likes.UpdateCommentLike(commentID, userID, isLike); err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/post/"+postID, http.StatusSeeOther)
}

// userFromSession gets the user from the current session
func (h *LikeHandler) userFromSession(r *http.Request) *model.User {
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
