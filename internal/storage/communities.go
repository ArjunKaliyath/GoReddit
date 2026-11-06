package storage

import (
	"fmt"

	"github.com/ArjunKaliyath/GoReddit/internal/model"
)

func (dm *Store) CreateSubreddit(name, description string, creatorID int) (int, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	tx, err := dm.db.Begin()
	if err != nil {
		return 0, err
	}

	// Create subreddit
	result, err := tx.Exec(`INSERT INTO subreddits (name, description) VALUES (?, ?)`, name, description)
	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to create subreddit: %w", err)
	}

	subredditID, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Add creator as first member
	_, err = tx.Exec(`
		INSERT INTO subreddit_members (subreddit_id, user_id) 
		VALUES (?, ?)
	`, subredditID, creatorID)

	if err != nil {
		tx.Rollback()
		return 0, fmt.Errorf("failed to add creator to subreddit: %w", err)
	}

	err = tx.Commit()
	return int(subredditID), err
}

func (dm *Store) JoinSubreddit(userID, subredditID int) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	_, err := dm.db.Exec(`
		INSERT OR IGNORE INTO subreddit_members (subreddit_id, user_id) 
		VALUES (?, ?)
	`, subredditID, userID)

	return err
}

func (dm *Store) LeaveSubreddit(userID, subredditID int) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	_, err := dm.db.Exec(`
		DELETE FROM subreddit_members 
		WHERE subreddit_id = ? AND user_id = ?
	`, subredditID, userID)

	return err
}

// Create Reddit model.Post
func (dm *Store) GetAllSubreddits() ([]model.Subreddit, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	query := `
		SELECT id, name, description, created_at
		FROM subreddits
		ORDER BY name
	`

	rows, err := dm.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subreddits []model.Subreddit
	for rows.Next() {
		var subreddit model.Subreddit
		err := rows.Scan(
			&subreddit.ID, &subreddit.Name,
			&subreddit.Description, &subreddit.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		subreddits = append(subreddits, subreddit)
	}

	return subreddits, rows.Err()
}

func (dm *Store) GetUserJoinedSubreddits(userID int) ([]model.Subreddit, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	query := `
		SELECT s.id, s.name, s.description, s.created_at
		FROM subreddits s
		JOIN subreddit_members sm ON s.id = sm.subreddit_id
		WHERE sm.user_id = ?
		ORDER BY s.name
	`

	rows, err := dm.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subreddits []model.Subreddit
	for rows.Next() {
		var subreddit model.Subreddit
		err := rows.Scan(
			&subreddit.ID, &subreddit.Name,
			&subreddit.Description, &subreddit.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		subreddits = append(subreddits, subreddit)
	}

	return subreddits, rows.Err()
}
