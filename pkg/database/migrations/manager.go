package migrations

import (
	"context"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v4/pgxpool"
)

// Migration represents a single database migration
type Migration struct {
	Version     int64
	Description string
	UpSQL       string
	DownSQL     string
}

// Manager handles database migrations
type Manager struct {
	db         *pgxpool.Pool
	migrations []Migration
}

// NewManager creates a new migration manager
func NewManager(db *pgxpool.Pool) *Manager {
	return &Manager{
		db:         db,
		migrations: []Migration{},
	}
}

// Register adds a migration to the manager
func (m *Manager) Register(version int64, description, upSQL, downSQL string) {
	m.migrations = append(m.migrations, Migration{
		Version:     version,
		Description: description,
		UpSQL:       upSQL,
		DownSQL:     downSQL,
	})

	// Sort migrations by version
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})
}

// EnsureMigrationTable creates the migration table if it doesn't exist
func (m *Manager) EnsureMigrationTable(ctx context.Context) error {
	_, err := m.db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT (now() AT TIME ZONE 'Asia/Kolkata')
		)
	`)
	return err
}

// GetAppliedMigrations returns a list of applied migration versions
func (m *Manager) GetAppliedMigrations(ctx context.Context) ([]int64, error) {
	rows, err := m.db.Query(ctx, "SELECT version FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []int64
	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}

	return versions, rows.Err()
}

// MigrateUp applies all pending migrations
func (m *Manager) MigrateUp(ctx context.Context) error {
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return err
	}

	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	// Create a map for quick lookup
	appliedMap := make(map[int64]bool)
	for _, v := range applied {
		appliedMap[v] = true
	}

	// Run pending migrations in a transaction
	for _, migration := range m.migrations {
		if appliedMap[migration.Version] {
			continue
		}

		tx, err := m.db.Begin(ctx)
		if err != nil {
			return err
		}

		// Apply migration
		if _, err := tx.Exec(ctx, migration.UpSQL); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migration %d failed: %w", migration.Version, err)
		}

		// Record migration
		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version, description) VALUES ($1, $2)",
			migration.Version, migration.Description); err != nil {
			tx.Rollback(ctx)
			return err
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		fmt.Printf("Applied migration %d: %s\n", migration.Version, migration.Description)
	}

	return nil
}

// MigrateDown rolls back the latest migration
func (m *Manager) MigrateDown(ctx context.Context) error {
	if err := m.EnsureMigrationTable(ctx); err != nil {
		return err
	}

	applied, err := m.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		return fmt.Errorf("no migrations to roll back")
	}

	// Get the latest applied migration
	latestVersion := applied[len(applied)-1]

	// Find the migration
	var migration Migration
	found := false
	for _, m := range m.migrations {
		if m.Version == latestVersion {
			migration = m
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("migration %d not found", latestVersion)
	}

	// Run in a transaction
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return err
	}

	// Apply down migration
	if _, err := tx.Exec(ctx, migration.DownSQL); err != nil {
		tx.Rollback(ctx)
		return err
	}

	// Remove record
	if _, err := tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", latestVersion); err != nil {
		tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	fmt.Printf("Rolled back migration %d: %s\n", latestVersion, migration.Description)
	return nil
}
