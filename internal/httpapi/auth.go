package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/ArjunKaliyath/GoReddit/internal/model"
	"github.com/ArjunKaliyath/GoReddit/internal/storage"
	"net/http"
	"strings"
	"time"
)

func (s *Server) session(w http.ResponseWriter, u *model.User) error {
	bytes := make([]byte, 32)
	if _, e := rand.Read(bytes); e != nil {
		return e
	}
	token := hex.EncodeToString(bytes)
	expires := time.Now().Add(7 * 24 * time.Hour)
	if e := s.db.CreateSession(token, u.ID, expires); e != nil {
		return e
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: token, Path: "/api", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteStrictMode, Expires: expires, MaxAge: 604800})
	return nil
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	v, e := decode(r)
	if e != nil {
		fail(w, e)
		return
	}
	if len(v.Username) < 3 || len(v.Username) > 32 || strings.IndexFunc(v.Username, func(c rune) bool {
		return !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_')
	}) >= 0 || len(v.Password) < 8 || len(v.Password) > 72 {
		fail(w, invalid("Use a 3–32 character username (letters, digits, underscores) and an 8–72 byte password"))
		return
	}
	value, e := s.pool.Do(r.Context(), func() (any, error) {
		id, e := s.db.RegisterUser(v.Username, v.Password)
		if errors.Is(e, storage.ErrUsernameTaken) {
			return nil, apiError{409, "Username unavailable"}
		}
		if e != nil {
			return nil, e
		}
		return &model.User{ID: id, Username: v.Username}, nil
	})
	if e != nil {
		fail(w, e)
		return
	}
	u := value.(*model.User)
	if e = s.session(w, u); e != nil {
		fail(w, e)
		return
	}
	send(w, 201, u)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	v, e := decode(r)
	if e != nil {
		fail(w, e)
		return
	}
	if len(v.Password) > 72 {
		fail(w, apiError{401, "Invalid username or password"})
		return
	}
	value, e := s.pool.Do(r.Context(), func() (any, error) { return s.db.Authenticate(v.Username, v.Password) })
	if e != nil {
		if errors.Is(e, storage.ErrCredentials) {
			e = apiError{401, e.Error()}
		}
		fail(w, e)
		return
	}
	u := value.(*model.User)
	if e = s.session(w, u); e != nil {
		fail(w, e)
		return
	}
	send(w, 200, u)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, e := r.Cookie("session"); e == nil {
		if e = s.db.DeleteSession(c.Value); e != nil {
			fail(w, e)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Path: "/api", MaxAge: -1, HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteStrictMode})
	send(w, 200, map[string]bool{"ok": true})
}
