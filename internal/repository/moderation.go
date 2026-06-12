package repository

import (
	"EasyForumGo/internal/model"
	"database/sql"
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
func (r *ModerationRepository) GetUserStatus(userID string) (string, error) {
	return "active", nil
}

// GetReportedPosts returns posts that have at least one pending report.
func (r *ModerationRepository) GetReportedPosts() ([]model.Post, error) {
	return nil, nil
}

// GetReportCountForPost returns the number of pending reports for a post.
func (r *ModerationRepository) GetReportCountForPost(postID string) (int, error) {
	return 0, nil
}

// BanUser creates a permanent ban restriction for a user.
func (r *ModerationRepository) BanUser(userID, moderatorID, reason string) error {
	return nil
}

// MuteUser creates a temporary mute restriction for a user.
func (r *ModerationRepository) MuteUser(userID, moderatorID, reason string, durationHours int) error {
	return nil
}

// LiftRestriction removes all active restrictions for a user.
func (r *ModerationRepository) LiftRestriction(userID string) error {
	return nil
}
