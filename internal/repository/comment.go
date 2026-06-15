package repository

import (
	"database/sql"

	"EasyForumGo/internal/model"
)

type CommentRepository struct {
	db *sql.DB
}

// NewCommentRepository creates a new instance
func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// GetByPostID gets stored data
func (r *CommentRepository) GetByPostID(postID string) ([]model.Comment, error) {
	rows, err := r.db.Query(`SELECT id, post_id, user_id, content, created_at, updated_at FROM comments WHERE post_id = ? AND status = 'approved' ORDER BY created_at ASC`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []model.Comment
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Content, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// GetByUserID gets stored data
func (r *CommentRepository) GetByUserID(userID string) ([]model.Comment, error) {
	rows, err := r.db.Query(`SELECT id, post_id, user_id, content, created_at, updated_at FROM comments WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []model.Comment
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Content, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

// CountByUserID counts comments created by a user
func (r *CommentRepository) CountByUserID(userID string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM comments WHERE user_id = ?`, userID).Scan(&count)
	return count, err
}

// Create creates a new record
func (r *CommentRepository) Create(c *model.Comment) error {
	_, err := r.db.Exec(`INSERT INTO comments (id, post_id, user_id, content, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.PostID, c.UserID, c.Content, c.Status, c.CreatedAt, c.UpdatedAt)
	return err
}

// Update updates an existing record
func (r *CommentRepository) Update(c *model.Comment) error {
	_, err := r.db.Exec(`UPDATE comments SET content = ?, updated_at = ? WHERE id = ?`, c.Content, c.UpdatedAt, c.ID)
	return err
}

// GetByID gets stored data
func (r *CommentRepository) GetByID(id string) (*model.Comment, error) {
	var c model.Comment
	err := r.db.QueryRow(`SELECT id, post_id, user_id, content, created_at, updated_at FROM comments WHERE id = ?`, id).
		Scan(&c.ID, &c.PostID, &c.UserID, &c.Content, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Delete deletes an existing record
func (r *CommentRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM comments WHERE id = ?`, id)
	return err
}
