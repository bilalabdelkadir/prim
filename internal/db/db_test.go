package db

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDatabaseURL(t *testing.T) {
	host, port, user, password, dbname, err := ParseDatabaseURL("postgresql://admin:2334@172.17.0.1:5432/newdb")
	require.NoError(t, err)
	assert.Equal(t, "172.17.0.1", host)
	assert.Equal(t, "5432", port)
	assert.Equal(t, "admin", user)
	assert.Equal(t, "2334", password)
	assert.Equal(t, "newdb", dbname)
}

func TestParseDatabaseURL_WithSSLMode(t *testing.T) {
	host, port, user, password, dbname, err := ParseDatabaseURL("postgresql://admin:2334@172.17.0.1:5432/newdb?sslmode=disable")
	require.NoError(t, err)
	assert.Equal(t, "172.17.0.1", host)
	assert.Equal(t, "5432", port)
	assert.Equal(t, "admin", user)
	assert.Equal(t, "2334", password)
	assert.Equal(t, "newdb", dbname)
}

func TestResolveDatabaseURL_EnvVar(t *testing.T) {
	const expected = "postgresql://admin:2334@172.17.0.1:5432/newdb"
	os.Setenv("DATABASE_URL", expected)
	defer os.Unsetenv("DATABASE_URL")

	got := ResolveDatabaseURL(`env("DATABASE_URL")`)
	assert.Equal(t, expected, got)
}

func TestResolveDatabaseURL_Literal(t *testing.T) {
	const literal = "postgresql://admin:2334@172.17.0.1:5432/newdb"
	got := ResolveDatabaseURL(`"` + literal + `"`)
	assert.Equal(t, literal, got)
}

func TestBuildMaintenanceURL(t *testing.T) {
	raw := "postgresql://admin:2334@172.17.0.1:5432/newdb"
	got := buildMaintenanceURL(raw, "newdb")
	assert.Equal(t, "postgresql://admin:2334@172.17.0.1:5432/postgres", got)
}

func TestBuildMaintenanceURL_WithQuery(t *testing.T) {
	raw := "postgresql://admin:2334@172.17.0.1:5432/newdb?sslmode=disable"
	got := buildMaintenanceURL(raw, "newdb")
	assert.Equal(t, "postgresql://admin:2334@172.17.0.1:5432/postgres?sslmode=disable", got)
}
