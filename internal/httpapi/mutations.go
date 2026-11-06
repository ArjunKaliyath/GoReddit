package httpapi

import (
	"net/http"
	"strings"
)

func (s *Server) mutate(r *http.Request, id int, action string) (any, error) {
	v, e := decode(r)
	if e != nil {
		return nil, e
	}
	switch action {
	case "subreddits":
		if v.Name == "" || strings.TrimSpace(v.Description) == "" {
			return nil, invalid("Name and description are required")
		}
		n, e := s.db.CreateSubreddit(v.Name, v.Description, id)
		return map[string]int{"subreddit_id": n}, e
	case "posts":
		if v.Title == "" || v.Content == "" || v.SubredditID < 1 {
			return nil, invalid("Title, content and community are required")
		}
		n, e := s.db.CreatePost(v.Title, v.Content, id, v.SubredditID)
		return map[string]int{"post_id": n}, e
	case "comments":
		if v.Content == "" || v.PostID < 1 {
			return nil, invalid("Content and post are required")
		}
		n, e := s.db.CreateComment(v.Content, id, v.PostID, v.ParentCommentID)
		return map[string]int{"comment_id": n}, e
	case "messages":
		if v.Content == "" || v.ToUsername == "" {
			return nil, invalid("Recipient and message are required")
		}
		to, e := s.db.UserID(v.ToUsername)
		if e != nil {
			return nil, apiError{404, "Recipient not found"}
		}
		n, e := s.db.SendDirectMessage(id, to, v.Content)
		return map[string]int{"message_id": n}, e
	case "vote":
		if v.TargetID < 1 || (v.TargetType != "post" && v.TargetType != "comment") || (v.Value != 1 && v.Value != -1) {
			return nil, invalid("Invalid vote")
		}
		e := s.db.Vote(id, v.TargetID, v.TargetType, v.Value)
		return map[string]bool{"ok": true}, e
	}
	return nil, invalid("Unknown operation")
}
