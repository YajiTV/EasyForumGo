package model

import "time"

type Library struct {
	ID        string
	UserID    string
	Name      string
	PostCount int
	CreatedAt time.Time
}
