# GoReddit

A Reddit-style application with a Go API, persistent SQLite storage, and a React + TypeScript interface. Create an account, join communities, publish posts, reply to comments, vote, follow people, and send private messages. No simulated users or generated activity are required.

## Run locally

Requirements: **Go 1.24+**, a C compiler with **CGO enabled** (the SQLite driver bundles SQLite), and **Node.js 22+** with npm.

From the repository root:

```sh
go mod download
npm --prefix web ci
npm --prefix web run build
go run ./cmd/server
```

The server listens on port **8080** on the loopback interface and serves both the built interface and `/api`. SQLite is created at `data/reddit.db`. Register your first account in the browser, create a community, then publish a post. Other people can register separate accounts and interact with the same data.

For frontend development, keep the Go server running and start Vite in another terminal:

```sh
npm --prefix web run dev
```

Vite uses port **5173** and proxies `/api` to the Go server on port 8080. Build the frontend once before starting the Go server; Vite then provides live frontend updates. Restart Go after backend changes.

In the prepared cloud environment, activate the installed toolchain first:

```sh
export PATH=/workspace/toolchains/go/bin:$PATH
export GOCACHE=/workspace/goreddit-local/cache
export GOMODCACHE=/workspace/goreddit-local/modcache
cd /workspace/GoReddit
```

## Organization

```text
cmd/server/             Process configuration and graceful shutdown
internal/
  model/                Shared domain and response types
  storage/              SQLite schema and domain-specific persistence
  worker/               Bounded concurrent mutation queue
  httpapi/              Routing, sessions, validation, JSON responses
web/
  src/
    api.ts              Typed API requests and errors
    hooks/              Session and application data lifecycle
    components/         Account, community, post, comment and inbox views
    App.tsx             Navigation and page composition
  e2e/                  Two-user Chromium workflow
```

HTTP handlers authenticate a session and validate input; mutations go through five workers by default, with a queue capacity of 128. Workers return values and errors to the requesting handler. They never receive an HTTP response writer. Overload returns **503** instead of growing an unbounded queue. Cancelled queued work is skipped; a mutation already executing can still commit after a client disconnects.

SQLite uses foreign keys, a busy timeout, and WAL mode. Transactions and a process-local write mutex protect multi-step changes. Database access uses one connection, so persistence is serialized even while HTTP requests and worker jobs run concurrently. This is a single-process application; increasing worker count does not turn SQLite into a distributed database.

Votes replace a user's prior vote atomically and apply only the karma difference. Repeating an upvote does not inflate karma. A reply must reference a comment on the same post.

## Authentication

Registration stores bcrypt password hashes. Login issues a random session token in an HttpOnly, SameSite=Strict cookie; only a SHA-256 hash of that token is stored in SQLite. Sessions expire after seven days and survive server restarts. Logout invalidates the token. The API rejects cross-origin mutation requests and no longer accepts `X-User-ID` as authentication. Private messages are retrieved only for the authenticated recipient.

Use `-secure-cookies` when serving over HTTPS. The default supports local HTTP development. Deployment still needs application-specific operational work such as TLS termination, abuse controls and backups.

## Server configuration

```sh
go run ./cmd/server -addr 127.0.0.1:8080 -db data/reddit.db -web web/dist -workers 5
```

| Flag | Default | Purpose |
| --- | --- | --- |
| `-addr` | `127.0.0.1:8080` | Listen address |
| `-db` | `data/reddit.db` | SQLite database path |
| `-web` | `web/dist` | Built React assets |
| `-workers` | `5` | Concurrent mutation workers |
| `-secure-cookies` | `false` | Require HTTPS for session cookies |

The server drains HTTP requests before stopping workers and closing the database on SIGINT/SIGTERM. Data survives ordinary restarts. There is deliberately no public database-reset endpoint.

## API

All JSON routes use the `/api` prefix. Field names use `snake_case`; IDs are integers. Except for health, registration and login, data endpoints require the session cookie returned by registration or login. Collection responses may be `null` when empty; the frontend API client normalizes them to arrays.

| Method | Route | Purpose |
| --- | --- | --- |
| GET | `/api/health` | Process readiness |
| POST | `/api/register`, `/api/login` | Username/password credentials |
| POST | `/api/logout` | Invalidate session |
| GET | `/api/me` | Current user |
| GET | `/api/feed`, `/api/posts/top` | Joined-community feed / top 100 posts |
| POST | `/api/posts` | `title`, `content`, `subreddit_id` |
| GET | `/api/posts/{id}/comments` | Read comments and parent relationships |
| POST | `/api/comments` | `content`, `post_id`, optional `parent_comment_id` |
| POST | `/api/vote` | `target_id`, `target_type` (`post`/`comment`), `value` (`1`/`-1`) |
| GET | `/api/subreddits`, `/api/subreddits/joined` | Community discovery / memberships |
| POST | `/api/subreddits` | `name`, `description`; creator joins automatically |
| POST | `/api/subreddits/{id}/join`, `/api/subreddits/{id}/leave` | Membership changes |
| GET | `/api/users/{username}`, `/api/users/top`, `/api/users/top-subscribed` | Profiles and rankings |
| GET | `/api/subscriptions` | Followed users |
| POST | `/api/users/{id}/subscribe`, `/api/users/{id}/unsubscribe` | Follow / unfollow |
| GET | `/api/messages` | Current user's inbox |
| POST | `/api/messages` | `to_username`, `content` |

Example using a local cookie jar:

```sh
curl -c /tmp/goreddit.cookies -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"choose-a-long-password"}' \
  http://127.0.0.1:8080/api/register
curl -b /tmp/goreddit.cookies http://127.0.0.1:8080/api/feed
```

## Validation

```sh
go test -race ./...
go vet ./...
npm --prefix web test
npm --prefix web run build
```

The Go tests exercise real SQLite persistence, sessions, access control, communities, posts, nested comments, private messages, follows, worker backpressure, cancellation, and concurrent post/vote requests. React tests cover post interactions and clearing stale private data after logout.

For the complete browser workflow, install Playwright's browser once (or use an installed Chromium):

```sh
# Standard local setup:
cd web
npx playwright install chromium
cd ..
npm --prefix web run test:e2e

# Prepared cloud environment:
CHROMIUM_PATH=/usr/bin/chromium npm --prefix web run test:e2e
```

Build the UI first. Playwright starts and stops its own server on port **18080**, using the separate ignored database `data/e2e.db`. Tests create unique accounts and preserve existing data. The test covers two browser sessions, account creation/login, a community, a post, a comment, voting, following, private messaging, persistent login, and a mobile viewport. Screenshots are written to `web/test-results/`.

## Changes from the original demo

The old `main.go` and interactive simulator have been replaced by the server command and browser interface. Gin and Proto.Actor are replaced by Go's standard HTTP server and an explicit bounded worker pool; the same concurrent request-processing purpose remains, with simpler lifecycle and error handling. The SQLite driver is now `github.com/mattn/go-sqlite3`, which requires CGO and a C compiler.

API consumers must migrate to `/api`, cookie authentication and the documented JSON names. The message-send payload now uses a recipient username. The unsafe reset endpoint has been removed.

The default database is new and does not overwrite an existing `reddit_clone.db`. To inspect old content, back up that file and use `-db reddit_clone.db`. **Legacy accounts stored plaintext passwords and cannot log in through the new password verifier.** Use fresh accounts for development; migrating legacy account credentials requires an explicit password reset or offline hash migration. Existing content tables are preserved and the sessions table is added automatically. Back up existing databases before using them with changed application code.
