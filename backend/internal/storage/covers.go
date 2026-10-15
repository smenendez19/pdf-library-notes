package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CoverKeyFor returns the sharded cover key for a document sha256 hex digest.
// Covers share the PDF sha so re-uploads deduplicate the same cover bytes.
func CoverKeyFor(sha string) string {
	sha = strings.ToLower(sha)
	return filepath.Join("covers", sha[:2], sha[2:4], sha+".png")
}

// SaveCoverAtomic streams src to <root>/tmp/*.part, fsyncs it and renames it
// to <root>/covers/ab/cd/<sha256>.png. It returns the cover key.
func SaveCoverAtomic(storageRoot, sha string, src io.Reader) (string, int64, error) {
	if !shaHexRe.MatchString(strings.ToLower(sha)) {
		return "", 0, fmt.Errorf("invalid sha256")
	}
	key := CoverKeyFor(sha)

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

// OpenCover opens the stored cover for reading after validating the key.
func OpenCover(storageRoot, coverKey string) (*os.File, error) {
	p, err := resolve(storageRoot, coverKey)
	if err != nil {
		return nil, err
	}
	if !coverKeyRe.MatchString(strings.ToLower(filepath.ToSlash(coverKey))) {
		return nil, fmt.Errorf("invalid storage key")
	}
	return os.Open(p)
}

// DeleteCover removes the stored cover. Missing files are not an error.
func DeleteCover(storageRoot, coverKey string) error {
	p, err := resolve(storageRoot, coverKey)
	if err != nil {
		return err
	}
	if !coverKeyRe.MatchString(strings.ToLower(filepath.ToSlash(coverKey))) {
		return fmt.Errorf("invalid storage key")
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
