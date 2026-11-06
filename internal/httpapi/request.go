package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type input struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Title           string `json:"title"`
	Content         string `json:"content"`
	SubredditID     int    `json:"subreddit_id"`
	PostID          int    `json:"post_id"`
	ParentCommentID *int   `json:"parent_comment_id"`
	TargetID        int    `json:"target_id"`
	TargetType      string `json:"target_type"`
	Value           int    `json:"value"`
	ToUsername      string `json:"to_username"`
}

func decode(r *http.Request) (input, error) {
	var v input
	d := json.NewDecoder(io.LimitReader(r.Body, 65537))
	d.DisallowUnknownFields()
	if err := d.Decode(&v); err != nil {
		return v, invalid("Invalid JSON request")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return v, invalid("Expected one JSON object")
	}
	v.Username = strings.TrimSpace(v.Username)
	v.Name = strings.TrimSpace(v.Name)
	v.Title = strings.TrimSpace(v.Title)
	v.Content = strings.TrimSpace(v.Content)
	if len(v.Content) > 10000 || len(v.Title) > 200 || len(v.Description) > 2000 || len(v.Name) > 64 {
		return v, invalid("Input exceeds maximum length")
	}
	return v, nil
}

func pathID(r *http.Request) (int, error) {
	id, e := strconv.Atoi(r.PathValue("id"))
	if e != nil || id < 1 {
		return 0, invalid("Invalid ID")
	}
	return id, nil
}
