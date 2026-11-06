package storage

import (
	"fmt"

	"github.com/ArjunKaliyath/GoReddit/internal/model"
)

func (dm *Store) SendDirectMessage(fromUserID, toUserID int, content string) (int, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	result, err := dm.db.Exec(`
		INSERT INTO direct_messages (from_user_id, to_user_id, content) 
		VALUES (?, ?, ?)
	`, fromUserID, toUserID, content)

	if err != nil {
		return 0, fmt.Errorf("failed to send message: %w", err)
	}

	id, err := result.LastInsertId()
	return int(id), err
}

// Function to retrieve a user's received direct messages
func (dm *Store) GetDirectMessages(userID int) ([]model.DirectMessage, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	query := `
		SELECT 
			dm.id, 
			dm.from_user_id, 
			u.username AS from_username, 
			dm.content, 
			dm.created_at
		FROM direct_messages dm
		JOIN users u ON dm.from_user_id = u.id
		WHERE dm.to_user_id = ?
		ORDER BY dm.created_at DESC
	`

	rows, err := dm.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []model.DirectMessage
	for rows.Next() {
		var msg model.DirectMessage
		err := rows.Scan(
			&msg.ID,
			&msg.FromUserID,
			&msg.FromUsername,
			&msg.Content,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// Functions to let user subscribe and unsubscribe to other users.
