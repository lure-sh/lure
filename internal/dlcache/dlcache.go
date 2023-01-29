package dlcache

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"go.arsenm.dev/lure/internal/config"
)

// BasePath stores the base path to the download cache
var BasePath = filepath.Join(config.CacheDir, "dl")

// New creates a new directory with the given ID in the cache.
// If a directory with the same ID already exists,
// it will be deleted before creating a new one.
func New(id string) (string, error) {
	h, err := hashID(id)
	if err != nil {
		return "", err
	}
	itemPath := filepath.Join(BasePath, h)

	fi, err := os.Stat(itemPath)
	if err == nil || (fi != nil && !fi.IsDir()) {
		err = os.RemoveAll(itemPath)
		if err != nil {
			return "", err
		}
	}

	err = os.MkdirAll(itemPath, 0o755)
	if err != nil {
		return "", err
	}

	return itemPath, nil
}

// Get checks if an entry with the given ID
// already exists in the cache, and if so,
// returns the directory and true. If it
// does not exist, it returns an empty string
// and false.
func Get(id string) (string, bool) {
	h, err := hashID(id)
	if err != nil {
		return "", false
	}
	itemPath := filepath.Join(BasePath, h)

	_, err = os.Stat(itemPath)
	if err != nil {
		return "", false
	}

	return itemPath, true
}

// hashID hashes the input ID with SHA1
// and returns the hex string of the hashed
// ID.
func hashID(id string) (string, error) {
	h := sha1.New()
	_, err := io.WriteString(h, id)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
