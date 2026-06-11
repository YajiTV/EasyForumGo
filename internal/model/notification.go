package model

import "time"

type Notification struct {
	ID            string
	UserID        string
	ActorID       string
	Type          string
	CommentID     string
	PostID        string
	IsRead        bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ActorUsername string
	PostTitle     string
}
