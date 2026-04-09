package cli

import (
	"fmt"
	"os"
)

const schemaTemplate = `datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?
  createdAt DateTime @default(now())
}
`

const envTemplate = `DATABASE_URL=postgresql://user:password@localhost:5432/mydb?sslmode=disable
`

const gitignoreTemplate = `generated/
migrations/
.env
bin/
`

// RunInit scaffolds a new prim project by creating template files.
func RunInit() error {
	if _, err := os.Stat("schema.prisma"); err == nil {
		return fmt.Errorf("schema.prisma already exists")
	}

	files := []struct {
		name    string
		content string
	}{
		{"schema.prisma", schemaTemplate},
		{".env", envTemplate},
		{".gitignore", gitignoreTemplate},
	}

	for _, f := range files {
		if err := os.WriteFile(f.name, []byte(f.content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", f.name, err)
		}
		fmt.Printf("created %s\n", f.name)
	}

	fmt.Print(`
next steps:
  1. edit .env with your database URL
  2. edit schema.prisma with your models
  3. prim generate    → generate Go code
  4. prim migrate     → create and apply migrations
  5. prim studio      → open visual query builder
`)

	return nil
}
