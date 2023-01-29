package dlcache

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"go.arsenm.dev/lure/internal/config"
)

var BasePath = filepath.Join(config.CacheDir, "dl")

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

func hashID(id string) (string, error) {
	h := sha1.New()
	_, err := io.WriteString(h, id)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
