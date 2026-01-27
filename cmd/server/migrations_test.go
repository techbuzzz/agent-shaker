package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"

	_ "github.com/lib/pq"
	"github.com/techbuzzz/agent-shaker/internal/database"
)

// TestMigrationConcurrentSafety verifies that concurrent migration attempts
// don't cause conflicts or duplicate executions
func TestMigrationConcurrentSafety(t *testing.T) {
	// Skip if no test database is available
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping integration test")
	}

	// Clean up any existing test schema
	cleanupDB(t, dbURL)

	// Create temporary test migration file
	tmpDir := t.TempDir()
	migrationFile := filepath.Join(tmpDir, "001_test.sql")
	migrationSQL := `CREATE TABLE IF NOT EXISTS test_table (id SERIAL PRIMARY KEY, name TEXT);`
	if err := os.WriteFile(migrationFile, []byte(migrationSQL), 0644); err != nil {
		t.Fatalf("Failed to create test migration: %v", err)
	}

	// Change to temp directory for migration discovery
	oldWd, _ := os.Getwd()
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.Mkdir(migrationsDir, 0755); err != nil {
		t.Fatalf("Failed to create migrations dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(migrationsDir, "001_test.sql"), []byte(migrationSQL), 0644); err != nil {
		t.Fatalf("Failed to create migration in migrations dir: %v", err)
	}
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Test concurrent migration attempts
	const numConcurrent = 5
	var wg sync.WaitGroup
	errors := make(chan error, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine creates its own DB connection
			db, err := database.NewDB(dbURL)
			if err != nil {
				errors <- err
				return
			}
			defer db.Close()

			// Attempt to run migrations
			if err := runMigrations(db); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Migration error: %v", err)
	}

	// Verify migration was applied exactly once
	db, err := database.NewDB(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to verify: %v", err)
	}
	defer db.Close()

	// Check schema_migrations table
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = '001_test.sql'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected migration to be recorded exactly once, got %d times", count)
	}

	// Verify the test table was created
	var exists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'test_table'
		)
	`).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}
	if !exists {
		t.Error("Expected test_table to exist after migration")
	}
}

func cleanupDB(t *testing.T, dbURL string) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to open DB for cleanup: %v", err)
	}
	defer db.Close()

	// Drop test tables
	_, _ = db.Exec("DROP TABLE IF EXISTS test_table CASCADE")
	_, _ = db.Exec("DROP TABLE IF EXISTS schema_migrations CASCADE")
}
