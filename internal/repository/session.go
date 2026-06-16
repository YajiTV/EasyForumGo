package repository

import (
	"database/sql"

	"EasyForumGo/internal/model"
)

type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository creates a new instance
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// GetByToken gets stored data
func (r *SessionRepository) GetByToken(token string) (*model.Session, error) {
	var s model.Session
	err := r.db.QueryRow(`SELECT id, user_id, session_token, expires_at, created_at FROM sessions WHERE session_token = ?`, token).Scan(&s.ID, &s.UserID, &s.SessionToken, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Create creates a new record
func (r *SessionRepository) Create(s *model.Session) error {
	_, err := r.db.Exec(`INSERT INTO sessions (id, user_id, session_token, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		s.ID, s.UserID, s.SessionToken, s.ExpiresAt, s.CreatedAt)
	return err
}

// DeleteByUserID deletes an existing record
func (r *SessionRepository) DeleteByUserID(userID string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}
