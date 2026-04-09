# prim

Prisma-inspired schema language, migration engine, and CRUD code generator for Go.

## What is prim?

prim takes the developer experience of [Prisma](https://www.prisma.io/) and brings it to Go. You define your data models in a familiar `.prisma` schema file, and prim generates type-safe Go structs and repository code with full CRUD operations, manages database migrations through schema diffing, and provides a visual query builder (studio) for constructing complex queries.

Key design decisions:

- **Not an ORM.** prim generates plain Go functions that use `database/sql` directly. There is no query builder abstraction, no reflection, no interface indirection. The generated code is straightforward SQL that you can read, debug, and modify.
- **Zero runtime dependencies.** Generated code imports only standard library packages: `database/sql`, `context`, and `time`. Your application does not take on any dependency from prim at runtime.
- **Currently supports PostgreSQL.** The schema parser, migration engine, and generated SQL all target PostgreSQL.

## Features

- `prim init` to scaffold a new project with schema, `.env`, and `.gitignore`
- Schema language with Prisma-compatible syntax for defining models and relations
- `prim generate` to auto-generate Go structs and repository files with `FindByID`, `Create`, `Update`, and `Delete` methods
- `prim migrate` to diff your schema, generate SQL migration files, and apply them
- `prim validate` to check your schema for errors without generating anything
- `prim studio` for a visual query builder with nested includes, live code preview, and save to file
- Auto-creates the target database if it does not exist
- Helpful error messages with actionable hints for common issues (SSL, auth, connection)
- Generated code uses only `database/sql` with no external dependencies
- `prim --version` to check your installed version

## Installation

```bash
go install github.com/bilalabdelkadir/prim/cmd/prim@latest
```

Make sure `$HOME/go/bin` is in your PATH:

```bash
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

## Quick Start

### 1. Initialize a project

```bash
mkdir myproject && cd myproject
go mod init myproject
prim init
```

This creates three files:
- `schema.prisma` with a starter User model
- `.env` with a placeholder `DATABASE_URL`
- `.gitignore` for generated files and secrets

Edit `.env` with your actual database URL:

```
DATABASE_URL=postgresql://user:password@localhost:5432/mydb?sslmode=disable
```

Then edit `schema.prisma` with your models:

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

You can validate your schema at any time:

```bash
prim validate
# schema is valid: 2 models found
```

### 2. Generate Go code

```bash
prim generate -schema schema.prisma -output generated
```

This produces one model file and one repository file per model. For the `User` model, the generated code looks like this:

**generated/user_model.go**

```go
package db

import "time"

type User struct {
    Id        int
    Email     string
    Name      *string
    CreatedAt time.Time
}
```

**generated/user_repository.go**

```go
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

func (r *UserRepository) FindByID(ctx context.Context, id int) (*User, error) {
    u := &User{}
    err := r.db.QueryRowContext(ctx,
        `SELECT "id", "email", "name", "createdAt" FROM "users" WHERE "id"=$1`,
        id,
    ).Scan(&u.Id, &u.Email, &u.Name, &u.CreatedAt)
    if err != nil {
        return nil, err
    }
    return u, nil
}

func (r *UserRepository) Create(ctx context.Context, email string, name *string, createdAt time.Time) (*User, error) {
    u := &User{}
    err := r.db.QueryRowContext(ctx,
        `INSERT INTO "users" ("email", "name", "createdAt") VALUES ($1, $2, $3) RETURNING "id", "email", "name", "createdAt"`,
        email, name, createdAt,
    ).Scan(&u.Id, &u.Email, &u.Name, &u.CreatedAt)
    if err != nil {
        return nil, err
    }
    return u, nil
}

func (r *UserRepository) Update(ctx context.Context, id int, email string, name *string, createdAt time.Time) (*User, error) {
    u := &User{}
    err := r.db.QueryRowContext(ctx,
        `UPDATE "users" SET "email"=$1, "name"=$2, "createdAt"=$3 WHERE "id"=$4 RETURNING "id", "email", "name", "createdAt"`,
        email, name, createdAt, id,
    ).Scan(&u.Id, &u.Email, &u.Name, &u.CreatedAt)
    if err != nil {
        return nil, err
    }
    return u, nil
}

func (r *UserRepository) Delete(ctx context.Context, id int) error {
    _, err := r.db.ExecContext(ctx,
        `DELETE FROM "users" WHERE "id"=$1`,
        id,
    )
    return err
}
```

Optional fields (marked with `?` in the schema) become pointer types in Go. The `@id` field is excluded from `Create` parameters since the database generates it via `autoincrement()`.

### 3. Run migrations

```bash
prim migrate -schema schema.prisma -dir migrations -db "postgresql://user:pass@localhost:5432/mydb?sslmode=disable"
```

The migration engine will:

1. Auto-create the database (`mydb`) if it does not exist.
2. Diff your schema against the current database state.
3. Generate a timestamped SQL file in the `migrations/` directory.
4. Apply the migration.

You can also set the `DATABASE_URL` environment variable and omit the `-db` flag. If both are provided, the `-db` flag takes precedence.

### 4. Use in your Go code

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    db "myproject/generated"
    _ "github.com/lib/pq"
)

func main() {
    conn, _ := sql.Open("postgres", "postgresql://user:pass@localhost:5432/mydb?sslmode=disable")

    userRepo := db.NewUserRepository(conn)

    // Create
    user, _ := userRepo.Create(context.Background(), "alice@example.com", nil, time.Now())
    fmt.Println(user.Id, user.Email)

    // Find
    found, _ := userRepo.FindByID(context.Background(), user.Id)
    fmt.Println(found.Email)

    // Update
    name := "Alice"
    updated, _ := userRepo.Update(context.Background(), user.Id, "alice@example.com", &name, time.Now())
    fmt.Println(*updated.Name)

    // Delete
    userRepo.Delete(context.Background(), user.Id)
}
```

## Schema Language

prim uses Prisma-compatible schema syntax.

### Datasource block

Configures the database connection:

```prisma
datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}
```

The `env("...")` function reads from environment variables at runtime.

### Model blocks

Each `model` block defines a database table and its columns:

```prisma
model Product {
  id          Int       @id @default(autoincrement())
  sku         String    @unique
  name        String
  description String?
  unitPrice   Float
  createdAt   DateTime  @default(now())
  updatedAt   DateTime
  inventory   Inventory[]
}
```

### Field types

| Type       | Description                     |
|------------|---------------------------------|
| `Int`      | Integer                         |
| `String`   | Text                            |
| `Boolean`  | True/false                      |
| `Float`    | Floating-point number           |
| `DateTime` | Timestamp                       |

### Modifiers

- `?` marks a field as optional (nullable). Example: `name String?`
- `[]` marks a relation field as a list. Example: `posts Post[]`

### Attributes

| Attribute                                      | Description                              |
|------------------------------------------------|------------------------------------------|
| `@id`                                          | Marks the primary key                    |
| `@default(autoincrement())`                    | Auto-incrementing integer                |
| `@default(now())`                              | Defaults to current timestamp            |
| `@default("value")`                            | Static default value                     |
| `@unique`                                      | Adds a unique constraint                 |
| `@relation(fields: [...], references: [...])`  | Defines a foreign key relation           |

### Relations

Relations are defined by pairing a reference field with a `@relation` attribute:

```prisma
model Inventory {
  id        Int     @id @default(autoincrement())
  productId Int
  product   Product @relation(fields: [productId], references: [id])
  warehouse String
  quantity  Int
}
```

The `productId` field stores the foreign key. The `product` field defines the relation and is not stored as a column. On the other side, `Product` has an `inventory Inventory[]` field representing the reverse relation.

## CLI Reference

### `prim init`

Scaffolds a new prim project in the current directory.

```
prim init
```

Creates `schema.prisma`, `.env`, and `.gitignore`. Will not overwrite if `schema.prisma` already exists.

### `prim generate`

Parses a schema file and generates Go model and repository files.

```
prim generate [flags]
  -schema string   Path to schema file (default "schema.prisma")
  -output string   Output directory (default "generated")
```

### `prim migrate`

Diffs the schema against the database, generates a SQL migration file, and applies it.

```
prim migrate [flags]
  -schema string   Path to schema file (default "schema.prisma")
  -dir string      Migrations directory (default "migrations")
  -db string       Database URL (overrides schema datasource and DATABASE_URL env)
```

If `-db` is not provided, prim falls back to the `DATABASE_URL` environment variable.

### `prim validate`

Checks a schema file for syntax errors without generating code or connecting to a database.

```
prim validate [flags]
  -schema string   Path to schema file (default "schema.prisma")
```

### `prim studio`

Launches the visual query builder. The studio UI is embedded in the binary.

```
prim studio [flags]
  -schema string   Path to schema file (default "schema.prisma")
  -port int        Studio port (default 4983)
  -db string       Database URL (overrides schema datasource and DATABASE_URL env)
```

### `prim --version`

Prints the installed version.

```
prim --version
# prim v0.1.0
```

## Prim Studio

Prim Studio is a web-based visual query builder that runs locally. It lets you construct complex database queries through a graphical interface and generates the corresponding Go code.

### Capabilities

- Browse tables and their schemas
- Build queries with nested includes across relations
- Apply WHERE conditions, ordering, and pagination at each include level
- Live preview of the generated Go code
- Save generated query methods directly to your repository files
- Supports Find One, Find Many, Count, Create, Update, and Delete operations

### Running Studio

The studio UI is embedded in the prim binary. One command starts everything:

```bash
prim studio -schema schema.prisma -db "postgresql://user:pass@localhost:5432/mydb?sslmode=disable"
```

Then open `http://localhost:4983` in your browser.

The `-db` flag is optional. Without it, studio runs in schema-only mode: you can browse models, build queries, and preview generated code, but Raw SQL will not work.

### Development Mode

If you are contributing to prim and want to work on the studio frontend with hot reloading:

Terminal 1 (Go backend):
```bash
prim studio -schema schema.prisma -db "postgresql://..."
```

Terminal 2 (React dev server with hot reload):
```bash
cd studio-ui
npm install  # first time only
npm run dev
```

Then open `http://localhost:5173` instead. The Vite dev server proxies API calls to the Go backend.

## Generated Code

### File structure

For each model in your schema, prim generates two files:

- `<model>_model.go` -- struct definition
- `<model>_repository.go` -- CRUD methods

All generated files use `package db` and import only standard library packages (`database/sql`, `context`, `time`).

### Repository methods

Every repository provides these methods:

| Method     | Signature                                                    | SQL Operation            |
|------------|--------------------------------------------------------------|--------------------------|
| `FindByID` | `FindByID(ctx context.Context, id int) (*Model, error)`     | `SELECT ... WHERE id=$1` |
| `Create`   | `Create(ctx context.Context, ...fields) (*Model, error)`    | `INSERT ... RETURNING`   |
| `Update`   | `Update(ctx context.Context, id int, ...fields) (*Model, error)` | `UPDATE ... RETURNING`   |
| `Delete`   | `Delete(ctx context.Context, id int) error`                 | `DELETE ... WHERE id=$1` |

The `Create` method excludes the `@id @default(autoincrement())` field from its parameters. Both `Create` and `Update` return the full row using PostgreSQL's `RETURNING` clause.

### Custom queries (via Studio)

Studio can generate additional methods for complex queries that involve nested includes. These methods are appended to the existing repository file. For example, a query that fetches inventory records with their related products generates dedicated result types and a new repository method:

```go
type FindInventorySalesResult struct {
    Id        int
    ProductId int
    Warehouse string
    Quantity  int
    MinStock  int
    UpdatedAt time.Time
    Product   *FindInventorySalesProductResult
}

type FindInventorySalesProductResult struct {
    Id          int
    Sku         string
    Name        string
    Description *string
    UnitPrice   float64
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

func (r *InventoryRepository) FindInventorySales(ctx context.Context) ([]*FindInventorySalesResult, error) {
    // ... query with joined relation data
}
```

## Example Project Structure

```
myproject/
├── schema.prisma              # Your schema definition
├── generated/                 # Auto-generated by prim
│   ├── product_model.go
│   ├── product_repository.go
│   ├── inventory_model.go
│   ├── inventory_repository.go
│   ├── customer_model.go
│   ├── customer_repository.go
│   └── ...
├── handlers/                  # Your application code
│   ├── products.go
│   └── ...
├── migrations/                # Auto-generated SQL
│   └── 20240101_migration.sql
├── go.mod
└── main.go
```

## Supported Field Types

| Schema Type  | Go Type      | PostgreSQL Type              |
|--------------|--------------|------------------------------|
| `Int`        | `int`        | `INTEGER` / `SERIAL`         |
| `String`     | `string`     | `TEXT`                       |
| `Boolean`    | `bool`       | `BOOLEAN`                    |
| `Float`      | `float64`    | `DOUBLE PRECISION`           |
| `DateTime`   | `time.Time`  | `TIMESTAMP WITH TIME ZONE`   |
| `String?`    | `*string`    | `TEXT` (nullable)            |
| `Int?`       | `*int`       | `INTEGER` (nullable)         |
| `Float?`     | `*float64`   | `DOUBLE PRECISION` (nullable)|
| `Boolean?`   | `*bool`      | `BOOLEAN` (nullable)         |
| `DateTime?`  | `*time.Time` | `TIMESTAMP WITH TIME ZONE` (nullable) |

## Environment Variables

| Variable       | Description                                                                 |
|----------------|-----------------------------------------------------------------------------|
| `DATABASE_URL` | PostgreSQL connection string. Used when the schema specifies `url = env("DATABASE_URL")`. |

## Roadmap

- [x] ~~Embed studio UI in the Go binary~~ (done)
- [x] ~~`prim init` to scaffold new projects~~ (done)
- [x] ~~`prim validate` to check schema~~ (done)
- [x] ~~Helpful error messages with hints~~ (done)
- [ ] Database introspection (currently treats DB as empty on each migrate)
- [ ] `CREATE TABLE IF NOT EXISTS` in migrations
- [ ] MySQL / SQLite support
- [ ] Enum types in schema
- [ ] Index definitions
- [ ] Seed data command
- [ ] `prim db push` (apply schema directly without migration files)
- [ ] Configurable package name for generated code

## License

MIT
