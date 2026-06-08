# Docker Persistence Verification

## Result

Database and upload persistence were verified successfully on June 8, 2026.

The verification used a dedicated Compose project and confirmed that:

- the Docker image builds from the repository without local certificate files
- the application starts and responds with `200 OK` over HTTP
- migrations create the SQLite database in the `db_data` named volume
- a database record remains available after container recreation
- a file in the uploads volume remains available after container recreation
- the application still responds with `200 OK` after recreation

## Environment

```text
Docker Engine: 29.5.2
Docker Compose: 5.1.4
Compose project: easy-persistence-check
```

## Procedure

Validate and start the application:

```bash
docker compose -p easy-persistence-check -f docker/docker-compose.yml config
docker compose -p easy-persistence-check -f docker/docker-compose.yml up -d --build
curl http://localhost:8080/
```

Insert a database marker and an upload marker:

```bash
docker compose -p easy-persistence-check -f docker/docker-compose.yml exec -T forum \
  sqlite3 /app/data/forum.db \
  "CREATE TABLE IF NOT EXISTS persistence_check (value TEXT PRIMARY KEY);
   INSERT OR REPLACE INTO persistence_check(value) VALUES ('database-persists');"

docker compose -p easy-persistence-check -f docker/docker-compose.yml exec -T forum \
  sh -c "printf 'upload-persists' > /app/uploads/persistence-check.txt"
```

Recreate the container without deleting its named volumes:

```bash
docker compose -p easy-persistence-check -f docker/docker-compose.yml up -d --force-recreate
```

Read both markers after recreation:

```bash
docker compose -p easy-persistence-check -f docker/docker-compose.yml exec -T forum \
  sqlite3 /app/data/forum.db "SELECT value FROM persistence_check;"

docker compose -p easy-persistence-check -f docker/docker-compose.yml exec -T forum \
  cat /app/uploads/persistence-check.txt
```

Observed output:

```text
HTTP status before recreation: 200
Database marker after recreation: database-persists
Upload marker after recreation: upload-persists
HTTP status after recreation: 200
```

## Volume Behavior

The Compose file declares two named volumes:

```text
db_data  -> /app/data
uploads  -> /app/uploads
```

These volumes survive container replacement and a normal `docker compose down`. They are deleted only when explicitly requested with `docker compose down --volumes`.

The dedicated verification project and its test volumes were removed after the successful check.
