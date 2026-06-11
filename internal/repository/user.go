package repository

import (
	"EasyForumGo/internal/model"
	"database/sql"
	"fmt"
	"strings"
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
		userColumns+` WHERE id = ?`, id,
	).Scan(userScanTargets(&u)...)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(email string) (*model.User, error) {
	var u model.User
	err := r.db.QueryRow(
		userColumns+` WHERE email = ?`, email,
	).Scan(userScanTargets(&u)...)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	var u model.User
	err := r.db.QueryRow(
		userColumns+` WHERE username = ? COLLATE NOCASE`, username,
	).Scan(userScanTargets(&u)...)
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

// UpdateEmail updates a user's email address
func (r *UserRepository) UpdateEmail(userID, email string) error {
	_, err := r.db.Exec(`UPDATE users SET email = ? WHERE id = ?`, email, userID)
	return err
}

// UpdatePasswordAndDeleteSessions updates a password and invalidates sessions
func (r *UserRepository) UpdatePasswordAndDeleteSessions(userID, hashedPassword string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE users SET password = ? WHERE id = ?`, hashedPassword, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *UserRepository) UpdateRole(userID string, role model.Role) error {
	_, err := r.db.Exec(
		`UPDATE users SET role = ? WHERE id = ?`,
		role, userID,
	)
	return err
}

// UpdateSocialProfile updates a user's public social preferences
func (r *UserRepository) UpdateSocialProfile(userID, biography string, followsVisible bool) error {
	_, err := r.db.Exec(`UPDATE users SET biography = ?, follows_visible = ? WHERE id = ?`, biography, followsVisible, userID)
	return err
}

// Search returns users matching a username
func (r *UserRepository) Search(query, excludeUserID string, limit int) ([]model.User, error) {
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("invalid limit")
	}
	rows, err := r.db.Query(userColumns+`
		WHERE id <> ? AND username LIKE ? ESCAPE '\'
		ORDER BY username COLLATE NOCASE ASC LIMIT ?`,
		excludeUserID, "%"+escapeLike(query)+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []model.User
	for rows.Next() {
		var user model.User
		if err := scanUser(rows, &user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

const userColumns = `SELECT id, email, username, password, role, profile_picture, biography, follows_visible, created_at FROM users`

// userScanTargets returns user scan destinations
func userScanTargets(user *model.User) []any {
	return []any{&user.ID, &user.Email, &user.Username, &user.Password, &user.Role, &user.ProfilePicture, &user.Biography, &user.FollowsVisible, &user.CreatedAt}
}

// scanUser scans a user row
func scanUser(scanner interface{ Scan(...any) error }, user *model.User) error {
	return scanner.Scan(userScanTargets(user)...)
}

// escapeLike escapes sqlite LIKE wildcard characters
func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
}
