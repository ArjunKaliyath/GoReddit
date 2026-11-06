package model

import "time"

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Karma    int    `json:"karma"`
}

type Post struct {
	ID             int       `json:"id"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	AuthorID       int       `json:"author_id"`
	AuthorUsername string    `json:"author_name"`
	SubredditID    int       `json:"subreddit_id"`
	SubredditName  string    `json:"subreddit_name"`
	CreatedAt      time.Time `json:"created_at"`
	VoteCount      struct {
		Upvotes   int `json:"upvotes"`
		Downvotes int `json:"downvotes"`
	} `json:"vote_count"`
}

type DirectMessage struct {
	ID           int       `json:"id"`
	FromUserID   int       `json:"from_user_id"`
	FromUsername string    `json:"from_username"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
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
