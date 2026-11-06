package storage

import (
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHashAndSessionPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persistent.db")
	db, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	id, e := db.RegisterUser("alice", "password-for-test")
	if e != nil {
		t.Fatal(e)
	}
	var hash string
	if e = db.db.QueryRow("SELECT password FROM users WHERE id=?", id).Scan(&hash); e != nil {
		t.Fatal(e)
	}
	if hash == "password-for-test" || bcrypt.CompareHashAndPassword([]byte(hash), []byte("password-for-test")) != nil {
		t.Fatal("password not hashed")
	}
	if e = db.MigrateSessions(); e != nil {
		t.Fatal(e)
	}
	if e = db.CreateSession("random-test-token", id, time.Now().Add(time.Hour)); e != nil {
		t.Fatal(e)
	}
	db.Close()
	db, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	u, e := db.SessionUser("random-test-token")
	if e != nil || u.ID != id {
		t.Fatalf("session not retained: %v", e)
	}
	if _, e = db.db.Exec("INSERT INTO users(username,password) VALUES('legacy','plaintext')"); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Authenticate("legacy", "plaintext"); e != ErrCredentials {
		t.Fatal("legacy plaintext password accepted")
	}
}
