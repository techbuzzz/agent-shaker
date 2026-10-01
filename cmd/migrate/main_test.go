package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNonMigrationFiles(t *testing.T) {
	tests := []struct {
		name       string
		files      []string
		dirs       []string
		want       []string
		missingDir bool
	}{
		{
			name:  "clean directory",
			files: []string{"001_init.up.sql", "002_sample_data.up.sql", "003_daily_standups.up.sql"},
			want:  nil,
		},
		{
			name: "split up/down migrations are valid",
			// golang-migrate also accepts <version>_<title>.<up|down>.sql
			files: []string{"001_init.up.sql", "001_init.down.sql"},
			want:  nil,
		},
		{
			name: "underscored titles are valid",
			// 002_sample_data.sql is a real migration: the title may contain
			// underscores, so the pattern must not exclude them.
			files: []string{"001_init.up.sql", "002_sample_data.up.sql", "003_daily_standups.up.sql"},
			want:  nil,
		},
		{
			name:  "three digit versions",
			files: []string{"001_a.up.sql", "010_b.up.sql", "100_c.up.sql"},
			want:  nil,
		},
		{
			// The real-world failure mode: the old single-file convention looks
			// correct to a human but is silently rejected by golang-migrate's
			// parser, which then reports "first .: file does not exist" and
			// makes it look like the directory is missing.
			name:  "legacy combined filename is reported",
			files: []string{"001_init.up.sql", "002_sample_data.sql"},
			want:  []string{"002_sample_data.sql"},
		},
		{
			// This is what actually broke `docker compose up`: a manual
			// operator script parked in the migrations dir.
			name:  "unversioned manual script is reported",
			files: []string{"001_init.up.sql", "bootstrap_existing_db.sql"},
			want:  []string{"bootstrap_existing_db.sql"},
		},
		{
			name:  "readme in the dir is reported",
			files: []string{"001_init.up.sql", "README.md"},
			want:  []string{"README.md"},
		},
		{
			name:  "dotfiles are ignored",
			files: []string{"001_init.up.sql", ".gitkeep", ".DS_Store"},
			want:  nil,
		},
		{
			name:  "subdirectories are ignored",
			files: []string{"001_init.up.sql"},
			dirs:  []string{"archive", "bootstrap"},
			want:  nil,
		},
		{
			// t.TempDir() always creates the directory, so point at a child
			// path that is never created.
			name:       "missing directory is an error",
			missingDir: true,
		},
		{
			name:  "offenders are sorted for stable output",
			files: []string{"001_init.up.sql", "zebra.sql", "alpha.sql"},
			want:  []string{"alpha.sql", "zebra.sql"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tc.files {
				if err := os.WriteFile(filepath.Join(dir, f), []byte("-- test\n"), 0o644); err != nil {
					t.Fatalf("write %s: %v", f, err)
				}
			}
			for _, d := range tc.dirs {
				if err := os.Mkdir(filepath.Join(dir, d), 0o755); err != nil {
					t.Fatalf("mkdir %s: %v", d, err)
				}
			}
			if tc.missingDir {
				// t.TempDir() exists, so descend into a child that does not.
				dir = filepath.Join(dir, "no-such-dir")
			}

			got, err := nonMigrationFiles(dir)
			if tc.missingDir {
				if err == nil {
					t.Fatalf("expected an error for a missing directory, got offenders=%v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("offender[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
