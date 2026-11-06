// Package backup snapshots files before MeMo-AI modifies them.
package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Stamp returns a sortable timestamp suitable for a backup directory.
func Stamp() string {
	return time.Now().Format("20060102-150405.000")
}

// Copy copies src to dst, creating parent directories.
func Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// Snapshot copies src into backupRoot/<stamp>/<rel> and returns the new path.
func Snapshot(backupRoot, stamp, src, rel string) (string, error) {
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	dst := filepath.Join(backupRoot, stamp, filepath.FromSlash(rel))
	if err := Copy(src, dst); err != nil {
		return "", fmt.Errorf("backup %s: %w", src, err)
	}
	return dst, nil
}
