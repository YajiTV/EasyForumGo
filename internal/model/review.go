package model

import "time"

type PendingContent struct {
	ID         string
	Type       string
	Title      string
	Content    string
	AuthorID   string
	AuthorName string
	CreatedAt  time.Time
}

type ContentReview struct {
	ID          string
	ContentType string
	ContentID   string
	ReviewerID  string
	Decision    string
	Reason      string
	CreatedAt   time.Time
}
