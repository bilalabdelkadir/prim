package cli

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/bilalabdelkadir/prim/internal/db"
	"github.com/bilalabdelkadir/prim/internal/migrator"
	"github.com/bilalabdelkadir/prim/internal/parser"
	"github.com/bilalabdelkadir/prim/internal/schema"
)

// RunMigrate reads the schema file, diffs against current state, generates SQL,
// and writes a migration file. If databaseURL is provided it also ensures the
// database exists and applies the migration.
func RunMigrate(schemaPath, migrationsDir, databaseURL string) error {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("reading schema: %w", err)
	}

	next, err := parser.Parse(string(data))
	if err != nil {
		return fmt.Errorf("parsing schema: %w", err)
	}

	// Resolve database URL from schema if not provided explicitly.
	if databaseURL == "" && next.Datasource != nil {
		databaseURL = db.ResolveDatabaseURL(next.Datasource.URL)
	}

	// Auto-create the database if a URL is available.
	if databaseURL != "" {
		if err := db.EnsureDatabase(databaseURL); err != nil {
			return fmt.Errorf("ensuring database: %w", err)
		}
	}

	// Connect to the database.
	var conn *sql.DB
	if databaseURL != "" {
		conn, err = db.Connect(databaseURL)
		if err != nil {
			return fmt.Errorf("connecting to database: %w", err)
		}
		defer conn.Close()
	}

	// For now, treat current state as empty (no DB introspection yet).
	var current *schema.Schema
	ops := migrator.Diff(current, next)
	if len(ops) == 0 {
		fmt.Println("no changes")
		return nil
	}

	sqlText := migrator.Generate(ops, next)

	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		return fmt.Errorf("creating migrations dir: %w", err)
	}

	filename := fmt.Sprintf("%s/%s_migration.sql", migrationsDir, time.Now().Format("20060102150405"))

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating migration file: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	if _, err := w.WriteString(sqlText); err != nil {
		return fmt.Errorf("writing migration: %w", err)
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flushing migration: %w", err)
	}

	fmt.Printf("created %s\n", filename)

	// Apply migration to the real database if connected.
	if conn != nil {
		if _, err := conn.Exec(sqlText); err != nil {
			return fmt.Errorf("applying migration: %w", err)
		}
		fmt.Println("migration applied")
	}

	return nil
}
