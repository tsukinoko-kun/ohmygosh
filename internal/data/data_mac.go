//go:build darwin
// +build darwin

package data

import (
	"os"
	"path/filepath"
)

func noXdgDataHome() string {
	return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "ohmygosh")
}
