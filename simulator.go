package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/manifoldco/promptui"
)

const baseURL = "http://localhost:8080"

type Client struct {
	userID     string
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{},
	}
}

func (c *Client) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, baseURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	// Add user ID to headers for authentication
	if c.userID != "" {
		req.Header.Set("X-User-ID", c.userID)
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

func (c *Client) Register() error {
	prompt := promptui.Prompt{
		Label: "Enter username",
	}
	username, err := prompt.Run()
	if err != nil {
		return err
	}

	passwordPrompt := promptui.Prompt{
		Label: "Enter password",
		Mask:  '*',
	}
	password, err := passwordPrompt.Run()
	if err != nil {
		return err
	}

	body := map[string]string{
		"username": username,
		"password": password,
	}

	resp, err := c.makeRequest("POST", "/register", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("registration failed: %v", response["error"])
	}

	c.userID = fmt.Sprintf("%v", response["user_id"])
	fmt.Printf("Registered successfully! Your User ID is: %s\n", c.userID)
	return nil
}

func (c *Client) CreateSubreddit() error {
	namePrompt := promptui.Prompt{
		Label: "Enter subreddit name",
	}
	name, err := namePrompt.Run()
	if err != nil {
		return err
	}

	descPrompt := promptui.Prompt{
		Label: "Enter subreddit description",
	}
	description, err := descPrompt.Run()
	if err != nil {
		return err
	}

	body := map[string]string{
		"name":        name,
		"description": description,
	}

	resp, err := c.makeRequest("POST", "/subreddits", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("subreddit creation failed: %v", response["error"])
	}

	fmt.Printf("Subreddit created successfully! Subreddit ID: %v\n", response["subreddit_id"])
	return nil
}

func (c *Client) CreatePost() error {

	resp, err := c.makeRequest("GET", "/subreddits/joined", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var joinedSubreddits []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&joinedSubreddits); err != nil {
		return fmt.Errorf("failed to decode joined subreddits: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch joined subreddits")
	}

	// Display joined subreddits
	fmt.Println("Subreddits You've Joined:")
	if len(joinedSubreddits) == 0 {
		fmt.Println("You haven't joined any subreddits yet. Please join a subreddit first.")
		return nil
	}

	for _, subreddit := range joinedSubreddits {
		fmt.Printf("ID: %v | Name: %v | Description: %v\n",
			subreddit["id"],
			subreddit["name"],
			subreddit["description"])
	}

	titlePrompt := promptui.Prompt{
		Label: "Enter post title",
	}
	title, err := titlePrompt.Run()
	if err != nil {
		return err
	}

	contentPrompt := promptui.Prompt{
		Label: "Enter post content",
	}
	content, err := contentPrompt.Run()
	if err != nil {
		return err
	}

	subredditIDPrompt := promptui.Prompt{
		Label: "Enter subreddit ID",
	}
	subredditIDStr, err := subredditIDPrompt.Run()
	if err != nil {
		return err
	}

	subredditID, err := strconv.Atoi(subredditIDStr)
	if err != nil {
		return fmt.Errorf("invalid subreddit ID")
	}


	body := map[string]interface{}{
		"title":        title,
		"content":      content,
		"subreddit_id": subredditID,
	}

	resp2, err := c.makeRequest("POST", "/posts", body)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	var response map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&response)

	if resp2.StatusCode != http.StatusCreated {
		return fmt.Errorf("post creation failed: %v", response["error"])
	}

	fmt.Printf("Post created successfully! Post ID: %v\n", response["post_id"])
	return nil
}

func (c *Client) ViewFeed() error {
	resp, err := c.makeRequest("GET", "/feed", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var posts []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&posts)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch feed")
	}

	fmt.Println("Feed Posts:")
	for _, post := range posts {
		fmt.Printf("Title: %v\n", post["Title"])
		fmt.Printf("Author: %v\n", post["author_name"])
		fmt.Printf("Subreddit: %v\n", post["subreddit_name"])
		fmt.Printf("Content: %v\n", post["Content"])
		fmt.Printf("Upvotes: %v, Downvotes: %v\n\n",
			post["vote_count"].(map[string]interface{})["upvotes"],
			post["vote_count"].(map[string]interface{})["downvotes"])
	}
	return nil
}

