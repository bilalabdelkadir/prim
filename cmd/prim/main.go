package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bilalabdelkadir/prim/internal/cli"
	"github.com/bilalabdelkadir/prim/internal/db"
)

const (
	defaultSchema = "schema.prisma"
	version       = "v0.1.0"
)

const helpText = `prim — prisma-like codegen for Go

Usage:
  prim <command> [flags]

Commands:
  init        Scaffold a new prim project
  generate    Generate Go code from schema
  migrate     Create and apply database migrations
  studio      Open the visual query builder
  validate    Check schema for errors

Flags:
  --version   Print version

Run 'prim <command> -h' for command-specific help.`

func main() {
	// Load .env file if present (OS env vars take precedence).
	db.LoadDotEnv(".env")

	if len(os.Args) < 2 {
		fmt.Println(helpText)
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "help", "-h", "--help":
		fmt.Println(helpText)
		os.Exit(0)

	case "--version", "-version":
		fmt.Printf("prim %s\n", version)
		os.Exit(0)

	case "init":
		if err := cli.RunInit(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "validate":
		fs := flag.NewFlagSet("validate", flag.ExitOnError)
		schema := fs.String("schema", defaultSchema, "path to schema file")
		fs.Parse(args)

		if err := cli.RunValidate(*schema); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "generate":
		fs := flag.NewFlagSet("generate", flag.ExitOnError)
		schema := fs.String("schema", defaultSchema, "path to schema file")
		out := fs.String("output", "generated", "output directory")
		fs.Parse(args)

		if err := cli.RunGenerate(*schema, *out); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "migrate":
		fs := flag.NewFlagSet("migrate", flag.ExitOnError)
		schema := fs.String("schema", defaultSchema, "path to schema file")
		dir := fs.String("dir", "migrations", "migrations directory")
		dbURL := fs.String("db", "", "database URL (overrides schema and DATABASE_URL env)")
		fs.Parse(args)

		databaseURL := *dbURL
		if databaseURL == "" {
			databaseURL = os.Getenv("DATABASE_URL")
		}

		if err := cli.RunMigrate(*schema, *dir, databaseURL); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "studio":
		fs := flag.NewFlagSet("studio", flag.ExitOnError)
		schema := fs.String("schema", defaultSchema, "path to schema file")
		port := fs.Int("port", 4983, "studio port")
		dbURL := fs.String("db", "", "database URL (overrides schema and DATABASE_URL env)")
		fs.Parse(args)

		databaseURL := *dbURL
		if databaseURL == "" {
			databaseURL = os.Getenv("DATABASE_URL")
		}

		if err := cli.RunStudio(*schema, *port, databaseURL); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s\n", cmd, helpText)
		os.Exit(1)
	}
}
