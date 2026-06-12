package model

import "time"

type ReportStatus string
type TargetType string
type ReportCategory string
type ActionType string
type RestrictionType string

const (
	ReportPending   ReportStatus = "pending"
	ReportResolved  ReportStatus = "resolved"
	ReportDismissed ReportStatus = "dismissed"
)

const (
	TargetPost    TargetType = "post"
	TargetComment TargetType = "comment"
	TargetUser    TargetType = "user"
)

const (
	CategorySpam                 ReportCategory = "spam"
	CategoryHarassment           ReportCategory = "harassment"
	CategoryInappropriateContent ReportCategory = "inappropriate_content"
	CategoryHateSpeech           ReportCategory = "hate_speech"
	CategoryOther                ReportCategory = "other"
)

const (
	ActionDeletePost    ActionType = "delete_post"
	ActionDeleteComment ActionType = "delete_comment"
	ActionWarnUser      ActionType = "warn_user"
	ActionMuteUser      ActionType = "mute_user"
	ActionBanUser       ActionType = "ban_user"
	ActionChangeRole    ActionType = "change_role"
)

const (
	RestrictionMute RestrictionType = "mute"
	RestrictionBan  RestrictionType = "ban"
)

type Report struct {
	ID           string
	ReporterID   string
	ReporterName string
	TargetType   TargetType
	TargetID     string
	Category     ReportCategory
	Reason       string
	Status       ReportStatus
	CreatedAt    time.Time
}

type ModerationAction struct {
	ID            string
	ModeratorID   string
	ModeratorName string
	ReportID      string
	ActionType    ActionType
	TargetID      string
	Reason        string
	DurationHours int
	CreatedAt     time.Time
}

type UserRestriction struct {
	ID        string
	UserID    string
	Type      RestrictionType
	Reason    string
	ExpiresAt *time.Time
	CreatedBy string
	CreatedAt time.Time
}

func (r *UserRestriction) IsActive() bool {
	if r.ExpiresAt == nil {
		return true
	}
	return time.Now().Before(*r.ExpiresAt)
}
