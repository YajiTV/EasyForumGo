package model

import "time"

type Session struct {
	ID           string
	UserID       string
	SessionToken string
	ExpiresAt    time.Time
	CreatedAt    time.Time
}
