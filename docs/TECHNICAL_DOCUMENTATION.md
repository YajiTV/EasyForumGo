# Easy Technical Documentation

## 1. Purpose and Scope

Easy is a server-rendered forum written in Go for the B1 Forum project. This document describes the implementation delivered in this repository and maps its technical decisions to the subject requirements.

The mandatory scope includes:

- SQLite persistence
- registration, login, logout, cookies, and one active session per user
- posts associated with one or more categories and an optional image
- comments
- likes and dislikes on posts and comments
- category, created-post, and liked-post filters
- Docker delivery
- HTTP and technical error handling

Implemented optional scope includes Google and GitHub OAuth, advanced image validation, personal activity pages, customizable post libraries, notifications, public social profiles, follows, discovery, complete account settings, moderation foundations, HTTPS, and rate limiting.

## 2. Architecture
The internal Go module is named `EasyForumGo`.

The application follows a layered organization while remaining deliberately small and based on the Go standard library.

```text
HTTP request
    |
    v
cmd/server router
    |
    v
handler --------> validator / utils
    |
    v
repository
    |
    v
SQLite
```

### Entry point and routing

`cmd/server/main.go` loads the configuration, opens SQLite, creates the rate limiters, registers routes, and starts HTTP or HTTPS.

`cmd/server/router.go` uses the Go 1.22 `http.ServeMux` method-and-path patterns. Static files and uploaded images are served with `http.FileServer`.

### Handlers

`internal/handler` owns HTTP behavior:

- reads request paths and form values
- resolves the current user from the session
- validates user input
- checks resource ownership
- calls repositories
- renders HTML templates or redirects

Handlers do not concatenate user input into SQL queries.

### Middleware

`internal/middleware/auth.go` resolves valid sessions for protected handlers.

`internal/middleware/rate_limiter.go` provides an in-memory fixed-window limiter keyed by a valid session user ID when possible, then by client IP. Forwarded client-address headers are only read when `TRUST_PROXY=true`. The application configures:

- 200 global requests per minute
- 10 login attempts per 15 minutes
- 20 write actions per hour

### Models and repositories

`internal/model` contains the Go structures representing database records.

`internal/repository` contains SQLite queries and maps rows to models. Queries use `database/sql` placeholders. SQLite foreign keys are enabled when the connection opens.

### Templates and static assets

`web/templates/layout/base.html` provides the shared page layout. Feature templates define title, optional CSS, and content blocks.

`web/static` contains plain CSS and static images. No front-end framework is used.

## 3. Application Startup

Startup follows this sequence:

1. `config.Load` reads environment variables and applies defaults.
2. `repository.InitDB` creates the parent data directory and opens SQLite.
3. SQLite foreign keys are enabled with `PRAGMA foreign_keys = ON`.
4. Migration files are sorted, executed once, and recorded in `schema_migrations`.
5. Handlers and rate limiters are created.
6. Routes and static file handlers are registered.
7. The server starts:
   - HTTP only when TLS paths are empty
   - HTTP redirect plus HTTPS when both TLS paths are configured

## 4. Core Functional Flows

### Registration

1. The visitor submits email, username, password, and password confirmation.
2. Server-side validation checks email syntax and domain, username format, password strength, and confirmation.
3. Existing email and username values are rejected.
4. bcrypt hashes the password.
5. The account is inserted with a UUID.
6. The visitor is redirected to login.

### Login and unique session

1. The user submits email and password.
2. The stored bcrypt hash is compared with the password.
3. A transaction deletes any existing session for this user.
4. A new UUID session token and expiration date are inserted.
5. The token is sent in an `HttpOnly`, `SameSite=Lax` cookie.

Creating a new session replaces the previous session, satisfying the subject's single active session rule.

### Posts and categories

1. A connected user submits a title, content, category IDs, and optional image.
2. The server validates text lengths and verifies that selected category IDs exist.
3. The image size, signature, and complete JPEG/PNG/GIF decoding are validated when an image is present.
4. The post and its category relations are stored atomically.
5. The user is redirected to the post detail page.

Only the author may edit or delete a post.

### Comments

A connected user can add a non-empty comment to an existing post. Edit and deletion handlers load the comment and compare its `user_id` with the current user's ID before changing it.

