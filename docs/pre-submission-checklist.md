# Pre-submission checklist

Date: 2026-06-04

Branch: `chore/security-validation-audit`

## Automated checks

- `go test ./...`: passed.
- `git diff --check`: passed.
- Docker image build with `docker compose -f docker/docker-compose.yml up --build -d`: passed.

## Docker runtime checks

- `http://localhost:8080/`: returns `301` to `https://localhost:8443/`.
- `https://localhost:8443/`: returns `200`.
- Container status: `forum_app` starts and stays up.
- Docker logs show HTTP redirect server on `8080` and HTTPS server on `8443`.
- Account creation in Docker: passed.
- Login in Docker: passed.
- Post creation in Docker: passed.
- Container restart: passed.
- Database persistence after restart: passed, created post remained visible on the home page.

## MVP checklist

- Visitor can read the home feed: passed.
- Visitor can open a post detail page: passed in manual test report.
- User can register: passed.
- User can log in: passed.
- User has a single active session: passed.
- User can log out: covered by route and session cleanup, not re-run in Docker pass.
- Authenticated user can create a post with a category: passed.
- Authenticated user can upload JPEG, PNG, and GIF images: passed in manual test report.
- Authenticated user can comment: passed.
- Author can edit/delete own posts: passed in manual test report.
- Author can edit/delete own comments: passed in manual test report.
- Non-author edit/delete is forbidden: passed.
- Authenticated user can like/dislike posts and comments: passed.
- Vote counters stay consistent: passed in manual test report.
- Category filter works: passed after router fix.
- Profile personal pages work: passed after router fix.
- Custom error pages exist for 400, 401, 403, 404, and 500: passed by file check and manual HTTP checks.
- User inputs are validated server-side: passed by validation changes and manual checks.
- SQL queries use placeholders for user-controlled values: passed in SQL audit.
- Passwords are hashed: passed in manual test report.
- README includes install and launch instructions: passed.
- SQLite database persists after Docker restart: passed.

## Notes

- Docker TLS certificates were generated with `sh scripts/gen_certs.sh`.
- Generated certificates are ignored by Git.
- Docker volumes were kept after validation so persistence was not destroyed by the checklist cleanup.
