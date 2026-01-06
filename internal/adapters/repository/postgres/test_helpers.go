package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// getProjectRoot finds the project root by searching upwards for the go.mod file.
func getProjectRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		_, err := os.Stat(filepath.Join(wd, "go.mod"))
		if err == nil {
			return wd, nil
		}
		if wd == filepath.Dir(wd) {
			return "", errors.New("go.mod not found in any parent directory")
		}
		wd = filepath.Dir(wd)
	}
}

// NewTestDB creates a new db in a container
func NewTestDB(t *testing.T) (*sql.DB, func(), func()) {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:13-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpassword",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(30 * time.Second),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Could not start postgres container: %v", err)
	}

	host, _ := postgresContainer.Host(ctx)
	p, _ := postgresContainer.MappedPort(ctx, "5432")
	dbURL := fmt.Sprintf("postgres://testuser:testpassword@%s:%s/testdb?sslmode=disable", host, p.Port())

	projectRoot, err := getProjectRoot()
	if err != nil {
		t.Fatalf("Could not find project root: %v", err)
	}
	migrationsPath := filepath.Join(projectRoot, "db", "migrations")

	u := &url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(migrationsPath),
	}

	m, err := migrate.New(u.String(), dbURL)
	if err != nil {
		t.Fatalf("failed to init migrate with URL %s: %v", u.String(), err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("failed to run up migrations: %v", err)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}

	cleanup := func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate postgres container: %v", err)
		}
		db.Close()
	}

	truncateAll := func() {
		query := `
           DO $$ 
           DECLARE 
               r RECORD;
           BEGIN
               FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') 
               LOOP
                   EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' RESTART IDENTITY CASCADE';
               END LOOP;
           END $$;
       `
		_, err := db.Exec(query)
		if err != nil {
			t.Fatalf("failed to truncate tables: %v", err)
		}
	}
	return db, cleanup, truncateAll
}
