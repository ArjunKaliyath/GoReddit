package storage

import (
	"crypto/sha256"
	"errors"
	"time"

	"github.com/ArjunKaliyath/GoReddit/internal/model"
	"golang.org/x/crypto/bcrypt"
)

var ErrCredentials = errors.New("invalid username or password")

func (dm *Store) Authenticate(username, password string) (*model.User, error) {
	dm.mu.RLock()
	var u model.User
	var hash string
	err := dm.db.QueryRow("SELECT id, username, karma, password FROM users WHERE username = ?", username).Scan(&u.ID, &u.Username, &u.Karma, &hash)
	dm.mu.RUnlock()
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return nil, ErrCredentials
	}
	return &u, nil
}

func (dm *Store) CreateSession(token string, userID int, expires time.Time) error {
	hash := sha256.Sum256([]byte(token))
	_, err := dm.db.Exec("INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)", hash[:], userID, expires.Unix())
	return err
}

func (dm *Store) SessionUser(token string) (*model.User, error) {
	hash := sha256.Sum256([]byte(token))
	var u model.User
	err := dm.db.QueryRow(`SELECT u.id,u.username,u.karma FROM users u JOIN sessions s ON s.user_id=u.id WHERE s.token_hash=? AND s.expires_at>?`, hash[:], time.Now().Unix()).Scan(&u.ID, &u.Username, &u.Karma)
	return &u, err
}

func (dm *Store) DeleteSession(token string) error {
	hash := sha256.Sum256([]byte(token))
	_, err := dm.db.Exec("DELETE FROM sessions WHERE token_hash=?", hash[:])
	return err
}

func (dm *Store) MigrateSessions() error {
	_, err := dm.db.Exec(`CREATE TABLE IF NOT EXISTS sessions(token_hash BLOB PRIMARY KEY,user_id INTEGER NOT NULL REFERENCES users(id),expires_at INTEGER NOT NULL); DELETE FROM sessions WHERE expires_at < strftime('%s','now');`)
	return err
}

// UserID resolves message recipients without exposing password data.
func (dm *Store) UserID(username string) (int, error) {
	var id int
	err := dm.db.QueryRow("SELECT id FROM users WHERE username=?", username).Scan(&id)
	return id, err
}
