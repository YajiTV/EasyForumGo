# Manual test report

Date: 2026-06-04

Branch: `chore/security-validation-audit`

## Environment

Local server:

```sh
PORT=18080 DB_PATH=/tmp/forumjs-manual/data/forum.db UPLOAD_DIR=/tmp/forumjs-manual/uploads go run ./cmd/server
```

Database and uploads were isolated under `/tmp/forumjs-manual`.

## Passed flows

### Guest

- `GET /` returns `200`.
- `GET /post/new` redirects to `/login` with `303`.
- Guest vote on a post redirects to `/login`.

### Registration and login

- Invalid registration data returns the registration page with `200`.
- Valid registration redirects to `/login`.
- Password stored in SQLite differs from the raw password.
- Valid login redirects to `/`.
- Re-login creates one active session and invalidates the previous cookie.
- Old cookie is refused and redirects to `/login`.

### Posts

- Post title shorter than 3 characters is rejected.
- Valid post without image is created and redirects to `/post/{id}`.
- Non-image upload is rejected.
- JPEG upload is accepted.
- PNG upload is accepted.
- GIF upload is accepted.
- Upload larger than 20 Mo is rejected.
- Author can edit a post.
- Non-author edit returns `403`.
- Non-author delete returns `403`.

### Comments

- Empty comment returns `400`.
- Valid comment is created and redirects to the post detail.
- Author can edit a comment.
- Non-author comment delete returns `403`.
- Author can delete a comment.

### Votes

- Post like creates one like.
- Post dislike switches the vote from like to dislike.
- Repeating the same post vote removes the vote.
- Comment like and dislike use the same switch/remove logic.
- Vote counters in the database match each step.

## Bugs identified

### Server router mismatch

`cmd/server/main.go` manually registers routes instead of using `setupRouter` from `cmd/server/router.go`.

Observed missing routes on the actual server:

- `GET /profile` returns `404`.
- `GET /profile/liked-posts` returns `404`.
- `GET /posts/category/general` returns `404`.
- `GET /about` returns `404`.

Expected behavior: these routes should be available because they are declared in `setupRouter`.

### Profile personal filters incomplete

The profile template links to:

- `/profile/my-posts`
- `/profile/my-comments`

These routes are not yet connected in `setupRouter`, and the profile handler currently renders liked posts for `/profile`.

Expected behavior: authenticated users can access their profile overview, their posts, their liked posts, and their comments.

## Not run in Docker during this pass

Docker validation is reserved for the pre-submission checklist after the bug fixes, so the final check can validate the corrected code path.
