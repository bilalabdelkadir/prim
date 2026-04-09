package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// readSchema reads a schema file and wraps common errors with helpful hints.
func readSchema(schemaPath string) ([]byte, error) {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s not found. Run 'prim init' to create one, or use -schema to specify a path", schemaPath)
		}
		return nil, fmt.Errorf("reading schema: %w", err)
	}
	return data, nil
}

// wrapConnError adds a hint to connection-related errors.
func wrapConnError(context string, err error) error {
	msg := err.Error()
	if strings.Contains(msg, "connection refused") || strings.Contains(msg, "connect: connection refused") {
		return fmt.Errorf("%s: %w\nhint: check that PostgreSQL is running and the URL is correct", context, err)
	}
	return fmt.Errorf("%s: %w", context, err)
}
