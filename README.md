# GoReddit Implementation

A backend service implementation of a Reddit-like platform using Go, Gin web framework, and SQLite. This application provides core social media functionality including user management, subreddits, posts, comments, voting, direct messaging, and user subscriptions.

## Architecture Overview

The solution consists of two main components:

1. **Server Process**: Implements the Reddit engine and API endpoints with an actor model implementation for request routing
2. **Client Process**: Provides a CLI-based UI for simulating user actions through REST API calls

## Key Components

### 1. Database Management

The solution uses SQLite for data persistence due to its:
- Ability to handle larger datasets that might not fit in memory
- Lightweight nature requiring no separate database server

#### Schema includes tables for:
- Users
- Subreddits
- Subreddit Members
- Posts
- Comments
- Votes
- Direct Messages
- User Subscriptions

### 2. Core Functionality

#### User Management
- User registration
- Karma tracking
- User subscriptions
- Top users ranking

#### Subreddit Management
- Create subreddits
- Join/leave subreddits
- Subreddit member tracking

#### Content Interaction
- Create posts
- Create comments (supports nested comments)
- Voting system (Upvotes and Downvotes)
- Direct messaging

### 3. Authentication and Security
- Basic authentication middleware
- User ID-based authentication

## API Endpoints

### User APIs
- `POST /register` - Register a new user
- `GET /users/:username` - Get user details by username
- `GET /users/top` - Get top users ranked by karma
- `POST /users/:user_id/subscribe` - Subscribe to another user
- `POST /users/:user_id/unsubscribe` - Unsubscribe from a user
- `GET /subscriptions` - Get list of users the current user is subscribed to
- `GET /users/top-subscribed` - Get top users with the most subscribers

### Subreddit APIs
- `POST /subreddits` - Create a new subreddit
- `POST /subreddits/:id/join` - Join a subreddit
- `POST /subreddits/:id/leave` - Leave a subreddit
- `GET /subreddits/all` - Displays all subreddits
- `GET /subreddits/joined` - Gets list of all subreddits that the user has joined

### Post APIs
- `POST /posts` - Create a new post
- `GET /feed` - Get personalized feed of posts from joined subreddits
- `GET /posts/top` - Get top posts ranked by votes

### Voting APIs
- `POST /vote` - Vote on a post or comment (upvote or downvote)

### Comment APIs
- `POST /comments` - Create a new comment on a post

### Direct Messaging APIs
- `POST /messages` - Send a direct message to another user
- `GET /messages` - Get direct messages for the current user

### Utility APIs
- `POST /reset-database` - Reset the entire database and clear all simulated records

## Installation and Setup

### Prerequisites
- Go installation is required

### Setup Steps

1. **Open Terminals**
   - Open two terminals in the project directory
   - One for the server, one for the client

2. **Initialize Go Module**
   ```bash
   go mod init module_name
   go mod tidy
   go get github.com/gin-gonic/gin
   go get github.com/asynkron/protoactor-go/actor
   go get github.com/manifoldco/promptui
   ```

3. **Run the Server**
   ```bash
   go run main.go
   ```
   Server will be available at `localhost:8080`

4. **Run the Client Simulator**
   ```bash
   go run simulator.go
   ```
