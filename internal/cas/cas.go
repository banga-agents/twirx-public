// Package cas provides a filesystem-backed content-addressed store.
package cas

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const Algorithm = "sha256"

// Store stores immutable blobs by SHA-256 digest.
type Store struct {
	Root string
}

func New(root string) *Store { return &Store{Root: root} }

func Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return Algorithm + ":" + hex.EncodeToString(sum[:])
}

func ParseDigest(digest string) ([32]byte, error) {
	var out [32]byte
	prefix := Algorithm + ":"
	if !strings.HasPrefix(digest, prefix) {
		return out, fmt.Errorf("cas: unsupported digest %q", digest)
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(digest, prefix))
	if err != nil {
		return out, fmt.Errorf("cas: decode digest: %w", err)
	}
	if len(raw) != len(out) {
		return out, fmt.Errorf("cas: digest length %d, want %d", len(raw), len(out))
	}
	copy(out[:], raw)
	return out, nil
}

func (s *Store) Path(digest string) (string, error) {
	sum, err := ParseDigest(digest)
	if err != nil {
		return "", err
	}
	hexDigest := hex.EncodeToString(sum[:])
	return filepath.Join(s.Root, Algorithm, hexDigest[:2], hexDigest[2:4], hexDigest), nil
}

func (s *Store) Put(data []byte) (string, string, error) {
	digest := Digest(data)
	path, err := s.Path(digest)
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", "", fmt.Errorf("cas: create parent: %w", err)
	}

	if existing, err := os.ReadFile(path); err == nil {
		if Digest(existing) != digest {
			return "", "", errors.New("cas: existing blob does not match its path digest")
		}
		return digest, path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", "", fmt.Errorf("cas: inspect existing blob: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".tw-cas-*")
	if err != nil {
		return "", "", fmt.Errorf("cas: create temporary blob: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(0o640); err != nil {
		_ = tmp.Close()
		return "", "", fmt.Errorf("cas: chmod temporary blob: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return "", "", fmt.Errorf("cas: write temporary blob: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", "", fmt.Errorf("cas: sync temporary blob: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", "", fmt.Errorf("cas: close temporary blob: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		if existing, readErr := os.ReadFile(path); readErr == nil && Digest(existing) == digest {
			return digest, path, nil
		}
		return "", "", fmt.Errorf("cas: commit blob: %w", err)
	}
	return digest, path, nil
}

func (s *Store) Open(digest string) (*os.File, error) {
	path, err := s.Path(digest)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cas: open blob: %w", err)
	}
	return f, nil
}

func (s *Store) Read(digest string, maxBytes int64) ([]byte, error) {
	f, err := s.Open(digest)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	limited := io.LimitReader(f, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("cas: read blob: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("cas: blob exceeds %d-byte read limit", maxBytes)
	}
	if Digest(data) != digest {
		return nil, errors.New("cas: blob digest mismatch")
	}
	return data, nil
}
