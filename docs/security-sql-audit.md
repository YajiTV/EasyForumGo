# SQL prepared statements audit

Date: 2026-06-04

## Scope

This audit covers application SQL calls in:

- `internal/repository`
- `internal/handler`
- `internal/middleware`
- `cmd/server`

Search command used:

```sh
rg -n "(Query|QueryRow|Exec|Prepare|Sprintf|fmt\\.Sprintf|\\+ .*SELECT|SELECT .*\\+|INSERT .*\\+|UPDATE .*\\+|DELETE .*\\+)" internal pkg cmd
```

## Result

No user-controlled value is interpolated into SQL strings.

All application queries that include request, form, route, cookie, or session data use `?` placeholders and pass values as separate arguments to `database/sql`.

Examples checked:

- Authentication lookups and session writes in `internal/handler/auth.go`.
- Session middleware lookup and cleanup in `internal/middleware/auth.go`.
- User, post, comment, category, session, and like repositories in `internal/repository`.
- Profile liked-post lookup in `internal/repository/like.go`.

## Migration exception

`internal/repository/db.go` executes migration file contents with `db.Exec(string(content))`.

This is expected and acceptable because migrations are static project files, not user input. Migration filenames are discovered from the configured migration directory and tracked through `schema_migrations` with a prepared `INSERT`.

## Follow-up rule

Any future SQL query that includes external data must keep using placeholders:

```go
db.QueryRow("SELECT id FROM users WHERE email = ?", email)
```

Do not build SQL with string concatenation or `fmt.Sprintf` when values come from users, route parameters, cookies, uploaded files, or sessions.
