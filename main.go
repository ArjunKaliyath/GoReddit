package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
	"github.com/asynkron/protoactor-go/actor"
)

// DatabaseManager handles all database operations
type DatabaseManager struct {
	db *sql.DB
	mu sync.RWMutex
}

// InitDatabase invoked to create and setup initial database tables. 
func InitDatabase(dbPath string) (*DatabaseManager, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
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

		-- Subreddit Members table
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
			PRIMARY KEY (user_id, target_id, target_type, vote_value),
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

		-- User Subscriptions table
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
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	return &DatabaseManager{db: db}, nil
}

// Register User
func (dm *DatabaseManager) RegisterUser(username, password string) (int, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	query := `INSERT INTO users (username, password) VALUES (?, ?)`
	result, err := dm.db.Exec(query, username, password) 
	if err != nil {
		return 0, fmt.Errorf("failed to register user: %v", err)
	}

	id, err := result.LastInsertId()
	return int(id), err
}

func (dm *DatabaseManager) GetUserByUsername(username string) (*User, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var user User
	query := `SELECT id, username, karma FROM users WHERE username = ?`
	err := dm.db.QueryRow(query, username).Scan(&user.ID, &user.Username, &user.Karma)
	if err != nil {
		return nil, fmt.Errorf("user not found: %v", err)
	}

	return &user, nil
}

// Subreddit Operations
func (dm *DatabaseManager) CreateSubreddit(name, description string, creatorID int) (int, error) {
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
		return 0, fmt.Errorf("failed to create subreddit: %v", err)
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
		return 0, fmt.Errorf("failed to add creator to subreddit: %v", err)
	}

	err = tx.Commit()
	return int(subredditID), err
}

func (dm *DatabaseManager) JoinSubreddit(userID, subredditID int) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	_, err := dm.db.Exec(`
		INSERT OR IGNORE INTO subreddit_members (subreddit_id, user_id) 
		VALUES (?, ?)
	`, subredditID, userID)

	return err
}

func (dm *DatabaseManager) LeaveSubreddit(userID, subredditID int) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	_, err := dm.db.Exec(`
		DELETE FROM subreddit_members 
		WHERE subreddit_id = ? AND user_id = ?
	`, subredditID, userID)

	return err
}

// Create Reddit Post
func (dm *DatabaseManager) CreatePost(title, content string, authorID, subredditID int) (int, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	result, err := dm.db.Exec(`
		INSERT INTO posts (title, content, author_id, subreddit_id) 
		VALUES (?, ?, ?, ?)
	`, title, content, authorID, subredditID)

	if err != nil {
		return 0, fmt.Errorf("failed to create post: %v", err)
	}

	id, err := result.LastInsertId()
	return int(id), err
}

//Function to retrieve user's top feed items 
func (dm *DatabaseManager) GetFeed(userID int) ([]Post, error) {
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

	var posts []Post
	for rows.Next() {
		var post Post
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

	return posts, nil
}

// Function to let user upvote or downvote on a post and calculate User Karma
func (dm *DatabaseManager) Vote(userID, targetID int, targetType string, value int) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	tx, err := dm.db.Begin()
	if err != nil {
		return err
	}

	// Upsert vote
	_, err = tx.Exec(`
		INSERT INTO votes (user_id, target_id, target_type, vote_value) 
		VALUES (?, ?, ?, ?)
	`, userID, targetID, targetType, value)

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record vote: %v", err)
	}

	// Update karma based on vote type and target
	var updateQuery string
	if targetType == "post" {
		updateQuery = `
			UPDATE users 
			SET karma = karma + ? 
			WHERE id = (SELECT author_id FROM posts WHERE id = ?)
		`
	} else { // comment
		updateQuery = `
			UPDATE users 
			SET karma = karma + ? 
			WHERE id = (SELECT author_id FROM comments WHERE id = ?)
		`
	}

	_, err = tx.Exec(updateQuery, value, targetID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update karma: %v", err)
	}

	return tx.Commit()
}

// Function to let user comment on a post or reply to a comment
func (dm *DatabaseManager) CreateComment(content string, authorID, postID int, parentCommentID *int) (int, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	query := `
		INSERT INTO comments (content, author_id, post_id, parent_comment_id) 
		VALUES (?, ?, ?, ?)
	`

	result, err := dm.db.Exec(query, content, authorID, postID, parentCommentID)
	if err != nil {
		return 0, fmt.Errorf("failed to create comment: %v", err)
	}

	id, err := result.LastInsertId()
	return int(id), err
}

