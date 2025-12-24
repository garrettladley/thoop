package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	dotConfig = ".config"
	appName   = "thoop"
	dbName    = "thoop.db"
)

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, dotConfig, appName), nil
}

func EnsureDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create %s directory: %w", appName, err)
	}
	return dir, nil
}

func DB() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, dbName), nil
}
