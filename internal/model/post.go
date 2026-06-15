package model

import "time"

type Post struct {
	ID        string
	UserID    string
	Title     string
	Content   string
	ImagePath string
	Approved  int
	CreatedAt time.Time
	UpdatedAt time.Time
}