### Votes

The database unique constraints allow one vote per user and resource.

- no current vote: insert the requested vote
- same vote selected again: remove it
- opposite vote selected: update it

The same behavior applies to posts and comments.

### Filters and profile

- category filters query posts through `post_categories`
- `/profile/my-posts` queries posts by the current user ID
- `/profile/liked-posts` joins posts with positive post votes
- `/profile/my-comments` queries comments by the current user ID
- `/profile/activity` consolidates created posts, liked and disliked posts, comments, and personal counters

### Personal libraries

Connected users can create, rename, delete, and empty named libraries from `/library`. A post detail page provides a library selector for saving the post.

Every library read and write query includes the current user ID. This prevents users from viewing or changing another user's libraries. Database constraints prevent duplicate library names per user and duplicate posts inside one library.

### Account settings

`/settings` centralizes account information, email changes, password changes, session logout, and permanent account deletion.

Local accounts must confirm their current password before changing their email, changing their password, or deleting the account. Password changes delete the active session and require a new login. OAuth-only accounts display provider-specific information and do not expose unusable email or password forms. Account deletion requires an explicit confirmation and relies on foreign key cascades to remove related forum data.

### OAuth authentication

Google and GitHub authorization-code flows use provider-specific anti-CSRF state cookies. GitHub requests `user:email` and selects a verified primary e-mail, falling back to another verified address. OAuth identities are stored separately from users so one account can be linked to both providers. A verified e-mail links to an existing account; otherwise a short-lived opaque completion token lets the user choose a public username.

### Social profiles and following feed

Every user has a public profile at `/user/{username}` with a biography, registration date, counters, and paginated posts. Connected users can follow or unfollow another user through POST routes. Followers and following lists are paginated and can be hidden by their owner.

The home feed accepts `feed=following` to display posts from followed users. Discovery combines follower popularity with categories from posts liked by the current user. The public search covers posts and profiles, with result-type, post-category, relevance, recency, and popularity controls. Discovery suggestions exclude the current user and already-followed users.

Following and publishing create idempotent social notifications.

## 5. Database Design

SQLite is used through `database/sql` and `github.com/mattn/go-sqlite3`. Application records use UUID strings as primary keys.

### Tables

| Table | Main relations and constraints |
| --- | --- |
| `users` | unique email and username |
| `sessions` | unique user and token; cascades on user deletion |
| `posts` | belongs to a user; cascades on user deletion |
| `categories` | unique category name |
| `post_categories` | composite primary key; links posts and categories |
| `comments` | belongs to a post and user |
| `post_likes` | unique post/user pair |
| `comment_likes` | unique comment/user pair |
| `libraries` | belongs to a user; unique name per user |
| `library_posts` | composite primary key linking libraries and posts |
| `schema_migrations` | records executed migration filenames |
| `user_follows` | unique directed following relation with self-follow prevention |
| `notifications` | forum, social, and moderation events |
| `reports` | user-submitted moderation reports |
| `moderation_actions` | moderation action history |
| `user_restrictions` | active mute and ban records |
| `oauth_identities` | unique Google and GitHub identities linked to users |
| `oauth_pending_flows` | short-lived server-side OAuth profile completion data |

Foreign key cascades remove dependent records when their parent is deleted. The visual entity-relationship diagram is stored in `docs/ERD.svg`.

### Migration strategy

Migration files in `migrations/` are ordered by filename. Each migration is executed once and then recorded in `schema_migrations`. Seed categories use `INSERT OR IGNORE`, making startup repeatable.

## 6. Security

### Authentication and passwords

- Passwords are hashed with bcrypt.
- Plain-text passwords are never stored.
- Session tokens are generated with UUIDs.
- Sessions expire and only one session per user is allowed.
- Cookies are `HttpOnly` and `SameSite=Lax`.

### Authorization

Guest users can read public content. Creation, comments, votes, profile pages, and modification actions require a valid user session.

Handlers compare the current user ID with the resource owner before editing or deleting posts and comments.

### Input and SQL safety

- All form input is validated server-side.
- SQL statements use placeholders.
- User input is rendered through `html/template`, which escapes HTML by default.
- Image validation checks detected content type rather than trusting the extension.
- Request body and file sizes are limited.

