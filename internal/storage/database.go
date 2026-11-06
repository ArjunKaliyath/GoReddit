package storage

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
	mu sync.RWMutex
}

// Open invoked to create and setup initial database tables.
func Open(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000; PRAGMA journal_mode = WAL;"); err != nil {
		db.Close()
		return nil, err
	}
	// Create tables
	_, err = db.Exec(`
		-- Users table
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password TEXT NOT NULL,
			karma INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		-- Subreddits table
		CREATE TABLE IF NOT EXISTS subreddits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		-- model.Subreddit Members table
		CREATE TABLE IF NOT EXISTS subreddit_members (
			subreddit_id INTEGER,
			user_id INTEGER,
			joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (subreddit_id, user_id),
			FOREIGN KEY (subreddit_id) REFERENCES subreddits(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		-- Posts table
		CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			author_id INTEGER NOT NULL,
			subreddit_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (author_id) REFERENCES users(id),
			FOREIGN KEY (subreddit_id) REFERENCES subreddits(id)
		);

		-- Comments table (supports hierarchical comments)
		CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL,
			author_id INTEGER NOT NULL,
			post_id INTEGER,
			parent_comment_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (author_id) REFERENCES users(id),
			FOREIGN KEY (post_id) REFERENCES posts(id),
			FOREIGN KEY (parent_comment_id) REFERENCES comments(id)
		);

		-- Votes table (for posts and comments)
		CREATE TABLE IF NOT EXISTS votes (
			user_id INTEGER NOT NULL,
			target_id INTEGER NOT NULL,
			target_type TEXT CHECK(target_type IN ('post', 'comment')) NOT NULL,
			vote_value INTEGER CHECK(vote_value IN (-1, 1)) NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, target_id, target_type),
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		-- Direct Messages table
		CREATE TABLE IF NOT EXISTS direct_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_user_id INTEGER NOT NULL,
			to_user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (from_user_id) REFERENCES users(id),
			FOREIGN KEY (to_user_id) REFERENCES users(id)
		);

		-- model.User Subscriptions table
        CREATE TABLE IF NOT EXISTS user_subscriptions (
            subscriber_id INTEGER NOT NULL,
            subscribed_user_id INTEGER NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (subscriber_id, subscribed_user_id),
            FOREIGN KEY (subscriber_id) REFERENCES users(id),
            FOREIGN KEY (subscribed_user_id) REFERENCES users(id)
        );
	`)

	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return &Store{db: db}, nil
}

func (dm *Store) Close() {
	if dm.db != nil {
		dm.db.Close()
	}
}
