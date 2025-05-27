//go:build !windows && !darwin
// +build !windows,!darwin

package config

import (
	"os"
	"path/filepath"
)

func defaultConfigDir() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "ohmygosh")
}
