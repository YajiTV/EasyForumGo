package utils

import "github.com/google/uuid"

// NewUUID creates a new instance
func NewUUID() string {
	return uuid.NewString()
}

// NewSessionToken creates a new instance
func NewSessionToken() string {
	return uuid.NewString()
}
