package model

import (
	"strings"
	"time"
)

type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	Username       string    `json:"username"`
	Password       string    `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	ProfilePicture string    `json:"profile_picture"`
}

// AvatarURL returns the correct URL for the profile picture,
// whether it's a local file or an external URL (e.g. Google OAuth).
func (u *User) AvatarURL() string {
	if strings.HasPrefix(u.ProfilePicture, "http") {
		return u.ProfilePicture
	}
	if u.ProfilePicture == "" {
		return ""
	}
	return "/uploads/" + u.ProfilePicture
}
