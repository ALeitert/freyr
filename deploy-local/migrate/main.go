package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var (
		connectionStr string
		migrationPath string
	)

	flag.StringVar(&connectionStr, "con", "", "Connection string to the PostgreSQL database.")
	flag.StringVar(&migrationPath, "mig", "", "Path to the directory with migration files.")
	flag.Parse()

	migrationPath, _ = filepath.Abs(migrationPath)

	mErr := tryMigrate(connectionStr, migrationPath)
	if mErr != nil && !errors.Is(mErr, migrate.ErrNoChange) {
		cErr := tryClean(connectionStr, migrationPath)

		fmt.Println("Error(s) while migrating database:", errors.Join(mErr, cErr))
		os.Exit(1)
	}
}

func tryMigrate(connectionStr, migrationPath string) (err error) {
	var m *migrate.Migrate
	m, err = migrate.New("file://"+migrationPath, connectionStr)
	if err != nil {
		return fmt.Errorf("failed to create migration object: %w", err)
	}

	defer func() {
		srcErr, dbErr := m.Close()
		err = errors.Join(err, srcErr, dbErr)
	}()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to migrate: %w", err)
	}

	return nil
}

func tryClean(connectionStr, migrationPath string) error {
	m, err := migrate.New("file://"+migrationPath, connectionStr)
	if err != nil {
		return fmt.Errorf("failed to create migration object for cleaning: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("failed to determine migration version: %w", err)
	}

	if dirty {
		m.Force(int(version) - 1)
	}

	return nil
}