### HTTPS and rate limiting

Docker development mode serves HTTPS with an automatically generated self-signed certificate and redirects HTTP to HTTPS. A trusted reverse proxy terminates HTTPS with a real certificate in production. The in-memory rate limiter protects general traffic, authentication, and all registered write actions.

### Known security limits

- CSRF tokens are not implemented.
- The rate limiter is local to one process and resets on restart.
- `TRUST_PROXY=true` is safe only when direct access is blocked and a trusted proxy replaces forwarded client-address headers.
- Production depends on the reverse proxy for HTTPS termination.
- Database encryption is not implemented.
- Complete subject-level moderation is not implemented.

## 7. Error Handling

`internal/handler/errors.go` centralizes custom error pages for:

- `400 Bad Request`
- `401 Unauthorized`
- `403 Forbidden`
- `404 Not Found`
- `500 Internal Server Error`
- `502 Bad Gateway`
- `503 Service Unavailable`

Error pages can preserve the current user's navigation state. Some feature handlers still use direct `http.Error` fallbacks for technical failures; these return the correct HTTP status without exposing internal database details.

## 8. Docker and Persistence

The Dockerfile uses a multi-stage build:

1. Alpine Go builder with GCC and musl development packages for CGO SQLite
2. Alpine runtime containing the application, templates, migrations, and SQLite tools

Compose binds HTTP `8080` and HTTPS `8443` to localhost and creates the stable `web` network. A reverse proxy can join this network when needed.

Named volumes persist:

- `/app/data` for `forum.db`
- `/app/uploads` for uploaded images

Container recreation does not remove named volumes. `docker compose down --volumes` deliberately deletes them.

## 9. Configuration

Configuration is read from environment variables in `config/config.go`. Invalid or missing numeric values fall back to safe defaults.

Session duration is configurable and defaults to 24 hours. Image uploads use the `MAX_UPLOAD_MB` limit, with the subject's 20 MB limit as the default.

`APP_ENV` accepts `dev` or `prod`. Production mode requires an HTTPS `APP_PUBLIC_URL` and `TRUST_PROXY=true`. `APP_BASE_PATH` mounts the application below an optional path prefix, `APP_PUBLIC_URL` defines the canonical OAuth callback base and enables secure cookies for HTTPS URLs, and `TRUST_PROXY` allows trusted forwarded client IP headers.

Development and production use the same `.env`, `docker/Dockerfile`, and `docker/docker-compose.yml`. Docker development uses the generated self-signed certificate; production can disable internal TLS and let the reverse proxy own TLS.

## 10. Verification Strategy

The repository must pass:

```bash
gofmt -w .
go test ./...
go vet ./...
docker compose -f docker/docker-compose.yml config
docker compose -f docker/docker-compose.yml up --build
```

`go test ./...` compiles every package. Manual validation must cover:

- guest read-only access
- registration, login, session replacement, and logout
- post and comment creation, editing, deletion, and ownership checks
- image validation
- post and comment votes
- mandatory filters
- personal activity and library ownership
- public profiles, social privacy, following feed, discovery, and notifications
- account settings, password confirmation, session invalidation, and deletion cascade
- custom error pages
- database and upload persistence across container recreation

No automated tests are included. `go test ./...` compiles all packages.

## 11. Subject Requirement Mapping

| Subject requirement | Implementation |
| --- | --- |
| Go without backend framework | Go standard library and `http.ServeMux` |
| SQLite | migrations and repositories using `database/sql` |
| Registration and login | auth handlers, bcrypt, unique account fields |
| Cookies and expiring unique session | `sessions` table and session cookie |
| Posts, categories, and images | post handlers, many-to-many categories, upload helper |
| Comments | comment handlers and repository |
| Likes and dislikes | post and comment vote toggle |
| Mandatory filters | category, current user's posts, liked posts |
| Personal activity and libraries | profile activity handler, library handler, and ownership-scoped repository |
| Account settings | settings handler with password confirmation and session invalidation |
| Guest read-only access | public feed and detail routes |
| Docker | multi-stage Dockerfile and Compose |
| HTTP errors | custom error renderer and templates |
| Clear README | root `README.md` |
