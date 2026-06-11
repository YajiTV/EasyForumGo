package model

import (
	"strings"
	"time"
)

type Role string

const (
	RoleGuest     Role = "guest"
	RoleUser      Role = "user"
	RoleModerator Role = "moderator"
	RoleAdmin     Role = "admin"
)

type User struct {
	ID             string
	Email          string
	Username       string
	Password       string
	Role           Role
	ProfilePicture string
	CreatedAt      time.Time
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u *User) IsModerator() bool {
	return u.Role == RoleModerator || u.Role == RoleAdmin
}

func (u *User) CanModerate() bool {
	return u.IsModerator()
}

func (u *User) AvatarURL() string {
	if strings.HasPrefix(u.ProfilePicture, "http") {
		return u.ProfilePicture
	}
	if u.ProfilePicture == "" {
		return ""
	}
	return "/uploads/" + u.ProfilePicture
}
