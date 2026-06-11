package repository

import (
	"database/sql"

	"EasyForumGo/internal/model"
)

type LibraryRepository struct {
	db *sql.DB
}

// NewLibraryRepository creates a new instance
func NewLibraryRepository(db *sql.DB) *LibraryRepository {
	return &LibraryRepository{db: db}
}

// GetByUserID gets a user's libraries
func (r *LibraryRepository) GetByUserID(userID string) ([]model.Library, error) {
	rows, err := r.db.Query(`
		SELECT l.id, l.user_id, l.name, COUNT(lp.post_id), l.created_at
		FROM libraries l
		LEFT JOIN library_posts lp ON lp.library_id = l.id
		WHERE l.user_id = ?
		GROUP BY l.id
		ORDER BY l.created_at ASC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var libraries []model.Library
	for rows.Next() {
		var library model.Library
		if err := rows.Scan(&library.ID, &library.UserID, &library.Name, &library.PostCount, &library.CreatedAt); err != nil {
			return nil, err
		}
		libraries = append(libraries, library)
	}
	return libraries, rows.Err()
}

// GetByIDAndUserID gets a library owned by a user
func (r *LibraryRepository) GetByIDAndUserID(id, userID string) (*model.Library, error) {
	var library model.Library
	err := r.db.QueryRow(`
		SELECT l.id, l.user_id, l.name, COUNT(lp.post_id), l.created_at
		FROM libraries l
		LEFT JOIN library_posts lp ON lp.library_id = l.id
		WHERE l.id = ? AND l.user_id = ?
		GROUP BY l.id
	`, id, userID).Scan(&library.ID, &library.UserID, &library.Name, &library.PostCount, &library.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &library, nil
}

// Create creates a new library
func (r *LibraryRepository) Create(library *model.Library) error {
	_, err := r.db.Exec(
		`INSERT INTO libraries (id, user_id, name, created_at) VALUES (?, ?, ?, ?)`,
		library.ID, library.UserID, library.Name, library.CreatedAt,
	)
	return err
}

// Rename renames a library owned by a user
func (r *LibraryRepository) Rename(id, userID, name string) error {
	_, err := r.db.Exec(`UPDATE libraries SET name = ? WHERE id = ? AND user_id = ?`, name, id, userID)
	return err
}

// Delete deletes a library owned by a user
func (r *LibraryRepository) Delete(id, userID string) error {
	_, err := r.db.Exec(`DELETE FROM libraries WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

// GetPosts gets posts from a library owned by a user
func (r *LibraryRepository) GetPosts(libraryID, userID string) ([]model.Post, error) {
	rows, err := r.db.Query(`
		SELECT p.id, p.user_id, p.title, p.content, p.image_path, p.created_at, p.updated_at
		FROM posts p
		JOIN library_posts lp ON lp.post_id = p.id
		JOIN libraries l ON l.id = lp.library_id
		WHERE l.id = ? AND l.user_id = ?
		ORDER BY lp.created_at DESC
	`, libraryID, userID)
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

// ContainsPost reports whether a user's library contains a post
func (r *LibraryRepository) ContainsPost(libraryID, userID, postID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1
			FROM library_posts lp
			JOIN libraries l ON l.id = lp.library_id
			WHERE lp.library_id = ? AND l.user_id = ? AND lp.post_id = ?
		)
	`, libraryID, userID, postID).Scan(&exists)
	return exists, err
}

// AddPost adds a post to a user's library
func (r *LibraryRepository) AddPost(libraryID, userID, postID string) error {
	_, err := r.db.Exec(`
		INSERT OR IGNORE INTO library_posts (library_id, post_id)
		SELECT l.id, p.id
		FROM libraries l, posts p
		WHERE l.id = ? AND l.user_id = ? AND p.id = ?
	`, libraryID, userID, postID)
	return err
}

// RemovePost removes a post from a user's library
func (r *LibraryRepository) RemovePost(libraryID, userID, postID string) error {
	_, err := r.db.Exec(`
		DELETE FROM library_posts
		WHERE library_id = ? AND post_id = ?
		AND EXISTS (SELECT 1 FROM libraries WHERE id = ? AND user_id = ?)
	`, libraryID, postID, libraryID, userID)
	return err
}
