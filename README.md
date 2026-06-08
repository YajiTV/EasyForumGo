# ForumJS

ForumJS is a server-rendered community forum built in Go. It provides the complete mandatory feature set from the B1 Forum project: authentication, one active session per user, posts with categories and images, comments, votes, personal filters, SQLite persistence, HTTP error handling, and Docker delivery.

## Team

- Axel: front-end and back-end development
- Mathys: front-end and back-end development
- Baptiste: front-end and back-end development
- Nine: front-end development

## Features

### Mandatory features

- Public post feed, category filters, post details, comments, likes, and dislikes
- Registration with a unique email and username
- Login, logout, expiring cookie sessions, and one active session per user
- Post creation with one or more categories and an optional image
- Post and comment editing or deletion restricted to their authors
- Like, dislike, vote change, and vote removal on posts and comments
- Personal pages for posts created and posts liked by the connected user
- Custom `400`, `401`, `403`, `404`, and `500` error pages
- Server-side input validation and prepared SQL statements
- SQLite data and uploaded image persistence through Docker volumes

### Additional features

- Personal comments page and editable user profile
- Profile picture upload
- Password strength feedback
- JPEG, PNG, and GIF validation with a 20 MB limit
- HTTPS with a self-signed certificate
- Global, login, and write-action rate limiting
- Informational, help, legal, privacy, terms, cookies, and contact pages

## Technology

- Go `1.22.2` and the standard library: `net/http`, `html/template`, `database/sql`
- SQLite through `github.com/mattn/go-sqlite3`
- bcrypt through `golang.org/x/crypto`
- UUIDs through `github.com/google/uuid`
- Server-rendered HTML and plain CSS without a front-end framework
- Docker and Docker Compose

## Run With Docker

### Prerequisites

- Docker
- Docker Compose

Clone and start the application:

```bash
git clone https://github.com/YajiTV/ForumJS.git
cd ForumJS
docker compose -f docker/docker-compose.yml up --build
```

Open:

```text
https://localhost:8443
```

The self-signed development certificate causes a browser warning. The HTTP endpoint at `http://localhost:8080` redirects to HTTPS.

Stop the application:

```bash
docker compose -f docker/docker-compose.yml down
```

The database and uploaded images remain available after a normal stop or container recreation. To deliberately remove all persisted Docker data:

```bash
docker compose -f docker/docker-compose.yml down --volumes
```

## Run Locally

Local development requires Go `1.22.2` or newer and a C compiler because the SQLite driver uses CGO.

```bash
go mod download
go run ./cmd/server
```

Open `http://localhost:8080`.

To enable HTTPS locally, generate a development certificate and load the example configuration:

```bash
sh scripts/gen_certs.sh
set -a
. ./.env.example
set +a
go run ./cmd/server
```

Then open `https://localhost:8443`.

## Configuration

Configuration is read from environment variables. The application uses defaults when variables are absent.

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP server port |
| `HTTPS_PORT` | `8443` | HTTPS server port |
| `TLS_CERT_FILE` | empty | TLS certificate path; TLS is enabled when both TLS paths are set |
| `TLS_KEY_FILE` | empty | TLS private key path |
| `DB_PATH` | `./data/forum.db` | SQLite database file |
| `MIGRATIONS_DIR` | `./migrations` | SQL migration directory |
| `STATIC_DIR` | `web/static` | Static asset directory |
| `TEMPLATES_DIR` | `web/templates` | HTML template directory |
| `UPLOAD_DIR` | `./uploads` | Uploaded image directory |
| `SESSION_DURATION_H` | `24` | Session duration in hours |

See [`.env.example`](.env.example) for a complete example.

## Project Structure

```text
cmd/server/          application entry point and route registration
config/              environment configuration
internal/handler/    HTTP request handling and template rendering
internal/middleware/ authentication and rate limiting
internal/model/      application data structures
internal/repository/ SQLite access and prepared SQL queries
migrations/          versioned schema and category seed
pkg/utils/           password, UUID, and upload helpers
pkg/validator/       server-side input validation
web/templates/       server-rendered HTML templates
web/static/          CSS and static images
uploads/             user-uploaded images
docker/              Dockerfile and Compose configuration
docs/                ERD and technical documentation
```