func (c *Client) Vote() error {
	resp, err := c.makeRequest("GET", "/feed", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var posts []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&posts)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch feed")
	}

	// Display feed posts with their IDs
	fmt.Println("Feed Posts:")
	if len(posts) == 0 {
		fmt.Println("No posts available. Please create or join a subreddit first.")
		return nil
	}

	for _, post := range posts {
		fmt.Printf("Post ID: %v\n", post["ID"])
		fmt.Printf("Title: %v\n", post["Title"])
		fmt.Printf("Author: %v\n", post["author_name"])
		fmt.Printf("Subreddit: %v\n", post["subreddit_name"])
		fmt.Printf("Content: %v\n", post["Content"])
		fmt.Printf("Upvotes: %v, Downvotes: %v\n\n",
			post["vote_count"].(map[string]interface{})["upvotes"],
			post["vote_count"].(map[string]interface{})["downvotes"])
	}

	targetIDPrompt := promptui.Prompt{
		Label: "Enter target ID (post/comment ID)",
	}
	targetIDStr, err := targetIDPrompt.Run()
	if err != nil {
		return err
	}
	targetID, err := strconv.Atoi(targetIDStr)
	if err != nil {
		return fmt.Errorf("invalid target ID")
	}

	if err != nil{
		return fmt.Errorf("invalid post ID")
	}

	typePrompt := promptui.Select{
		Label: "Select target type",
		Items: []string{"post", "comment"},
	}
	_, targetType, err := typePrompt.Run()
	if err != nil {
		return err
	}

	valuePrompt := promptui.Select{
		Label: "Select vote",
		Items: []string{"Upvote (+1)", "Downvote (-1)"},
	}
	_, voteStr, err := valuePrompt.Run()
	if err != nil {
		return err
	}

	var voteValue int
	if voteStr == "Upvote (+1)" {
		voteValue = 1
	} else {
		voteValue = -1
	}

	body := map[string]interface{}{
		"target_id":   targetID,
		"target_type": targetType,
		"value":       voteValue,
	}

	resp2, err := c.makeRequest("POST", "/vote", body)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	var response map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&response)

	if resp2.StatusCode != http.StatusOK {
		return fmt.Errorf("voting failed: %v", response["error"])
	}

	fmt.Println("Vote recorded successfully!")
	return nil
}

func (c *Client) SendMessage() error {

	resp, err := c.makeRequest("GET", "/subscriptions", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var subscriptions []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&subscriptions); err != nil {
		return fmt.Errorf("failed to decode subscriptions: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch subscriptions")
	}

	// Display subscribed users
	fmt.Println("Users You're Subscribed To:")
	if len(subscriptions) == 0 {
		fmt.Println("You haven't subscribed to any users yet.")
	} else {
		for _, user := range subscriptions {
			fmt.Printf("User ID: %v | Username: %v\n",
				user["ID"],
				user["Username"])
		}
	}

	userIDPrompt := promptui.Prompt{
		Label: "Enter recipient user ID",
	}
	userIDStr, err := userIDPrompt.Run()
	if err != nil {
		return err
	}
	toUserID, err := strconv.Atoi(userIDStr)
	if err != nil{
		return fmt.Errorf("invalid user ID")
	}

	contentPrompt := promptui.Prompt{
		Label: "Enter message content",
	}
	content, err := contentPrompt.Run()
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"to_user_id": toUserID,
		"content":    content,
	}

	resp2, err := c.makeRequest("POST", "/messages", body)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	var response map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&response)

	if resp2.StatusCode != http.StatusCreated {
		return fmt.Errorf("message sending failed: %v", response["error"])
	}

	fmt.Println("Message sent successfully!")
	return nil
}

func (c *Client) ViewMessages() error {
	resp, err := c.makeRequest("GET", "/messages", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var messages []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&messages)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch messages")
	}

	fmt.Println("Received Messages:")
	for _, msg := range messages {
		fmt.Printf("From: %v\n", msg["FromUsername"])
		fmt.Printf("Content: %v\n", msg["Content"])
		fmt.Printf("Sent at: %v\n\n", msg["CreatedAt"])
	}
	return nil
}

