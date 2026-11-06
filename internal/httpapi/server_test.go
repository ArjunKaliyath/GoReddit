package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ArjunKaliyath/GoReddit/internal/model"
	"github.com/ArjunKaliyath/GoReddit/internal/storage"
	"github.com/ArjunKaliyath/GoReddit/internal/worker"
)

type fixture struct {
	handler http.Handler
	db      *storage.Store
}

func setup(t *testing.T) *fixture {
	t.Helper()
	db, e := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if e != nil {
		t.Fatal(e)
	}
	if e = db.MigrateSessions(); e != nil {
		t.Fatal(e)
	}
	pool := worker.New(5, 128)
	static := t.TempDir()
	if e = os.WriteFile(filepath.Join(static, "index.html"), []byte("GoReddit UI"), 0600); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { pool.Close(); db.Close() })
	return &fixture{New(db, pool, false, static), db}
}
func (f *fixture) request(method, path string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
	var data bytes.Buffer
	if body != nil {
		json.NewEncoder(&data).Encode(body)
	}
	req := httptest.NewRequest(method, path, &data)
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	res := httptest.NewRecorder()
	f.handler.ServeHTTP(res, req)
	return res
}
func check(t *testing.T, r *httptest.ResponseRecorder, status int) {
	t.Helper()
	if r.Code != status {
		t.Fatalf("status %d, want %d: %s", r.Code, status, r.Body.String())
	}
}
func register(t *testing.T, f *fixture, name string) (*http.Cookie, model.User) {
	t.Helper()
	r := f.request("POST", "/api/register", map[string]string{"username": name, "password": "a-real-test-password"}, nil)
	check(t, r, 201)
	var u model.User
	if e := json.Unmarshal(r.Body.Bytes(), &u); e != nil {
		t.Fatal(e)
	}
	return r.Result().Cookies()[0], u
}
func number(t *testing.T, r *httptest.ResponseRecorder, key string) int {
	t.Helper()
	var v map[string]int
	if e := json.Unmarshal(r.Body.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	return v[key]
}

func TestAuthentication(t *testing.T) {
	f := setup(t)
	cookie, u := register(t, f, "alice")
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/api" {
		t.Fatal("missing session cookie protections")
	}
	check(t, f.request("GET", "/api/me", nil, cookie), 200)
	forged := httptest.NewRequest("GET", "/api/feed", nil)
	forged.Header.Set("X-User-ID", fmt.Sprint(u.ID))
	r := httptest.NewRecorder()
	f.handler.ServeHTTP(r, forged)
	check(t, r, 401)
	check(t, f.request("POST", "/api/login", map[string]string{"username": "alice", "password": "wrong"}, nil), 401)
	check(t, f.request("POST", "/api/login", map[string]string{"username": "alice", "password": "a-real-test-password"}, nil), 200)
	check(t, f.request("POST", "/api/register", map[string]string{"username": "alice", "password": "a-real-test-password"}, nil), 409)
	check(t, f.request("POST", "/api/logout", map[string]string{}, cookie), 200)
	check(t, f.request("GET", "/api/me", nil, cookie), 401)
	if e := f.db.CreateSession("expired", u.ID, time.Now().Add(-time.Hour)); e != nil {
		t.Fatal(e)
	}
	check(t, f.request("GET", "/api/me", nil, &http.Cookie{Name: "session", Value: "expired"}), 401)
}
func TestCommunityContentAndMessagingWorkflow(t *testing.T) {
	f := setup(t)
	alice, au := register(t, f, "alice")
	bob, bu := register(t, f, "bobby")
	r := f.request("POST", "/api/subreddits", map[string]any{"name": "golang", "description": "Talk Go"}, alice)
	check(t, r, 201)
	community := number(t, r, "subreddit_id")
	check(t, f.request("POST", fmt.Sprintf("/api/subreddits/%d/join", community), map[string]any{}, bob), 200)
	r = f.request("POST", "/api/posts", map[string]any{"title": "Hello", "content": "An actual post", "subreddit_id": community}, alice)
	check(t, r, 201)
	post := number(t, r, "post_id")
	r = f.request("GET", "/api/feed", nil, bob)
	check(t, r, 200)
	var posts []model.Post
	if e := json.Unmarshal(r.Body.Bytes(), &posts); e != nil {
		t.Fatal(e)
	}
	if len(posts) != 1 || posts[0].Title != "Hello" || posts[0].AuthorID != au.ID {
		t.Fatalf("incorrect feed: %s", r.Body.String())
	}
	r = f.request("POST", "/api/comments", map[string]any{"post_id": post, "content": "A comment"}, bob)
	check(t, r, 201)
	comment := number(t, r, "comment_id")
	check(t, f.request("POST", "/api/comments", map[string]any{"post_id": post, "content": "A reply", "parent_comment_id": comment}, alice), 201)
	r = f.request("GET", fmt.Sprintf("/api/posts/%d/comments", post), nil, alice)
	check(t, r, 200)
	var comments []model.Comment
	if e := json.Unmarshal(r.Body.Bytes(), &comments); e != nil {
		t.Fatal(e)
	}
	if len(comments) != 2 || comments[1].ParentCommentID == nil || *comments[1].ParentCommentID != comment {
		t.Fatal("nested comments not persisted")
	}
	check(t, f.request("POST", fmt.Sprintf("/api/users/%d/subscribe", bu.ID), map[string]any{}, alice), 200)
	r = f.request("GET", "/api/subscriptions", nil, alice)
	check(t, r, 200)
	if !bytes.Contains(r.Body.Bytes(), []byte("bobby")) {
		t.Fatal("follow missing")
	}
	check(t, f.request("POST", "/api/messages", map[string]any{"to_username": "bobby", "content": "Hello privately"}, alice), 201)
	r = f.request("GET", "/api/messages", nil, bob)
	check(t, r, 200)
	if !bytes.Contains(r.Body.Bytes(), []byte("Hello privately")) {
		t.Fatal("message missing")
	}
	r = f.request("GET", "/api/messages", nil, alice)
	check(t, r, 200)
	if bytes.Contains(r.Body.Bytes(), []byte("Hello privately")) {
		t.Fatal("recipient inbox leaked")
	}
	check(t, f.request("POST", fmt.Sprintf("/api/users/%d/unsubscribe", bu.ID), map[string]any{}, alice), 200)
	check(t, f.request("POST", fmt.Sprintf("/api/subreddits/%d/leave", community), map[string]any{}, bob), 200)
	r = f.request("GET", "/api/feed", nil, bob)
	check(t, r, 200)
	if bytes.Contains(r.Body.Bytes(), []byte("An actual post")) {
		t.Fatal("left community still in feed")
	}
}
func TestConcurrentMutationsAndVoteReplacement(t *testing.T) {
	f := setup(t)
	author, au := register(t, f, "author")
	voter, _ := register(t, f, "voter")
	r := f.request("POST", "/api/subreddits", map[string]any{"name": "concurrency", "description": "Parallel requests"}, author)
	check(t, r, 201)
	community := number(t, r, "subreddit_id")
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := f.request("POST", "/api/posts", map[string]any{"title": fmt.Sprintf("Post %d", i), "content": "Concurrent", "subreddit_id": community}, author)
			if r.Code != 201 {
				t.Errorf("concurrent create: %d %s", r.Code, r.Body.String())
			}
		}(i)
	}
	wg.Wait()
	r = f.request("GET", "/api/posts/top", nil, author)
	check(t, r, 200)
	var posts []model.Post
	if e := json.Unmarshal(r.Body.Bytes(), &posts); e != nil {
		t.Fatal(e)
	}
	if len(posts) != 24 {
		t.Fatalf("lost writes: got %d posts", len(posts))
	}
	post := posts[0].ID
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := f.request("POST", "/api/vote", map[string]any{"target_id": post, "target_type": "post", "value": 1}, voter)
			if r.Code != 200 {
				t.Errorf("concurrent vote: %d %s", r.Code, r.Body.String())
			}
		}()
	}
	wg.Wait()
	u, e := f.db.GetUserByUsername(au.Username)
	if e != nil {
		t.Fatal(e)
	}
	if u.Karma != 1 {
		t.Fatalf("duplicate votes inflated karma: %d", u.Karma)
	}
	check(t, f.request("POST", "/api/vote", map[string]any{"target_id": post, "target_type": "post", "value": -1}, voter), 200)
	u, e = f.db.GetUserByUsername(au.Username)
	if e != nil {
		t.Fatal(e)
	}
	if u.Karma != -1 {
		t.Fatalf("vote replacement karma: %d", u.Karma)
	}
	posts, e = f.db.GetTopPosts(100)
	if e != nil {
		t.Fatal(e)
	}
	for _, p := range posts {
		if p.ID == post && (p.VoteCount.Upvotes != 0 || p.VoteCount.Downvotes != 1) {
			t.Fatalf("incorrect vote counts: %+v", p.VoteCount)
		}
	}
}
func TestValidationAndBrowserProtection(t *testing.T) {
	f := setup(t)
	cookie, _ := register(t, f, "alice")
	for _, body := range []any{map[string]any{"title": "", "content": "x", "subreddit_id": 1}, map[string]any{"title": "x", "content": "x", "subreddit_id": 1, "author_id": 999}} {
		check(t, f.request("POST", "/api/posts", body, cookie), 400)
	}
	check(t, f.request("POST", "/api/vote", map[string]any{"target_id": 1, "target_type": "post", "value": 12}, cookie), 400)
	check(t, f.request("POST", "/api/reset-database", map[string]any{}, cookie), 404)
	req := httptest.NewRequest("POST", "/api/logout", nil)
	req.AddCookie(cookie)
	req.Header.Set("Origin", "https://attacker.example")
	r := httptest.NewRecorder()
	f.handler.ServeHTTP(r, req)
	check(t, r, 403)
	r = f.request("GET", "/", nil, nil)
	check(t, r, 200)
	if !bytes.Contains(r.Body.Bytes(), []byte("GoReddit UI")) {
		t.Fatal("frontend not served")
	}
}
