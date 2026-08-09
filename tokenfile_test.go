package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Readers open the file at arbitrary moments, so a rewrite must never expose a
// partial token or a world-readable file.
func TestWriteTokenIsAtomicAndPrivate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "token")

	if err := writeToken(path, "ghs_first"); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := fi.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}

	if err := writeToken(path, "ghs_second"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "ghs_second\n" {
		t.Errorf("content = %q", b)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("%d entries left in the directory, want 1", len(entries))
	}
}
