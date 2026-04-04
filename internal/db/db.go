package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

// Connect opens a connection to the database at the given URL and pings it.
func Connect(databaseURL string) (*sql.DB, error) {
	fmt.Println("connecting to database...")
	conn, err := sql.Open("postgres", withTimeout(databaseURL))
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}
	fmt.Println("connected to database")
	return conn, nil
}

// EnsureDatabase checks whether the target database exists and creates it if
// it doesn't. It connects to the server's default "postgres" database to run
// the check and the CREATE DATABASE statement.
func EnsureDatabase(databaseURL string) error {
	_, _, _, _, dbname, err := ParseDatabaseURL(databaseURL)
	if err != nil {
		return fmt.Errorf("parsing database URL: %w", err)
	}

	maintenanceURL := buildMaintenanceURL(databaseURL, dbname)
	fmt.Printf("checking if database %q exists...\n", dbname)

	conn, err := sql.Open("postgres", withTimeout(maintenanceURL))
	if err != nil {
		return fmt.Errorf("connecting to maintenance db: %w", err)
	}
	defer conn.Close()

	var exists int
	err = conn.QueryRow("SELECT 1 FROM pg_database WHERE datname = $1", dbname).Scan(&exists)
	if err == nil {
		// Database already exists.
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("checking database existence: %w", err)
	}

	// Database does not exist — create it.
	// Use quoted identifier to handle special characters in the name.
	_, err = conn.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, strings.ReplaceAll(dbname, `"`, `""`)))
	if err != nil {
		return fmt.Errorf("creating database: %w", err)
	}

	fmt.Printf("created database %q\n", dbname)
	return nil
}

// ResolveDatabaseURL resolves a datasource URL value from a schema file.
// If the value looks like env("VAR_NAME"), the named environment variable is
// read. Otherwise the value is returned as a literal (with surrounding quotes
// stripped).
func ResolveDatabaseURL(schemaURL string) string {
	s := strings.TrimSpace(schemaURL)

	// Handle env("VAR_NAME") pattern.
	if strings.HasPrefix(s, `env("`) && strings.HasSuffix(s, `")`) {
		varName := s[5 : len(s)-2]
		return os.Getenv(varName)
	}

	// Strip surrounding quotes from a literal string.
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}

	return s
}

// ParseDatabaseURL extracts the individual components from a PostgreSQL URL.
func ParseDatabaseURL(rawURL string) (host, port, user, password, dbname string, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", "", "", fmt.Errorf("parsing URL: %w", err)
	}

	host = u.Hostname()
	port = u.Port()
	if port == "" {
		port = "5432"
	}

	user = u.User.Username()
	password, _ = u.User.Password()

	// The database name is the path without the leading slash.
	dbname = strings.TrimPrefix(u.Path, "/")

	return host, port, user, password, dbname, nil
}

// withTimeout appends connect_timeout=5 to the URL if not already set.
func withTimeout(rawURL string) string {
	if strings.Contains(rawURL, "connect_timeout") {
		return rawURL
	}
	if strings.Contains(rawURL, "?") {
		return rawURL + "&connect_timeout=5"
	}
	return rawURL + "?connect_timeout=5"
}

// buildMaintenanceURL replaces the database name in the URL with "postgres".
func buildMaintenanceURL(rawURL string, dbname string) string {
	// Replace the last occurrence of /dbname with /postgres, preserving
	// any query parameters.
	idx := strings.LastIndex(rawURL, "/"+dbname)
	if idx == -1 {
		return rawURL
	}
	return rawURL[:idx] + "/postgres" + rawURL[idx+1+len(dbname):]
}
