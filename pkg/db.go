package pkg

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	_ "github.com/lib/pq"
)

// DB wraps sql.DB for easier usage
var DB *sql.DB

// Connect connects to the database using a DSN
func Connect(dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		return err
	}
	DB = db
	return nil
}

func checkMigrationsApplied() (bool, error) {
	// Проверяем существование ключевой таблицы
	var tableExists bool
	err := DB.QueryRow(`
        SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_schema = 'public' 
            AND table_name = 'works'
        )
    `).Scan(&tableExists)

	if err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}

	return tableExists, nil
}

// Migrate runs all .up.sql migrations in the given directory
func Migrate(migrationsDir string) error {
	applied, err := checkMigrationsApplied()
	if err != nil {
		return err
	}

	if applied {
		fmt.Println("Migrations already applied")
		return nil
	}
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrationFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" && filepath.Base(entry.Name())[len(entry.Name())-7:len(entry.Name())-4] == ".up" {
			migrationFiles = append(migrationFiles, filepath.Join(migrationsDir, entry.Name()))
		}
	}

	slices.Reverse(migrationFiles)

	for _, file := range migrationFiles {
		fmt.Printf("Running migration: %s\n", file)
		query, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		_, err = DB.Exec(string(query))
		if err != nil {
			return fmt.Errorf("error in migration %s: %w", file, err)
		}
	}
	return nil
}

// Exec executes a query without returning any rows.
func Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return DB.ExecContext(ctx, query, args...)
}

// Query executes a query that returns rows.
func Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return DB.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that is expected to return at most one row.
func QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	if DB == nil {
		return nil
	}
	return DB.QueryRowContext(ctx, query, args...)
}

// Begin starts a new transaction.
func Begin() (*sql.Tx, error) {
	if DB == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return DB.Begin()
}

// Close closes the database connection.
func Close() error {
	if DB == nil {
		return nil
	}
	return DB.Close()
}
