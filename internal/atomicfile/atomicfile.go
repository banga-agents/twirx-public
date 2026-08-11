// Package atomicfile publishes bounded files through a synchronized rename.
package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// Write writes data to path without exposing a partially written destination.
// The destination directory is created when necessary.
func Write(path string, data []byte, maxBytes int, mode os.FileMode) error {
	if path == "" {
		return fmt.Errorf("atomicfile: path is required")
	}
	if maxBytes <= 0 {
		return fmt.Errorf("atomicfile: byte limit must be positive")
	}
	if len(data) > maxBytes {
		return fmt.Errorf("atomicfile: data size %d exceeds %d-byte limit", len(data), maxBytes)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("atomicfile: create destination directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".tw-write-*")
	if err != nil {
		return fmt.Errorf("atomicfile: create temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("atomicfile: chmod temporary file: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("atomicfile: write temporary file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("atomicfile: sync temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("atomicfile: close temporary file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("atomicfile: commit file: %w", err)
	}
	directory, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("atomicfile: open destination directory: %w", err)
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return fmt.Errorf("atomicfile: sync destination directory: %w", err)
	}
	if err := directory.Close(); err != nil {
		return fmt.Errorf("atomicfile: close destination directory: %w", err)
	}
	return nil
}
