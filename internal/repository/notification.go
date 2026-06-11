package repository

import (
	"EasyForumGo/internal/model"
	"database/sql"
	"time"
)

type NotificationRepository struct {
	db *sql.DB
}

// create new instance
func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create creates a new record
func (r *NotificationRepository) Create(n *model.Notification) error {
	var postID, commentID interface{}
	if n.PostID != "" {
		postID = n.PostID
	}
	if n.CommentID != "" {
		commentID = n.CommentID
	}
	_, err := r.db.Exec(
		`INSERT INTO notifications (id, user_id, actor_id, type, post_id, comment_id, is_read, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, 0, ?)`,
		n.ID, n.UserID, n.ActorID, n.Type, postID, commentID, n.CreatedAt,
	)
	return err
}

// GetByUserID gets stored data with actor username and post title
func (r *NotificationRepository) GetByUserID(userID string) ([]model.Notification, error) {
	rows, err := r.db.Query(`
		SELECT n.id, n.user_id, n.actor_id, n.type, n.post_id, n.comment_id, n.is_read, n.created_at,
		       u.username, COALESCE(p.title, '') AS post_title
		FROM notifications n
		JOIN users u ON u.id = n.actor_id
		LEFT JOIN posts p ON p.id = n.post_id
		WHERE n.user_id = ?
		ORDER BY n.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []model.Notification
	for rows.Next() {
		var n model.Notification
		var postID, commentID sql.NullString
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.ActorID, &n.Type, &postID, &commentID, &n.IsRead, &n.CreatedAt,
			&n.ActorUsername, &n.PostTitle,
		); err != nil {
			return nil, err
		}
		n.PostID = postID.String
		n.CommentID = commentID.String
		notifs = append(notifs, n)
	}
	return notifs, rows.Err()
}

// CountUnread counts stored records
func (r *NotificationRepository) CountUnread(userID string) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND is_read = 0`, userID,
	).Scan(&count)
	return count, err
}

// MarkAllAsRead updates stored records
func (r *NotificationRepository) MarkAllRead(userID string) error {
	_, err := r.db.Exec(`UPDATE notifications SET is_read = 1 WHERE user_id = ?`, userID)
	return err
}

// MarkAsRead updates stored record
func (r *NotificationRepository) MarkAsRead(id string) error {
	_, err := r.db.Exec(`UPDATE notifications SET is_read = 1 WHERE id = ?`, id)
	return err
}

// Delete deletes stored record
func (r *NotificationRepository) DeleteOlderThan(t time.Time) error {
	_, err := r.db.Exec(`DELETE FROM notifications WHERE created_at < ?`, t)
	return err
}