// Function to let users send messages to other users
func (dm *DatabaseManager) SendDirectMessage(fromUserID, toUserID int, content string) (int, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	result, err := dm.db.Exec(`
		INSERT INTO direct_messages (from_user_id, to_user_id, content) 
		VALUES (?, ?, ?)
	`, fromUserID, toUserID, content)

	if err != nil {
		return 0, fmt.Errorf("failed to send message: %v", err)
	}

	id, err := result.LastInsertId()
	return int(id), err
}

//Function to retrieve a user's received direct messages
func (dm *DatabaseManager) GetDirectMessages(userID int) ([]DirectMessage, error) {
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

	var messages []DirectMessage
	for rows.Next() {
		var msg DirectMessage
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

	return messages, nil
}

// Functions to let user subscribe and unsubscribe to other users.
func (dm *DatabaseManager) SubscribeToUser(subscriberID, subscribedUserID int) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	_, err := dm.db.Exec(`
        INSERT OR IGNORE INTO user_subscriptions 
        (subscriber_id, subscribed_user_id) 
        VALUES (?, ?)
    `, subscriberID, subscribedUserID)

	return err
}

func (dm *DatabaseManager) UnsubscribeFromUser(subscriberID, subscribedUserID int) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	_, err := dm.db.Exec(`
        DELETE FROM user_subscriptions 
        WHERE subscriber_id = ? AND subscribed_user_id = ?
    `, subscriberID, subscribedUserID)

	return err
}

func (dm *DatabaseManager) GetUserSubscriptions(userID int) ([]User, error) {
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

	var subscriptions []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.Karma)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, user)
	}

	return subscriptions, nil
}

// Function to close the database 
func (dm *DatabaseManager) Close() {
	if dm.db != nil {
		dm.db.Close()
	}
}

// Structs for database operations
type User struct {
	ID       string
	Username string
	Karma    int
}

type Post struct {
	ID             int
	Title          string
	Content        string
	AuthorID       int    `json:"author_id"`
	AuthorUsername string `json:"author_name"`
	SubredditID    int    `json:"subreddit_id"`
	SubredditName  string `json:"subreddit_name"`
	CreatedAt      time.Time
	VoteCount      struct {
		Upvotes   int `json:"upvotes"`
		Downvotes int `json:"downvotes"`
	} `json:"vote_count"`
}

type DirectMessage struct {
	ID           int
	FromUserID   int `json:"from_user_id"`
	FromUsername string
	Content      string
	CreatedAt    time.Time
}

// Request/Response structs
type RegisterUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type CreateSubredditRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
}

type CreatePostRequest struct {
	Title       string `json:"title" binding:"required"`
	Content     string `json:"content" binding:"required"`
	SubredditID int    `json:"subreddit_id" binding:"required"`
}

type CreateCommentRequest struct {
	Content         string `json:"content" binding:"required"`
	PostID          int    `json:"post_id" binding:"required"`
	ParentCommentID *int   `json:"parent_comment_id"`
}

type VoteRequest struct {
	TargetID   int    `json:"target_id" binding:"required"`
	TargetType string `json:"target_type" binding:"required,oneof=post comment"`
	Value      int    `json:"value" binding:"required,oneof=-1 1"`
}

type SendMessageRequest struct {
	ToUserID int    `json:"to_user_id" binding:"required"`
	Content  string `json:"content" binding:"required"`
}

type PostWithDetails struct {
	Post
	Votes     int       `json:"votes"`
	UserVote  *int      `json:"user_vote"` 
	Comments  []Comment `json:"comments"`
	VoteCount struct {
		Upvotes   int `json:"upvotes"`
		Downvotes int `json:"downvotes"`
	} `json:"vote_count"`
}

type Comment struct {
	ID              int       `json:"id"`
	Content         string    `json:"content"`
	AuthorID        int       `json:"author_id"`
	AuthorUsername  string    `json:"author_username"`
	PostID          int       `json:"post_id"`
	ParentCommentID *int      `json:"parent_comment_id"`
	CreatedAt       time.Time `json:"created_at"`
	Votes           int       `json:"votes"`
	UserVote        *int      `json:"user_vote"` 
}

type TopUser struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	Karma        int    `json:"karma"`
	PostCount    int    `json:"post_count"`
	CommentCount int    `json:"comment_count"`
}

