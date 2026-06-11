package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"EasyForumGo/internal/model"
	"EasyForumGo/internal/repository"
	"EasyForumGo/pkg/utils"
)

const socialPageSize = 10

type SocialHandler struct {
	users         *repository.UserRepository
	posts         *repository.PostRepository
	follows       *repository.FollowRepository
	likes         *repository.LikeRepository
	sessions      *repository.SessionRepository
	renderer      *PageRenderer
	notifications *repository.NotificationRepository
	blocks        *repository.BlockRepository
}

type PublicProfilePageData struct {
	User         *model.User
	ProfileUser  *model.User
	Posts        []PostWithMeta
	Stats        model.SocialStats
	IsFollowing  bool
	IsOwner      bool
	HasBlocked   bool
	IsBlocked    bool
	Page         int
	PreviousPage int
	NextPage     int
	HasPrevious  bool
	HasNext      bool
}

type SocialListPageData struct {
	User         *model.User
	ProfileUser  *model.User
	Users        []model.User
	Title        string
	ListType     string
	Hidden       bool
	Page         int
	PreviousPage int
	NextPage     int
	HasPrevious  bool
	HasNext      bool
}

type DiscoverPageData struct {
	User        *model.User
	Query       string
	Results     []model.User
	Popular     []model.User
	Suggestions []model.User
}

// NewSocialHandler creates a new instance
func NewSocialHandler(db *sql.DB, renderer *PageRenderer) *SocialHandler {
	return &SocialHandler{
		users:         repository.NewUserRepository(db),
		posts:         repository.NewPostRepository(db),
		follows:       repository.NewFollowRepository(db),
		likes:         repository.NewLikeRepository(db),
		sessions:      repository.NewSessionRepository(db),
		renderer:      renderer,
		notifications: repository.NewNotificationRepository(db),
		blocks:        repository.NewBlockRepository(db),
	}
}

