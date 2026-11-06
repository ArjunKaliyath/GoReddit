package storage

import (
	"database/sql"
	"fmt"

	"github.com/ArjunKaliyath/GoReddit/internal/model"
)

func (dm *Store) CreatePost(title, content string, authorID, subredditID int) (int, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	result, err := dm.db.Exec(`
		INSERT INTO posts (title, content, author_id, subreddit_id)
		VALUES (?, ?, ?, ?)
	`, title, content, authorID, subredditID)

	if err != nil {
		return 0, fmt.Errorf("failed to create post: %w", err)
	}

	id, err := result.LastInsertId()
	return int(id), err
}

// Function to retrieve user's top feed items
func (dm *Store) GetFeed(userID int) ([]model.Post, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	query := `
		SELECT p.id, p.title, p.content, p.author_id, p.subreddit_id, p.created_at,
               u.username AS author_username, s.name AS subreddit_name,
			(SELECT COUNT(*) FROM votes WHERE target_id = p.id AND target_type = 'post' AND vote_value = 1) AS upvotes,
            (SELECT COUNT(*) FROM votes WHERE target_id = p.id AND target_type = 'post' AND vote_value = -1) AS downvotes
		FROM posts p
		JOIN subreddit_members sm ON p.subreddit_id = sm.subreddit_id
		JOIN users u ON p.author_id = u.id
		JOIN subreddits s ON p.subreddit_id = s.id
		WHERE sm.user_id = ?
		ORDER BY p.created_at DESC
	`

	rows, err := dm.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		var post model.Post
		err := rows.Scan(
			&post.ID, &post.Title, &post.Content, &post.AuthorID,
			&post.SubredditID, &post.CreatedAt,
			&post.AuthorUsername, &post.SubredditName, &post.VoteCount.Upvotes,
			&post.VoteCount.Downvotes,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, rows.Err()
}

// Vote atomically replaces a user's vote and applies only the karma delta.
func (dm *Store) Vote(userID, targetID int, targetType string, value int) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if (targetType != "post" && targetType != "comment") || (value != 1 && value != -1) {
		return fmt.Errorf("invalid vote")
	}
	tx, err := dm.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	table := "posts"
	if targetType == "comment" {
		table = "comments"
	}
	var authorID, previous int
	if err = tx.QueryRow("SELECT author_id FROM "+table+" WHERE id=?", targetID).Scan(&authorID); err != nil {
		return err
	}
	if err = tx.QueryRow("SELECT COALESCE(SUM(vote_value),0) FROM votes WHERE user_id=? AND target_id=? AND target_type=?", userID, targetID, targetType).Scan(&previous); err != nil {
		return err
	}
	if _, err = tx.Exec("DELETE FROM votes WHERE user_id=? AND target_id=? AND target_type=?", userID, targetID, targetType); err != nil {
		return err
	}
	if _, err = tx.Exec("INSERT INTO votes(user_id,target_id,target_type,vote_value) VALUES(?,?,?,?)", userID, targetID, targetType, value); err != nil {
		return err
	}
	if _, err = tx.Exec("UPDATE users SET karma=karma+? WHERE id=?", value-previous, authorID); err != nil {
		return err
	}
	return tx.Commit()
}

func (dm *Store) CreateComment(content string, authorID, postID int, parentCommentID *int) (int, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if parentCommentID != nil {
		var parentPost int
		if err := dm.db.QueryRow("SELECT post_id FROM comments WHERE id=?", *parentCommentID).Scan(&parentPost); err != nil {
			return 0, err
		}
		if parentPost != postID {
			return 0, fmt.Errorf("parent comment belongs to another post")
		}
	}
	query := `
		INSERT INTO comments (content, author_id, post_id, parent_comment_id)
		VALUES (?, ?, ?, ?)
	`

	result, err := dm.db.Exec(query, content, authorID, postID, parentCommentID)
	if err != nil {
		return 0, fmt.Errorf("failed to create comment: %w", err)
	}

	id, err := result.LastInsertId()
	return int(id), err
}

func (dm *Store) GetTopPosts(limit int) ([]model.Post, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	query := `
        SELECT p.id, p.title, p.content, p.author_id, p.subreddit_id, p.created_at,
               u.username AS author_username, s.name AS subreddit_name,
               (SELECT COUNT(*) FROM votes WHERE target_id = p.id AND target_type = 'post' AND vote_value = 1) AS upvotes,
               (SELECT COUNT(*) FROM votes WHERE target_id = p.id AND target_type = 'post' AND vote_value = -1) AS downvotes
        FROM posts p
        JOIN users u ON p.author_id = u.id
        JOIN subreddits s ON p.subreddit_id = s.id
        ORDER BY upvotes - downvotes DESC
        LIMIT ?
    `

	rows, err := dm.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []model.Post
	for rows.Next() {
		var post model.Post
		err := rows.Scan(
			&post.ID, &post.Title, &post.Content, &post.AuthorID,
			&post.SubredditID, &post.CreatedAt,
			&post.AuthorUsername, &post.SubredditName,
			&post.VoteCount.Upvotes, &post.VoteCount.Downvotes,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	return posts, rows.Err()
}

func (dm *Store) Comments(postID int) ([]model.Comment, error) {
	rows, err := dm.db.Query(`SELECT c.id,c.content,c.author_id,u.username,c.post_id,c.parent_comment_id,c.created_at FROM comments c JOIN users u ON u.id=c.author_id WHERE c.post_id=? ORDER BY c.id`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []model.Comment{}
	for rows.Next() {
		var c model.Comment
		if err := rows.Scan(&c.ID, &c.Content, &c.AuthorID, &c.AuthorUsername, &c.PostID, &c.ParentCommentID, &c.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (dm *Store) PostExists(id int) bool {
	var found int
	return dm.db.QueryRow("SELECT id FROM posts WHERE id=?", id).Scan(&found) != sql.ErrNoRows && found == id
}
