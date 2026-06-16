package repository

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	// go-sqlcipher is a drop-in replacement for go-sqlite3 that adds AES-256 encryption.
	// It registers itself under the "sqlite3" driver name.
	// IMPORTANT: a database created without encryption cannot be opened with a key and vice-versa.
	// To encrypt an existing plaintext database, use sqlcipher-tools or:
	//   sqlite3 plain.db .dump | sqlcipher encrypted.db "PRAGMA key='...'; .read /dev/stdin"
	_ "github.com/mutecomm/go-sqlcipher/v4"
)

// InitDB opens the database and runs migrations.
// encryptionKey is passed as PRAGMA key; an empty key disables encryption.
func InitDB(dbPath, migrationsDir, encryptionKey string) (*sql.DB, error) {
	if dir := filepath.Dir(dbPath); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", sqliteDSN(dbPath, encryptionKey))
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("pragma foreign_keys: %w", err)
	}

	if err := runMigrations(db, migrationsDir); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return db, nil
}

// sqliteDSN builds the SQLite DSN with foreign-key enforcement and optional encryption.
func sqliteDSN(dbPath, encryptionKey string) string {
	separator := "?"
	if strings.Contains(dbPath, "?") {
		separator = "&"
	}
	dsn := dbPath + separator + "_foreign_keys=on"
	if encryptionKey != "" {
		dsn += "&_pragma_key=" + url.QueryEscape(encryptionKey) + "&_pragma_cipher_page_size=4096"
	}
	return dsn
}

// runMigrations runs pending database migrations
func runMigrations(db *sql.DB, dir string) error {
	// track migrations to prevent duplicate execution
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (filename TEXT PRIMARY KEY)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, f := range files {
		name := filepath.Base(f)

		applied, err := migrationApplied(db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", f, err)
		}
		if err := applyMigration(db, name, content); err != nil {
			return err
		}
	}
	return nil
}

// migrationApplied checks whether a migration has already run
func migrationApplied(db *sql.DB, name string) (bool, error) {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE filename = ?`, name).Scan(&count); err != nil {
		return false, fmt.Errorf("check migration %s: %w", name, err)
	}
	return count > 0, nil
}

// applyMigration executes and records one migration atomically
func applyMigration(db *sql.DB, name string, content []byte) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("exec migration %s: %w", name, err)
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations (filename) VALUES (?)`, name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}
