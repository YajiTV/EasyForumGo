package repository

import "database/sql"

type BlockRepository struct {
	db *sql.DB
}

// NewBlockRepository creates a new instance
func NewBlockRepository(db *sql.DB) *BlockRepository {
	return &BlockRepository{db: db}
}

// Block blocks a user and removes following relations in both directions
func (r *BlockRepository) Block(blockerID, blockedID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT OR IGNORE INTO user_blocks (blocker_id, blocked_id) VALUES (?, ?)`, blockerID, blockedID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM user_follows WHERE (follower_id = ? AND followed_id = ?) OR (follower_id = ? AND followed_id = ?)`,
		blockerID, blockedID, blockedID, blockerID); err != nil {
		return err
	}
	return tx.Commit()
}

// Unblock removes a user block
func (r *BlockRepository) Unblock(blockerID, blockedID string) error {
	_, err := r.db.Exec(`DELETE FROM user_blocks WHERE blocker_id = ? AND blocked_id = ?`, blockerID, blockedID)
	return err
}

// IsBlockedBetween reports whether either user blocked the other
func (r *BlockRepository) IsBlockedBetween(firstID, secondID string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM user_blocks WHERE (blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)`,
		firstID, secondID, secondID, firstID).Scan(&count)
	return count > 0, err
}

// HasBlocked reports whether the first user blocked the second
func (r *BlockRepository) HasBlocked(blockerID, blockedID string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM user_blocks WHERE blocker_id = ? AND blocked_id = ?`, blockerID, blockedID).Scan(&count)
	return count > 0, err
}
