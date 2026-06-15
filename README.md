# Easy

Easy is a server-rendered community forum written in Go for the B1 Forum project. The product name is Easy, the repository slug is `EasyForumGo`, and the internal Go module is `EasyForumGo`.

## Team

- Axel: front-end and back-end development
- Mathys: front-end and back-end development
- Baptiste: front-end and back-end development
- Nine: front-end development

## Subject Coverage

### Mandatory Scope

- SQLite persistence with versioned migrations
- Registration, login, logout, expiring cookie sessions, and one active session per user
- Public post and comment reading for guests
- Posts with one or more categories and an optional image
- Comments
- Post and comment likes and dislikes
- Author-only post and comment editing and deletion
- Category, created-post, and liked-post filters
- Custom HTTP error pages and server-side validation
- Autonomous Docker delivery with persistent database and upload volumes
- Clear installation and launch instructions

### Additional Features

- Personal activity page and customizable post libraries
- Complete account settings and profile picture upload
- Public profiles, follows, following feed, discovery, and search
- Notifications for forum, social, and moderation events
- CSS-only light, system, and dark themes
- Google and GitHub OAuth authentication with verified-email account linking
- Moderation data model, reports, roles, restrictions, and actions
- Optional base-path deployment behind a trusted reverse proxy

## Bonus Status

| Subject bonus | Status |
| --- | --- |
| Google and GitHub OAuth | Implemented |
| Advanced image upload | Implemented: JPEG, PNG, GIF, decoded-image validation, configurable limit |
| Moderation | Partial: roles, reports, actions, mute, and ban exist; pre-publication approval and complete administration are not implemented |
| HTTPS and rate limiting | Implemented: Docker development HTTPS, HTTP redirect, reverse-proxy support, and in-memory limits |
| Notifications and personal activity | Implemented |
| Database encryption | Not implemented |

## Technology

- Go `1.22.2`
- Go standard library: `net/http`, `html/template`, `database/sql`, image decoders, and related packages
- SQLite through `github.com/mattn/go-sqlite3`
- bcrypt through `golang.org/x/crypto`
- UUIDs through `github.com/google/uuid`
- Server-rendered HTML and plain CSS without a front-end framework
- Docker and Docker Compose

No automated test suite is included. `go test ./...` is still used to compile every package.

## Run With Docker

### Prerequisites

- Docker
- Docker Compose

Start the complete stack without any preparatory command:

```bash
docker compose -f docker/docker-compose.yml up --build
```

Open:

```text
http://localhost:8080   -> redirects to HTTPS
https://localhost:8443  -> application with a self-signed development certificate
```

The browser warning for the self-signed localhost certificate is expected.

Stop the application without deleting persisted data:

```bash
docker compose -f docker/docker-compose.yml down
```

Delete the database and uploaded files deliberately:

```bash
docker compose -f docker/docker-compose.yml down --volumes
```

Compose automatically creates the stable `web` network. Database and uploaded images persist in the `db_data` and `uploads` named volumes.

## Run Locally

Local development requires Go `1.22.2` or newer and a C compiler because the SQLite driver uses CGO.

```bash
go mod download
go run ./cmd/server
```

Without TLS environment variables, the local process serves HTTP on `http://localhost:8080`.

## Configuration

Configuration is loaded from environment variables and an optional local `.env` file.

| Variable | Default | Description |
| --- | --- | --- |
| `APP_ENV` | `dev` | Runtime mode: `dev` or `prod` |
| `PORT` | `8080` | HTTP port |
| `HTTPS_PORT` | `8443` | HTTPS port when internal TLS is enabled |
| `TLS_CERT_FILE` | empty locally | TLS certificate path |
| `TLS_KEY_FILE` | empty locally | TLS private key path |
| `HOST_BIND` | `127.0.0.1` | Docker host bind address |
| `HOST_PORT` | `8080` | Docker HTTP host port |
| `HOST_HTTPS_PORT` | `8443` | Docker HTTPS host port |
| `DB_PATH` | `./data/forum.db` | SQLite database path |
| `MIGRATIONS_DIR` | `./migrations` | Migration directory |
| `STATIC_DIR` | `web/static` | Static asset directory |
| `TEMPLATES_DIR` | `web/templates` | HTML template directory |
| `UPLOAD_DIR` | `./uploads` | Uploaded image directory |
| `MAX_UPLOAD_MB` | `20` | Maximum image size in megabytes |
| `SESSION_DURATION_H` | `24` | Session duration in hours |
| `GOOGLE_OAUTH_ID` | empty | Google OAuth client ID |
| `GOOGLE_OAUTH_KEY` | empty | Google OAuth client secret |
| `GITHUB_OAUTH_ID` | empty | GitHub OAuth app client ID |
| `GITHUB_OAUTH_KEY` | empty | GitHub OAuth app client secret |
| `APP_BASE_PATH` | empty | Optional public path prefix, for example `/easy` |
| `APP_PUBLIC_URL` | empty locally; `https://localhost:8443` in Docker | Canonical public application URL |
| `TRUST_PROXY` | `false` | Trust forwarded client-address headers |

See [`.env.example`](.env.example).

## Production Behind a Reverse Proxy

Docker development mode provides a self-signed certificate so HTTPS can be evaluated without manual setup. In production, a trusted reverse proxy should terminate HTTPS with a real certificate.

Set:

```env
APP_ENV=prod
APP_BASE_PATH=/easy
APP_PUBLIC_URL=https://example.com/easy
TRUST_PROXY=true
TLS_CERT_FILE=
TLS_KEY_FILE=
```

