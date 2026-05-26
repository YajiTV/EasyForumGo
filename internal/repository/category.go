package repository

import (
	"database/sql"

	"ForumJS/internal/model"
)

type CategoryRepository struct {
	db *sql.DB
}

func NewCategoryRepository(db *sql.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) GetAll() ([]model.Category, error) {
	rows, err := r.db.Query(`SELECT id, name, description FROM categories ORDER BY name ASC`)
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

func (r *CategoryRepository) GetByID(id string) (*model.Category, error) {
	var c model.Category
	err := r.db.QueryRow(`SELECT id, name, description FROM categories WHERE id = ?`, id).Scan(&c.ID, &c.Name, &c.Description)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepository) Create(c *model.Category) error {
	_, err := r.db.Exec(`INSERT INTO categories (id, name, description) VALUES (?, ?, ?)`, c.ID, c.Name, c.Description)
	return err
}
