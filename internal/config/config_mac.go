//go:build darwin
// +build darwin

package config

import (
	"os"
	"path/filepath"
)

func defaultConfigDir() string {
	return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "ohmygosh")
}
