package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ArjunKaliyath/GoReddit/internal/httpapi"
	"github.com/ArjunKaliyath/GoReddit/internal/storage"
	"github.com/ArjunKaliyath/GoReddit/internal/worker"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	database := flag.String("db", "data/reddit.db", "SQLite database file")
	assets := flag.String("web", "web/dist", "Built React assets")
	workers := flag.Int("workers", 5, "Concurrent mutation workers")
	secure := flag.Bool("secure-cookies", false, "Require HTTPS for session cookies")
	flag.Parse()
	if *workers < 1 {
		return errors.New("workers must be positive")
	}
	if _, err := os.Stat(filepath.Join(*assets, "index.html")); err != nil {
		return errors.New("build the UI first: npm --prefix web ci && npm --prefix web run build")
	}
	if err := os.MkdirAll(filepath.Dir(*database), 0700); err != nil {
		return err
	}
	db, err := storage.Open(*database)
	if err != nil {
		return err
	}
	defer db.Close()
	if err = db.MigrateSessions(); err != nil {
		return err
	}
	pool := worker.New(*workers, 128)
	defer pool.Close()
	srv := &http.Server{Addr: *addr, Handler: httpapi.New(db, pool, *secure, *assets), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("GoReddit listening on %s", *addr)
	result := make(chan error, 1)
	go func() { result <- srv.ListenAndServe() }()
	select {
	case err := <-result:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdown); err != nil {
			log.Printf("Shutdown: %v", err)
			srv.Close()
		}
		<-result
	}
	return nil
}