The reverse proxy can join the automatically created `web` network and forward the unchanged public path to `forum:8080`. Setting both `TLS_CERT_FILE` and `TLS_KEY_FILE` explicitly to empty disables internal TLS so the proxy can own HTTPS. Production mode requires an HTTPS `APP_PUBLIC_URL` and `TRUST_PROXY=true`.

The deployment script automatically applies
`docker/docker-compose.prod.yml`. This production override treats the existing
shared `web` reverse-proxy network as external and disables internal TLS. The
base Compose file remains autonomous for local development.

OAuth callback URLs must match:

```text
https://example.com/easy/auth/google/callback
https://example.com/easy/auth/github/callback
```

For Docker development, create a GitHub OAuth App with homepage URL
`https://localhost:8443` and callback URL
`https://localhost:8443/auth/github/callback`, then set `GITHUB_OAUTH_ID` and
`GITHUB_OAUTH_KEY`. GitHub accounts must expose at least one verified e-mail to
the application; private primary e-mails are retrieved through the requested
`user:email` scope.

Run one application instance because Easy uses SQLite and in-memory rate limiting.

## Project Structure

```text
cmd/server/          application startup and route registration
config/              environment configuration
internal/handler/    HTTP handlers and rendering
internal/middleware/ authentication, restrictions, proxy, base path, and rate limiting
internal/model/      application data structures
internal/repository/ SQLite access and migrations
migrations/          versioned SQL schema and seed data
pkg/utils/           password, UUID, and upload helpers
pkg/validator/       server-side input validation
web/templates/       server-rendered HTML templates
web/static/          CSS and static images
docker/              Dockerfile and Compose configuration
docs/                technical and persistence documentation
```

## Database

Easy applies 22 migration files on startup and records them in `schema_migrations`.

| Table | Purpose |
| --- | --- |
| `users` | Accounts, roles, profile, and social preferences |
| `oauth_identities`, `oauth_pending_flows` | Multiple OAuth providers per account and short-lived completion flows |
| `sessions` | Unique expiring user sessions |
| `posts`, `categories`, `post_categories` | Forum posts and categories |
| `comments` | Post comments |
| `post_likes`, `comment_likes` | Likes and dislikes |
| `libraries`, `library_posts` | Personal saved-post collections |
| `notifications` | Forum, social, and moderation notifications |
| `reports`, `moderation_actions`, `user_restrictions` | Moderation reports, history, mute, and ban |
| `user_follows` | Social following relationships |
| `schema_migrations` | Applied migration filenames |

SQLite foreign keys and cascading deletion are enabled. The existing visual diagram is available at [`docs/ERD.svg`](docs/ERD.svg), but it does not yet include every later social and moderation table.

## Main Routes

| Method | Route | Access | Purpose |
| --- | --- | --- | --- |
| `GET` | `/`, `/?category={id}`, `/?feed=following` | Public or connected | Post feeds |
| `GET` | `/post/{id}` | Public | Post detail and comments |
| `GET`, `POST` | `/register`, `/login` | Public | Authentication |
| `POST` | `/logout` | Connected | End the session |
| `GET`, `POST` | `/post/new` | Connected | Create a post |
| `GET`, `POST` | `/post/{id}/edit`, `/post/{id}/delete` | Author | Manage a post |
| `POST` | `/post/{id}/comment` | Connected | Comment |
| `GET`, `POST` | `/comment/{id}/edit`, `/comment/{id}/delete` | Author | Manage a comment |
| `POST` | `/post/{id}/like`, `/post/{id}/dislike` | Connected | Vote on a post |
| `POST` | `/comment/{id}/like`, `/comment/{id}/dislike` | Connected | Vote on a comment |
| `GET` | `/profile/*`, `/settings`, `/library/*`, `/notifications` | Connected | Personal features |
| `GET` | `/user/{username}`, `/discover`, `/search` | Public or connected | Social features |
| `GET`, `POST` | `/auth/google/*`, `/auth/github/*` | Public | Google and GitHub OAuth |
| `POST` | `/{post|comment|user}/{id}/report` | Connected | Report content or a user |
| `GET`, `POST` | `/moderation/*`, `/admin/*` | Moderator or admin | Moderation actions |

## Security

- bcrypt password hashes
- UUID session tokens, expiration, and one active session per user
- `HttpOnly`, `SameSite=Lax`, and HTTPS `Secure` cookies
- server-side input validation and SQL placeholders
- author ownership checks
- SQLite foreign keys and cascades
- decoded-image validation and request-size limits
- global, authentication, and write-action rate limiting, keyed by valid session user then client IP
- role checks, mute enforcement, and login-time ban enforcement
- forwarded client headers trusted only when `TRUST_PROXY=true`

## Verification

```bash
gofmt -w .
go test ./...
go vet ./...
go build ./cmd/server
docker compose -f docker/docker-compose.yml config
docker compose -f docker/docker-compose.yml up --build -d
curl -I http://localhost:8080/
curl -k -I https://localhost:8443/
```

`go test ./...` compiles every package; no automated tests are included. Important workflows and persistence must be verified manually.

## Known Limits

- Moderation does not implement pre-publication approval or complete category/user administration.
- The moderation dashboard template is currently missing.
- CSRF tokens are not implemented for ordinary state-changing forms.
- The in-memory rate limiter resets on restart and is intended for one application instance.
- `TRUST_PROXY=true` must only be used when direct access is blocked and a trusted proxy replaces forwarded client-address headers.
- Database encryption is not implemented.
- The ERD does not include every table added after the mandatory MVP.
- Some direct technical-error fallbacks remain while error handling is progressively centralized.

## Demo Accounts

No demo or administrator account is seeded. Create an account from `/register`. Testing administration currently requires assigning the `admin` role directly in SQLite.
