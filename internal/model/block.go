package model

import "time"

type Block struct {
	BlockerID string
	BlockedID string
	CreatedAt time.Time
}
