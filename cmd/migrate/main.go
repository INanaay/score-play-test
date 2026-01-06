package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var (
		databaseURL string
		source      string
		up          bool
		down        bool
	)

	flag.StringVar(&databaseURL, "database", "", "Database connection URL (ex: postgresql://user:pass@host:port/dbname)")
	flag.StringVar(&source, "source", "", "Path to migrations directory (ex: db/migrations)")
	flag.BoolVar(&up, "up", false, "Run up migrations")
	flag.BoolVar(&down, "down", false, "Run down migrations")
	flag.Parse()

	if databaseURL == "" {
		log.Fatal("-database flag is required")
	}
	if source == "" {
		log.Fatal("-source flag is required")
	}
	if !up && !down {
		log.Fatal("either -up or -down flag is required")
	}
	if up && down {
		log.Fatal("cannot specify both -up and -down flags")
	}

	//Open database connection
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Create postgres driver instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Fatalf("failed to create database driver: %v", err)
	}
	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", source),
		"postgres",
		driver,
	)
	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}
	// Run migrations
	if up {
		log.Println("running UP migrations...")
		if err := m.Up(); err != nil {
			if err == migrate.ErrNoChange {
				log.Println("no new migrations to apply")
				os.Exit(0)
			}
			log.Fatalf("failed to run up migrations: %v", err)
		}
		log.Println("UP migrations completed successfully")
	}

	if down {
		log.Println("running DOWN migrations...")
		if err := m.Down(); err != nil {
			if err == migrate.ErrNoChange {
				log.Println("no migrations to rollback")
				os.Exit(0)
			}
			log.Fatalf("failed to run down migrations: %v", err)
		}
		log.Println("DOWN migrations completed successfully")
	}

}
