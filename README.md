# prim

Prisma-inspired schema language, migration engine, and CRUD code generator for Go.

---

## What is prim?

prim takes the developer experience of [Prisma](https://www.prisma.io/) and brings it to Go. You write a `.prisma` schema file, and prim handles the rest: generates type-safe Go structs and repositories, manages database migrations, auto-creates databases, and gives you a visual query builder for complex queries.

- **Not an ORM.** Generates plain Go functions using `database/sql`. No reflection, no magic.
- **Zero runtime dependencies.** Generated code imports only stdlib: `database/sql`, `context`, `time`.
- **PostgreSQL.** Parser, migrations, and generated SQL all target PostgreSQL.

---

## Installation

```bash
go install github.com/bilalabdelkadir/prim/cmd/prim@latest
```

Make sure `$HOME/go/bin` is in your PATH:

```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

Verify:

```bash
prim --version
# prim v0.1.0
```

---

## Getting Started

This walks you through the full flow from zero to a working Go app with generated code.

### Step 1: Initialize

Create a new Go project and scaffold prim files:

```bash
mkdir myapp && cd myapp
go mod init myapp
prim init
```

This creates:

| File | Purpose |
|---|---|
| `schema.prisma` | Your data model definition |
| `.env` | Database connection URL |
| `.gitignore` | Ignores generated files, migrations, and `.env` |

### Step 2: Configure your database

Edit `.env` with your PostgreSQL connection:

```
DATABASE_URL=postgresql://admin:password@localhost:5432/myapp?sslmode=disable
```

prim reads `.env` automatically. You never need to pass `-db` flags if this file is set up.

### Step 3: Define your schema

Edit `schema.prisma`:

```prisma
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String?
  posts     Post[]
  createdAt DateTime @default(now())
}

model Post {
  id        Int     @id @default(autoincrement())
  title     String
  content   String?
  published Boolean @default(false)
  authorId  Int
  author    User    @relation(fields: [authorId], references: [id])
}
```

Validate it:

```bash
prim validate
# schema is valid: 2 models found
```

### Step 4: Generate Go code

```bash
prim generate
```

This reads `schema.prisma` and writes Go files into `generated/`:

```
generated/
  user_model.go
  user_repository.go
  post_model.go
  post_repository.go
```

Each model gets a struct and a repository with four methods: `FindByID`, `Create`, `Update`, `Delete`.

Here's what the generated User code looks like:

```go
// generated/user_model.go
package db

import "time"

type User struct {
    Id        int
    Email     string
    Name      *string
    CreatedAt time.Time
}
```

```go
// generated/user_repository.go
package db

import (
    "context"
    "database/sql"
    "time"
)

type UserRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
    return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id int) (*User, error) { ... }
func (r *UserRepository) Create(ctx context.Context, email string, name *string, createdAt time.Time) (*User, error) { ... }
func (r *UserRepository) Update(ctx context.Context, id int, email string, name *string, createdAt time.Time) (*User, error) { ... }
func (r *UserRepository) Delete(ctx context.Context, id int) error { ... }
```

Optional fields (`String?`) become pointer types (`*string`). The `@id` field is excluded from `Create` params since the database generates it.

### Step 5: Run migrations

```bash
prim migrate
```

This will:
1. Auto-create the database if it doesn't exist
2. Diff your schema against the current DB state
3. Generate a timestamped SQL file in `migrations/`
4. Apply the migration

No flags needed — it reads `DATABASE_URL` from `.env` and `schema.prisma` from the current directory.

### Step 6: Use the generated code

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "time"

    db "myapp/generated"
    _ "github.com/lib/pq"
)

func main() {
    conn, err := sql.Open("postgres", "postgresql://admin:password@localhost:5432/myapp?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }

    userRepo := db.NewUserRepository(conn)
    ctx := context.Background()

    // Create a user
    user, err := userRepo.Create(ctx, "alice@example.com", nil, time.Now())
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("created user %d: %s\n", user.Id, user.Email)

    // Find by ID
    found, err := userRepo.FindByID(ctx, user.Id)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("found: %s\n", found.Email)

    // Update
    name := "Alice"
    updated, err := userRepo.Update(ctx, user.Id, "alice@example.com", &name, time.Now())
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("updated: %s\n", *updated.Name)

    // Delete
    err = userRepo.Delete(ctx, user.Id)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("deleted")
}
```

### Step 7: Build complex queries with Studio

For anything beyond basic CRUD — nested includes, filtered queries, joins across relations — use the visual query builder:

```bash
prim studio
```

Open `http://localhost:4983` in your browser. Studio lets you:

- Browse all your models and fields
- Build queries visually with nested includes (e.g. User -> Posts -> Comments)
- Add WHERE conditions, ORDER BY, and LIMIT at each level
- See generated Go code update live as you build
- Save the generated method directly to your repository file

Studio reads `.env` for the database connection. Without a database, it still works for schema browsing and code generation — only Raw SQL requires a live connection.

---

## Schema Language

prim uses Prisma-compatible schema syntax.

### Datasource

```prisma
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}
```

`env("...")` reads from environment variables (including `.env` files).

### Models

Each `model` defines a database table:

```prisma
model Product {
  id          Int       @id @default(autoincrement())
  sku         String    @unique
  name        String
  description String?
  unitPrice   Float
  createdAt   DateTime  @default(now())
}
```

### Field types

| Type | Go type | PostgreSQL type |
|---|---|---|
| `Int` | `int` | `INTEGER` / `SERIAL` |
| `String` | `string` | `TEXT` |
| `Boolean` | `bool` | `BOOLEAN` |
| `Float` | `float64` | `DOUBLE PRECISION` |
| `DateTime` | `time.Time` | `TIMESTAMP WITH TIME ZONE` |