type TopSubscribedUser struct {
	ID              int    `json:"id"`
	Username        string `json:"username"`
	Karma           int    `json:"karma"`
	SubscriberCount int    `json:"subscriber_count"`
}

// Subreddit represents a subreddit in the system
type Subreddit struct {
    ID          int       `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    CreatedAt   time.Time `json:"created_at"`
}

// API handler struct
type APIHandler struct {
	db *DatabaseManager
}


func NewAPIHandler(dbPath string) (*APIHandler, error) {
	dbManager, err := InitDatabase(dbPath)
	if err != nil {
		return nil, err
	}
	return &APIHandler{db: dbManager}, nil
}

// Middleware to authenticate user based on user ID as a parameter
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// In a real application, implement proper authentication
		// For now, we'll use a simple user_id header
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID required"})
			c.Abort()
			return
		}
		c.Set("user_id", userID)
		c.Next()
	}
}

//Function to get users with highest karma after the simulation 
func (dm *DatabaseManager) GetTopUsers(limit int) ([]TopUser, error) {
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

	var users []TopUser
	for rows.Next() {
		var user TopUser
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

	return users, nil
}

//Function to get details of most subscribed users
func (dm *DatabaseManager) GetTopSubscribedUsers(limit int) ([]TopSubscribedUser, error) {
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

	var users []TopSubscribedUser
	for rows.Next() {
		var user TopSubscribedUser
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

	return users, nil
}

//Function to get posts with highest difference between upvotes and downvotes
func (dm *DatabaseManager) GetTopPosts(limit int) ([]Post, error) {
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

	var posts []Post
	for rows.Next() {
		var post Post
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

	return posts, nil
}

// GetAllSubreddits retrieves all subreddits with their IDs
func (dm *DatabaseManager) GetAllSubreddits() ([]Subreddit, error) {
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

	var subreddits []Subreddit
	for rows.Next() {
		var subreddit Subreddit
		err := rows.Scan(
			&subreddit.ID, &subreddit.Name, 
			&subreddit.Description, &subreddit.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		subreddits = append(subreddits, subreddit)
	}

	return subreddits, nil
}

// GetUserJoinedSubreddits retrieves subreddits a user has joined
func (dm *DatabaseManager) GetUserJoinedSubreddits(userID int) ([]Subreddit, error) {
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

	var subreddits []Subreddit
	for rows.Next() {
		var subreddit Subreddit
		err := rows.Scan(
			&subreddit.ID, &subreddit.Name, 
			&subreddit.Description, &subreddit.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		subreddits = append(subreddits, subreddit)
	}

	return subreddits, nil
}

//Function to clear the database after all simulation operations are done. 
func (dm *DatabaseManager) ResetDatabase() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	tables := []string{
		"direct_messages",
		"votes",
		"comments",
		"posts",
		"subreddit_members",
		"subreddits",
		"users",
	}

	tx, err := dm.db.Begin()
	if err != nil {
		return err
	}

	// Delete all rows from tables
	for _, table := range tables {
		_, err = tx.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete from %s: %v", table, err)
		}
	}


	for _, table := range tables {
		_, err = tx.Exec(fmt.Sprintf("DELETE FROM sqlite_sequence WHERE name='%s'", table))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to reset auto-increment for %s: %v", table, err)
		}
	}

	return tx.Commit()
}

// API handlers
func (h *APIHandler) getTopPosts(c *gin.Context) {
	limit := 5 // Default to top 5 posts
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	posts, err := h.db.GetTopPosts(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, posts)
}

func (h *APIHandler) resetDatabase(c *gin.Context) {
	
	err := h.db.ResetDatabase()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Database reset successfully"})
}

func (h *APIHandler) registerUser(c *gin.Context) {
	var req RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := h.db.RegisterUser(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user_id":  userID,
		"username": req.Username,
	})
}

func (h *APIHandler) getUserByUsername(c *gin.Context) {
	username := c.Param("username")
	user, err := h.db.GetUserByUsername(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *APIHandler) getFeed(c *gin.Context) {
	userID, _ := strconv.Atoi(c.GetString("user_id"))
	posts, err := h.db.GetFeed(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, posts)
}


func (h *APIHandler) getDirectMessages(c *gin.Context) {
	userID, _ := strconv.Atoi(c.GetString("user_id"))
	messages, err := h.db.GetDirectMessages(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}
func (h *APIHandler) getTopUsers(c *gin.Context) {
	limit := 10 // Default limit
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	users, err := h.db.GetTopUsers(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (h *APIHandler) subscribeToUser(c *gin.Context) {
	userToSubscribe, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	subscriberID, _ := strconv.Atoi(c.GetString("user_id"))
	err = h.db.SubscribeToUser(subscriberID, userToSubscribe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully subscribed to user"})
}

func (h *APIHandler) unsubscribeFromUser(c *gin.Context) {
	userToUnsubscribe, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	subscriberID, _ := strconv.Atoi(c.GetString("user_id"))
	err = h.db.UnsubscribeFromUser(subscriberID, userToUnsubscribe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully unsubscribed from user"})
}

func (h *APIHandler) getUserSubscriptions(c *gin.Context) {
	userID, _ := strconv.Atoi(c.GetString("user_id"))
	subscriptions, err := h.db.GetUserSubscriptions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, subscriptions)
}

func (h *APIHandler) getTopSubscribedUsers(c *gin.Context) {
	limit := 10 // Default limit
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	users, err := h.db.GetTopSubscribedUsers(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// RequestProcessingActor represents a worker actor in the pool
type RequestProcessingActor struct {
	handler *APIHandler
	id      int
}

// Request represents a generic request to be processed by the actor
type Request struct {
	Type    string
	Payload interface{}
	Context *gin.Context
	Result  chan error
}

// ActorPool manages a pool of request processing actors
type ActorPool struct {
	system     *actor.ActorSystem
	actors     []*actor.PID
	roundRobin int
	mu         sync.Mutex
}

// NewActorPool creates a pool of actors
func NewActorPool(system *actor.ActorSystem, handler *APIHandler, poolSize int) *ActorPool {
	pool := &ActorPool{
		system: system,
		actors: make([]*actor.PID, poolSize),
	}

	// Create pool of actors
	for i := 0; i < poolSize; i++ {
		props := actor.PropsFromProducer(func() actor.Actor {
			return &RequestProcessingActor{
				handler: handler,
				id:      i,
			}
		})
		pool.actors[i] = system.Root.Spawn(props)
	}

	return pool
}

// ProcessRequest sends a request to the next actor in a round-robin fashion
func (p *ActorPool) ProcessRequest(requestType string, payload interface{}, context *gin.Context) error {
	p.mu.Lock()
	actor := p.actors[p.roundRobin]
	p.roundRobin = (p.roundRobin + 1) % len(p.actors)
	p.mu.Unlock()

	// Create a channel to receive the result
	resultChan := make(chan error, 1)

	// Send request to the selected actor
	p.system.Root.Send(actor, &Request{
		Type:    requestType,
		Payload: payload,
		Context: context,
		Result:  resultChan,
	})

	// Wait for and return the result
	return <-resultChan
}

// Create a custom Gin handler that uses the actor pool
func ActorPoolHandler(pool *ActorPool, requestType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload interface{}
		var err error

		// Parse payload based on request type
		switch requestType {
		case "create_post":
			var req CreatePostRequest
			err = c.ShouldBindJSON(&req)
			payload = req
		case "create_comment":
			var req CreateCommentRequest
			err = c.ShouldBindJSON(&req)
			payload = req
		case "send_message":
			var req SendMessageRequest
			err = c.ShouldBindJSON(&req)
			payload = req
		case "join_subreddit":
			var req JoinSubredditRequest
			subredditID, parseErr := strconv.Atoi(c.Param("id"))
			if parseErr != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subreddit ID"})
                return
            }
			req.SubredditID = subredditID
            payload = req
		case "leave_subreddit":
            var req LeaveSubredditRequest
            // Parse the subreddit ID from the URL parameter
            subredditID, parseErr := strconv.Atoi(c.Param("id"))
            if parseErr != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subreddit ID"})
                return
            }
            req.SubredditID = subredditID
            payload = req
		case "create_subreddit":
			var req CreateSubredditRequest
			err = c.ShouldBindJSON(&req)
			payload = req
		case "vote":
			var req VoteRequest
			err = c.ShouldBindJSON(&req)
			payload = req
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request type"})
			return
		}

		// Handle parsing error
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Process request through actor pool
		if err := pool.ProcessRequest(requestType, payload, c); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
	}
}

// Additional request type structs (if not already defined)
type JoinSubredditRequest struct {
	SubredditID int `json:"subreddit_id" binding:"required"`
}

type LeaveSubredditRequest struct {
    SubredditID int `json:"subreddit_id" binding:"required"`
}

func (a *RequestProcessingActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *Request:
		log.Printf("Worker %d processing request of type %s", a.id, msg.Type)
		
		var err error
		switch msg.Type {
		case "create_post":
			err = a.processCreatePost(msg)
		case "create_comment":
			err = a.processCreateComment(msg)
		case "send_message":
			err = a.processSendMessage(msg)
		case "join_subreddit":
			err = a.processJoinSubreddit(msg)
		case "create_subreddit":
			err = a.processCreateSubreddit(msg)
		case "vote":
			err = a.processVote(msg)
		case "leave_subreddit":
            err = a.processLeaveSubreddit(msg)  
		default:
			err = fmt.Errorf("unhandled request type: %s", msg.Type)
		}

		// If an error occurred during processing, send it back through the result channel
		if err != nil {
			msg.Result <- err
		} else {
			msg.Result <- nil
		}
	}
}

// getUserJoinedSubreddits handles retrieving subreddits user has joined
func (h *APIHandler) getUserJoinedSubreddits(c *gin.Context) {
	userID, _ := strconv.Atoi(c.GetString("user_id"))
	subreddits, err := h.db.GetUserJoinedSubreddits(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, subreddits)
}

// getAllSubreddits handles retrieving all subreddits
func (h *APIHandler) getAllSubreddits(c *gin.Context) {
	subreddits, err := h.db.GetAllSubreddits()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, subreddits)
}

//Actor API handlers
func (a *RequestProcessingActor) processCreatePost(req *Request) error {
	postReq, ok := req.Payload.(CreatePostRequest)
	if !ok {
		req.Context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return fmt.Errorf("invalid payload")
	}

	userID, _ := strconv.Atoi(req.Context.GetString("user_id"))
	postID, err := a.handler.db.CreatePost(postReq.Title, postReq.Content, userID, postReq.SubredditID)
	if err != nil {
		req.Context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return err
	}

	req.Context.JSON(http.StatusCreated, gin.H{
		"post_id": postID,
		"title":   postReq.Title,
	})
	return nil
}

func (a *RequestProcessingActor) processCreateComment(req *Request) error {
	// Type assert the payload to CreateCommentRequest
	commentReq, ok := req.Payload.(CreateCommentRequest)
	if !ok {
		return fmt.Errorf("invalid payload for create comment")
	}

	// Extract user ID from context
	userID, _ := strconv.Atoi(req.Context.GetString("user_id"))

	// Call database method to create comment
	commentID, err := a.handler.db.CreateComment(
		commentReq.Content, 
		userID, 
		commentReq.PostID, 
		commentReq.ParentCommentID,
	)
	if err != nil {
		req.Context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return err
	}

	// Respond with created comment details
	req.Context.JSON(http.StatusCreated, gin.H{
		"comment_id": commentID,
		"content":    commentReq.Content,
	})
	return nil
}

func (a *RequestProcessingActor) processSendMessage(req *Request) error {
	// Type assert the payload to SendMessageRequest
	messageReq, ok := req.Payload.(SendMessageRequest)
	if !ok {
		return fmt.Errorf("invalid payload for send message")
	}

	// Extract sender user ID from context
	userID, _ := strconv.Atoi(req.Context.GetString("user_id"))

	// Call database method to send direct message
	messageID, err := a.handler.db.SendDirectMessage(
		userID, 
		messageReq.ToUserID, 
		messageReq.Content,
	)
	if err != nil {
		req.Context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return err
	}

	// Respond with sent message details
	req.Context.JSON(http.StatusCreated, gin.H{
		"message_id": messageID,
		"content":    messageReq.Content,
	})
	return nil
}

// Additional actor-based handlers for other complex operations

func (a *RequestProcessingActor) processJoinSubreddit(req *Request) error {
	// Type assert the payload to JoinSubredditRequest
	joinReq, ok := req.Payload.(JoinSubredditRequest)
	if !ok {
		return fmt.Errorf("invalid payload for join subreddit")
	}

	// Extract user ID from context
	userID, _ := strconv.Atoi(req.Context.GetString("user_id"))

	// Call database method to join subreddit
	err := a.handler.db.JoinSubreddit(userID, joinReq.SubredditID)
	if err != nil {
		req.Context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return err
	}

	req.Context.JSON(http.StatusOK, gin.H{"message": "Successfully joined subreddit"})
	return nil
}

func (a *RequestProcessingActor) processLeaveSubreddit(req *Request) error {
    // Type assert the payload to LeaveSubredditRequest
    leaveReq, ok := req.Payload.(LeaveSubredditRequest)
    if !ok {
        return fmt.Errorf("invalid payload for leave subreddit")
    }

    // Extract user ID from context
    userID, _ := strconv.Atoi(req.Context.GetString("user_id"))

    // Call database method to leave subreddit
    err := a.handler.db.LeaveSubreddit(userID, leaveReq.SubredditID)
    if err != nil {
        req.Context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return err
    }

    req.Context.JSON(http.StatusOK, gin.H{"message": "Successfully left subreddit"})
    return nil
}

func (a *RequestProcessingActor) processCreateSubreddit(req *Request) error {
	// Type assert the payload to CreateSubredditRequest
	subredditReq, ok := req.Payload.(CreateSubredditRequest)
	if !ok {
		return fmt.Errorf("invalid payload for create subreddit")
	}

	// Extract user ID from context
	userID, _ := strconv.Atoi(req.Context.GetString("user_id"))

	// Call database method to create subreddit
	subredditID, err := a.handler.db.CreateSubreddit(
		subredditReq.Name, 
		subredditReq.Description, 
		userID,
	)
	if err != nil {
		req.Context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return err
	}

	req.Context.JSON(http.StatusCreated, gin.H{
		"subreddit_id": subredditID,
		"name":         subredditReq.Name,
	})
	return nil
}

func (a *RequestProcessingActor) processVote(req *Request) error {
	// Type assert the payload to VoteRequest
	voteReq, ok := req.Payload.(VoteRequest)
	if !ok {
		return fmt.Errorf("invalid payload for vote")
	}

	// Extract user ID from context
	userID, _ := strconv.Atoi(req.Context.GetString("user_id"))

	// Call database method to record vote
	err := a.handler.db.Vote(
		userID, 
		voteReq.TargetID, 
		voteReq.TargetType, 
		voteReq.Value,
	)
	if err != nil {
		req.Context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return err
	}

	req.Context.JSON(http.StatusOK, gin.H{"message": "Vote recorded successfully"})
	return nil
}


//main function - code invocation starts from here 
func main() {
	// Create actor system
	actorSystem := actor.NewActorSystem()

	handler, err := NewAPIHandler("reddit_clone.db")
	if err != nil {
		log.Fatalf("Failed to initialize API handler: %v", err)
	}
	defer handler.db.Close()

	r := gin.Default()

	// Create actor pool (with 5 workers)
	actorPool := NewActorPool(actorSystem, handler, 5)

	// Public routes
	r.POST("/register", handler.registerUser)
	r.GET("/users/:username", handler.getUserByUsername)

	// Protected routes 
	authorized := r.Group("/")
	authorized.Use(authMiddleware())
	{
		// Use actor pool handlers for more complex operations
		authorized.POST("/posts", ActorPoolHandler(actorPool, "create_post"))
		authorized.POST("/comments", ActorPoolHandler(actorPool, "create_comment"))
		authorized.POST("/messages", ActorPoolHandler(actorPool, "send_message"))
		authorized.POST("/subreddits", ActorPoolHandler(actorPool, "create_subreddit"))
		authorized.POST("/subreddits/:id/join", ActorPoolHandler(actorPool, "join_subreddit"))
		authorized.POST("/vote", ActorPoolHandler(actorPool, "vote"))
		authorized.POST("/subreddits/:id/leave", ActorPoolHandler(actorPool, "leave_subreddit"))

		// other routes that don't need complex processing
		authorized.GET("/feed", handler.getFeed)
		authorized.GET("/messages", handler.getDirectMessages)
		authorized.GET("/users/top", handler.getTopUsers)
		authorized.GET("/posts/top", handler.getTopPosts)
		authorized.POST("/reset-database", handler.resetDatabase)
		authorized.GET("/subscriptions", handler.getUserSubscriptions)
		authorized.GET("/users/top-subscribed", handler.getTopSubscribedUsers)
		authorized.POST("/users/:user_id/subscribe", handler.subscribeToUser)
		authorized.POST("/users/:user_id/unsubscribe", handler.unsubscribeFromUser)
		authorized.GET("/subreddits/all", handler.getAllSubreddits)
		authorized.GET("/subreddits/joined", handler.getUserJoinedSubreddits)
		
	}

	r.Run(":8080") // start running backend server on port 8080
}
