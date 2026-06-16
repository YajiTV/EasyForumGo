package repository

import (
	"database/sql"

	"EasyForumGo/internal/model"
)

type PostCategoryRepository struct {
	db *sql.DB
}

// NewPostCategoryRepository creates a new instance
func NewPostCategoryRepository(db *sql.DB) *PostCategoryRepository {
	return &PostCategoryRepository{db: db}
}

// AddCategory adds a new association
func (r *PostCategoryRepository) AddCategory(postID, categoryID string) error {
	_, err := r.db.Exec(`INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)`, postID, categoryID)
	return err
}

// GetCategoriesByPostID gets stored data
func (r *PostCategoryRepository) GetCategoriesByPostID(postID string) ([]model.Category, error) {
	rows, err := r.db.Query(`SELECT c.id, c.name, c.description FROM categories c JOIN post_categories pc ON c.id = pc.category_id WHERE pc.post_id = ?`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []model.Category
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Description); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

// GetPopularByUserID gets the categories an author posts in most often
func (r *PostCategoryRepository) GetPopularByUserID(userID string, limit int) ([]model.Category, error) {
	rows, err := r.db.Query(`SELECT c.id, c.name, c.description
		FROM categories c
		JOIN post_categories pc ON pc.category_id = c.id
		JOIN posts p ON p.id = pc.post_id
		WHERE p.user_id = ?
		GROUP BY c.id
		ORDER BY COUNT(DISTINCT p.id) DESC, c.name ASC
		LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []model.Category
	for rows.Next() {
		var category model.Category
		if err := rows.Scan(&category.ID, &category.Name, &category.Description); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, rows.Err()
}

// DeleteByPostID deletes an existing record
func (r *PostCategoryRepository) DeleteByPostID(postID string) error {
	_, err := r.db.Exec(`DELETE FROM post_categories WHERE post_id = ?`, postID)
	return err
}
