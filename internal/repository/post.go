package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"EasyForumGo/internal/model"
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
	rows, err := r.db.Query(`SELECT id, user_id, title, content, image_path, created_at, updated_at FROM posts WHERE status = 'approved' ORDER BY created_at DESC`)
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

// GetAllPaginated gets a page of posts
func (r *PostRepository) GetAllPaginated(limit, offset int) ([]model.Post, error) {
	return r.queryPosts(`SELECT id, user_id, title, content, image_path, created_at, updated_at
		FROM posts WHERE status = 'approved' ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, max(offset, 0))
}

// GetByUserID gets stored data
func (r *PostRepository) GetByUserID(userID string) ([]model.Post, error) {
	rows, err := r.db.Query(`SELECT id, user_id, title, content, image_path, created_at, updated_at FROM posts WHERE user_id = ? AND status = 'approved' ORDER BY created_at DESC`, userID)
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

// GetByUserIDPaginated gets a page of a user's posts
func (r *PostRepository) GetByUserIDPaginated(userID string, limit, offset int) ([]model.Post, error) {
	return r.queryPosts(`SELECT id, user_id, title, content, image_path, created_at, updated_at
		FROM posts WHERE user_id = ? AND status = 'approved' ORDER BY created_at DESC LIMIT ? OFFSET ?`, userID, limit, max(offset, 0))
}

// GetFollowing gets a page of posts from followed users
func (r *PostRepository) GetFollowing(userID string, limit, offset int) ([]model.Post, error) {
	return r.queryPosts(`SELECT p.id, p.user_id, p.title, p.content, p.image_path, p.created_at, p.updated_at
		FROM posts p JOIN user_follows f ON f.followed_id = p.user_id
		WHERE f.follower_id = ? AND p.status = 'approved' ORDER BY p.created_at DESC LIMIT ? OFFSET ?`, userID, limit, max(offset, 0))
}

// CountFollowing counts posts from followed users
func (r *PostRepository) CountFollowing(userID string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM posts p JOIN user_follows f ON f.followed_id = p.user_id WHERE f.follower_id = ?`, userID).Scan(&count)
	return count, err
}

// CountByUserID counts posts created by a user
func (r *PostRepository) CountByUserID(userID string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM posts WHERE user_id = ?`, userID).Scan(&count)
	return count, err
}

// GetByCategory gets stored data
func (r *PostRepository) GetByCategory(categoryID string) ([]model.Post, error) {
	rows, err := r.db.Query(`SELECT p.id, p.user_id, p.title, p.content, p.image_path, p.created_at, p.updated_at FROM posts p JOIN post_categories pc ON p.id = pc.post_id WHERE pc.category_id = ? AND p.status = 'approved' ORDER BY p.created_at DESC`, categoryID)
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

// GetByCategoryPaginated gets a page of posts in a category
func (r *PostRepository) GetByCategoryPaginated(categoryID string, limit, offset int) ([]model.Post, error) {
	return r.queryPosts(`SELECT p.id, p.user_id, p.title, p.content, p.image_path, p.created_at, p.updated_at
		FROM posts p JOIN post_categories pc ON p.id = pc.post_id
		WHERE pc.category_id = ? AND p.status = 'approved' ORDER BY p.created_at DESC LIMIT ? OFFSET ?`, categoryID, limit, max(offset, 0))
}

// Search returns posts matching a query and optional category
func (r *PostRepository) Search(query, categoryID, sort string, limit int) ([]model.Post, error) {
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("invalid limit")
	}

	orderBy := `CASE
			WHEN LOWER(p.title) = LOWER(?) THEN 0
			WHEN LOWER(p.title) LIKE LOWER(?) ESCAPE '\' THEN 1
			ELSE 2
		END, p.created_at DESC`
	switch sort {
	case "recent":
		orderBy = "p.created_at DESC"
	case "popular":
		orderBy = `(SELECT COUNT(*) FROM post_likes pl WHERE pl.post_id = p.id AND pl.is_like = 1) DESC, p.created_at DESC`
	}

	escapedQuery := escapePostLike(query)
	containsQuery := "%" + escapedQuery + "%"
	startsWithQuery := escapedQuery + "%"
	args := []any{query, containsQuery, containsQuery, containsQuery}
	categoryFilter := ""
	if categoryID != "" {
		categoryFilter = ` AND EXISTS (
			SELECT 1 FROM post_categories filtered_categories
			WHERE filtered_categories.post_id = p.id AND filtered_categories.category_id = ?
		)`
		args = append(args, categoryID)
	}
	if sort != "recent" && sort != "popular" {
		args = append(args, query, startsWithQuery)
	}
	args = append(args, limit)

	return r.queryPosts(`SELECT p.id, p.user_id, p.title, p.content, p.image_path, p.created_at, p.updated_at
		FROM posts p
		JOIN users author ON author.id = p.user_id
		WHERE (
			? = ''
			OR p.title LIKE ? ESCAPE '\'
			OR p.content LIKE ? ESCAPE '\'
			OR author.username LIKE ? ESCAPE '\'
		)`+categoryFilter+`
		ORDER BY `+orderBy+` LIMIT ?`, args...)
}

// Create creates a new record
func (r *PostRepository) Create(p *model.Post) error {
	_, err := r.db.Exec(`INSERT INTO posts (id, user_id, title, content, image_path, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.UserID, p.Title, p.Content, p.ImagePath, p.CreatedAt, p.UpdatedAt)
	return err
}

// CreateWithCategories creates a post and its category associations atomically
func (r *PostRepository) CreateWithCategories(p *model.Post, categoryIDs []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin create post: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`INSERT INTO posts (id, user_id, title, content, image_path, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.UserID, p.Title, p.Content, p.ImagePath, p.Status, p.CreatedAt, p.UpdatedAt); err != nil {
		return fmt.Errorf("insert post: %w", err)
	}
	for _, categoryID := range categoryIDs {
		if _, err := tx.Exec(`INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)`, p.ID, categoryID); err != nil {
			return fmt.Errorf("insert post category: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create post: %w", err)
	}
	return nil
}

// Update updates an existing record
func (r *PostRepository) Update(p *model.Post) error {
	_, err := r.db.Exec(`UPDATE posts SET title = ?, content = ?, image_path = ?, updated_at = ? WHERE id = ?`,
		p.Title, p.Content, p.ImagePath, p.UpdatedAt, p.ID)
	return err
}

// UpdateWithCategories updates a post and its category associations atomically
func (r *PostRepository) UpdateWithCategories(p *model.Post, categoryIDs []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin update post: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE posts SET title = ?, content = ?, image_path = ?, updated_at = ? WHERE id = ?`,
		p.Title, p.Content, p.ImagePath, p.UpdatedAt, p.ID); err != nil {
		return fmt.Errorf("update post: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM post_categories WHERE post_id = ?`, p.ID); err != nil {
		return fmt.Errorf("delete post categories: %w", err)
	}
	for _, categoryID := range categoryIDs {
		if _, err := tx.Exec(`INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)`, p.ID, categoryID); err != nil {
			return fmt.Errorf("insert post category: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update post: %w", err)
	}
	return nil
}

// Delete deletes an existing record
func (r *PostRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM posts WHERE id = ?`, id)
	return err
}

// queryPosts runs a post list query
func (r *PostRepository) queryPosts(query string, args ...any) ([]model.Post, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []model.Post
	for rows.Next() {
		var post model.Post
		if err := rows.Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.ImagePath, &post.CreatedAt, &post.UpdatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

// escapePostLike escapes sqlite LIKE wildcard characters
func escapePostLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}
