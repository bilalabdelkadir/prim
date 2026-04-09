package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bilalabdelkadir/prim/internal/codegen"
	"github.com/bilalabdelkadir/prim/internal/parser"
)

// RunGenerate reads the schema file, parses it, and writes generated Go files
// to the output directory.
func RunGenerate(schemaPath, outDir string) error {
	data, err := readSchema(schemaPath)
	if err != nil {
		return err
	}

	s, err := parser.Parse(string(data))
	if err != nil {
		return fmt.Errorf("parsing schema: %w", err)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	for _, model := range s.Models {
		modelCode, err := codegen.GenerateModel(model)
		if err != nil {
			return fmt.Errorf("generating model %s: %w", model.Name, err)
		}
		repoCode, err := codegen.GenerateRepository(model)
		if err != nil {
			return fmt.Errorf("generating repository %s: %w", model.Name, err)
		}

		name := strings.ToLower(model.Name)

		if err := writeFile(filepath.Join(outDir, name+"_model.go"), modelCode); err != nil {
			return err
		}
		if err := writeFile(filepath.Join(outDir, name+"_repository.go"), repoCode); err != nil {
			return err
		}

		fmt.Printf("generated %s\n", model.Name)
	}

	return nil
}

// writeFile writes content to path using buffered I/O.
func writeFile(path, content string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	if _, err := w.WriteString(content); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return w.Flush()
}
