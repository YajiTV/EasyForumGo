package repository

import (
	"database/sql"

	"ForumJS/internal/model"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) GetByToken(token string) (*model.Session, error) {
	var s model.Session
	err := r.db.QueryRow(`SELECT id, user_id, session_token, expires_at, created_at FROM sessions WHERE session_token = ?`, token).Scan(&s.ID, &s.UserID, &s.SessionToken, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepository) Create(s *model.Session) error {
	_, err := r.db.Exec(`INSERT INTO sessions (id, user_id, session_token, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		s.ID, s.UserID, s.SessionToken, s.ExpiresAt, s.CreatedAt)
	return err
}

func (r *SessionRepository) DeleteByUserID(userID string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}
