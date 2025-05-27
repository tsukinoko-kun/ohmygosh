//go:build !windows && !darwin
// +build !windows,!darwin

package data

import (
	"os"
	"path/filepath"
)

func noXdgDataHome() string {
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "ohmygosh")
}
