// Command migrate is a thin wrapper around github.com/golang-migrate/migrate/v4
// that reads SQL files from a directory and applies them against the database
// specified by DATABASE_URL.
//
// Migrations are forward-only. Files must be named <version>_<title>.up.sql —
// that is the only form golang-migrate's file source driver parses.
//
// Usage:
//
//	cmd/migrate -cmd up -dir migrations
//	cmd/migrate -cmd version -dir migrations
//	cmd/migrate -cmd force -version 3 -dir migrations
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

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

	// Fail loudly, and by name, before handing the directory to golang-migrate.
	// Its file source driver silently mis-parses any file that does not follow
	// the NNN_title.sql convention and reports only a cryptic
	// "first .: file does not exist", which is very hard to act on.
	if offenders, err := nonMigrationFiles(*dir); err != nil {
		fatal("cannot read migrations dir %q: %v", *dir, err)
	} else if len(offenders) > 0 {
		fatal("migrations dir %q contains %d file(s) golang-migrate cannot parse: %s\n"+
			"every file must be named <version>_<title>.sql (e.g. 008_add_widget.sql); "+
			"manual/one-off SQL belongs outside this directory",
			*dir, len(offenders), strings.Join(offenders, ", "))
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
		// This project is deliberately forward-only. Shipping untested .down.sql
		// files would be worse than refusing to roll back: a wrong or no-op
		// down migration either destroys data or marks a version as reverted
		// while leaving the schema in place, which is the worst of both.
		//
		// To recover from a half-applied migration, use -cmd force -version N,
		// which is the operation this tool actually supports for that case.
		fatal("-cmd down is not supported: this project's migrations are forward-only " +
			"(no .down.sql files are shipped on purpose). " +
			"Use -cmd force -version N to resynchronise the recorded version after a failed run.")
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

// migrationFilePattern matches the only filenames golang-migrate's file source
// driver understands. Its parser is
//
//	^([0-9]+)_(.*)\.(down|up)\.(.*)$
//
// so BOTH the .up and .down infixes are mandatory. The older single-file
// convention (001_init.sql) is silently rejected: the driver then reports
// "no migration" for -cmd version and "up: first .: file does not exist" for
// -cmd up, which reads like a missing directory rather than a naming problem.
//
// Titles may contain underscores (002_sample_data.up.sql), so the middle group
// is permissive.
var migrationFilePattern = regexp.MustCompile(`^\d+_.+\.(up|down)\.[^.]+$`)

// nonMigrationFiles returns the sorted names of regular files in dir that
// golang-migrate would refuse to parse. An empty result means the directory is
// safe to hand to the driver.
func nonMigrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var offenders []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			// Editor/OS droppings; the driver skips dotfiles.
			continue
		}
		if !migrationFilePattern.MatchString(name) {
			offenders = append(offenders, name)
		}
	}
	sort.Strings(offenders)
	return offenders, nil
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
