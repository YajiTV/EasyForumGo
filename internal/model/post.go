package model

import "time"

type Post struct {
	ID        string
	UserID    string
	Title     string
	Content   string
	ImagePath string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