// PublicProfile renders a user's public profile
func (h *SocialHandler) PublicProfile(w http.ResponseWriter, r *http.Request) {
	profileUser, err := h.users.GetByUsername(r.PathValue("username"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	currentUser := h.userFromSession(r)
	page := pageNumber(r)
	posts, err := h.posts.GetByUserIDPaginated(profileUser.ID, socialPageSize+1, (page-1)*socialPageSize)
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	hasNext := len(posts) > socialPageSize
	if hasNext {
		posts = posts[:socialPageSize]
	}
	postCount, _ := h.posts.CountByUserID(profileUser.ID)
	followerCount, _ := h.follows.CountFollowers(profileUser.ID)
	followingCount, _ := h.follows.CountFollowing(profileUser.ID)
	isFollowing := false
	hasBlocked := false
	isBlocked := false
	if currentUser != nil {
		isFollowing, _ = h.follows.IsFollowing(currentUser.ID, profileUser.ID)
		hasBlocked, _ = h.blocks.HasBlocked(currentUser.ID, profileUser.ID)
		blockedBetween, _ := h.blocks.IsBlockedBetween(currentUser.ID, profileUser.ID)
		isBlocked = blockedBetween && !hasBlocked
	}
	h.renderer.Render(w, "social/public_profile.html", PublicProfilePageData{
		User: currentUser, ProfileUser: profileUser, Posts: h.postsWithMeta(posts),
		Stats:       model.SocialStats{PostCount: postCount, FollowerCount: followerCount, FollowingCount: followingCount},
		IsFollowing: isFollowing, IsOwner: currentUser != nil && currentUser.ID == profileUser.ID,
		HasBlocked: hasBlocked, IsBlocked: isBlocked,
		Page: page, PreviousPage: page - 1, NextPage: page + 1, HasPrevious: page > 1, HasNext: hasNext,
	})
}

// Followers renders a user's followers
func (h *SocialHandler) Followers(w http.ResponseWriter, r *http.Request) {
	h.renderUserList(w, r, "followers")
}

// Following renders users followed by a user
func (h *SocialHandler) Following(w http.ResponseWriter, r *http.Request) {
	h.renderUserList(w, r, "following")
}

// Follow creates a following relation
func (h *SocialHandler) Follow(w http.ResponseWriter, r *http.Request) {
	h.changeFollow(w, r, true)
}

// Unfollow removes a following relation
func (h *SocialHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	h.changeFollow(w, r, false)
}

// Block blocks a user
func (h *SocialHandler) Block(w http.ResponseWriter, r *http.Request) {
	h.changeBlock(w, r, true)
}

// Unblock unblocks a user
func (h *SocialHandler) Unblock(w http.ResponseWriter, r *http.Request) {
	h.changeBlock(w, r, false)
}

// Discover renders user discovery suggestions
func (h *SocialHandler) Discover(w http.ResponseWriter, r *http.Request) {
	currentUser := h.userFromSession(r)
	if currentUser == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	popular, _ := h.users.Popular(currentUser.ID, 8)
	suggestions, _ := h.users.SuggestedByLikedCategories(currentUser.ID, 8)
	h.renderer.Render(w, "social/discover.html", DiscoverPageData{User: currentUser, Popular: popular, Suggestions: suggestions})
}

// Search renders user search results
func (h *SocialHandler) Search(w http.ResponseWriter, r *http.Request) {
	currentUser := h.userFromSession(r)
	if currentUser == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	var results []model.User
	if query != "" {
		results, _ = h.users.Search(query, currentUser.ID, 30)
	}
	h.renderer.Render(w, "social/discover.html", DiscoverPageData{User: currentUser, Query: query, Results: results})
}

// renderUserList renders followers or following users
func (h *SocialHandler) renderUserList(w http.ResponseWriter, r *http.Request, listType string) {
	profileUser, err := h.users.GetByUsername(r.PathValue("username"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	currentUser := h.userFromSession(r)
	isOwner := currentUser != nil && currentUser.ID == profileUser.ID
	hidden := !profileUser.FollowsVisible && !isOwner
	page := pageNumber(r)
	var users []model.User
	if !hidden {
		if listType == "followers" {
			users, err = h.follows.ListFollowers(profileUser.ID, socialPageSize+1, (page-1)*socialPageSize)
		} else {
			users, err = h.follows.ListFollowing(profileUser.ID, socialPageSize+1, (page-1)*socialPageSize)
		}
		if err != nil {
			http.Error(w, "Erreur serveur", http.StatusInternalServerError)
			return
		}
	}
	hasNext := len(users) > socialPageSize
	if hasNext {
		users = users[:socialPageSize]
	}
	title := "Abonnés"
	if listType == "following" {
		title = "Abonnements"
	}
	h.renderer.Render(w, "social/user_list.html", SocialListPageData{
		User: currentUser, ProfileUser: profileUser, Users: users, Title: title, ListType: listType,
		Hidden: hidden, Page: page, PreviousPage: page - 1, NextPage: page + 1, HasPrevious: page > 1, HasNext: hasNext,
	})
}

// changeFollow changes a following relation
func (h *SocialHandler) changeFollow(w http.ResponseWriter, r *http.Request, follow bool) {
	currentUser := h.userFromSession(r)
	if currentUser == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	target, err := h.users.GetByUsername(r.PathValue("username"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if target.ID == currentUser.ID {
		http.Error(w, "Vous ne pouvez pas vous suivre.", http.StatusBadRequest)
		return
	}
	if follow {
		err = h.follows.Follow(currentUser.ID, target.ID)
		if errors.Is(err, repository.ErrFollowBlocked) {
			http.Error(w, "Cette relation est bloquée.", http.StatusForbidden)
			return
		}
		if err == nil {
			_ = h.notifications.Create(&model.Notification{
				ID: utils.NewUUID(), UserID: target.ID, ActorID: currentUser.ID,
				Type: "new_follower", CreatedAt: time.Now(),
			})
		}
	} else {
		err = h.follows.Unfollow(currentUser.ID, target.ID)
	}
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	redirect := strings.TrimSpace(r.FormValue("redirect"))
	if !strings.HasPrefix(redirect, "/") || strings.HasPrefix(redirect, "//") {
		redirect = "/user/" + target.Username
	}
	http.Redirect(w, r, redirect, http.StatusSeeOther)
}

// changeBlock changes a user block
func (h *SocialHandler) changeBlock(w http.ResponseWriter, r *http.Request, block bool) {
	currentUser := h.userFromSession(r)
	if currentUser == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	target, err := h.users.GetByUsername(r.PathValue("username"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if target.ID == currentUser.ID {
		http.Error(w, "Vous ne pouvez pas vous bloquer.", http.StatusBadRequest)
		return
	}
	if block {
		err = h.blocks.Block(currentUser.ID, target.ID)
	} else {
		err = h.blocks.Unblock(currentUser.ID, target.ID)
	}
	if err != nil {
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/user/"+target.Username, http.StatusSeeOther)
}

// postsWithMeta adds display metadata to posts
func (h *SocialHandler) postsWithMeta(posts []model.Post) []PostWithMeta {
	result := make([]PostWithMeta, 0, len(posts))
	for _, post := range posts {
		author, _ := h.users.GetByID(post.UserID)
		username := "Inconnu"
		if author != nil {
			username = author.Username
		}
		likes, _ := h.likes.CountPostLikes(post.ID)
		dislikes, _ := h.likes.CountPostDislikes(post.ID)
		result = append(result, PostWithMeta{ID: post.ID, UserID: post.UserID, Title: post.Title, Content: post.Content, ImagePath: post.ImagePath, CreatedAt: post.CreatedAt, Username: username, LikeCount: likes, DislikeCount: dislikes})
	}
	return result
}

// userFromSession gets the user from the current session
func (h *SocialHandler) userFromSession(r *http.Request) *model.User {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}
	session, err := h.sessions.GetByToken(cookie.Value)
	if err != nil || session.ExpiresAt.Before(time.Now()) {
		return nil
	}
	user, _ := h.users.GetByID(session.UserID)
	return user
}

// pageNumber returns a validated pagination page
func pageNumber(r *http.Request) int {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		return 1
	}
	return page
}
