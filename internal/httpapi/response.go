package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/ArjunKaliyath/GoReddit/internal/worker"
	"github.com/mattn/go-sqlite3"
	"log"
	"net/http"
)

type apiError struct {
	status  int
	message string
}

func (e apiError) Error() string { return e.message }

func invalid(message string) error { return apiError{400, message} }

func send(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}

func fail(w http.ResponseWriter, err error) {
	status, message := 500, "The operation could not be completed"
	var e apiError
	var constraint sqlite3.Error
	if errors.As(err, &e) {
		status, message = e.status, e.message
	} else if errors.Is(err, sql.ErrNoRows) {
		status, message = 404, "Requested resource not found"
	} else if errors.As(err, &constraint) && constraint.Code == sqlite3.ErrConstraint {
		status, message = 400, "Invalid reference or conflicting data"
	} else if errors.Is(err, worker.ErrClosed) {
		status, message = 503, "Server is shutting down"
	} else if errors.Is(err, worker.ErrBusy) {
		status, message = 503, err.Error()
	} else {
		log.Printf("API error: %v", err)
	}
	send(w, status, map[string]string{"error": message})
}
