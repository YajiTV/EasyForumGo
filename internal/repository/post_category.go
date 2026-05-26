package repository

import (
	"database/sql"

	"ForumJS/internal/model"
)

type PostCategoryRepository struct {
	db *sql.DB
}

func NewPostCategoryRepository(db *sql.DB) *PostCategoryRepository {
	return &PostCategoryRepository{db: db}
}

// AddCategory associe une categorie a un post.
func (r *PostCategoryRepository) AddCategory(postID, categoryID string) error {
	_, err := r.db.Exec(`INSERT INTO post_categories (post_id, category_id) VALUES (?, ?)`, postID, categoryID)
	return err
}

// GetCategoriesByPostID retourne toutes les categories d'un post.
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

// DeleteByPostID supprime toutes les associations d'un post (utile pour Update).
func (r *PostCategoryRepository) DeleteByPostID(postID string) error {
	_, err := r.db.Exec(`DELETE FROM post_categories WHERE post_id = ?`, postID)
	return err
}
