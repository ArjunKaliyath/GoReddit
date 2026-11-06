package httpapi

import (
	"bytes"
	"github.com/ArjunKaliyath/GoReddit/internal/storage"
	"github.com/ArjunKaliyath/GoReddit/internal/worker"
	"io"
	"net/http"
	"net/url"
)

type Server struct {
	db     *storage.Store
	pool   *worker.Pool
	secure bool
}

type operation func(*http.Request, int) (any, error)

func New(db *storage.Store, pool *worker.Pool, secure bool, staticDir string) http.Handler {
	s := &Server{db, pool, secure}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) { send(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /api/register", s.register)
	mux.HandleFunc("POST /api/login", s.login)
	mux.HandleFunc("POST /api/logout", s.logout)
	s.route(mux, "GET /api/me", 200, false, func(r *http.Request, id int) (any, error) {
		cookie, _ := r.Cookie("session")
		return db.SessionUser(cookie.Value)
	})
	s.route(mux, "GET /api/feed", 200, false, func(r *http.Request, id int) (any, error) { return db.GetFeed(id) })
	s.route(mux, "GET /api/posts/top", 200, false, func(r *http.Request, id int) (any, error) { return db.GetTopPosts(100) })
	s.route(mux, "GET /api/subreddits", 200, false, func(r *http.Request, id int) (any, error) { return db.GetAllSubreddits() })
	s.route(mux, "GET /api/subreddits/joined", 200, false, func(r *http.Request, id int) (any, error) { return db.GetUserJoinedSubreddits(id) })
	s.route(mux, "GET /api/messages", 200, false, func(r *http.Request, id int) (any, error) { return db.GetDirectMessages(id) })
	s.route(mux, "GET /api/users/top", 200, false, func(r *http.Request, id int) (any, error) { return db.GetTopUsers(20) })
	s.route(mux, "GET /api/users/top-subscribed", 200, false, func(r *http.Request, id int) (any, error) { return db.GetTopSubscribedUsers(20) })
	s.route(mux, "GET /api/subscriptions", 200, false, func(r *http.Request, id int) (any, error) { return db.GetUserSubscriptions(id) })
	s.route(mux, "GET /api/users/{username}", 200, false, func(r *http.Request, id int) (any, error) {
		u, e := db.GetUserByUsername(r.PathValue("username"))
		if e != nil {
			return nil, apiError{404, "User not found"}
		}
		return u, nil
	})
	s.route(mux, "GET /api/posts/{id}/comments", 200, false, func(r *http.Request, id int) (any, error) {
		pid, e := pathID(r)
		if e != nil {
			return nil, e
		}
		if !db.PostExists(pid) {
			return nil, apiError{404, "Post not found"}
		}
		return db.Comments(pid)
	})
	for _, action := range []string{"subreddits", "posts", "comments", "messages", "vote"} {
		action := action
		status := 201
		if action == "vote" {
			status = 200
		}
		s.route(mux, "POST /api/"+action, status, true, func(r *http.Request, id int) (any, error) { return s.mutate(r, id, action) })
	}
	for _, action := range []string{"join", "leave"} {
		action := action
		s.route(mux, "POST /api/subreddits/{id}/"+action, 200, true, func(r *http.Request, id int) (any, error) {
			target, e := pathID(r)
			if e != nil {
				return nil, e
			}
			if action == "join" {
				e = db.JoinSubreddit(id, target)
			} else {
				e = db.LeaveSubreddit(id, target)
			}
			return map[string]bool{"ok": true}, e
		})
	}
	for _, action := range []string{"subscribe", "unsubscribe"} {
		action := action
		s.route(mux, "POST /api/users/{id}/"+action, 200, true, func(r *http.Request, id int) (any, error) {
			target, e := pathID(r)
			if e != nil {
				return nil, e
			}
			if target == id {
				return nil, invalid("Cannot subscribe to yourself")
			}
			if action == "subscribe" {
				e = db.SubscribeToUser(id, target)
			} else {
				e = db.UnsubscribeFromUser(id, target)
			}
			return map[string]bool{"ok": true}, e
		})
	}
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		send(w, 404, map[string]string{"error": "Endpoint not found"})
	})
	mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		if r.Method != "GET" && r.Method != "HEAD" {
			if origin := r.Header.Get("Origin"); origin != "" {
				u, e := url.Parse(origin)
				if e != nil || u.Host != r.Host {
					send(w, 403, map[string]string{"error": "Cross-origin request rejected"})
					return
				}
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, 65536)
		mux.ServeHTTP(w, r)
	})
}

func (s *Server) route(m *http.ServeMux, pattern string, status int, mutation bool, op operation) {
	m.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		cookie, e := r.Cookie("session")
		if e != nil {
			send(w, 401, map[string]string{"error": "Please sign in"})
			return
		}
		u, e := s.db.SessionUser(cookie.Value)
		if e != nil {
			send(w, 401, map[string]string{"error": "Session expired; please sign in"})
			return
		}

		if mutation {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				fail(w, invalid("Request too large"))
				return
			}
			r = r.Clone(r.Context())
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
		var value any
		if mutation {
			value, e = s.pool.Do(r.Context(), func() (any, error) { return op(r, u.ID) })
		} else {
			value, e = op(r, u.ID)
		}
		if e != nil {
			fail(w, e)
			return
		}
		if value == nil {
			value = []any{}
		}
		send(w, status, value)
	})
}
