package repository

import (
	"database/sql"
	"errors"
	"fmt"

	"EasyForumGo/internal/model"
)

type FollowRepository struct {
	db *sql.DB
}

// NewFollowRepository creates a new instance
func NewFollowRepository(db *sql.DB) *FollowRepository {
	return &FollowRepository{db: db}
}

// Follow creates a following relation
func (r *FollowRepository) Follow(followerID, followedID string) error {
	if followerID == followedID {
		return errors.New("users cannot follow themselves")
	}
	var blocked int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM user_blocks WHERE (blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)`,
		followerID, followedID, followedID, followerID).Scan(&blocked); err != nil {
		return err
	}
	if blocked > 0 {
		return errors.New("blocked users cannot follow each other")
	}
	_, err := r.db.Exec(`INSERT OR IGNORE INTO user_follows (follower_id, followed_id) VALUES (?, ?)`, followerID, followedID)
	return err
}

// Unfollow removes a following relation
func (r *FollowRepository) Unfollow(followerID, followedID string) error {
	_, err := r.db.Exec(`DELETE FROM user_follows WHERE follower_id = ? AND followed_id = ?`, followerID, followedID)
	return err
}

// IsFollowing reports whether a following relation exists
func (r *FollowRepository) IsFollowing(followerID, followedID string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM user_follows WHERE follower_id = ? AND followed_id = ?`, followerID, followedID).Scan(&count)
	return count > 0, err
}

// CountFollowers counts a user's followers
func (r *FollowRepository) CountFollowers(userID string) (int, error) {
	return r.count(`SELECT COUNT(*) FROM user_follows WHERE followed_id = ?`, userID)
}

// CountFollowing counts users followed by a user
func (r *FollowRepository) CountFollowing(userID string) (int, error) {
	return r.count(`SELECT COUNT(*) FROM user_follows WHERE follower_id = ?`, userID)
}

// ListFollowers returns a paginated follower list
func (r *FollowRepository) ListFollowers(userID string, limit, offset int) ([]model.User, error) {
	return r.list(`
		SELECT u.id, u.email, u.username, u.password, u.role, u.profile_picture, u.biography, u.follows_visible, u.created_at
		FROM users u JOIN user_follows f ON f.follower_id = u.id
		WHERE f.followed_id = ? ORDER BY f.created_at DESC LIMIT ? OFFSET ?`, userID, limit, offset)
}

// ListFollowing returns a paginated following list
func (r *FollowRepository) ListFollowing(userID string, limit, offset int) ([]model.User, error) {
	return r.list(`
		SELECT u.id, u.email, u.username, u.password, u.role, u.profile_picture, u.biography, u.follows_visible, u.created_at
		FROM users u JOIN user_follows f ON f.followed_id = u.id
		WHERE f.follower_id = ? ORDER BY f.created_at DESC LIMIT ? OFFSET ?`, userID, limit, offset)
}

// FollowerIDs returns the ids of a user's followers
func (r *FollowRepository) FollowerIDs(userID string) ([]string, error) {
	rows, err := r.db.Query(`SELECT follower_id FROM user_follows WHERE followed_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// RemoveBetween removes following relations in both directions
func (r *FollowRepository) RemoveBetween(firstID, secondID string) error {
	_, err := r.db.Exec(`DELETE FROM user_follows WHERE (follower_id = ? AND followed_id = ?) OR (follower_id = ? AND followed_id = ?)`,
		firstID, secondID, secondID, firstID)
	return err
}

// count counts matching following relations
func (r *FollowRepository) count(query, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(query, userID).Scan(&count)
	return count, err
}

// list returns users matching a following relation query
func (r *FollowRepository) list(query, userID string, limit, offset int) ([]model.User, error) {
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("invalid limit")
	}
	rows, err := r.db.Query(query, userID, limit, max(offset, 0))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []model.User
	for rows.Next() {
		var user model.User
		if err := scanUser(rows, &user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}
