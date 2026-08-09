package main

import (
	"os"
	"path/filepath"
)

// writeToken writes to a temporary file in the destination directory and
// renames it into place, so a reader never sees a partially written token and
// never sees the file with default permissions.
func writeToken(path, tok string) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".github-app-tokend-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer func() {
		f.Close()
		os.Remove(tmp)
	}()

	if err := f.Chmod(0o600); err != nil {
		return err
	}
	if _, err := f.WriteString(tok + "\n"); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
