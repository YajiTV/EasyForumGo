package repository

import (
	"database/sql"
	"EasyForumGo/internal/model"
)

type ContentReviewRepository struct {
	db *sql.DB
}

func NewContentReviewRepository(db *sql.DB) *ContentReviewRepository {
	return &ContentReviewRepository{db: db}
}

func (r *ContentReviewRepository) GetKeywords() ([]string, error) {
	rows, err := r.db.Query(`SELECT word FROM flagged_keywords ORDER BY word ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keywords []string
	for rows.Next() {
		var w string
		if err := rows.Scan(&w); err != nil {
			return nil, err
		}
		keywords = append(keywords, w)
	}
	return keywords, rows.Err()
}

func (r *ContentReviewRepository) GetPendingPosts(sort string) ([]model.PendingContent, error) {
	order := "ASC"
	if sort == "desc" {
		order = "DESC"
	}
	rows, err := r.db.Query(`
		SELECT p.id, 'post', p.title, p.content, p.user_id, u.username, p.created_at
		FROM posts p
		JOIN users u ON u.id = p.user_id
		WHERE p.status = 'pending'
		ORDER BY p.created_at `+order, )
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPending(rows)
}

func (r *ContentReviewRepository) GetPendingComments(sort string) ([]model.PendingContent, error) {
	order := "ASC"
	if sort == "desc" {
		order = "DESC"
	}
	rows, err := r.db.Query(`
		SELECT c.id, 'comment', '', c.content, c.user_id, u.username, c.created_at
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.status = 'pending'
		ORDER BY c.created_at `+order, )
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPending(rows)
}

func (r *ContentReviewRepository) ApprovePost(id string) error {
	_, err := r.db.Exec(`UPDATE posts SET status = 'approved' WHERE id = ?`, id)
	return err
}

func (r *ContentReviewRepository) ApproveComment(id string) error {
	_, err := r.db.Exec(`UPDATE comments SET status = 'approved' WHERE id = ?`, id)
	return err
}

func (r *ContentReviewRepository) RejectPost(id string) error {
	_, err := r.db.Exec(`DELETE FROM posts WHERE id = ?`, id)
	return err
}

func (r *ContentReviewRepository) RejectComment(id string) error {
	_, err := r.db.Exec(`DELETE FROM comments WHERE id = ?`, id)
	return err
}

func (r *ContentReviewRepository) CreateReview(review *model.ContentReview) error {
	_, err := r.db.Exec(`
		INSERT INTO content_reviews (id, content_type, content_id, reviewer_id, decision, reason, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		review.ID, review.ContentType, review.ContentID, review.ReviewerID,
		review.Decision, review.Reason, review.CreatedAt)
	return err
}

func (r *ContentReviewRepository) GetPostAuthor(postID string) (string, error) {
	var userID string
	err := r.db.QueryRow(`SELECT user_id FROM posts WHERE id = ?`, postID).Scan(&userID)
	return userID, err
}

func (r *ContentReviewRepository) GetCommentAuthor(commentID string) (string, error) {
	var userID string
	err := r.db.QueryRow(`SELECT user_id FROM comments WHERE id = ?`, commentID).Scan(&userID)
	return userID, err
}

func scanPending(rows *sql.Rows) ([]model.PendingContent, error) {
	var items []model.PendingContent
	for rows.Next() {
		var p model.PendingContent
		if err := rows.Scan(&p.ID, &p.Type, &p.Title, &p.Content, &p.AuthorID, &p.AuthorName, &p.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}
