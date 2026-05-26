package repository

import (
	"database/sql"

	"ForumJS/internal/model"
)

type LikeRepository struct {
	db *sql.DB
}

func NewLikeRepository(db *sql.DB) *LikeRepository {
	return &LikeRepository{db: db}
}

// --- PostLike ---

func (r *LikeRepository) CreatePostLike(l *model.PostLike) error {
	_, err := r.db.Exec(`INSERT INTO post_likes (id, post_id, user_id, is_like, created_at) VALUES (?, ?, ?, ?, ?)`,
		l.ID, l.PostID, l.UserID, l.IsLike, l.CreatedAt)
	return err
}

func (r *LikeRepository) DeletePostLike(postID, userID string) error {
	_, err := r.db.Exec(`DELETE FROM post_likes WHERE post_id = ? AND user_id = ?`, postID, userID)
	return err
}

func (r *LikeRepository) GetByPostID(postID string) ([]model.PostLike, error) {
	rows, err := r.db.Query(`SELECT id, post_id, user_id, is_like, created_at FROM post_likes WHERE post_id = ?`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var likes []model.PostLike
	for rows.Next() {
		var l model.PostLike
		if err := rows.Scan(&l.ID, &l.PostID, &l.UserID, &l.IsLike, &l.CreatedAt); err != nil {
			return nil, err
		}
		likes = append(likes, l)
	}
	return likes, rows.Err()
}

func (r *LikeRepository) CountPostLikes(postID string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND is_like = 1`, postID).Scan(&count)
	return count, err
}

func (r *LikeRepository) CountPostDislikes(postID string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND is_like = 0`, postID).Scan(&count)
	return count, err
}

// --- CommentLike ---

func (r *LikeRepository) CreateCommentLike(l *model.CommentLike) error {
	_, err := r.db.Exec(`INSERT INTO comment_likes (id, comment_id, user_id, is_like, created_at) VALUES (?, ?, ?, ?, ?)`,
		l.ID, l.CommentID, l.UserID, l.IsLike, l.CreatedAt)
	return err
}

func (r *LikeRepository) DeleteCommentLike(commentID, userID string) error {
	_, err := r.db.Exec(`DELETE FROM comment_likes WHERE comment_id = ? AND user_id = ?`, commentID, userID)
	return err
}

func (r *LikeRepository) GetByCommentID(commentID string) ([]model.CommentLike, error) {
	rows, err := r.db.Query(`SELECT id, comment_id, user_id, is_like, created_at FROM comment_likes WHERE comment_id = ?`, commentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var likes []model.CommentLike
	for rows.Next() {
		var l model.CommentLike
		if err := rows.Scan(&l.ID, &l.CommentID, &l.UserID, &l.IsLike, &l.CreatedAt); err != nil {
			return nil, err
		}
		likes = append(likes, l)
	}
	return likes, rows.Err()
}

func (r *LikeRepository) CountCommentLikes(commentID string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM comment_likes WHERE comment_id = ? AND is_like = 1`, commentID).Scan(&count)
	return count, err
}

func (r *LikeRepository) CountCommentDislikes(commentID string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM comment_likes WHERE comment_id = ? AND is_like = 0`, commentID).Scan(&count)
	return count, err
}
