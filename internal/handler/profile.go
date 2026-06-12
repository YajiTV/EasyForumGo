package handler

import (
	"database/sql"
	"net/http"
	"time"

	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
)

type ProfileHandler struct {
	users    *repository.UserRepository
	posts    *repository.PostRepository
	comments *repository.CommentRepository
	likes    *repository.LikeRepository
	sessions *repository.SessionRepository
	renderer *PageRenderer
}

// NewProfileHandler creates a new instance
func NewProfileHandler(db *sql.DB, renderer *PageRenderer) *ProfileHandler {
	return &ProfileHandler{
		users:    repository.NewUserRepository(db),
		posts:    repository.NewPostRepository(db),
		comments: repository.NewCommentRepository(db),
		likes:    repository.NewLikeRepository(db),
		sessions: repository.NewSessionRepository(db),
		renderer: renderer,
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

type ProfileActivityStats struct {
	PostCount    int
	CommentCount int
	LikeCount    int
	DislikeCount int
}

type ProfileActivityPageData struct {
	User          *model.User
	CreatedPosts  []PostWithMeta
	LikedPosts    []PostWithMeta
	DislikedPosts []PostWithMeta
	Comments      []ProfileCommentWithPost
	Stats         ProfileActivityStats
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

	h.renderProfile(w, ProfilePageData{
		User:         user,
		Comments:     h.commentsWithPost(comments),
		ActiveTab:    "my-comments",
		SectionTitle: "Mes commentaires",
		EmptyTitle:   "Aucun commentaire",
		EmptyMessage: "Vos commentaires apparaîtront ici.",
	})
}

// Activity renders the current user's activity summary
func (h *ProfileHandler) Activity(w http.ResponseWriter, r *http.Request) {
	user := h.userFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	createdPosts, err := h.posts.GetByUserID(user.ID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	likedPosts, err := h.likes.GetPostsByUserVote(user.ID, true)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	dislikedPosts, err := h.likes.GetPostsByUserVote(user.ID, false)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	comments, err := h.comments.GetByUserID(user.ID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	stats, err := h.activityStats(user.ID)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	h.renderActivity(w, ProfileActivityPageData{
		User:          user,
		CreatedPosts:  h.postsWithMeta(createdPosts),
		LikedPosts:    h.postsWithMeta(likedPosts),
		DislikedPosts: h.postsWithMeta(dislikedPosts),
		Comments:      h.commentsWithPost(comments),
		Stats:         stats,
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

// renderProfile renders the requested page
func (h *ProfileHandler) renderProfile(w http.ResponseWriter, data ProfilePageData) {
	h.renderer.Render(w, "profile/profile.html", data)
}

// renderActivity renders the activity page
func (h *ProfileHandler) renderActivity(w http.ResponseWriter, data ProfileActivityPageData) {
	h.renderer.Render(w, "profile/activity.html", data)
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

// commentsWithPost adds post details to comments
func (h *ProfileHandler) commentsWithPost(comments []model.Comment) []ProfileCommentWithPost {
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
	return commentsWithPost
}

// activityStats gets the current user's activity counters
func (h *ProfileHandler) activityStats(userID string) (ProfileActivityStats, error) {
	postCount, err := h.posts.CountByUserID(userID)
	if err != nil {
		return ProfileActivityStats{}, err
	}
	commentCount, err := h.comments.CountByUserID(userID)
	if err != nil {
		return ProfileActivityStats{}, err
	}
	likeCount, err := h.likes.CountPostVotesByUserID(userID, true)
	if err != nil {
		return ProfileActivityStats{}, err
	}
	commentLikeCount, err := h.likes.CountCommentVotesByUserID(userID, true)
	if err != nil {
		return ProfileActivityStats{}, err
	}
	dislikeCount, err := h.likes.CountPostVotesByUserID(userID, false)
	if err != nil {
		return ProfileActivityStats{}, err
	}
	commentDislikeCount, err := h.likes.CountCommentVotesByUserID(userID, false)
	if err != nil {
		return ProfileActivityStats{}, err
	}
	return ProfileActivityStats{
		PostCount:    postCount,
		CommentCount: commentCount,
		LikeCount:    likeCount + commentLikeCount,
		DislikeCount: dislikeCount + commentDislikeCount,
	}, nil
}
