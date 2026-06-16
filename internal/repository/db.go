package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
		if encryptionKey != "" && isNotDatabaseError(err) {
			if rekeyed, rekeyErr := rekeyLegacySQLCipherDB(dbPath, encryptionKey); rekeyErr != nil {
				return nil, fmt.Errorf("rekey legacy SQLCipher database after open failure %q: %w", err, rekeyErr)
			} else if rekeyed {
				db, err = sql.Open("sqlite3", sqliteDSN(dbPath, encryptionKey))
				if err != nil {
					return nil, fmt.Errorf("open rekeyed db: %w", err)
				}
				if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
					db.Close()
					return nil, fmt.Errorf("pragma foreign_keys after rekey: %w", err)
				}
				if err := runMigrations(db, migrationsDir); err != nil {
					db.Close()
					return nil, fmt.Errorf("migrations: %w", err)
				}
				return db, nil
			}

			if migrationErr := migratePlaintextDBToSQLCipher(dbPath, encryptionKey); migrationErr != nil {
				return nil, fmt.Errorf("migrate plaintext database to SQLCipher after open failure %q: %w", err, migrationErr)
			}
			db, err = sql.Open("sqlite3", sqliteDSN(dbPath, encryptionKey))
			if err != nil {
				return nil, fmt.Errorf("open encrypted db after migration: %w", err)
			}
			if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
				db.Close()
				return nil, fmt.Errorf("pragma foreign_keys after migration: %w", err)
			}
		} else {
			return nil, fmt.Errorf("pragma foreign_keys: %w", err)
		}
	}

	if err := runMigrations(db, migrationsDir); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return db, nil
}

func rekeyLegacySQLCipherDB(dbPath, encryptionKey string) (bool, error) {
	for _, candidate := range legacyEncryptionKeyCandidates(encryptionKey) {
		db, err := sql.Open("sqlite3", sqliteDSN(dbPath, candidate))
		if err != nil {
			return false, fmt.Errorf("open legacy encrypted db: %w", err)
		}
		if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
			db.Close()
			if isNotDatabaseError(err) {
				continue
			}
			return false, fmt.Errorf("pragma foreign_keys with legacy key: %w", err)
		}
		if _, err := db.Exec("PRAGMA rekey = " + sqlStringLiteral(encryptionKey)); err != nil {
			db.Close()
			return false, fmt.Errorf("rekey db: %w", err)
		}
		if err := db.Close(); err != nil {
			return false, fmt.Errorf("close rekeyed db: %w", err)
		}
		log.Printf("Rekeyed SQLCipher database from legacy deployment key")
		return true, nil
	}
	return false, nil
}

func legacyEncryptionKeyCandidates(encryptionKey string) []string {
	legacy := composeUnbracedDollarExpansion(encryptionKey)
	if legacy == "" || legacy == encryptionKey {
		return nil
	}
	return []string{legacy}
}

func composeUnbracedDollarExpansion(value string) string {
	var out strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '$' {
			out.WriteByte(value[i])
			continue
		}
		if i+1 >= len(value) || !isEnvNameStart(value[i+1]) {
			out.WriteByte(value[i])
			continue
		}
		i++
		for i+1 < len(value) && isEnvNameChar(value[i+1]) {
			i++
		}
	}
	return out.String()
}

func isEnvNameStart(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isEnvNameChar(c byte) bool {
	return isEnvNameStart(c) || (c >= '0' && c <= '9')
}

func sqlStringLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func migratePlaintextDBToSQLCipher(dbPath, encryptionKey string) error {
	info, err := os.Stat(dbPath)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return nil
	}

	dir := filepath.Dir(dbPath)
	base := filepath.Base(dbPath)
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	encryptedPath := filepath.Join(dir, base+".encrypted-"+timestamp+".tmp")
	backupPath := filepath.Join(dir, base+".plaintext-backup-"+timestamp)

	plainDB, err := sql.Open("sqlite3", sqliteDSN(dbPath, ""))
	if err != nil {
		return fmt.Errorf("open plaintext db for encryption migration: %w", err)
	}
	defer plainDB.Close()

	if _, err := plainDB.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("disable foreign_keys for encryption migration: %w", err)
	}
	if _, err := plainDB.Exec("ATTACH DATABASE ? AS encrypted KEY ?", encryptedPath, encryptionKey); err != nil {
		os.Remove(encryptedPath)
		return fmt.Errorf("attach encrypted db for migration: %w", err)
	}
	if _, err := plainDB.Exec("SELECT sqlcipher_export('encrypted')"); err != nil {
		plainDB.Exec("DETACH DATABASE encrypted")
		os.Remove(encryptedPath)
		return fmt.Errorf("export encrypted db: %w", err)
	}
	if _, err := plainDB.Exec("DETACH DATABASE encrypted"); err != nil {
		os.Remove(encryptedPath)
		return fmt.Errorf("detach encrypted db: %w", err)
	}
	if err := plainDB.Close(); err != nil {
		os.Remove(encryptedPath)
		return fmt.Errorf("close plaintext db before swap: %w", err)
	}

	if err := os.Rename(dbPath, backupPath); err != nil {
		os.Remove(encryptedPath)
		return fmt.Errorf("backup plaintext db: %w", err)
	}
	if err := os.Rename(encryptedPath, dbPath); err != nil {
		if restoreErr := os.Rename(backupPath, dbPath); restoreErr != nil {
			return fmt.Errorf("install encrypted db: %w; restore plaintext backup: %v", err, restoreErr)
		}
		return fmt.Errorf("install encrypted db: %w", err)
	}

	log.Printf("Migrated plaintext SQLite database to SQLCipher; backup kept at %s", backupPath)
	return nil
}

func isNotDatabaseError(err error) bool {
	for err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "file is not a database") {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
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
