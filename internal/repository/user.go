package repository

import (
	"EasyForumGo/internal/model"
	"database/sql"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByID(id string) (*model.User, error) {
	var u model.User
	err := r.db.QueryRow(
		`SELECT id, email, username, password, role, profile_picture, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.Username, &u.Password, &u.Role, &u.ProfilePicture, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(email string) (*model.User, error) {
	var u model.User
	err := r.db.QueryRow(
		`SELECT id, email, username, password, role, profile_picture, created_at FROM users WHERE email = ?`, email,
	).Scan(&u.ID, &u.Email, &u.Username, &u.Password, &u.Role, &u.ProfilePicture, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	var u model.User
	err := r.db.QueryRow(
		`SELECT id, email, username, password, role, profile_picture, created_at FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Email, &u.Username, &u.Password, &u.Role, &u.ProfilePicture, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(u *model.User) error {
	_, err := r.db.Exec(
		`INSERT INTO users (id, email, username, password, role, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.Username, u.Password, u.Role, u.CreatedAt,
	)
	return err
}

func (r *UserRepository) Update(u *model.User) error {
	_, err := r.db.Exec(
		`UPDATE users SET email = ?, username = ? WHERE id = ?`,
		u.Email, u.Username, u.ID,
	)
	return err
}

func (r *UserRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

func (r *UserRepository) UpdateProfile(userID, username, profilePicture string) error {
	_, err := r.db.Exec(
		`UPDATE users SET username = ?, profile_picture = ? WHERE id = ?`,
		username, profilePicture, userID,
	)
	return err
}

func (r *UserRepository) UpdateRole(userID string, role model.Role) error {
	_, err := r.db.Exec(
		`UPDATE users SET role = ? WHERE id = ?`,
		role, userID,
	)
	return err
}