Add `?` for optional (nullable) fields: `String?` becomes `*string` in Go.

### Attributes

| Attribute | Purpose |
|---|---|
| `@id` | Primary key |
| `@default(autoincrement())` | Auto-incrementing integer |
| `@default(now())` | Default to current timestamp |
| `@unique` | Unique constraint |
| `@relation(fields: [...], references: [...])` | Foreign key relation |

### Relations

```prisma
model Post {
  id       Int  @id @default(autoincrement())
  authorId Int
  author   User @relation(fields: [authorId], references: [id])
}

model User {
  id    Int    @id @default(autoincrement())
  posts Post[]
}
```

`authorId` is the foreign key column. `author` defines the relation (not stored as a column). `posts Post[]` is the reverse side.

---

## CLI Reference

All commands use sensible defaults. If you have `schema.prisma` and `.env` in your current directory, most flags are optional.

| Command | What it does |
|---|---|
| `prim init` | Scaffold a new project (`schema.prisma`, `.env`, `.gitignore`) |
| `prim generate` | Generate Go code from schema |
| `prim migrate` | Create and apply database migrations |
| `prim validate` | Check schema for syntax errors |
| `prim studio` | Open the visual query builder |
| `prim --version` | Print installed version |

### Flags

**`prim generate`**
```
-schema string   Path to schema file (default "schema.prisma")
-output string   Output directory (default "generated")
```

**`prim migrate`**
```
-schema string   Path to schema file (default "schema.prisma")
-dir string      Migrations directory (default "migrations")
-db string       Database URL override (default: reads .env)
```

**`prim studio`**
```
-schema string   Path to schema file (default "schema.prisma")
-port int        Port to serve on (default 4983)
-db string       Database URL override (default: reads .env)
```

**`prim validate`**
```
-schema string   Path to schema file (default "schema.prisma")
```

---

## Prim Studio

A web-based visual query builder, embedded in the prim binary.

```bash
prim studio
```

Open `http://localhost:4983`.

### What you can do

- **Tables tab** — browse models, view field types, attributes, relations
- **Query Builder** — construct complex queries visually:
  - Choose operation: Find One, Find Many, Count, Create, Update, Delete
  - Select fields to return
  - Add WHERE conditions with operators (=, !=, >, <, LIKE, IN, IS NULL)
  - Add nested includes at any depth (User -> Posts -> Comments -> ...)
  - Each include level has its own select, where, orderBy, limit
  - Data fields for Create/Update operations
  - Live Go code preview updates as you build
  - Fullscreen code view (click EXPAND)
  - Save generated methods directly to repository files
- **Raw SQL** — execute queries against the connected database
- **Settings** — adjust font size (Compact / Default / Comfortable)

### How includes work internally

prim uses the **multi-query pattern** (same as Prisma) instead of JOINs:

```sql
-- Query 1: fetch users
SELECT "id", "email", "name" FROM "users" WHERE ...

-- Query 2: fetch posts for those users
SELECT "id", "title", "authorId" FROM "posts" WHERE "authorId" = ANY($1)

-- Query 3: fetch comments for those posts
SELECT "id", "text", "postId" FROM "comments" WHERE "postId" = ANY($1)
```

Then assembles the tree in Go code using ID maps. This avoids the cartesian explosion problem that JOINs cause with nested one-to-many relations.

### Contributing to Studio

For frontend development with hot reload:

```bash
# Terminal 1: Go backend
prim studio

# Terminal 2: React dev server
cd studio-ui && npm install && npm run dev
```

Open `http://localhost:5173` — Vite proxies API calls to the Go backend.

---

## Project Structure

A typical project using prim:

```
myapp/
├── schema.prisma          # Your schema (you write this)
├── .env                   # DATABASE_URL (created by prim init)
├── .gitignore             # Ignores generated/, migrations/, .env
├── generated/             # Auto-generated by prim generate
│   ├── user_model.go
│   ├── user_repository.go
│   ├── post_model.go
│   └── post_repository.go
├── migrations/            # Auto-generated by prim migrate
│   └── 20240101120000_migration.sql
├── handlers/              # Your application code (you write this)
│   └── users.go           # Imports generated repos
├── go.mod
└── main.go
```

---

## Environment Variables

| Variable | Description |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string. Set in `.env` or your shell. |

prim automatically loads `.env` from the current directory. OS environment variables take precedence over `.env` values.

---

## Error Messages

prim gives actionable hints for common problems:

```
error: schema.prisma not found
hint: run 'prim init' to create one, or use -schema to specify a path

error: SSL is not enabled on the server
hint: add ?sslmode=disable to your database URL

error: could not connect to database
hint: check that PostgreSQL is running and the host/port are correct

error: authentication failed
hint: check the username and password in your database URL

line 5: expected model name after "model" keyword, got "{"
```

---

## Roadmap

- [x] `prim init` project scaffolding
- [x] `prim validate` schema checking
- [x] Embedded studio UI (single binary)
- [x] `.env` file auto-loading
- [x] Helpful error messages with hints
- [x] Belongs-to and has-many relation support in query builder
- [ ] Database introspection for migrations
- [ ] `CREATE TABLE IF NOT EXISTS` in migrations
- [ ] MySQL / SQLite support
- [ ] Enum types in schema
- [ ] Index definitions
- [ ] Seed data command
- [ ] `prim db push` (apply schema without migration files)
- [ ] Configurable package name for generated code

---

## License

MIT
