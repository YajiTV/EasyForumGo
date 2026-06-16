package model

import "time"

type Notification struct {
	ID        string
	UserID    string
	ActorID   string
	Type      string
	PostID    string
	CommentID string
	IsRead    bool
	CreatedAt time.Time

	ActorUsername string
	PostTitle     string
}
