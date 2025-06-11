package postgres

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations runs all migrations in the specified directory
func RunMigrations(dbURL, migrationsDir string) error {
	log.Printf("Running migrations from: %s", migrationsDir)

	// Check if directory exists
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory does not exist: %s", migrationsDir)
	}

	// Convert to absolute path for file:// URL
	absPath, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Step 3: Create a `migrate` instance, which ties together the source and database
	m, err := migrate.New(
		fmt.Sprintf("file://%s", filepath.ToSlash(absPath)),
		dbURL,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	// Ensure resources are cleaned up after migrations run
	defer m.Close()

	// Step 4: Execute the migrations (apply all "Up" migrations)
	if err := m.Up(); err != nil {
		// If there are no new migrations, ErrNoChange is returned
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("No migrations to apply")
			return nil
		}
		// For any other error, wrap and return
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Println("Migrations completed successfully")
	return nil
}
