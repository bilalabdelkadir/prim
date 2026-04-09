package cli

import (
	"fmt"

	"github.com/bilalabdelkadir/prim/internal/parser"
)

// RunValidate parses the schema file and reports whether it is valid.
func RunValidate(schemaPath string) error {
	data, err := readSchema(schemaPath)
	if err != nil {
		return err
	}

	s, err := parser.Parse(string(data))
	if err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}

	fmt.Printf("schema is valid: %d models found\n", len(s.Models))
	return nil
}
