package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mohamedfawas/qubool-kallyanam/tools/migration/postgres"
)

func main() {
	// Parse flags
	var dbURL, migrationsDir string
	flag.StringVar(&dbURL, "db-url", "", "Database URL (postgres://user:pass@host:port/dbname?sslmode=disable)")
	flag.StringVar(&migrationsDir, "dir", "", "Migrations directory")
	flag.Parse()

	// Use environment variables if flags not provided
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
		if dbURL == "" {
			host := getEnvOrDefault("DB_HOST", "localhost")
			port := getEnvOrDefault("DB_PORT", "5432")
			user := getEnvOrDefault("DB_USER", "postgres")
			password := getEnvOrDefault("DB_PASSWORD", "postgres")
			dbname := getEnvOrDefault("DB_NAME", "postgres")
			sslmode := getEnvOrDefault("DB_SSLMODE", "disable")

			dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
				user, password, host, port, dbname, sslmode)
		}
	}

	if migrationsDir == "" {
		migrationsDir = os.Getenv("MIGRATIONS_DIR")
		if migrationsDir == "" {
			fmt.Println("Error: Migrations directory is required")
			flag.Usage()
			os.Exit(1)
		}
	}

	// Run migrations
	if err := postgres.RunMigrations(dbURL, migrationsDir); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
