package repository

import (
	"EasyForumGo/internal/model"
	"database/sql"
)

type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) Create(report *model.Report) error {
	_, err := r.db.Exec(
		`INSERT INTO reports (id, reporter_id, target_type, target_id, category, reason, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		report.ID, report.ReporterID, report.TargetType, report.TargetID,
		report.Category, report.Reason, report.Status, report.CreatedAt,
	)
	return err
}

func (r *ReportRepository) GetPending() ([]model.Report, error) {
	return r.getByStatus("pending")
}

func (r *ReportRepository) GetAll() ([]model.Report, error) {
	rows, err := r.db.Query(
		`SELECT r.id, r.reporter_id, u.username, r.target_type, r.target_id,
		        r.category, COALESCE(r.reason, ''), r.status, r.created_at
		 FROM reports r
		 JOIN users u ON u.id = r.reporter_id
		 ORDER BY r.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReports(rows)
}

func (r *ReportRepository) getByStatus(status string) ([]model.Report, error) {
	rows, err := r.db.Query(
		`SELECT r.id, r.reporter_id, u.username, r.target_type, r.target_id,
		        r.category, COALESCE(r.reason, ''), r.status, r.created_at
		 FROM reports r
		 JOIN users u ON u.id = r.reporter_id
		 WHERE r.status = ?
		 ORDER BY r.created_at DESC`,
		status,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReports(rows)
}

func (r *ReportRepository) GetByID(id string) (*model.Report, error) {
	var rep model.Report
	var reason string
	err := r.db.QueryRow(
		`SELECT r.id, r.reporter_id, u.username, r.target_type, r.target_id,
		        r.category, COALESCE(r.reason, ''), r.status, r.created_at
		 FROM reports r
		 JOIN users u ON u.id = r.reporter_id
		 WHERE r.id = ?`,
		id,
	).Scan(&rep.ID, &rep.ReporterID, &rep.ReporterName, &rep.TargetType,
		&rep.TargetID, &rep.Category, &reason, &rep.Status, &rep.CreatedAt)
	if err != nil {
		return nil, err
	}
	rep.Reason = reason
	return &rep, nil
}

func (r *ReportRepository) UpdateStatus(id string, status model.ReportStatus) error {
	_, err := r.db.Exec(`UPDATE reports SET status = ? WHERE id = ?`, status, id)
	return err
}

func (r *ReportRepository) AlreadyReported(reporterID string, targetType model.TargetType, targetID string) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM reports WHERE reporter_id = ? AND target_type = ? AND target_id = ?`,
		reporterID, targetType, targetID,
	).Scan(&count)
	return count > 0, err
}

func (r *ReportRepository) CreateAction(action *model.ModerationAction) error {
	_, err := r.db.Exec(
		`INSERT INTO moderation_actions (id, moderator_id, report_id, action_type, target_id, reason, duration_hours, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		action.ID, action.ModeratorID, nullableString(action.ReportID), action.ActionType,
		action.TargetID, action.Reason, action.DurationHours, action.CreatedAt,
	)
	return err
}

func (r *ReportRepository) GetActionHistory() ([]model.ModerationAction, error) {
	rows, err := r.db.Query(
		`SELECT a.id, a.moderator_id, u.username, COALESCE(a.report_id, ''),
		        a.action_type, a.target_id, COALESCE(a.reason, ''), COALESCE(a.duration_hours, 0), a.created_at
		 FROM moderation_actions a
		 JOIN users u ON u.id = a.moderator_id
		 ORDER BY a.created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []model.ModerationAction
	for rows.Next() {
		var a model.ModerationAction
		if err := rows.Scan(&a.ID, &a.ModeratorID, &a.ModeratorName, &a.ReportID,
			&a.ActionType, &a.TargetID, &a.Reason, &a.DurationHours, &a.CreatedAt); err != nil {
			return nil, err
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

func (r *ReportRepository) CreateRestriction(restriction *model.UserRestriction) error {
	_, err := r.db.Exec(
		`INSERT INTO user_restrictions (id, user_id, type, reason, expires_at, created_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		restriction.ID, restriction.UserID, restriction.Type, restriction.Reason,
		restriction.ExpiresAt, restriction.CreatedBy, restriction.CreatedAt,
	)
	return err
}

func (r *ReportRepository) GetActiveRestriction(userID string, restrictionType model.RestrictionType) (*model.UserRestriction, error) {
	var res model.UserRestriction
	var expiresAt sql.NullTime
	err := r.db.QueryRow(
		`SELECT id, user_id, type, reason, expires_at, created_by, created_at
		 FROM user_restrictions
		 WHERE user_id = ? AND type = ?
		 AND (expires_at IS NULL OR expires_at > datetime('now'))
		 ORDER BY created_at DESC LIMIT 1`,
		userID, restrictionType,
	).Scan(&res.ID, &res.UserID, &res.Type, &res.Reason,
		&expiresAt, &res.CreatedBy, &res.CreatedAt)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		res.ExpiresAt = &expiresAt.Time
	}
	return &res, nil
}

func scanReports(rows *sql.Rows) ([]model.Report, error) {
	var reports []model.Report
	for rows.Next() {
		var rep model.Report
		if err := rows.Scan(&rep.ID, &rep.ReporterID, &rep.ReporterName, &rep.TargetType,
			&rep.TargetID, &rep.Category, &rep.Reason, &rep.Status, &rep.CreatedAt); err != nil {
			return nil, err
		}
		reports = append(reports, rep)
	}
	return reports, rows.Err()
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
