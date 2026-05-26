package repository

import (
	"database/sql"

	"ForumJS/internal/model"
)

type CommentRepository struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) GetByPostID(postID string) ([]model.Comment, error) {
	rows, err := r.db.Query(`SELECT id, post_id, user_id, content, created_at, updated_at FROM comments WHERE post_id = ? ORDER BY created_at ASC`, postID)
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

func (r *CommentRepository) Create(c *model.Comment) error {
	_, err := r.db.Exec(`INSERT INTO comments (id, post_id, user_id, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID, c.PostID, c.UserID, c.Content, c.CreatedAt, c.UpdatedAt)
	return err
}

func (r *CommentRepository) Update(c *model.Comment) error {
	_, err := r.db.Exec(`UPDATE comments SET content = ?, updated_at = ? WHERE id = ?`, c.Content, c.UpdatedAt, c.ID)
	return err
}

func (r *CommentRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM comments WHERE id = ?`, id)
	return err
}
