package repository

import (
	"EasyForumGo/internal/model"
	"EasyForumGo/pkg/utils"
	"database/sql"
	"time"
)

type ModerationRepository struct {
	db *sql.DB
}

func NewModerationRepository(db *sql.DB) *ModerationRepository {
	return &ModerationRepository{db: db}
}

// GetAllUsers returns all users with their role and restriction status.
func (r *ModerationRepository) GetAllUsers() ([]model.User, error) {
	rows, err := r.db.Query(`SELECT id, email, username, role FROM users`)
	if err != nil {
		return nil, err
	}
		defer rows.Close()
		var users []model.User
		for rows.Next(){
			var user model.User
		if err := rows.Scan(&user.ID, &user.Email, &user.Username, &user.Role); err != nil {
			return nil, err
		}
   users = append(users, user)
	}
	return users, rows.Err()
}


// GetUserStatus returns the active restriction type for a user ("active", "muted", "banned").
func (r *ModerationRepository) GetUserStatus(userID string) (string, error){
    var restrictionType string
    
    err := r.db.QueryRow(`
        SELECT type FROM user_restrictions
        WHERE user_id = ?
        AND type = 'ban'
        AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
        LIMIT 1
    `, userID).Scan(&restrictionType)
    if err == nil {
        return "banned", nil
    }

    err = r.db.QueryRow(`
        SELECT type FROM user_restrictions
        WHERE user_id = ?
        AND type = 'mute'
        AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
        LIMIT 1
    `, userID).Scan(&restrictionType)
    if err == nil {
        return "muted", nil
    }

    return "active", nil
}

func (r *ModerationRepository) GetReportedPosts() ([]model.Post, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT p.id, p.user_id, p.title, p.content, p.created_at
		FROM posts p
		JOIN reports rep ON rep.target_id = p.id
		WHERE rep.target_type = 'post'
		AND rep.status = 'pending'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		var p model.Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.CreatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

// GetReportCountForPost returns the number of pending reports for a post.
func (r *ModerationRepository) GetReportCountForPost(postID string) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM reports
		WHERE target_id = ?
		AND target_type = 'post'
		AND status = 'pending'
	`, postID).Scan(&count)
	return count, err
}

// BanUser creates a permanent ban restriction for a user.
func (r *ModerationRepository) BanUser(userID, moderatorID, reason string) error {
	_, err := r.db.Exec(`
        INSERT INTO user_restrictions (id, user_id, type, reason, expires_at, created_by)
        VALUES (?, ?, 'ban', ?, NULL, ?)
    `, utils.NewUUID(), userID, reason, moderatorID)
    return err
}


// MuteUser creates a temporary mute restriction for a user.
func (r *ModerationRepository) MuteUser(userID, moderatorID, reason string, durationHours int) error {
    expiresAt := time.Now().Add(time.Duration(durationHours) * time.Hour)
    _, err := r.db.Exec(`
        INSERT INTO user_restrictions (id, user_id, type, reason, expires_at, created_by)
        VALUES (?, ?, 'mute', ?, ?, ?)
    `, utils.NewUUID(), userID, reason, expiresAt, moderatorID)
    return err
}

// LiftRestriction removes all active restrictions for a user.
func (r *ModerationRepository) LiftRestriction(userID string) error {
	_, err := r.db.Exec(`DELETE FROM user_restrictions WHERE user_id = ?`, userID)
    return err
}