func (c *Client) SubscribeToUser() error {
	userIDPrompt := promptui.Prompt{
		Label: "Enter user ID to subscribe to",
	}
	userIDStr, err := userIDPrompt.Run()
	if err != nil {
		return err
	}
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return fmt.Errorf("invalid user ID")
	}

	resp, err := c.makeRequest("POST", fmt.Sprintf("/users/%d/subscribe", userID), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("subscription failed: %v", response["error"])
	}

	fmt.Println("Successfully subscribed to user!")
	return nil
}

func (c *Client) ViewTopUsers() error {
	resp, err := c.makeRequest("GET", "/users/top", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var users []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&users)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch top users")
	}

	fmt.Println("Top Users:")
	for _, user := range users {
		fmt.Printf("Username: %v\n", user["username"])
		fmt.Printf("Karma: %v\n", user["karma"])
		fmt.Printf("Posts: %v\n", user["post_count"])
		fmt.Printf("Comments: %v\n\n", user["comment_count"])
	}
	return nil
}

func (c *Client) JoinSubreddit() error {

	resp, err := c.makeRequest("GET", "/subreddits/all", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var subreddits []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&subreddits); err != nil {
		return fmt.Errorf("failed to decode subreddits: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch subreddits")
	}

	// Display available subreddits
	fmt.Println("Available Subreddits:")
	for _, subreddit := range subreddits {
		fmt.Printf("ID: %v | Name: %v | Description: %v \n",
			subreddit["id"],
			subreddit["name"],
			subreddit["description"])
	}

	subredditIDPrompt := promptui.Prompt{
		Label: "Enter subreddit ID to join",
	}
	subredditIDStr, err := subredditIDPrompt.Run()
	if err != nil {
		return err
	}

	subredditID, err := strconv.Atoi(subredditIDStr)

	if err != nil {
		return fmt.Errorf("invalid subreddit ID")
	}

	resp2, err := c.makeRequest("POST", fmt.Sprintf("/subreddits/%d/join", subredditID), nil)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	var response map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&response)

	if resp2.StatusCode != http.StatusOK {
		return fmt.Errorf("subreddit join failed: %v", response["error"])
	}

	fmt.Println("Successfully joined the subreddit!")
	return nil
}

func (c *Client) LeaveSubreddit() error {

	resp, err := c.makeRequest("GET", "/subreddits/joined", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var joinedSubreddits []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&joinedSubreddits); err != nil {
		return fmt.Errorf("failed to decode joined subreddits: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch joined subreddits")
	}

	// Display joined subreddits
	fmt.Println("Subreddits You've Joined:")
	if len(joinedSubreddits) == 0 {
		fmt.Println("You haven't joined any subreddits yet.")
		return nil
	}

	for _, subreddit := range joinedSubreddits {
		fmt.Printf("ID: %v | Name: %v | Description: %v \n",
			subreddit["id"],
			subreddit["name"],
			subreddit["description"])
	}

	subredditIDPrompt := promptui.Prompt{
		Label: "Enter subreddit ID to leave",
	}
	subredditIDStr, err := subredditIDPrompt.Run()
	if err != nil {
		return err
	}
	subredditID, err := strconv.Atoi(subredditIDStr)

	if err != nil{
		return fmt.Errorf("invalid subreddit ID")
	}
	resp2, err := c.makeRequest("POST", fmt.Sprintf("/subreddits/%d/leave", subredditID), nil)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	var response map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&response)

	if resp2.StatusCode != http.StatusOK {
		return fmt.Errorf("subreddit leave failed: %v", response["error"])
	}

	fmt.Println("Successfully left the subreddit!")
	return nil
}

