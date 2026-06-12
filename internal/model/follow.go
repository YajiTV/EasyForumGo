package model

import "time"

type Follow struct {
	FollowerID string
	FollowedID string
	CreatedAt  time.Time
}

type SocialStats struct {
	PostCount      int
	FollowerCount  int
	FollowingCount int
}
