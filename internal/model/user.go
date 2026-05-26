package model

import "time"

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Password  string    `json:"-"` //password is never includ
	CreatedAt time.Time `json:"created_at"`
}
