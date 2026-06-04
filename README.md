# ForumJS

ForumJS is a small forum built in Go with server-rendered HTML, SQLite, sessions, categories, comments, votes, profile pages, and image uploads.

## Team

- Baptiste

## Features

- Public post feed and post detail pages.
- Account registration, login, logout, and one active session per user.
- Post creation, edit, delete, categories, and optional JPEG/PNG/GIF image upload.
- Comment creation, edit, and delete.
- Like/dislike votes on posts and comments.
- Category filters.
- Profile pages for personal posts, liked posts, and comments.
- Custom 400, 401, 403, 404, and 500 error pages.

## Stack

- Go standard library with `net/http`.
- SQLite with `database/sql`.
- HTML templates and CSS.
- Docker and Docker Compose.

## Prerequisites

- Go 1.22 or newer for local development.
- Docker and Docker Compose for container launch.
- OpenSSL if local TLS certificates need to be generated.

## Run With Docker

Generate local certificates once:

```bash
sh scripts/gen_certs.sh
```

Start the forum:

```bash
docker compose -f docker/docker-compose.yml up --build
```

Open:

```text
https://localhost:8443
```

The HTTP endpoint `http://localhost:8080` redirects to HTTPS when the Docker TLS configuration is used.

Stop the forum:

```bash
docker compose -f docker/docker-compose.yml down
```

The SQLite database and uploads persist in Docker volumes.

## Run Locally

```bash
go run ./cmd/server
```

Default local URL:

```text
http://localhost:8080
```

## Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port |
| `HTTPS_PORT` | `8443` | HTTPS port when TLS is enabled |
| `TLS_CERT_FILE` | empty | TLS certificate path |
| `TLS_KEY_FILE` | empty | TLS key path |
| `DB_PATH` | `./data/forum.db` | SQLite database path |
| `MIGRATIONS_DIR` | `./migrations` | SQL migrations directory |
| `STATIC_DIR` | `web/static` | Static assets directory |
| `TEMPLATES_DIR` | `web/templates` | HTML templates directory |
| `UPLOAD_DIR` | `./uploads` | User uploads directory |
| `MAX_UPLOAD_MB` | `20` | Configured upload limit |
| `SESSION_DURATION_H` | `24` | Session duration in hours |

## Project Structure

```text
cmd/server       HTTP server and routes
config           Environment configuration
internal/handler HTTP handlers
internal/model   Data models
internal/repository SQLite repositories and migrations runner
pkg/utils        Hashing, UUID, upload helpers
pkg/validator    Server-side validation rules
web/templates    HTML templates
web/static       CSS assets
migrations       SQLite schema and seed data
docker           Dockerfile and Compose file
docs             Project diagrams and validation reports
```

## Main Routes

- `GET /`
- `GET /login`, `POST /login`
- `GET /register`, `POST /register`
- `POST /logout`
- `GET /post/new`, `POST /post/new`
- `GET /post/{id}`
- `GET /post/{id}/edit`, `POST /post/{id}/edit`
- `POST /post/{id}/delete`
- `POST /post/{id}/comment`
- `GET /comment/{id}/edit`, `POST /comment/{id}/edit`
- `POST /comment/{id}/delete`
- `POST /post/{id}/like`, `POST /post/{id}/dislike`
- `POST /comment/{id}/like`, `POST /comment/{id}/dislike`
- `GET /posts/category/{id}`
- `GET /profile`
- `GET /profile/my-posts`
- `GET /profile/liked-posts`
- `GET /profile/my-comments`
- `GET /profile/edit`, `POST /profile/edit`

## Database Summary

The database is SQLite. Migrations create:

- users
- categories
- posts
- post_categories
- sessions
- post_likes
- comments
- comment_likes

Seed data creates the default categories.

## Tests

Compile all packages:

```bash
go test ./...
```

Manual validation reports:

- `docs/security-sql-audit.md`
- `docs/manual-test-report.md`
- `docs/pre-submission-checklist.md`

## Demo Accounts

No demo user is seeded. Create an account from `/register`.

## Known Limits

- There are no automated `*_test.go` unit tests yet; `go test ./...` currently validates package compilation.
- CSRF tokens are not implemented; forms use POST actions and cookies use `SameSite=Lax`.
- Vote removal is done by clicking the same like/dislike action again.

## Bonus

No bonus feature is required for the MVP phase.
