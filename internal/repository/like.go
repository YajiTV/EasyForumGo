package repository

import (
	"database/sql"
	"errors"

	"EasyForumGo/internal/model"
)

type LikeRepository struct {
	db *sql.DB
}

// NewLikeRepository creates a new instance
func NewLikeRepository(db *sql.DB) *LikeRepository {
	return &LikeRepository{db: db}
}

// CreatePostLike creates a new record
func (r *LikeRepository) CreatePostLike(l *model.PostLike) error {
	_, err := r.db.Exec(`INSERT INTO post_likes (id, post_id, user_id, is_like, created_at) VALUES (?, ?, ?, ?, ?)`,
		l.ID, l.PostID, l.UserID, l.IsLike, l.CreatedAt)
	return err
}

// DeletePostLike deletes an existing record
func (r *LikeRepository) DeletePostLike(postID, userID string) error {
	_, err := r.db.Exec(`DELETE FROM post_likes WHERE post_id = ? AND user_id = ?`, postID, userID)
	return err
}

// GetUserPostLike gets stored data
func (r *LikeRepository) GetUserPostLike(postID, userID string) (*model.PostLike, error) {
	var l model.PostLike
	err := r.db.QueryRow(
		`SELECT id, post_id, user_id, is_like, created_at FROM post_likes WHERE post_id = ? AND user_id = ?`,
		postID, userID,
	).Scan(&l.ID, &l.PostID, &l.UserID, &l.IsLike, &l.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// UpdatePostLike updates an existing record
func (r *LikeRepository) UpdatePostLike(postID, userID string, isLike bool) error {
	_, err := r.db.Exec(
		`UPDATE post_likes SET is_like = ? WHERE post_id = ? AND user_id = ?`,
		isLike, postID, userID,
	)
	return err
}

// GetByPostID gets stored data
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

// CountPostLikes counts stored records
func (r *LikeRepository) CountPostLikes(postID string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND is_like = 1`, postID).Scan(&count)
	return count, err
}

// CountPostDislikes counts stored records
func (r *LikeRepository) CountPostDislikes(postID string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND is_like = 0`, postID).Scan(&count)
	return count, err
}

// CreateCommentLike creates a new record
func (r *LikeRepository) CreateCommentLike(l *model.CommentLike) error {
	_, err := r.db.Exec(`INSERT INTO comment_likes (id, comment_id, user_id, is_like, created_at) VALUES (?, ?, ?, ?, ?)`,
		l.ID, l.CommentID, l.UserID, l.IsLike, l.CreatedAt)
	return err
}

// DeleteCommentLike deletes an existing record
func (r *LikeRepository) DeleteCommentLike(commentID, userID string) error {
	_, err := r.db.Exec(`DELETE FROM comment_likes WHERE comment_id = ? AND user_id = ?`, commentID, userID)
	return err
}

// GetUserCommentLike gets stored data
func (r *LikeRepository) GetUserCommentLike(commentID, userID string) (*model.CommentLike, error) {
	var l model.CommentLike
	err := r.db.QueryRow(
		`SELECT id, comment_id, user_id, is_like, created_at FROM comment_likes WHERE comment_id = ? AND user_id = ?`,
		commentID, userID,
	).Scan(&l.ID, &l.CommentID, &l.UserID, &l.IsLike, &l.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// UpdateCommentLike updates an existing record
func (r *LikeRepository) UpdateCommentLike(commentID, userID string, isLike bool) error {
	_, err := r.db.Exec(
		`UPDATE comment_likes SET is_like = ? WHERE comment_id = ? AND user_id = ?`,
		isLike, commentID, userID,
	)
	return err
}

// GetByCommentID gets stored data
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

// CountCommentLikes counts stored records
func (r *LikeRepository) CountCommentLikes(commentID string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM comment_likes WHERE comment_id = ? AND is_like = 1`, commentID).Scan(&count)
	return count, err
}

// CountCommentDislikes counts stored records
func (r *LikeRepository) CountCommentDislikes(commentID string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM comment_likes WHERE comment_id = ? AND is_like = 0`, commentID).Scan(&count)
	return count, err
}

// GetLikedPostsByUserID gets stored data
func (r *LikeRepository) GetLikedPostsByUserID(userID string) ([]model.Post, error) {
	return r.GetPostsByUserVote(userID, true)
}

// GetPostsByUserVote gets posts matching a user's vote
func (r *LikeRepository) GetPostsByUserVote(userID string, isLike bool) ([]model.Post, error) {
	rows, err := r.db.Query(`
		SELECT p.id, p.user_id, p.title, p.content, p.image_path, p.created_at, p.updated_at
		FROM posts p
		JOIN post_likes pl ON p.id = pl.post_id
		WHERE pl.user_id = ? AND pl.is_like = ?
		ORDER BY pl.created_at DESC
	`, userID, isLike)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		var p model.Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Content, &p.ImagePath, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}

// CountPostVotesByUserID counts a user's post votes by type
func (r *LikeRepository) CountPostVotesByUserID(userID string, isLike bool) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM post_likes WHERE user_id = ? AND is_like = ?`,
		userID, isLike,
	).Scan(&count)
	return count, err
}
