package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	shaHexRe   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	keyRe      = regexp.MustCompile(`^[0-9a-f]{2}/[0-9a-f]{2}/[0-9a-f]{64}\.pdf$`)
	coverKeyRe = regexp.MustCompile(`^covers/[0-9a-f]{2}/[0-9a-f]{2}/[0-9a-f]{64}\.png$`)
)

// KeyFor returns the sharded storage key for a sha256 hex digest.
func KeyFor(sha string) string {
	return filepath.Join(sha[:2], sha[2:4], sha+".pdf")
}

// SaveAtomic streams src to <root>/tmp/*.part, fsyncs it and renames it to
// <root>/ab/cd/<sha256>.pdf. It returns the storage key and bytes written.
func SaveAtomic(storageRoot, sha string, src io.Reader) (string, int64, error) {
	if !shaHexRe.MatchString(strings.ToLower(sha)) {
		return "", 0, fmt.Errorf("invalid sha256")
	}
	sha = strings.ToLower(sha)
	key := KeyFor(sha)

	if err := os.MkdirAll(filepath.Join(storageRoot, "tmp"), 0o755); err != nil {
		return "", 0, err
	}
	if err := os.MkdirAll(filepath.Join(storageRoot, filepath.Dir(key)), 0o755); err != nil {
		return "", 0, err
	}

	tmp, err := os.CreateTemp(filepath.Join(storageRoot, "tmp"), "*.part")
	if err != nil {
		return "", 0, err
	}
	tmpName := tmp.Name()
	size, copyErr := io.Copy(tmp, src)
	syncErr := tmp.Sync()
	closeErr := tmp.Close()
	if copyErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(tmpName)
		if copyErr != nil {
			return "", 0, copyErr
		}
		if syncErr != nil {
			return "", 0, syncErr
		}
		return "", 0, closeErr
	}

	if err := os.Rename(tmpName, filepath.Join(storageRoot, key)); err != nil {
		_ = os.Remove(tmpName)
		return "", 0, err
	}
	return key, size, nil
}

// resolve validates the key and returns its absolute path, ensuring it stays
// under storageRoot.
func resolve(storageRoot, storageKey string) (string, error) {
	slashKey := filepath.ToSlash(storageKey)
	lower := strings.ToLower(slashKey)
	if !keyRe.MatchString(lower) && !coverKeyRe.MatchString(lower) {
		return "", fmt.Errorf("invalid storage key")
	}
	cleanRoot, err := filepath.Abs(storageRoot)
	if err != nil {
		return "", err
	}
	p := filepath.Join(cleanRoot, filepath.FromSlash(slashKey))
	rel, err := filepath.Rel(cleanRoot, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("storage key escapes root")
	}
	return p, nil
}

// Open opens the stored file for reading after validating the key.
func Open(storageRoot, storageKey string) (*os.File, error) {
	p, err := resolve(storageRoot, storageKey)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

// Delete removes the stored file. Missing files are not an error.
func Delete(storageRoot, storageKey string) error {
	p, err := resolve(storageRoot, storageKey)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
