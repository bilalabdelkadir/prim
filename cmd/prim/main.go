package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bilalabdelkadir/prim/internal/cli"
)

const defaultSchema = "schema.prisma"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: prim <command> [flags]")
		fmt.Println("commands: generate, migrate, studio")
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
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
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}
}
