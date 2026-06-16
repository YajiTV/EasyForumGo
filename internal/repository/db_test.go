//go:build cgo

package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitDBMigratesPlaintextDatabaseToSQLCipher(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "forum.db")
	migrationsDir := filepath.Join(dir, "migrations")
	if err := os.Mkdir(migrationsDir, 0755); err != nil {
		t.Fatalf("create migrations dir: %v", err)
	}

	plainDB, err := InitDB(dbPath, migrationsDir, "")
	if err != nil {
		t.Fatalf("create plaintext db: %v", err)
	}
	if _, err := plainDB.Exec(`CREATE TABLE sample (id INTEGER PRIMARY KEY, value TEXT)`); err != nil {
		t.Fatalf("create sample table: %v", err)
	}
	if _, err := plainDB.Exec(`INSERT INTO sample (value) VALUES (?)`, "kept"); err != nil {
		t.Fatalf("insert sample row: %v", err)
	}
	if err := plainDB.Close(); err != nil {
		t.Fatalf("close plaintext db: %v", err)
	}

	encryptedDB, err := InitDB(dbPath, migrationsDir, "test-secret")
	if err != nil {
		t.Fatalf("migrate plaintext db: %v", err)
	}
	defer encryptedDB.Close()

	var value string
	if err := encryptedDB.QueryRow(`SELECT value FROM sample WHERE id = 1`).Scan(&value); err != nil {
		t.Fatalf("read migrated data: %v", err)
	}
	if value != "kept" {
		t.Fatalf("migrated value = %q, want kept", value)
	}

	backups, err := filepath.Glob(filepath.Join(dir, "forum.db.plaintext-backup-*"))
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("backup count = %d, want 1", len(backups))
	}
}