func (c *Client) CreateComment() error {

	resp, err := c.makeRequest("GET", "/feed", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var posts []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&posts)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch feed")
	}

	// Display feed posts with their IDs
	fmt.Println("Feed Posts:")
	if len(posts) == 0 {
		fmt.Println("No posts available. Please create or join a subreddit first.")
		return nil
	}

	for _, post := range posts {
		fmt.Printf("Post ID: %v\n", post["ID"])
		fmt.Printf("Title: %v\n", post["Title"])
		fmt.Printf("Author: %v\n", post["author_name"])
		fmt.Printf("Subreddit: %v\n", post["subreddit_name"])
		fmt.Printf("Content: %v\n", post["Content"])
		fmt.Printf("Upvotes: %v, Downvotes: %v\n\n",
			post["vote_count"].(map[string]interface{})["upvotes"],
			post["vote_count"].(map[string]interface{})["downvotes"])
	}

	postIDPrompt := promptui.Prompt{
		Label: "Enter post ID to comment on",
	}
	postIDStr, err := postIDPrompt.Run()
	if err != nil {
		return err
	}
	postID, err := strconv.Atoi(postIDStr)


	if err != nil{
		return fmt.Errorf("invalid post ID")
	}

	contentPrompt := promptui.Prompt{
		Label: "Enter comment content",
	}
	content, err := contentPrompt.Run()
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"post_id": postID,
		"content": content,
	}

	resp2, err := c.makeRequest("POST", "/comments", body)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	var response map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&response)

	if resp2.StatusCode != http.StatusCreated {
		return fmt.Errorf("comment creation failed: %v", response["error"])
	}

	fmt.Printf("Comment created successfully! Comment ID: %v\n", response["comment_id"])
	return nil
}

func main() {
	client := NewClient()

	log.SetOutput(os.Stdout)
    log.SetFlags(0)

	for {
		prompt := promptui.Select{
			Label: "Reddit Clone API Client",
			Items: []string{
				"Register",
				"Create Subreddit",
				"Create Post",
				"Comment",
				"View Feed",
				"Join Subreddit",
				"Leave Subreddit",
				"Vote",
				"Send Message",
				"View Messages",
				"Subscribe to User",
				"View Top Users",
				"Exit",
			},
		}

		_, result, err := prompt.Run()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return
		}

		var actionErr error
		switch result {
		case "Register":
			if client.userID == "" {
				actionErr = client.Register()
			} else {
				fmt.Printf("You have already registered.\n")
			}
		case "Create Subreddit":
			if client.userID == "" {
				log.Printf("You need to register before accessing the system.")
			} else {
				actionErr = client.CreateSubreddit()
			}
		case "Create Post":
			if client.userID == "" {
				log.Printf("You need to register before accessing the system.")
			} else {
				actionErr = client.CreatePost()
			}
		case "View Feed":
			if client.userID == "" {
				log.Printf("You need to register before accessing the system.")
			} else {
				actionErr = client.ViewFeed()
			}
		case "Vote":
			if client.userID == "" {
				log.Printf("You need to register before accessing the system.")
			} else {
				actionErr = client.Vote()
			}
		case "Send Message":
			if client.userID == "" {
				log.Printf("You need to register before accessing the system.")
			} else {
				actionErr = client.SendMessage()
			}
		case "View Messages":
			if client.userID == "" {
				log.Printf("You need to register before accessing the system.")
			} else {
				actionErr = client.ViewMessages()
			}
		case "Subscribe to User":
			if client.userID == "" {
				log.Printf("You need to register before accessing the system.")
			} else {
				actionErr = client.SubscribeToUser()
			}
		case "View Top Users":
			if client.userID == "" {
				log.Printf("You need to register before accessing the system.")
			} else {
				actionErr = client.ViewTopUsers()
			}
		case "Join Subreddit":
			if client.userID == "" {
				log.Printf("You need to register before accessing the system.")
			} else {
				actionErr = client.JoinSubreddit()
			}
		case "Leave Subreddit":
			if client.userID == "" {
				log.Printf("You need to register before accessing the system.")
			} else {
				actionErr = client.LeaveSubreddit()
			}
		case "Comment":
			if client.userID == "" {
				log.Printf("You need to register before accessing the system.")
			} else {
				actionErr = client.CreateComment()
			}
		case "Exit":
			fmt.Println("Exiting...")
			os.Exit(0)

		}

		if actionErr != nil {
			fmt.Printf("Error: %v\n", actionErr)
		}

		fmt.Println("\nPress Enter to continue...")
		fmt.Scanln()
	}
}
