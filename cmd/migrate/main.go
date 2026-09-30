// Command migrate is a thin wrapper around github.com/golang-migrate/migrate/v4
// that reads SQL files from a directory and applies (or rolls back) them
// against the database specified by DATABASE_URL.
//
// Usage:
//
//	cmd/migrate -cmd up -dir migrations
//	cmd/migrate -cmd down -steps 1 -dir migrations
//	cmd/migrate -cmd version -dir migrations
//	cmd/migrate -cmd force -version 3 -dir migrations
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	cmd := flag.String("cmd", "up", "command: up | down | version | force")
	dir := flag.String("dir", "migrations", "directory holding the migration SQL files")
	steps := flag.Int("steps", 0, "step count for up/down (0 = all)")
	version := flag.Int("version", 0, "target version for force")
	flag.Parse()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fatal("DATABASE_URL not set")
	}
	// golang-migrate expects the postgres scheme to be present; most DSNs
	// already start with postgres:// so no rewrite is needed.
	src := "file://" + *dir

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	m, err := migrate.New(src, dsn)
	if err != nil {
		fatal("migrate.New: %v", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			slog.Error("source close", "error", srcErr)
		}
		if dbErr != nil {
			slog.Error("db close", "error", dbErr)
		}
	}()

	switch *cmd {
	case "up":
		if *steps > 0 {
			err = m.Steps(*steps)
		} else {
			err = m.Up()
		}
	case "down":
		if *steps > 0 {
			err = m.Steps(-*steps)
		} else {
			err = m.Down()
		}
	case "version":
		v, dirty, err2 := m.Version()
		if err2 != nil {
			fatal("version: %v", err2)
		}
		fmt.Printf("version=%d dirty=%v\n", v, dirty)
		return
	case "force":
		if *version == 0 {
			fatal("force requires -version N")
		}
		err = m.Force(*version)
	default:
		fatal("unknown -cmd %q", *cmd)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		fatal("%s: %v", *cmd, err)
	}
	fmt.Println(strconv.Quote(*cmd), "succeeded")
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
