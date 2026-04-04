package cli

import (
	"fmt"
	"os"

	"github.com/bilalabdelkadir/prim/internal/db"
	"github.com/bilalabdelkadir/prim/internal/parser"
	"github.com/bilalabdelkadir/prim/internal/studio"
)

// RunStudio starts the studio web server. If databaseURL is provided, a live
// database connection is established. Otherwise studio runs in schema-only mode.
func RunStudio(schemaPath string, port int, databaseURL string) error {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("reading schema: %w", err)
	}

	s, err := parser.Parse(string(data))
	if err != nil {
		return fmt.Errorf("parsing schema: %w", err)
	}

	// Resolve database URL from schema if not provided explicitly.
	if databaseURL == "" && s.Datasource != nil {
		databaseURL = db.ResolveDatabaseURL(s.Datasource.URL)
	}

	if databaseURL != "" {
		if err := db.EnsureDatabase(databaseURL); err != nil {
			return fmt.Errorf("ensuring database: %w", err)
		}

		conn, err := db.Connect(databaseURL)
		if err != nil {
			return fmt.Errorf("connecting to database: %w", err)
		}
		defer conn.Close()

		srv := studio.NewServer(conn, s)
		return srv.Start(port)
	}

	// No database URL available — start studio without a DB connection.
	srv := studio.NewServer(nil, s)
	return srv.Start(port)
}
