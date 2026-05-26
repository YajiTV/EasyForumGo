package model

import "time"

type PostLike struct {
	ID        string
	PostID    string
	UserID    string
	IsLike    bool
	CreatedAt time.Time
}

type CommentLike struct {
	ID        string
	CommentID string
	UserID    string
	IsLike    bool
	CreatedAt time.Time
}
