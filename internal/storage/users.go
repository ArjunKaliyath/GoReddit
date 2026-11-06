package storage

import (
	"errors"
	"fmt"

	"github.com/ArjunKaliyath/GoReddit/internal/model"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var ErrUsernameTaken = errors.New("username unavailable")

func (dm *Store) RegisterUser(username, password string) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	dm.mu.Lock()
	defer dm.mu.Unlock()
	query := `INSERT INTO users (username, password) VALUES (?, ?)`
	result, err := dm.db.Exec(query, username, string(hash))
	if err != nil {
		var constraint sqlite3.Error
		if errors.As(err, &constraint) && constraint.ExtendedCode == sqlite3.ErrConstraintUnique {
			return 0, ErrUsernameTaken
		}
		return 0, fmt.Errorf("failed to register user: %w", err)
	}

	id, err := result.LastInsertId()
	return int(id), err
}

func (dm *Store) GetUserByUsername(username string) (*model.User, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var user model.User
	query := `SELECT id, username, karma FROM users WHERE username = ?`
	err := dm.db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Karma)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &user, nil
}

func (dm *Store) SubscribeToUser(subscriberID, subscribedUserID int) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	_, err := dm.db.Exec(`
        INSERT OR IGNORE INTO user_subscriptions 
        (subscriber_id, subscribed_user_id) 
        VALUES (?, ?)
    `, subscriberID, subscribedUserID)

	return err
}

func (dm *Store) UnsubscribeFromUser(subscriberID, subscribedUserID int) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	_, err := dm.db.Exec(`
        DELETE FROM user_subscriptions 
        WHERE subscriber_id = ? AND subscribed_user_id = ?
    `, subscriberID, subscribedUserID)

	return err
}

func (dm *Store) GetUserSubscriptions(userID int) ([]model.User, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	query := `
        SELECT u.id, u.username, u.karma
        FROM users u
        JOIN user_subscriptions us ON u.id = us.subscribed_user_id
        WHERE us.subscriber_id = ?
    `

	rows, err := dm.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subscriptions []model.User
	for rows.Next() {
		var user model.User
		err := rows.Scan(&user.ID, &user.Username, &user.Karma)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, user)
	}

	return subscriptions, rows.Err()
}

func (dm *Store) GetTopUsers(limit int) ([]model.TopUser, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	query := `
        SELECT 
            u.id,
            u.username,
            u.karma,
            (SELECT COUNT(*) FROM posts WHERE author_id = u.id) as post_count,
            (SELECT COUNT(*) FROM comments WHERE author_id = u.id) as comment_count
        FROM users u
        ORDER BY u.karma DESC
        LIMIT ?
    `

	rows, err := dm.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.TopUser
	for rows.Next() {
		var user model.TopUser
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Karma,
			&user.PostCount,
			&user.CommentCount,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// Function to get details of most subscribed users
func (dm *Store) GetTopSubscribedUsers(limit int) ([]model.TopSubscribedUser, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	query := `
        SELECT 
            u.id,
            u.username,
            u.karma,
            COUNT(us.subscriber_id) as subscriber_count
        FROM users u
        LEFT JOIN user_subscriptions us ON u.id = us.subscribed_user_id
        GROUP BY u.id, u.username, u.karma
        ORDER BY subscriber_count DESC
        LIMIT ?
    `

	rows, err := dm.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.TopSubscribedUser
	for rows.Next() {
		var user model.TopSubscribedUser
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Karma,
			&user.SubscriberCount,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}
