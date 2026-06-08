package repository

import (
	"database/sql"

	"ForumJS/internal/model"
)

type PostRepository struct {
	db *sql.DB
}

// NewPostRepository creates a new instance
func NewPostRepository(db *sql.DB) *PostRepository {
	return &PostRepository{db: db}
}

// GetByID gets stored data
func (r *PostRepository) GetByID(id string) (*model.Post, error) {
	var p model.Post
	err := r.db.QueryRow(`SELECT id, user_id, title, content, image_path, created_at, updated_at FROM posts WHERE id = ?`, id).
		Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.ImagePath, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetAll gets stored data
func (r *PostRepository) GetAll() ([]model.Post, error) {
	rows, err := r.db.Query(`SELECT id, user_id, title, content, image_path, created_at, updated_at FROM posts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		var p model.Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.ImagePath, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

// GetByUserID gets stored data
func (r *PostRepository) GetByUserID(userID string) ([]model.Post, error) {
	rows, err := r.db.Query(`SELECT id, user_id, title, content, image_path, created_at, updated_at FROM posts WHERE user_id = ? ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		var p model.Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.ImagePath, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

// GetByCategory gets stored data
func (r *PostRepository) GetByCategory(categoryID string) ([]model.Post, error) {
	rows, err := r.db.Query(`SELECT p.id, p.user_id, p.title, p.content, p.image_path, p.created_at, p.updated_at FROM posts p JOIN post_categories pc ON p.id = pc.post_id WHERE pc.category_id = ? ORDER BY p.created_at DESC`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		var p model.Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.ImagePath, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

// Create creates a new record
func (r *PostRepository) Create(p *model.Post) error {
	_, err := r.db.Exec(`INSERT INTO posts (id, user_id, title, content, image_path, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.UserID, p.Title, p.Content, p.ImagePath, p.CreatedAt, p.UpdatedAt)
	return err
}

// Update updates an existing record
func (r *PostRepository) Update(p *model.Post) error {
	_, err := r.db.Exec(`UPDATE posts SET title = ?, content = ?, image_path = ?, updated_at = ? WHERE id = ?`,
		p.Title, p.Content, p.ImagePath, p.UpdatedAt, p.ID)
	return err
}

// Delete deletes an existing record
func (r *PostRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM posts WHERE id = ?`, id)
	return err
}