## Database

ForumJS uses SQLite with foreign keys enabled. Migrations run automatically at startup and are recorded in `schema_migrations`.

| Table | Purpose |
| --- | --- |
| `users` | Accounts, bcrypt password hashes, and profile pictures |
| `sessions` | Unique expiring user sessions |
| `posts` | Forum posts and optional image paths |
| `categories` | Available post categories |
| `post_categories` | Many-to-many relation between posts and categories |
| `comments` | Comments associated with posts and users |
| `post_likes` | One like or dislike per user and post |
| `comment_likes` | One like or dislike per user and comment |

The entity-relationship diagram is available at [`docs/ERD.svg`](docs/ERD.svg).

## Main Routes

| Method | Route | Access | Purpose |
| --- | --- | --- | --- |
| `GET` | `/` | Public | Post feed |
| `GET` | `/posts/category/{id}` | Public | Posts filtered by category |
| `GET` | `/post/{id}` | Public | Post details and comments |
| `GET`, `POST` | `/register` | Public | Registration form and account creation |
| `GET`, `POST` | `/login` | Public | Login form and authentication |
| `POST` | `/logout` | Connected | End the current session |
| `GET`, `POST` | `/post/new` | Connected | Create a post |
| `GET`, `POST` | `/post/{id}/edit` | Author | Edit a post |
| `GET`, `POST` | `/post/{id}/delete` | Author | Delete a post |
| `POST` | `/post/{id}/comment` | Connected | Create a comment |
| `GET`, `POST` | `/comment/{id}/edit` | Author | Edit a comment |
| `GET`, `POST` | `/comment/{id}/delete` | Author | Delete a comment |
| `POST` | `/post/{id}/like`, `/post/{id}/dislike` | Connected | Vote on a post |
| `POST` | `/comment/{id}/like`, `/comment/{id}/dislike` | Connected | Vote on a comment |
| `GET` | `/profile/my-posts` | Connected | Current user's posts |
| `GET` | `/profile/liked-posts` | Connected | Posts liked by the current user |
| `GET` | `/profile/my-comments` | Connected | Current user's comments |
| `GET`, `POST` | `/profile/edit` | Connected | Edit the current user's profile |

## Security

- Passwords are hashed with bcrypt and never stored in plain text.
- SQL inputs use placeholders.
- Registration, login, posts, comments, categories, and uploads are validated server-side.
- Cookies are `HttpOnly`, use `SameSite=Lax`, and have an expiration date.
- Creating a new session removes the user's previous session.
- Author ownership is checked before post or comment modification.
- SQLite foreign keys enforce relation integrity and cascading deletion.
- Image content is checked with `http.DetectContentType`.
- Rate limiting protects global traffic, login attempts, and write actions.

## Verification

Run the Go checks:

```bash
gofmt -w $(find . -name '*.go' -type f)
go test ./...
go vet ./...
```

Validate the Compose configuration:

```bash
docker compose -f docker/docker-compose.yml config
```

`go test ./...` validates compilation and includes a repository test covering fresh migrations, foreign key activation, and cascade deletion. The main HTTP workflows must also be checked manually: guest access, authentication, posts, comments, votes, personal filters, uploads, errors, and Docker persistence.

## Demo Accounts

No demo account is seeded. Create an account from `/register`.

## Known Limits

- The self-signed HTTPS certificate is intended for development and evaluation only.
- CSRF tokens are not implemented; state-changing actions use `POST` and session cookies use `SameSite=Lax`.
- The in-memory rate limiter resets when the application restarts and is designed for a single application instance.

## Implemented Bonuses

- Advanced image upload validation
- HTTPS and rate limiting
- Personal activity pages

OAuth, moderation roles, notifications, and database encryption are not implemented.
